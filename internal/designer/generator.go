package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
	"gopkg.in/yaml.v3"
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
	// Provider is an optional LLM provider name to use for this request, overriding
	// the default provider.
	Provider string `json:"provider,omitempty"`
}

// GenerateResult holds the LLM-generated workflow.
type GenerateResult struct {
	// YAML is the raw workflow YAML string, ready to save or edit.
	YAML string `json:"yaml"`
	// Definition is the parsed and validated WorkflowDef.
	Definition *asmtypes.WorkflowDef `json:"definition"`
	// Interactions is the full conversation trace for this generation.
	Interactions []llm.Message `json:"interactions"`
}

// Generator uses an LLM to produce WorkflowDef YAML from a plain-language description.
type Generator struct {
	provider llm.Provider
	registry []MCPServerEntry
	wfStore  store.WorkflowStore
	prompts  *PromptManager
}

// NewGenerator creates a Generator backed by the given LLM provider.
// registry should come from LoadMCPRegistry().
// wfStore is used to inject the reusable process catalog into prompts.
func NewGenerator(provider llm.Provider, registry []MCPServerEntry, wfStore store.WorkflowStore) *Generator {
	return &Generator{
		provider: provider,
		registry: registry,
		wfStore:  wfStore,
		prompts:  NewPromptManager(),
	}
}

// WithProvider returns a shallow clone of the generator using a different LLM provider.
func (g *Generator) WithProvider(p llm.Provider) *Generator {
	return &Generator{
		provider: p,
		registry: g.registry,
		wfStore:  g.wfStore,
		prompts:  g.prompts,
	}
}

func (g *Generator) GetPrompt(id string) string {
	if g == nil || g.prompts == nil {
		return GetDefaultPrompt(id)
	}
	return g.prompts.Get(id, GetDefaultPrompt(id))
}

func (g *Generator) SetPrompt(id, content string) {
	if g == nil {
		return
	}
	if g.prompts == nil {
		g.prompts = NewPromptManager()
	}
	g.prompts.Set(id, content)
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
func (g *Generator) getFullRegistry(ctx context.Context) []MCPServerEntry {
	full := make([]MCPServerEntry, len(g.registry))
	copy(full, g.registry)

	if g.wfStore != nil {
		if s, ok := g.wfStore.(store.Store); ok {
			if dynamic, err := s.ListMCPServers(ctx); err == nil {
				for _, d := range dynamic {
					exists := false
					for _, existing := range g.registry {
						if existing.Name == d.Name {
							exists = true
							break
						}
					}
					if exists {
						continue
					}
					tools := make([]string, len(d.Tools))
					for i, t := range d.Tools {
						tools[i] = t.Name
					}
					entry := MCPServerEntry{
						Name:        d.Name,
						Description: d.Description,
						EnvVar:      strings.ToUpper(strings.ReplaceAll(d.Name, "-", "_")) + "_URL",
						URL:         d.URL,
						Command:     d.Command,
						Args:        d.Args,
						Tools:       tools,
					}
					full = append(full, entry)
				}
			}
		}
	}
	return full
}

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

	interaction := make([]llm.Message, 0)
	interaction = append(interaction, llm.Message{Role: "system", Content: system})
	interaction = append(interaction, messages...)

	// Call the LLM with the full conversation.
	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    g.provider.MaxOutputTokens(),
	})
	if err != nil {
		return &GenerateResult{Interactions: interaction}, fmt.Errorf("LLM call failed: %w", err)
	}

	interaction = append(interaction, llm.Message{Role: "assistant", Content: resp.Content})

	if isRefinement {
		return g.handleRefinementResponse(req, resp.Content, interaction)
	}

	// Legacy/Initial extraction for new workflows
	yamlStr, def, err := g.extractAndValidate(resp.Content)
	if err != nil {
		return &GenerateResult{
			YAML:         yamlStr,
			Definition:   def,
			Interactions: interaction,
		}, fmt.Errorf("generated workflow is invalid: %w", err)
	}

	// Ensure process description is persisted.
	if def != nil && def.Metadata.ProcessDescription == "" {
		def.Metadata.ProcessDescription = req.Description
	}

	return &GenerateResult{
		YAML:         yamlStr,
		Definition:   def,
		Interactions: interaction,
	}, nil
}

