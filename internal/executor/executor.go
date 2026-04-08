package executor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/metrics"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/tools"
	"github.com/asm-platform/asm/internal/tools/mcp"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// maxToolIterations caps the tool-calling loop to prevent runaway agents.
const maxToolIterations = 10

// DirectExecutor runs agent activities in-process without Temporal.
type DirectExecutor struct {
	llmRegistry  *llm.Registry
	bus          events.Bus
	defaultProv  string
	globalTools  *tools.Registry // tools available to every agent
}

func NewDirectExecutor(registry *llm.Registry, bus events.Bus, defaultProvider string) *DirectExecutor {
	return &DirectExecutor{
		llmRegistry: registry,
		bus:         bus,
		defaultProv: defaultProvider,
		globalTools: tools.NewRegistry(),
	}
}

// RegisterTool adds a tool to the global registry, making it available to all agents.
func (e *DirectExecutor) RegisterTool(t tools.Tool) {
	e.globalTools.Register(t)
}

func (e *DirectExecutor) Execute(ctx context.Context, task orchestrator.AgentTask) (*asmtypes.AgentOutput, error) {
	agent := task.AgentDef
	start := time.Now()

	result := func(err error) {
		dur := time.Since(start).Seconds()
		metrics.AgentDurationSeconds.WithLabelValues(task.StateDef.Name, agent.Name).Observe(dur)
		lbl := "success"
		if err != nil {
			lbl = "error"
		}
		metrics.AgentExecutionsTotal.WithLabelValues(task.StateDef.Name, agent.Name, lbl).Inc()
	}

	// Resolve LLM provider: per-agent config → registry default → env default → ollama fallback.
	providerName := e.llmRegistry.Default()
	if providerName == "" {
		providerName = e.defaultProv
	}
	if agent.Config["provider"] != "" {
		providerName = agent.Config["provider"]
	}
	provider, err := e.llmRegistry.Get(providerName)
	if err != nil {
		// Fallback to Ollama before giving up.
		if ollamaProv, ollamaErr := e.llmRegistry.Get("ollama"); ollamaErr == nil {
			slog.Warn("LLM provider unavailable, falling back to ollama",
				"requested", providerName, "error", err)
			provider = ollamaProv
		} else {
			err = fmt.Errorf("LLM provider %q not available (ollama fallback also failed): %w", providerName, err)
			result(err)
			return nil, err
		}
	}

	// Build system prompt.
	system, err := e.buildSystemPrompt(agent, task)
	if err != nil {
		err = fmt.Errorf("build system prompt: %w", err)
		result(err)
		return nil, err
	}

	// Load agent-specific MCP tools and merge with global tools.
	agentRegistry, err := e.loadAgentTools(ctx, agent)
	if err != nil {
		slog.Warn("Failed to load MCP tools", "agent", agent.Name, "error", err)
		agentRegistry = tools.NewRegistry()
	}
	activeRegistry := e.globalTools.Merge(agentRegistry)

	// Convert registry to LLM tool definitions.
	llmTools := toLLMTools(activeRegistry.All())

	// Initial user message.
	bbJSON, _ := json.MarshalIndent(task.Blackboard, "", "  ")
	userMsg := fmt.Sprintf(
		"Current blackboard state:\n```json\n%s\n```\n\nExecute your role for state: %s",
		string(bbJSON), task.StateDef.Name,
	)

	messages := []llm.Message{{Role: "user", Content: userMsg}}

	// toolCallCache deduplicates within a single Execute call.
	// Key: "toolName|sha256(inputJSON)". When the LLM requests the same tool
	// with identical inputs more than once in a conversation turn, we return
	// the cached result instead of re-executing the tool.
	//
	// Note: this cache is local to one Execute invocation and does NOT survive
	// Temporal activity retries. Tool implementations must be idempotent so
	// that a retried activity produces the same observable outcome.
	toolCallCache := make(map[string]json.RawMessage)

	// ── Tool-calling loop ────────────────────────────────────────────────────
	for iteration := 0; iteration < maxToolIterations; iteration++ {
		req := llm.CompletionRequest{
			Model:        agent.Model,
			SystemPrompt: system,
			Messages:     messages,
			Tools:        llmTools,
			MaxTokens:    4096,
		}

		// Stream thinking tokens to the event bus.
		tokenCh := make(chan string, 128)
		go func() {
			for token := range tokenCh {
				_ = e.bus.Publish(ctx, events.New(events.AgentThinking, events.AgentThinkingPayload{
					RunID:     task.RunID,
					AgentName: agent.Name,
					Token:     token,
				}))
			}
		}()

		resp, err := provider.Stream(ctx, req, tokenCh)
		close(tokenCh)
		if err != nil {
			err = fmt.Errorf("LLM call failed (iteration %d): %w", iteration+1, err)
			result(err)
			return nil, err
		}

		// No tool calls → final response; extract AgentOutput JSON.
		if len(resp.ToolCalls) == 0 {
			output, parseErr := parseAgentOutput(resp.Content)
			if parseErr != nil {
				result(nil)
				return &asmtypes.AgentOutput{
					Reasoning: resp.Content,
					Error:     fmt.Sprintf("failed to parse structured output: %v", parseErr),
				}, nil
			}
			result(nil)
			return output, nil
		}

		// Execute every tool call the model requested.
		assistantBlocks := buildAssistantBlocks(resp)
		toolResultBlocks := make([]llm.ContentBlock, 0, len(resp.ToolCalls))

		for _, tc := range resp.ToolCalls {
			inputJSON, _ := json.Marshal(tc.Input)

			// Build a deduplication key: tool name + SHA-256 of the input.
			h := sha256.Sum256(inputJSON)
			cacheKey := fmt.Sprintf("%s|%x", tc.Name, h)

			_ = e.bus.Publish(ctx, events.New(events.AgentToolCall, events.AgentToolCallPayload{
				RunID:     task.RunID,
				AgentName: agent.Name,
				ToolName:  tc.Name,
				Input:     tc.Input,
			}))

			var resultJSON json.RawMessage
			if cached, hit := toolCallCache[cacheKey]; hit {
				slog.Debug("Tool call cache hit", "tool", tc.Name)
				resultJSON = cached
			} else {
				raw, toolErr := e.executeTool(ctx, activeRegistry, tc.Name, inputJSON)
				if toolErr != nil {
					slog.Warn("Tool error", "tool", tc.Name, "error", toolErr)
					raw, _ = json.Marshal(fmt.Sprintf("error: %v", toolErr))
				}
				resultJSON = json.RawMessage(raw)
				toolCallCache[cacheKey] = resultJSON
			}

			var resultVal interface{}
			_ = json.Unmarshal(resultJSON, &resultVal)

			_ = e.bus.Publish(ctx, events.New(events.AgentToolCall, events.AgentToolCallPayload{
				RunID:     task.RunID,
				AgentName: agent.Name,
				ToolName:  tc.Name,
				Input:     tc.Input,
				Output:    resultVal,
			}))

			toolResultBlocks = append(toolResultBlocks, llm.ContentBlock{
				Type:      "tool_result",
				ToolUseID: tc.ID,
				Output:    resultVal,
			})
		}

		// Append the assistant turn and tool results to the conversation.
		messages = append(messages,
			llm.Message{Role: "assistant", Content: assistantBlocks},
			llm.Message{Role: "user", Content: toolResultBlocks},
		)
	}

	err = fmt.Errorf("agent '%s' exceeded maximum tool iterations (%d)", agent.Name, maxToolIterations)
	result(err)
	return nil, err
}

