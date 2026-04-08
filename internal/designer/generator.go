package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ChatMessage represents one turn in the AI assistant conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // the text the user typed or the assistant replied
}

// GenerateRequest is the input to the AI workflow generator.
type GenerateRequest struct {
	// Description is the user's latest instruction — either a full process
	// description (new workflow) or a refinement request (existing workflow).
	Description string `json:"description"`
	// ExistingYAML is the current workflow YAML when refining an existing workflow.
	// When set, the LLM modifies this workflow instead of creating from scratch.
	ExistingYAML string `json:"existing_yaml,omitempty"`
	// WorkflowDescription is the verbal/business description from the workflow
	// metadata. Included so the LLM understands the workflow's purpose even
	// when the user's latest instruction is a small change like "add a timeout".
	WorkflowDescription string `json:"workflow_description,omitempty"`
	// History is the prior conversation between user and assistant in this
	// editing session. Sent so the LLM can maintain context across turns.
	// The latest user message (Description) is NOT included here — it's
	// appended automatically.
	History []ChatMessage `json:"history,omitempty"`
	// MCPServers is an optional list of logical MCP server names from the registry
	// that should be available to agents in the generated workflow.
	MCPServers []string `json:"mcp_servers,omitempty"`
}

// GenerateResult holds the LLM-generated workflow.
type GenerateResult struct {
	// YAML is the raw workflow YAML string, ready to save or edit.
	YAML string `json:"yaml"`
	// Definition is the parsed and validated WorkflowDef.
	Definition *asmtypes.WorkflowDef `json:"definition"`
}

// Generator uses an LLM to produce WorkflowDef YAML from a plain-language description.
type Generator struct {
	provider llm.Provider
	registry []MCPServerEntry
}

// NewGenerator creates a Generator backed by the given LLM provider.
// registry should come from LoadMCPRegistry().
func NewGenerator(provider llm.Provider, registry []MCPServerEntry) *Generator {
	return &Generator{provider: provider, registry: registry}
}

