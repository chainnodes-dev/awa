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
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/tools"
	"github.com/asm-platform/asm/internal/tools/mcp"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// maxToolIterations caps the tool-calling loop to prevent runaway agents.
const maxToolIterations = 10

// Executor runs agent activities.
type Executor struct {
	llmRegistry  *llm.Registry
	bus          events.Bus
	store        store.Store
	defaultProv  string
	globalTools  *tools.Registry // tools available to every agent
}

func NewExecutor(registry *llm.Registry, bus events.Bus, s store.Store, defaultProvider string) *Executor {
	e := &Executor{
		llmRegistry: registry,
		bus:         bus,
		store:       s,
		defaultProv: defaultProvider,
		globalTools: tools.NewRegistry(),
	}
	e.RegisterTool(&tools.ReadUploadedFileTool{})
	return e
}

// RegisterTool adds a tool to the global registry, making it available to all agents.
func (e *Executor) RegisterTool(t tools.Tool) {
	e.globalTools.Register(t)
}

func (e *Executor) Execute(ctx context.Context, task orchestrator.AgentTask) (*asmtypes.AgentOutput, error) {
	agent := task.AgentDef
	start := time.Now()

	result := func(err error) {
		dur := time.Since(start).Seconds()
		metrics.AgentDurationSeconds.WithLabelValues(task.StateDef.Name, agent.Name, task.TenantID).Observe(dur)
		lbl := "success"
		if err != nil {
			lbl = "error"
		}
		metrics.AgentExecutionsTotal.WithLabelValues(task.StateDef.Name, agent.Name, lbl, task.TenantID).Inc()
	}

	// Load tenant secrets for tool/prompt use.
	var secrets map[string]string
	if tenant, err := e.store.GetTenant(ctx, task.TenantID); err == nil {
		secrets = tenant.Secrets
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

	// Derive valid triggers from the workflow definition so they can be injected
	// into the system prompt AND validated against the LLM's final response.
	outgoing := task.Def.TransitionsFrom(task.StateDef.Name)
	validTriggers := make([]string, 0, len(outgoing))
	for _, t := range outgoing {
		validTriggers = append(validTriggers, t.Trigger)
	}
	validTriggersStr := strings.Join(validTriggers, ", ")

	// Build system prompt.
	system, err := e.buildSystemPrompt(agent, task, validTriggers, validTriggersStr, secrets)
	if err != nil {
		err = fmt.Errorf("build system prompt: %w", err)
		result(err)
		return nil, err
	}

	// Load agent-specific MCP tools and merge with global tools.
	agentRegistry, err := e.loadAgentTools(ctx, agent, task)
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

	// Publish the full prompt once, before the first LLM call, so admins can
	// inspect exactly what was sent regardless of how many tool iterations follow.
	msgSlice := make([]interface{}, len(messages))
	for i, m := range messages {
		msgSlice[i] = m
	}
	_ = e.bus.Publish(ctx, events.New(events.AgentPrompt, events.AgentPromptPayload{
		RunID:     task.RunID,
		StateName: task.StateDef.Name,
		AgentName: agent.Name,
		System:    system,
		Messages:  msgSlice,
	}))

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
				err = fmt.Errorf("failed to parse agent JSON output: %w", parseErr)
				result(err)
				return nil, err
			}
			// Validate the trigger is non-empty and matches an outgoing transition.
			if output.Trigger == "" {
				err = fmt.Errorf("agent returned empty trigger; valid triggers for state %q are: [%s]",
					task.StateDef.Name, validTriggersStr)
				result(err)
				return nil, err
			}
			if len(validTriggers) > 0 && !containsTrigger(validTriggers, output.Trigger) {
				err = fmt.Errorf("agent returned unknown trigger %q; valid triggers for state %q are: [%s]",
					output.Trigger, task.StateDef.Name, validTriggersStr)
				result(err)
				return nil, err
			}
			// Publish the full response so admins can see exactly what the model returned.
			_ = e.bus.Publish(ctx, events.New(events.AgentResponse, events.AgentResponsePayload{
				RunID:     task.RunID,
				StateName: task.StateDef.Name,
				AgentName: agent.Name,
				Content:   resp.Content,
				Trigger:   output.Trigger,
				Reasoning: output.Reasoning,
			}))
			output.Content = resp.Content
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
func (e *Executor) loadAgentTools(ctx context.Context, agent asmtypes.AgentDef, task orchestrator.AgentTask) (*tools.Registry, error) {
	reg := tools.NewRegistry()

	serverList := agent.Config["mcp_servers"]
	if serverList == "" {
		return reg, nil
	}

	for _, rawRef := range strings.Split(serverList, ",") {
		serverRef := strings.TrimSpace(rawRef)
		if serverRef == "" {
			continue
		}

		serverURL := serverRef
		// Resolve logical name from DB if it doesn't look like a URL or an expanded env var.
		// Also handle the case where it might be a template placeholder that wasn't expanded.
		cleanRef := serverRef
		if strings.HasPrefix(cleanRef, "{{") && strings.HasSuffix(cleanRef, "}}") {
			// Extract name from {{ env.NAME_URL }}
			parts := strings.Fields(strings.Trim(cleanRef, "{} "))
			if len(parts) >= 2 && parts[0] == "env." {
				cleanRef = parts[1]
			} else if len(parts) >= 1 {
				cleanRef = parts[len(parts)-1]
			}
			// Strip common suffixes like _URL or _ENDPOINT
			cleanRef = strings.TrimSuffix(cleanRef, "_URL")
			cleanRef = strings.TrimSuffix(cleanRef, "_ENDPOINT")
		}

		if !strings.HasPrefix(serverRef, "http") && !strings.HasPrefix(serverRef, "/") && !strings.HasPrefix(serverRef, ".") {
			if dynamic, err := e.store.ListMCPServers(ctx); err == nil {
				for _, d := range dynamic {
					// Match by Name (case-insensitive) or by the cleaned-up Ref
					if strings.EqualFold(d.Name, serverRef) || strings.EqualFold(strings.ReplaceAll(d.Name, "-", "_"), cleanRef) {
						if d.Transport == "sse" {
							serverURL = d.URL
						} else {
							serverURL = d.URL
						}
						break
					}
				}
			}
		}

		if serverURL == "" {
			continue
		}

		// Create an observer that records every MCP call to the database.
		observer := func(method string, toolName string, input, output interface{}, dur time.Duration, err error) {
			if e.store == nil {
				return
			}
			log := &asmtypes.MCPAuditLog{
				RunID:      task.RunID,
				StateName:  task.StateDef.Name,
				AgentName:  agent.Name,
				ServerURL:  serverURL,
				Method:     method,
				ToolName:   toolName,
				Input:      input,
				Output:     output,
				DurationMs: int(dur.Milliseconds()),
				IsError:    err != nil,
			}
			if err != nil {
				log.ErrorMsg = err.Error()
			}
			_ = e.store.RecordMCPCall(context.Background(), log)
		}

		serverReg, err := mcp.LoadRegistry(ctx, serverURL, observer)
		if err != nil {
			return nil, fmt.Errorf("connect to MCP server %s: %w", serverURL, err)
		}
		for _, t := range serverReg.All() {
			reg.Register(t)
		}
	}
	return reg, nil
}

func (e *Executor) executeTool(ctx context.Context, reg *tools.Registry, name string, inputJSON json.RawMessage) (json.RawMessage, error) {
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
	if resp.Content != "" || resp.Reasoning != "" {
		blocks = append(blocks, llm.ContentBlock{
			Type:      "text",
			Text:      resp.Content,
			Reasoning: resp.Reasoning,
		})
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

func (e *Executor) buildSystemPrompt(agent asmtypes.AgentDef, task orchestrator.AgentTask, validTriggers []string, validTriggersStr string, secrets map[string]string) (string, error) {
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
		"SystemPrompt":     task.Def.Metadata.SystemPrompt,
		"AgentName":        agent.Name,
		"StateName":        task.StateDef.Name,
		"Instructions":     task.StateDef.Instructions,
		"Blackboard":       task.Blackboard,
		"Rules":            agent.Rules,
		"ValidTriggers":    validTriggers,
		"ValidTriggersStr": validTriggersStr,
		"Secrets":          secrets,
	})
	return buf.String(), nil
}

func parseAgentOutput(content string) (*asmtypes.AgentOutput, error) {
	start := -1
	depth := 0
	for i, ch := range content {
		switch ch {
		case '{':
			if start == -1 {
				start = i
			}
			depth++
		case '}':
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
const agentSystemPrompt = `{{if .SystemPrompt}}
{{.SystemPrompt}}
{{else}}
You are an autonomous agent in an Phaxa (ASM) platform.
{{end}}

Your current role: {{.AgentName}}
Current state: {{.StateName}}

{{if .Instructions}}
State instructions (TACTICAL TASK):
{{.Instructions}}
{{end}}

{{if .Blackboard._last_user_message}}
ACTIVE CHAT MESSAGE from human operator:
"{{.Blackboard._last_user_message}}"
Please respond to this message directly in your reasoning or as part of your blackboard updates if appropriate.
{{end}}

{{if .Rules}}
Business rules you MUST enforce:
{{range .Rules}}- {{.}}
{{end}}
{{end}}

{{if .Blackboard}}
You have access to tools. Use them as needed to complete your task, then produce your final response.
If the blackboard contains fields with a "file_id", these represent uploaded documents. You can use the "read_file" tool to inspect their contents (extract text from PDFs, CSVs, etc.).
{{end}}

{{if .ValidTriggers}}
VALID TRIGGERS — you MUST fire exactly one of these (no others are accepted):
{{range .ValidTriggers}}  - {{.}}
{{end}}
{{end}}
Your FINAL response (after any tool calls) MUST be a JSON object in exactly this format:
{
  "blackboard_updates": {
    "field_name": "value"
  },
  "trigger": "{{.ValidTriggersStr}}",
  "reasoning": "Brief explanation of your decision"
}

- "blackboard_updates": fields to write to the shared blackboard (may be empty {})
- "trigger": MUST be exactly one of [{{.ValidTriggersStr}}] — any other value will break the workflow
- "reasoning": brief explanation (for the audit trail)

Do not include any text outside the final JSON object.`

// containsTrigger reports whether name is present in the list.
func containsTrigger(list []string, name string) bool {
	for _, t := range list {
		if t == name {
			return true
		}
	}
	return false
}