func (g *Generator) handleRefinementResponse(req GenerateRequest, content string, interactions []llm.Message) (*GenerateResult, error) {
	var result struct {
		Explanation string `json:"explanation"`
		StateUpdates []struct {
			Name                  string            `json:"name"`
			Type                  string            `json:"type"`
			Instructions          string            `json:"instructions"`
			TechnicalRequirements string            `json:"technical_requirements"`
			Code                  string            `json:"code"`
			Script                struct {
				Trigger string            `json:"trigger"`
				Updates map[string]string `json:"updates"`
			} `json:"script"`
			Agent      string   `json:"agent"`
			MCPServers []string `json:"mcp_servers"`
		} `json:"state_updates"`
		NewStates []struct {
			Name                  string `json:"name"`
			Type                  string `json:"type"`
			Instructions          string `json:"instructions"`
			TechnicalRequirements string `json:"technical_requirements"`
		} `json:"new_states"`
		DeletedStates     []string `json:"deleted_states"`
		TransitionUpdates []struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Trigger string `json:"trigger"`
			Guard   string `json:"guard"`
		} `json:"transition_updates"`
		WorkflowCapabilities    []string                    `json:"workflow_capabilities"`
		BlackboardSchemaUpdates map[string]asmtypes.FieldDef `json:"blackboard_schema_updates"`
	}

	jsonStr, err := extractJSONBlock(content)
	if err != nil {
		return &GenerateResult{Interactions: interactions}, fmt.Errorf("failed to extract JSON from AI response: %w", err)
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return &GenerateResult{Interactions: interactions}, fmt.Errorf("failed to parse AI refinement response: %w", err)
	}

	// Apply patches via WorkflowEditor
	def, _, err := asmtypes.LoadFromYAML([]byte(req.ExistingYAML))
	if err != nil {
		return &GenerateResult{Interactions: interactions}, fmt.Errorf("failed to load current workflow for refinement: %w", err)
	}
	editor := NewWorkflowEditor(def)

	// 1. Deleted States
	for _, name := range result.DeletedStates {
		editor.DeleteState(name)
	}

	// 2. New States
	for _, s := range result.NewStates {
		editor.AddState(s.Name, asmtypes.StateType(s.Type), s.Instructions, s.TechnicalRequirements)
	}

	// 3. State Updates
	for _, s := range result.StateUpdates {
		editor.UpdateState(s.Name, asmtypes.StateType(s.Type), s.Instructions, s.TechnicalRequirements)
		if s.Type == string(asmtypes.StateCode) && s.Code != "" {
			editor.SetStateCode(s.Name, s.Code)
		}
		if s.Type == string(asmtypes.StateScript) && s.Script.Trigger != "" {
			editor.SetStateScript(s.Name, s.Script.Trigger, s.Script.Updates)
		}
		if s.Agent != "" {
			editor.SetStateAgent(s.Name, s.Agent, s.MCPServers)
		}
	}

	// 4. Transitions
	for _, t := range result.TransitionUpdates {
		editor.AddTransition(t.From, t.To, t.Trigger, t.Guard)
	}

	// 5. Workflow Capabilities
	if len(result.WorkflowCapabilities) > 0 {
		editor.SetCapabilities(result.WorkflowCapabilities)
	}

	// 6. Blackboard Schema
	if len(result.BlackboardSchemaUpdates) > 0 {
		if def.Blackboard.Schema == nil {
			def.Blackboard.Schema = make(map[string]asmtypes.FieldDef)
		}
		for k, v := range result.BlackboardSchemaUpdates {
			def.Blackboard.Schema[k] = v
		}
	}

	newYAML, err := yaml.Marshal(def)
	if err != nil {
		return &GenerateResult{Interactions: interactions}, fmt.Errorf("failed to marshal refined workflow: %w", err)
	}

	return &GenerateResult{
		YAML:         string(newYAML),
		Definition:   def,
		Interactions: interactions,
	}, nil
}