// Generate calls the LLM and returns a validated WorkflowDef.
//
// Context sent to the LLM (in order of priority):
//  1. System prompt — YAML schema, validation rules, output format (always)
//  2. Refinement rules — appended when ExistingYAML is set
//  3. Conversation history — prior user/assistant turns from the chat sidebar
//  4. Latest user message — includes the current YAML + workflow description
//     (if refining) or the process description (if creating from scratch)
//
// If the LLM returns invalid YAML, it is automatically retried once with the
// validation error appended. If the retry also fails, the error is returned.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	isRefinement := req.ExistingYAML != ""
	system := g.buildSystemPrompt(req.MCPServers)
	if isRefinement {
		system += g.refinementAddendum()
	}

	// Build message history: replay prior conversation turns so the LLM
	// maintains context across the editing session.
	messages := make([]llm.Message, 0, len(req.History)+1)
	for _, m := range req.History {
		messages = append(messages, llm.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Build the latest user message with full context.
	var userMsg string
	if isRefinement {
		userMsg = fmt.Sprintf("Here is the current workflow YAML:\n\n```yaml\n%s\n```\n", req.ExistingYAML)
		if req.WorkflowDescription != "" {
			userMsg += fmt.Sprintf("\nWorkflow purpose: %s\n", req.WorkflowDescription)
		}
		userMsg += fmt.Sprintf("\nPlease apply the following change:\n\n%s", req.Description)
	} else {
		userMsg = fmt.Sprintf("Design a workflow for the following process:\n\n%s", req.Description)
		if req.WorkflowDescription != "" {
			userMsg += fmt.Sprintf("\n\nAdditional context about the workflow: %s", req.WorkflowDescription)
		}
	}
	messages = append(messages, llm.Message{Role: "user", Content: userMsg})

	// Call the LLM with the full conversation.
	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract and validate. On failure, retry once with the error feedback.
	yamlStr, def, err := g.extractAndValidate(resp.Content)
	if err != nil {
		// Automatic retry: feed the error back to the LLM and ask for a fix.
		retryMessages := append(messages,
			llm.Message{Role: "assistant", Content: resp.Content},
			llm.Message{Role: "user", Content: fmt.Sprintf(
				"The YAML you produced is invalid:\n\n%s\n\nPlease fix the issues and return the corrected, complete YAML.",
				err.Error(),
			)},
		)
		resp2, err2 := g.provider.Complete(ctx, llm.CompletionRequest{
			SystemPrompt: system,
			Messages:     retryMessages,
			MaxTokens:    4096,
		})
		if err2 != nil {
			return nil, fmt.Errorf("LLM retry failed: %w (original error: %w)", err2, err)
		}
		yamlStr, def, err = g.extractAndValidate(resp2.Content)
		if err != nil {
			return nil, fmt.Errorf("generated workflow is still invalid after retry: %w", err)
		}
	}

	return &GenerateResult{
		YAML:       yamlStr,
		Definition: def,
	}, nil
}

// extractAndValidate extracts a YAML block from the LLM response and validates
// it by round-tripping through the workflow loader.
func (g *Generator) extractAndValidate(content string) (string, *asmtypes.WorkflowDef, error) {
	yamlStr, err := extractYAMLBlock(content)
	if err != nil {
		return "", nil, fmt.Errorf("no valid YAML block found in response: %w", err)
	}
	def, _, err := asmtypes.LoadFromYAML([]byte(yamlStr))
	if err != nil {
		return "", nil, fmt.Errorf("YAML validation failed: %w", err)
	}
	return yamlStr, def, nil
}

// buildSystemPrompt produces the system prompt that teaches the LLM the YAML schema
// and lists the available MCP servers.
func (g *Generator) buildSystemPrompt(requestedServers []string) string {
	var sb strings.Builder

	sb.WriteString(`You are an expert workflow designer for the ASM (Agentic State Machine) platform.
Your task is to produce a valid ASM workflow YAML definition based on the user's description.

## YAML Schema

The workflow YAML must conform to this structure:

` + "```yaml" + `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: workflow-name          # kebab-case, unique
  version: "1.0.0"
  description: "Optional description"

blackboard:
  schema:
    field_name:
      type: string             # string | number | bool | object
      required: true
      default: ""              # optional default value

states:
  - name: STATE_NAME           # SCREAMING_SNAKE_CASE recommended
    type: initial              # initial | intermediate | hitl | terminal | script
    instructions: |            # optional: plain-language context for the LLM agent
      Describe what the agent should do in this state.
    agent: agent-name          # omit for HITL and script states
    timeout: 30m               # optional
    on_timeout: trigger_name   # optional; trigger to fire when timeout expires

  - name: REVIEW_STATE
    type: hitl                 # human-in-the-loop: pauses until a human resolves it
    assignee: ""               # optional: specific user to notify

  - name: CALC_STATE
    type: script               # deterministic expr-lang evaluation; no LLM
    script:
      trigger: 'amount > 1000 ? "needs_review" : "auto_approve"'
      updates:
        vat: "amount * 0.2"

transitions:
  - from: SOURCE_STATE
    to:   TARGET_STATE
    trigger: trigger_name      # snake_case
    guard: "amount > 0"        # optional expr-lang boolean expression

agents:
  - name: agent-name
    model: claude-sonnet-4-6   # optional; uses platform default if omitted
    task_queue: ""             # optional; omit to use the default LLM worker queue
    tools: []                  # optional list of tool names
    rules:
      - "Business rule to enforce"
    config:
      mcp_servers: "{{ env.MCP_SERVER_URL }}"   # comma-separated env var references
      prompt: |                # optional inline system prompt override
        You are a specialist agent for...
` + "```" + `

## Rules

1. Every workflow must have exactly ONE state of type "initial".
2. Every workflow must have at least one state of type "terminal".
3. Every transition trigger must be a non-empty string in snake_case.
4. All state names referenced in transitions must exist in the states list.
5. All agent names referenced in states must exist in the agents list.
6. Use "hitl" for any step that requires a human decision or approval.
7. Use "script" ONLY for purely deterministic computations (arithmetic, comparisons, field mapping) where no reasoning is needed. Scripts use expr-lang expressions, not code.
8. Use "initial" or "intermediate" for ALL steps that require reasoning, validation, enrichment, classification, extraction, or any non-trivial logic. These are LLM-powered states.
9. Every state of type "initial" or "intermediate" MUST have an "agent" field referencing an agent defined in the "agents:" list. **CRITICAL: Reasoning states cannot be automated without an agent.**
10. The "agents:" list MUST contain a definition for every agent referenced by any state. At minimum include the agent name; model is optional (defaults to the platform default).
11. The agent's "instructions" field on the state tells the LLM agent what to do. Write clear, specific instructions.
12. **NEVER** omit the "agent" field for states that require analysis or automated processing. Empty agents are only valid if a state is explicitly intended to be resolved by an external API call (not a standard reasoning step).
13. The LLM agent decides which trigger to fire based on its analysis. Mention the available trigger names in the instructions so the agent knows its options.
14. The "initial" state should capture the input and start processing. **Assign an agent to the initial state to begin work immediately.**
15. Keep the blackboard schema minimal — only fields that flow between states.
16. Prefer intermediate (LLM-powered) states over script states. Only use script when the logic is a simple expression like "amount > threshold".

## How Triggers Work

- For "intermediate" and "initial" states: The LLM agent analyzes the blackboard, performs its task, updates blackboard fields, and returns a trigger name. The engine then follows the matching transition.
- For "script" states: The script's trigger expression is evaluated against the blackboard and must return a trigger name string.
- For "hitl" states: A human resolves the request with "approved" or "rejected" (or custom triggers).
- For "terminal" states: No trigger — the workflow ends.

Example: a state VALIDATE_DATA with instructions "Validate the invoice data. If valid emit 'valid', if there are issues emit 'invalid'" will have two transitions: VALIDATE_DATA→PROCESS (trigger: valid) and VALIDATE_DATA→REJECT (trigger: invalid).

`)

	// Add MCP server context if any servers were requested or all are listed.
	relevantServers := g.registry
	if len(requestedServers) > 0 {
		relevantServers = filterServers(g.registry, requestedServers)
	}

	if len(relevantServers) > 0 {
		sb.WriteString("## Available MCP Servers\n\n")
		sb.WriteString("The following MCP servers are registered and can be assigned to agents via `config.mcp_servers`.\n")
		sb.WriteString("Reference them as `{{ env.ENV_VAR_KEY }}` in the YAML. Choose servers whose capabilities match the agent's role.\n\n")
		for _, s := range relevantServers {
			sb.WriteString(fmt.Sprintf("- **%s** (`{{ env.%s }}`): %s\n", s.Name, s.EnvVar, s.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## Output Format

Respond with ONLY the workflow YAML, wrapped in a ` + "```yaml" + ` code block.
Do not include any explanation before or after the code block.
The YAML must be complete and valid according to the schema above.
`)

	return sb.String()
}

// refinementAddendum returns additional instructions appended to the system prompt
// when the user is refining an existing workflow rather than generating from scratch.
func (g *Generator) refinementAddendum() string {
	return `
## Refinement Mode

You are modifying an existing workflow. The user will provide the current YAML and a change request.

Rules for refinement:
1. Preserve the workflow's name and overall structure unless the change explicitly requires restructuring.
2. Only modify the parts of the workflow that are relevant to the user's request.
3. Keep all existing states, transitions, agents, and blackboard fields that are not affected by the change.
4. If adding new states, integrate them naturally into the existing flow.
5. Ensure all transitions remain valid after the change — no dangling references.
6. Output the complete updated YAML (not just the diff).
`
}

// extractYAMLBlock finds the first ```yaml ... ``` block in the LLM response,
// or falls back to the raw content if no code block is present.
func extractYAMLBlock(content string) (string, error) {
	// Try ```yaml block first.
	const startMarker = "```yaml"
	const endMarker = "```"

	start := strings.Index(content, startMarker)
	if start != -1 {
		inner := content[start+len(startMarker):]
		end := strings.Index(inner, endMarker)
		if end != -1 {
			return strings.TrimSpace(inner[:end]), nil
		}
	}

	// Try plain ``` block.
	const plainMarker = "```"
	start = strings.Index(content, plainMarker)
	if start != -1 {
		inner := content[start+len(plainMarker):]
		end := strings.Index(inner, plainMarker)
		if end != -1 {
			candidate := strings.TrimSpace(inner[:end])
			if strings.HasPrefix(candidate, "apiVersion:") {
				return candidate, nil
			}
		}
	}

	// Last resort: check if the raw content looks like YAML.
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "apiVersion:") {
		return trimmed, nil
	}

	return "", fmt.Errorf("response does not contain a YAML block")
}

// filterServers returns only the entries whose Name is in the requested list.
func filterServers(entries []MCPServerEntry, names []string) []MCPServerEntry {
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[n] = true
	}
	var out []MCPServerEntry
	for _, e := range entries {
		if wanted[e.Name] {
			out = append(out, e)
		}
	}
	return out
}

// MCPServerListItem is the public API representation of an MCP server entry.
// The env_var field is omitted intentionally.
type MCPServerListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToListItems converts registry entries to the public API format.
func ToListItems(entries []MCPServerEntry) []MCPServerListItem {
	out := make([]MCPServerListItem, len(entries))
	for i, e := range entries {
		out[i] = MCPServerListItem{Name: e.Name, Description: e.Description}
	}
	return out
}

// MarshalGenerateResult serialises the result to JSON-friendly map for the API response.
func MarshalGenerateResult(r *GenerateResult) (map[string]interface{}, error) {
	defBytes, err := json.Marshal(r.Definition)
	if err != nil {
		return nil, err
	}
	var defMap interface{}
	_ = json.Unmarshal(defBytes, &defMap)
	return map[string]interface{}{
		"yaml":       r.YAML,
		"definition": defMap,
	}, nil
}
