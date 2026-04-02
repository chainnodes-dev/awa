package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// GenerateRequest is the input to the AI workflow generator.
type GenerateRequest struct {
	// Description is a plain-language description of the business process to model.
	Description string `json:"description"`
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
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	system := g.buildSystemPrompt(req.MCPServers)
	userMsg := fmt.Sprintf("Design a workflow for the following process:\n\n%s", req.Description)

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages: []llm.Message{
			{Role: "user", Content: userMsg},
		},
		MaxTokens: 4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract the YAML block from the response.
	yamlStr, err := extractYAMLBlock(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("no valid YAML found in LLM response: %w", err)
	}

	// Validate by round-tripping through the loader.
	def, _, err := asmtypes.LoadFromYAML([]byte(yamlStr))
	if err != nil {
		return nil, fmt.Errorf("generated workflow is invalid: %w\n\nRaw output:\n%s", err, yamlStr)
	}

	return &GenerateResult{
		YAML:       yamlStr,
		Definition: def,
	}, nil
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
7. Use "script" for purely deterministic transformations (no LLM needed).
8. Use "intermediate" for steps that require LLM reasoning or tool use.
9. The "initial" state should capture the input and start processing.
10. Keep the blackboard schema minimal — only fields that flow between states.

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