// DebugWorkflowRequest is the input to the AI debugger.
type DebugWorkflowRequest struct {
	WorkflowYAML       string                 `json:"workflow_yaml"`
	FailedNodeName     string                 `json:"failed_node_name"`
	ErrorMessage       string                 `json:"error_message"`
	BlackboardSnapshot map[string]interface{} `json:"blackboard_snapshot,omitempty"`
}

// DebugWorkflowResult holds the fix proposed by the LLM.
type DebugWorkflowResult struct {
	YAML        string `json:"yaml"`
	Explanation string `json:"explanation"`
	// Interaction is the trace for this debug session.
	Interactions []llm.Message `json:"interactions"`
}

// DebugWorkflow calls the LLM to analyze a failure and propose a fix.
func (g *Generator) DebugWorkflow(ctx context.Context, req DebugWorkflowRequest) (*DebugWorkflowResult, error) {
	system := g.buildCoreRules()
	system += "\n\n" + g.GetPrompt(PromptIDDebug)

	userMsg := fmt.Sprintf("Workflow YAML:\n```yaml\n%s\n```\n\nFailed Node: %s\nError: %s\n", 
		req.WorkflowYAML, req.FailedNodeName, req.ErrorMessage)
	
	if len(req.BlackboardSnapshot) > 0 {
		bbJson, _ := json.MarshalIndent(req.BlackboardSnapshot, "", "  ")
		userMsg += fmt.Sprintf("\nBlackboard Snapshot:\n```json\n%s\n```\n", string(bbJson))
	}

	messages := []llm.Message{
		{Role: "user", Content: userMsg},
	}

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     messages,
		MaxTokens:    g.provider.MaxOutputTokens(),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM debug call failed: %w", err)
	}

	// The debugger prompt asks for structured JSON fixes.
	var result struct {
		Explanation string `json:"explanation"`
		StateUpdates []struct {
			Name                  string            `json:"name"`
			Type                  string            `json:"type"`
			Instructions          string            `json:"instructions"`
			TechnicalRequirements string            `json:"technical_requirements"`
			Code                  string            `json:"code"`
			Script                struct {
				Trigger string            `json:"trigger"`
				Updates map[string]string `json:"updates"`
			} `json:"script"`
		} `json:"state_updates"`
		NewStates []struct {
			Name                  string `json:"name"`
			Type                  string `json:"type"`
			Instructions          string `json:"instructions"`
			TechnicalRequirements string `json:"technical_requirements"`
		} `json:"new_states"`
		TransitionUpdates []struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Trigger string `json:"trigger"`
			Guard   string `json:"guard"`
		} `json:"transition_updates"`
		BlackboardSchemaUpdates map[string]asmtypes.FieldDef `json:"blackboard_schema_updates"`
	}

	jsonStr, err := extractJSONBlock(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from AI debug response: %w", err)
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI debug response: %w\nResponse: %s", err, resp.Content)
	}

	// Apply fixes via WorkflowEditor
	def, _, err := asmtypes.LoadFromYAML([]byte(req.WorkflowYAML))
	if err != nil {
		// Fallback if current YAML is broken, but it shouldn't be if it was running.
		def = &asmtypes.WorkflowDef{}
	}
	editor := NewWorkflowEditor(def)

	// 1. New States
	for _, s := range result.NewStates {
		editor.AddState(s.Name, asmtypes.StateType(s.Type), s.Instructions, s.TechnicalRequirements)
	}

	// 2. State Updates
	for _, s := range result.StateUpdates {
		editor.UpdateState(s.Name, asmtypes.StateType(s.Type), s.Instructions, s.TechnicalRequirements)
		if s.Type == string(asmtypes.StateCode) && s.Code != "" {
			editor.SetStateCode(s.Name, s.Code)
		}
		if s.Type == string(asmtypes.StateScript) && s.Script.Trigger != "" {
			editor.SetStateScript(s.Name, s.Script.Trigger, s.Script.Updates)
		}
	}

	// 3. Transitions
	for _, t := range result.TransitionUpdates {
		editor.AddTransition(t.From, t.To, t.Trigger, t.Guard)
	}

	// 4. Blackboard Schema
	if len(result.BlackboardSchemaUpdates) > 0 {
		if def.Blackboard.Schema == nil {
			def.Blackboard.Schema = make(map[string]asmtypes.FieldDef)
		}
		for k, v := range result.BlackboardSchemaUpdates {
			def.Blackboard.Schema[k] = v
		}
	}

	newYAML, err := yaml.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fixed workflow: %w", err)
	}

	interactions := []llm.Message{
		{Role: "user", Content: userMsg},
		{Role: "assistant", Content: resp.Content},
	}

	return &DebugWorkflowResult{
		YAML:         string(newYAML),
		Explanation:  result.Explanation,
		Interactions: interactions,
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

	// Tidy up: marshal back to YAML to ensure consistent indentation/format.
	tidied, err := yaml.Marshal(def)
	if err != nil {
		// If re-marshaling fails (rare), return the raw one as fallback.
		return yamlStr, def, nil
	}

	return string(tidied), def, nil
}