// loadAgentTools connects to any MCP servers declared in the agent's config and
// returns a Registry of their tools. MCP server URLs must be environment variable
// values — they arrive here already expanded by the YAML loader.
//
// Config key: "mcp_servers" — comma-separated list of server URLs.
// Example workflow YAML:
//
//	agents:
//	  - name: validator
//	    config:
//	      mcp_servers: "{{ env.VALIDATE_MCP_URL }}"
func (e *DirectExecutor) loadAgentTools(ctx context.Context, agent asmtypes.AgentDef) (*tools.Registry, error) {
	reg := tools.NewRegistry()

	serverList := agent.Config["mcp_servers"]
	if serverList == "" {
		return reg, nil
	}

	for _, rawURL := range strings.Split(serverList, ",") {
		serverURL := strings.TrimSpace(rawURL)
		if serverURL == "" {
			continue
		}
		serverReg, err := mcp.LoadRegistry(ctx, serverURL)
		if err != nil {
			return nil, fmt.Errorf("connect to MCP server %s: %w", serverURL, err)
		}
		for _, t := range serverReg.All() {
			reg.Register(t)
		}
	}
	return reg, nil
}

func (e *DirectExecutor) executeTool(ctx context.Context, reg *tools.Registry, name string, inputJSON json.RawMessage) (json.RawMessage, error) {
	t, err := reg.Get(name)
	if err != nil {
		return nil, fmt.Errorf("tool '%s' not available: %w", name, err)
	}
	return t.Execute(ctx, inputJSON)
}