// buildSystemPrompt produces the system prompt that teaches the LLM the YAML schema
// and lists the available MCP servers.
func (g *Generator) buildCoreRules() string {
	return DefaultBasePrompt() + "\n\n" + DefaultSkillPrompt() + `

### YAML Schema Overview
` + "```yaml" + `
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: kebab-case-name
  version: "1"
  description: "1-sentence summary"
blackboard:
  schema:
    input_field: { type: string, required: true }
states:
  - name: START
    type: initial
  - name: CALC_LOGIC
    type: code
    code: { code: "return {trigger: 'done'};" }
  - name: HUMAN_REVIEW
    type: hitl
    assignee: "team-name"
    form_schema: { type: object, properties: { approved: { type: boolean } } }
transitions:
  - from: START
    to: CALC_LOGIC
    trigger: done
agents:
  - name: primary-agent
    config: { mcp_servers: ["server-name"] }
` + "```" + `

### Efficiency Rules
1. **Deterministic Logic**: Use 'code' (JavaScript) or 'script' (expr-lang) for math, string manipulation, and branching.
2. **LLM Usage**: Use 'prompt' (Agent) nodes ONLY for reasoning or MCP tool calling.
3. **Subprocesses**: Always check the Reusable Process Catalog before implementing logic.
`
}

func (g *Generator) buildSystemPrompt(requestedServers []string) string {
	return g.AssembleSystemPrompt(context.Background(), requestedServers, nil, "")
}

// AssembleSystemPrompt is the unified entry point for building generation system prompts.
func (g *Generator) AssembleSystemPrompt(ctx context.Context, requestedServers []string, registeredProcesses []*asmtypes.WorkflowDef, specificPrompt string) string {
	var sb strings.Builder

	// 1. Base persona and Mental Model
	sb.WriteString(DefaultBasePrompt())
	sb.WriteString("\n\n")
	sb.WriteString(DefaultSkillPrompt())
	sb.WriteString("\n\n")

	// 2. YAML Schema Overview
	sb.WriteString("### YAML Schema Overview\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: chainnodes/v1\n")
	sb.WriteString("kind: Workflow\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: kebab-case-name\n")
	sb.WriteString("  version: \"1\"\n")
	sb.WriteString("  description: \"1-sentence summary\"\n")
	sb.WriteString("blackboard:\n")
	sb.WriteString("  schema:\n")
	sb.WriteString("    input_field: { type: string, required: true }\n")
	sb.WriteString("states:\n")
	sb.WriteString("  - name: START\n")
	sb.WriteString("    type: initial\n")
	sb.WriteString("  - name: CALC_LOGIC\n")
	sb.WriteString("    type: code\n")
	sb.WriteString("    code: { code: \"return {trigger: 'done'};\" }\n")
	sb.WriteString("  - name: HUMAN_REVIEW\n")
	sb.WriteString("    type: hitl\n")
	sb.WriteString("    assignee: \"team-name\"\n")
	sb.WriteString("    form_schema: { type: object, properties: { approved: { type: boolean } } }\n")
	sb.WriteString("transitions:\n")
	sb.WriteString("  - from: START\n")
	sb.WriteString("    to: CALC_LOGIC\n")
	sb.WriteString("    trigger: done\n")
	sb.WriteString("agents:\n")
	sb.WriteString("  - name: primary-agent\n")
	sb.WriteString("    config: { mcp_servers: [\"server-name\"] }\n")
	sb.WriteString("```\n\n")

	// 3. Reusable Process Catalog
	if len(registeredProcesses) > 0 {
		sb.WriteString(g.buildProcessCatalogSection(registeredProcesses))
		sb.WriteString("\n\n")
	} else if g.wfStore != nil {
		processes, err := g.wfStore.ListDefinitions(ctx, store.DefinitionFilter{ReusableOnly: true})
		if err == nil && len(processes) > 0 {
			sb.WriteString(g.buildProcessCatalogSection(processes))
			sb.WriteString("\n\n")
		}
	}

	// 4. MCP Servers
	registry := g.getFullRegistry(ctx)
	sb.WriteString(g.buildMCPSection(requestedServers, registry))
	sb.WriteString("\n\n")

	// 5. Step Specific Instructions
	if specificPrompt != "" {
		sb.WriteString(specificPrompt)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// refinementAddendum returns additional instructions appended to the system prompt
// when the user is refining an existing workflow rather than generating from scratch.
func (g *Generator) refinementAddendum() string {
	return g.prompts.Get(PromptIDRefine, `
## Refinement Mode

You are modifying an existing workflow. The user will provide the current YAML and a change request.

Rules for refinement:
1. Preserve the workflow's name and overall structure unless the change explicitly requires restructuring.
2. Only modify the parts of the workflow that are relevant to the user's request.
3. Keep all existing states, transitions, agents, and blackboard fields that are not affected by the change.
4. If adding new states, integrate them naturally into the existing flow.
5. Ensure all transitions remain valid after the change — no dangling references.
6. Output the complete updated YAML (not just the diff).
`)
}

// extractYAMLBlock finds the first ```yaml ... ``` block in the LLM response,
// or falls back to the raw content if no code block is present.
func extractYAMLBlock(content string) (string, error) {
	content = sanitizeYAML(content)
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

// sanitizeYAML replaces tabs with spaces and removes common invisible nuisances
// that cause YAML unmarshal errors.
func sanitizeYAML(y string) string {
	// Replace tabs with 2 spaces
	y = strings.ReplaceAll(y, "\t", "  ")
	// Remove zero-width non-breaking spaces (\u00A0)
	y = strings.ReplaceAll(y, "\u00A0", " ")
	// Remove carriage returns
	y = strings.ReplaceAll(y, "\r", "")
	// Fix fancy Unicode dashes that sometimes creep in from LLM hallucinations
	y = strings.ReplaceAll(y, "—", "-")
	y = strings.ReplaceAll(y, "–", "-")
	return y
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
type MCPServerListItem struct {
	Name        string `json:"name"`
	EnvVar      string `json:"env_var,omitempty"`
	Description string `json:"description"`
}

// ToListItems converts registry entries to the public API format.
func ToListItems(entries []MCPServerEntry) []MCPServerListItem {
	out := make([]MCPServerListItem, len(entries))
	for i, e := range entries {
		out[i] = MCPServerListItem{Name: e.Name, EnvVar: e.EnvVar, Description: e.Description}
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
		"yaml":         r.YAML,
		"definition":   defMap,
		"interactions": r.Interactions,
	}, nil
}