// buildAssistantBlocks converts a CompletionResponse into the content-block
// array used when appending the assistant turn to the conversation history.
func buildAssistantBlocks(resp *llm.CompletionResponse) []llm.ContentBlock {
	var blocks []llm.ContentBlock
	if resp.Content != "" {
		blocks = append(blocks, llm.ContentBlock{Type: "text", Text: resp.Content})
	}
	for _, tc := range resp.ToolCalls {
		blocks = append(blocks, llm.ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Name,
			Input: tc.Input,
		})
	}
	return blocks
}

// toLLMTools converts Tool instances to the llm.Tool format sent to the LLM.
func toLLMTools(ts []tools.Tool) []llm.Tool {
	out := make([]llm.Tool, len(ts))
	for i, t := range ts {
		out[i] = llm.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return out
}

func (e *DirectExecutor) buildSystemPrompt(agent asmtypes.AgentDef, task orchestrator.AgentTask) (string, error) {
	basePrompt := agentSystemPrompt

	// File-based prompt template takes priority (complex, multi-paragraph prompts).
	if agent.PromptTemplate != "" {
		data, err := os.ReadFile(agent.PromptTemplate)
		if err == nil {
			basePrompt = string(data)
		}
	} else if agent.Config["prompt"] != "" {
		// Inline prompt defined in the workflow YAML (agent.config.prompt).
		basePrompt = agent.Config["prompt"]
	}

	tmpl, err := template.New("system").Parse(basePrompt)
	if err != nil {
		return basePrompt, nil
	}

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, map[string]interface{}{
		"AgentName":    agent.Name,
		"StateName":    task.StateDef.Name,
		"Instructions": task.StateDef.Instructions,
		"Blackboard":   task.Blackboard,
		"Rules":        agent.Rules,
	})
	return buf.String(), nil
}

func parseAgentOutput(content string) (*asmtypes.AgentOutput, error) {
	start := -1
	depth := 0
	for i, ch := range content {
		if ch == '{' {
			if start == -1 {
				start = i
			}
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 && start != -1 {
				var output asmtypes.AgentOutput
				if err := json.Unmarshal([]byte(content[start:i+1]), &output); err == nil {
					return &output, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid JSON found in agent response")
}

// agentSystemPrompt is the default system prompt template.
// When tools are available the model should use them before producing the final
// JSON output. The final response MUST be a JSON object — nothing else.
const agentSystemPrompt = `You are an autonomous agent in an Agentic State Machine (ASM) platform.

Your current role: {{.AgentName}}
Current state: {{.StateName}}

{{if .Instructions}}
State instructions:
{{.Instructions}}
{{end}}

{{if .Rules}}
Business rules you MUST enforce:
{{range .Rules}}- {{.}}
{{end}}
{{end}}

{{if .Blackboard}}
You have access to tools. Use them as needed to complete your task, then produce your final response.
{{end}}

Your FINAL response (after any tool calls) MUST be a JSON object in exactly this format:
{
  "blackboard_updates": {
    "field_name": "value"
  },
  "trigger": "trigger_name",
  "reasoning": "Brief explanation of your decision"
}

- "blackboard_updates": fields to write to the shared blackboard (may be empty {})
- "trigger": the exact trigger name to fire to advance the workflow
- "reasoning": brief explanation (for the audit trail)

Do not include any text outside the final JSON object.`
