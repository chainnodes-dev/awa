package designer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/pkg/asmtypes"
	"gopkg.in/yaml.v3"
)

// PipelineRequest is the shared input for every step of the multi-step generator.
type PipelineRequest struct {
	// ProcessDescription is the original plain-language business process description.
	// Carried through every step so the LLM always has full context.
	ProcessDescription string `json:"process_description"`
	// CurrentYAML is the YAML produced by the previous step.
	// Empty only for Step 1 (Decompose).
	CurrentYAML string `json:"current_yaml,omitempty"`
	// Provider optionally overrides the default LLM provider.
	Provider string `json:"provider,omitempty"`
	// MCPServers optionally restricts which MCP servers are injected into the prompt.
	MCPServers []string `json:"mcp_servers,omitempty"`
}

// PipelineStepResult is the output of any pipeline step.
type PipelineStepResult struct {
	YAML         string                `json:"yaml"`
	Definition   *asmtypes.WorkflowDef `json:"definition"`
	Interactions []llm.Message         `json:"interactions"`
	// ValidationError is populated when YAML parses but fails semantic validation.
	// Steps 1–3 tolerate this because the workflow is not yet runnable by design.
	ValidationError string `json:"validation_error,omitempty"`
}

// MarshalPipelineStepResult serialises a PipelineStepResult to the standard API shape.
func MarshalPipelineStepResult(r *PipelineStepResult) (map[string]interface{}, error) {
	defBytes, err := json.Marshal(r.Definition)
	if err != nil {
		return nil, err
	}
	var defMap interface{}
	_ = json.Unmarshal(defBytes, &defMap)
	out := map[string]interface{}{
		"yaml":         r.YAML,
		"definition":   defMap,
		"interactions": r.Interactions,
	}
	if r.ValidationError != "" {
		out["validation_error"] = r.ValidationError
	}
	return out, nil
}

// codeJobResult holds the outcome of a parallel Codegen call.
type codeJobResult struct {
	stateName string
	result    *CodegenResult
	err       error
}

// scriptJobResult holds the outcome of a parallel Scaffold call.
type scriptJobResult struct {
	stateName string
	result    *ScaffoldResult
	err       error
}

// ── Step 1: Decompose ────────────────────────────────────────────────────────

// DecomposeWorkflow identifies the states/nodes of a process from a plain-language
// description.  All states are typed as 'intermediate' so the canvas can render
// them immediately.  No transitions, code, or expressions are generated yet.
func (g *Generator) DecomposeWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	system := g.prompts.Get(PromptIDDecompose, decomposeSystemPrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf("Process description:\n\n%s", req.ProcessDescription)
	return g.runPipelineStep(ctx, system, userMsg, true)
}

// ── Step 2: Categorise ───────────────────────────────────────────────────────

// CategoriseNodes assigns the correct ASM node type to every state produced by
// DecomposeWorkflow.  Placeholder implementations are injected for code/script
// nodes so the YAML remains loadable throughout the pipeline.
func (g *Generator) CategoriseNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the categorise step")
	}
	system := g.prompts.Get(PromptIDCategorise, categoriseSystemPrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nSkeleton workflow YAML (all nodes currently typed as 'intermediate'):\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)
	return g.runPipelineStep(ctx, system, userMsg, true)
}

// ── Step 3: Wire ─────────────────────────────────────────────────────────────

// WireWorkflow defines all transitions and the blackboard schema.  Node
// implementations remain as placeholders — this step produces the shared data
// contract that Step 4's code generators depend on.
func (g *Generator) WireWorkflow(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the wire step")
	}
	system := g.prompts.Get(PromptIDWire, wireSystemPrompt())
	system += g.buildMCPSection(req.MCPServers)
	system += pipelineOutputFormat()

	userMsg := fmt.Sprintf(
		"Process description:\n\n%s\n\nCategorised workflow YAML (types assigned, needs transitions and blackboard schema):\n\n```yaml\n%s\n```",
		req.ProcessDescription, req.CurrentYAML,
	)
	return g.runPipelineStep(ctx, system, userMsg, true)
}

// ── Step 4: Implement ────────────────────────────────────────────────────────

// ImplementNodes generates the final implementation for every node that needs it.
//
// Strategy:
//  1. Parse the wired YAML to extract BB schema and per-state trigger lists.
//  2. Fan out parallel Codegen() calls for code nodes and Scaffold() calls for
//     script nodes (both have dedicated specialised prompts).
//  3. Patch the generated code back into the YAML.
//  4. Run one final LLM pass to implement intermediate, hitl, wait, and
//     emit_event nodes and produce a complete, runnable workflow.
func (g *Generator) ImplementNodes(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	if req.CurrentYAML == "" {
		return nil, fmt.Errorf("current_yaml is required for the implement step")
	}

	def, _, parseErr := asmtypes.LoadFromYAML([]byte(req.CurrentYAML))
	if parseErr != nil || def == nil {
		return g.implementFallback(ctx, req)
	}

	bbSchema := buildBBSchemaMap(def)
	triggersPerState := buildTriggersPerState(def)

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		codeJobs   []codeJobResult
		scriptJobs []scriptJobResult
	)

	for _, s := range def.States {
		switch s.Type {
		case asmtypes.StateCode:
			if isPlaceholderCode(s) {
				wg.Add(1)
				go func(state asmtypes.StateDef) {
					defer wg.Done()
					res, err := g.Codegen(ctx, CodegenRequest{
						Instructions:  state.Instructions,
						StateName:     state.Name,
						ValidTriggers: triggersPerState[state.Name],
						BBSchema:      bbSchema,
					})
					mu.Lock()
					codeJobs = append(codeJobs, codeJobResult{state.Name, res, err})
					mu.Unlock()
				}(s)
			}
		case asmtypes.StateScript:
			if isPlaceholderScript(s) {
				wg.Add(1)
				go func(state asmtypes.StateDef) {
					defer wg.Done()
					res, err := g.Scaffold(ctx, ScaffoldRequest{
						StateName:   state.Name,
						Description: state.Instructions,
						Triggers:    triggersPerState[state.Name],
						BBSchema:    bbSchema,
					})
					mu.Lock()
					scriptJobs = append(scriptJobs, scriptJobResult{state.Name, res, err})
					mu.Unlock()
				}(s)
			}
		}
	}
	wg.Wait()

	patchedYAML, patchErr := patchImplementations(req.CurrentYAML, codeJobs, scriptJobs)
	if patchErr != nil {
		patchedYAML = req.CurrentYAML
	}

	finishSystem := g.prompts.Get(PromptIDImplFinish, implementFinishSystemPrompt())
	finishSystem += g.buildMCPSection(req.MCPServers)
	finishSystem += pipelineOutputFormat()
	finishMsg := fmt.Sprintf(
		"Original process description:\n\n%s\n\nWorkflow with code and script nodes already implemented (remaining nodes need finalising):\n\n```yaml\n%s\n```",
		req.ProcessDescription, patchedYAML,
	)
	return g.runPipelineStep(ctx, finishSystem, finishMsg, false)
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// runPipelineStep performs a single LLM call and extracts a YAML block from the response.
// When softValidate is true, validation failures populate ValidationError but are not fatal.
func (g *Generator) runPipelineStep(ctx context.Context, system, userMsg string, softValidate bool) (*PipelineStepResult, error) {
	interaction := []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: userMsg},
	}

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: system,
		Messages:     []llm.Message{{Role: "user", Content: userMsg}},
		MaxTokens:    g.provider.MaxOutputTokens(),
	})
	if err != nil {
		return &PipelineStepResult{Interactions: interaction}, fmt.Errorf("LLM call failed: %w", err)
	}

	interaction = append(interaction, llm.Message{Role: "assistant", Content: resp.Content})

	yamlStr, err := extractYAMLBlock(resp.Content)
	if err != nil {
		return &PipelineStepResult{Interactions: interaction}, fmt.Errorf("no YAML block in response: %w", err)
	}

	def, _, valErr := asmtypes.LoadFromYAML([]byte(yamlStr))
	validationError := ""
	if valErr != nil {
		if !softValidate {
			return &PipelineStepResult{YAML: yamlStr, Interactions: interaction},
				fmt.Errorf("YAML validation failed: %w", valErr)
		}
		validationError = valErr.Error()
	}

	return &PipelineStepResult{
		YAML:            yamlStr,
		Definition:      def,
		Interactions:    interaction,
		ValidationError: validationError,
	}, nil
}

// implementFallback runs a single refinement LLM call when the wired YAML cannot
// be structurally parsed for the parallel code-generation fanout.
func (g *Generator) implementFallback(ctx context.Context, req PipelineRequest) (*PipelineStepResult, error) {
	system := g.buildSystemPrompt(req.MCPServers) + g.refinementAddendum()
	userMsg := fmt.Sprintf(
		"Here is the current workflow YAML:\n\n```yaml\n%s\n```\n\nProcess context: %s\n\nPlease apply the following change:\n\nImplement all placeholder nodes. Replace any TODO code with working JavaScript. Replace placeholder script triggers with real expr-lang expressions. Finalise hitl assignees, wait conditions, and emit_event definitions. Produce a complete runnable workflow.",
		req.CurrentYAML, req.ProcessDescription,
	)
	return g.runPipelineStep(ctx, system, userMsg, false)
}

// buildMCPSection builds the MCP server context block for a pipeline step prompt.
func (g *Generator) buildMCPSection(requestedServers []string) string {
	relevantServers := g.registry
	if len(requestedServers) > 0 {
		relevantServers = filterServers(g.registry, requestedServers)
	}
	if len(relevantServers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Available MCP Servers\n\n")
	for _, s := range relevantServers {
		sb.WriteString(fmt.Sprintf("- **%s** (`{{ env.%s }}`): %s\n", s.Name, s.EnvVar, s.Description))
	}
	return sb.String()
}

// buildBBSchemaMap extracts a flat fieldName→type map from the workflow definition.
func buildBBSchemaMap(def *asmtypes.WorkflowDef) map[string]string {
	out := make(map[string]string, len(def.Blackboard.Schema))
	for name, fd := range def.Blackboard.Schema {
		out[name] = fd.Type
	}
	return out
}

// buildTriggersPerState returns outgoing trigger names indexed by state name.
func buildTriggersPerState(def *asmtypes.WorkflowDef) map[string][]string {
	out := make(map[string][]string)
	for _, t := range def.Transitions {
		out[t.From] = append(out[t.From], t.Trigger)
	}
	return out
}

// isPlaceholderCode returns true when a code node still has a placeholder body.
func isPlaceholderCode(s asmtypes.StateDef) bool {
	if s.Code == nil {
		return true
	}
	code := strings.TrimSpace(s.Code.Code)
	return code == "" ||
		strings.Contains(code, "// TODO: implement") ||
		code == "return {trigger: 'done'}" ||
		code == "return { trigger: 'done' }"
}

// isPlaceholderScript returns true when a script node still has a placeholder trigger.
func isPlaceholderScript(s asmtypes.StateDef) bool {
	if s.Script == nil {
		return true
	}
	t := strings.TrimSpace(s.Script.Trigger)
	return t == "" || t == `"done"` || t == "'done'"
}

// patchImplementations merges parallel code/script generation results into the YAML
// by round-tripping through a generic yaml map.
func patchImplementations(yamlStr string, codeJobs []codeJobResult, scriptJobs []scriptJobResult) (string, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return yamlStr, fmt.Errorf("parse YAML for patching: %w", err)
	}

	statesRaw, ok := doc["states"]
	if !ok {
		return yamlStr, nil
	}
	states, ok := statesRaw.([]interface{})
	if !ok {
		return yamlStr, nil
	}

	codeByName := make(map[string]*CodegenResult)
	for _, j := range codeJobs {
		if j.err == nil && j.result != nil {
			codeByName[j.stateName] = j.result
		}
	}
	scriptByName := make(map[string]*ScaffoldResult)
	for _, j := range scriptJobs {
		if j.err == nil && j.result != nil {
			scriptByName[j.stateName] = j.result
		}
	}

	for _, raw := range states {
		state, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := state["name"].(string)
		if res, ok := codeByName[name]; ok {
			// Ensure we always produce the nested {code: {code: "..."}} structure
			state["code"] = map[string]interface{}{"code": res.Code}
		}
		if res, ok := scriptByName[name]; ok {
			// Ensure we always produce the nested {script: {trigger: "...", updates: [...]}} structure
			script := map[string]interface{}{"trigger": res.Trigger}
			if len(res.Updates) > 0 {
				script["updates"] = res.Updates
			}
			state["script"] = script
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return yamlStr, fmt.Errorf("re-marshal patched YAML: %w", err)
	}
	return string(out), nil
}

func pipelineOutputFormat() string {
	return `
## Output Format

Respond with ONLY the workflow YAML, wrapped in a ` + "```yaml" + ` code block.
Do not include any explanation before or after the code block.
The YAML must be complete and valid according to the ASM schema.
`
}

// ── Default system prompts ────────────────────────────────────────────────────

func decomposeSystemPrompt() string {
	return `You are a workflow decomposition expert for the ASM (Agentic State Machine) platform.

Your task is to break a business process description into discrete workflow states (nodes).

## What to produce

For each state you identify:
- A concise SCREAMING_SNAKE_CASE name
- A clear 1–2 sentence description (set as the 'instructions' field) explaining what happens at this step
- Type: follow the rules below for node typing

### Node typing rules for this step
- The FIRST state (entry point) MUST have type: initial
- End states (success, failure, done, etc.) MUST have type: terminal
- ALL other states: set type: intermediate — specific typing happens in the next step

Additionally:
- Create a single 'primary-agent' entry in the agents block (as a YAML list item)
- **REQUIRED**: Add minimal transitions connecting states in a logical, linear sequence. Use trigger name 'next' for now.
- Set metadata.name, metadata.description, metadata.version: "1", apiVersion: asm/v1, kind: Workflow

## What NOT to produce

- Do NOT assign specific types (code, script, hitl, etc.) to middle states — only 'intermediate'
- Do NOT generate any JavaScript, expr-lang expressions, or form schemas
- Do NOT generate a blackboard schema yet
- Do NOT add complex guards or conditional branching yet

## Critical YAML format rules

The 'agents' field MUST be a YAML list, not a map:
` + "```yaml" + `
agents:
  - name: primary-agent   # CORRECT — list item with 'name' field
    model: ""
` + "```" + `
NOT:
` + "```yaml" + `
agents:
  primary-agent:          # WRONG — this is a map key, not a list item
    model: ""
` + "```" + `

Transitions also MUST be a list in the top-level ` + "`transitions:`" + ` block, never inside a state.

` + "```yaml" + `
transitions:
  - from: STATE_A      # CORRECT: Top-level list item
    to: STATE_B
    trigger: next
` + "```" + `

The 'states' field MUST be a YAML list (sequence), NOT a map:
` + "```yaml" + `
states:
  - name: START        # CORRECT: list item
    type: initial
` + "```" + `
WRONG:
` + "```yaml" + `
states:
  START:               # WRONG: map key
    type: initial
` + "```" + `

## Rules

1. Exactly one 'initial' state and at least one 'terminal' state
2. **CRITICAL**: Every 'instructions' field MUST be wrapped in double-quotes to prevent YAML parsing errors (e.g. ` + "`" + `instructions: "Task: verify invoice"` + "`" + `).
3. **Every non-terminal state MUST have at least one outgoing transition** (usually to the next logical step)
4. Be thorough — identify all meaningful steps in the process
`
}

func categoriseSystemPrompt() string {
	return `You are classifying workflow states into the correct ASM node types.

Given a skeleton workflow YAML (all states typed as 'intermediate') and the original process description,
assign the correct ASM type to every state.

## Available types and when to use them

| Type | Use when |
|------|----------|
| initial | Single entry point of the workflow |
| terminal | End states (success, failure, cancelled, etc.) |
| intermediate | Genuine LLM reasoning/decision-making that is non-deterministic |
| code | Deterministic computation — math, string manipulation, data transformation (JavaScript) |
| script | Simple one-line deterministic logic — conditions or mappings (expr-lang) |
| hitl | A human must take an action or approve something |
| wait | Pause until an external event arrives or a condition is met |
| emit_event | Fire a platform event to trigger other workflows |
| skill_call | Invoke a registered sub-workflow/skill |

## Classification rules

1. **Prioritize MCP Servers**: If a step requires external data retrieval or an external action (e.g., looking up a company in GLEIF, checking a stock price, sending an email) and a matching tool is available in the **MCP Servers** list, categorize that state as 'intermediate' so an agent can handle the tool interaction.
2. Prefer 'code' over 'intermediate' for internal deterministic computation — math, string manipulation, data transformation.
3. Prefer 'script' over 'code' for logic that fits in a single expr-lang expression.
4. Use 'intermediate' only when the step genuinely requires LLM reasoning or an MCP tool call.
5. 'initial' and 'terminal' should stay as determined by the structure.

## Placeholder implementations for non-intermediate types

For any type that requires a code body, inject a minimal placeholder so the YAML stays loadable:

- code nodes: ` + "`code: { code: \"// TODO: implement\\nreturn {trigger: 'done'}\" }`" + `
- script nodes: ` + "`script: { trigger: '\"done\"' }`" + `
- hitl nodes: add ` + "`assignee: \"team-placeholder\"`" + `
- wait nodes: add ` + "`condition: \"true\"` and `on_condition: \"done\"`" + `
- emit_event nodes: ` + "`emit_event: { event_name: placeholder_event, completion_trigger: done }`" + `

## Critical YAML format rules

The 'agents' field MUST be a YAML list, not a map:
` + "```yaml" + `
agents:
  - name: primary-agent   # CORRECT
    model: ""
` + "```" + `

The 'states' field MUST be a YAML list (sequence), NOT a map:
` + "```yaml" + `
states:
  - name: START        # CORRECT
    type: initial
` + "```" + `

## Rules

- Do NOT change state names, instructions, or transitions
- Do NOT change the type of 'initial' or 'terminal' states
- Do NOT generate a blackboard schema yet
- **Timeouts**: If a step implies a time limit (e.g., "must be approved within 2 days"), note that it will need a 'timeout' and 'on_timeout' field in the next step.
- Output the complete updated YAML with all types correctly assigned
`
}

func wireSystemPrompt() string {
	return `You are a workflow architect defining the control flow and data contract for an ASM workflow.

Given a categorised workflow YAML and the original process description, your task is to define:

## 1. Complete transitions block

Replace the placeholder transitions with the correct control flow.

**Critical YAML format rules for Transitions:**
Transitions MUST be a list of objects in the top-level ` + "`transitions:`" + ` block.

` + "```yaml" + `
transitions:
  - from: START        # Hyphen for list item
    to: NEXT_STATE
    trigger: approved
  - from: BRANCH_STATE
    to: SUCCESS
    trigger: ok
    guard: "amount > 100" # Use quotes for expressions
` + "```" + `

- Use meaningful trigger names (e.g. 'approved', 'rejected', 'error', 'timeout_occurred')
- **HITL Nodes**: MUST have both 'approved' and 'rejected' triggers unless specified otherwise.
- **Timeouts**: For any state requiring a time limit, set 'timeout: "<duration>"' (e.g. "1h") and 'on_timeout: timeout_occurred' on the state. Add a matching transition from that state with 'trigger: timeout_occurred'.
- Add guards (expr-lang conditions) where conditional branching is needed
- Add parallel fan-out with 'to_nodes' where steps can run concurrently
- Ensure every non-terminal state has at least one outgoing transition
- Ensure every state is reachable from the initial state

## 2. Blackboard schema

Define ONLY the fields required as initial inputs to start the workflow or those needing a specific default value.

**CRITICAL**: Do NOT include fields that are produced or updated during the runtime of the workflow (e.g., by LLM agents, scripts, or tool calls). The platform handles dynamic field creation automatically.

**Critical YAML format rules for Blackboard Schema:**
The 'schema' field MUST be a YAML map (object), NOT a list (sequence):

` + "```yaml" + `
blackboard:
  schema:
    vendor_name:      # CORRECT: This is a map key
      type: string
` + "```" + `

WRONG (causes unmarshal error):
` + "```yaml" + `
blackboard:
  schema:
    - name: vendor_name # WRONG: This is a list item
      type: string
` + "```" + `

Available field types: 'string', 'number', 'bool', 'object'

Do NOT add fields that are only used internally by LLM agents.

## 3. Identify MCP Tool Requirements

For every 'intermediate' node (agent) that was categorized to use an MCP server:
- Explicitly mention the name of the required MCP server and the specific tool in the 'instructions' field.
- The next step (Implementation) will use this to correctly wire the agent.

## 4. Update instructions for non-intermediate nodes

For each code/script/hitl/wait/emit_event node, update the 'instructions' field to include:
- A clear description of what the node does
- The exact list of valid trigger names it may fire (so Step 4 code generators have full context)

## Rules

- Keep placeholder code/script/hitl/wait/emit_event implementations as-is
- Keep all existing state names and types
- Transitions MUST be in the top-level 'transitions:' block, NEVER inside a state
- The 'states' field MUST be a YAML list (sequence), NOT a map.
- **CRITICAL**: ALL 'instructions' and 'guard' strings MUST be wrapped in double-quotes. This prevents colons in descriptions or ternary expressions from breaking the YAML.
`
}

func implementFinishSystemPrompt() string {
	return `You are finalising an ASM workflow that already has its structure, types, wiring, and
code/script node implementations defined.

Your task is to implement the remaining nodes and produce a clean, complete, runnable workflow.

## What needs implementing

1. **intermediate nodes**: 
   - Assign an ` + "`agent`" + ` by name (e.g., ` + "`agent: researcher`" + `). Ensure this agent is defined in the top-level ` + "`agents:`" + ` block.
   - Write a production-quality ` + "`instructions`" + ` prompt for the LLM agent. Include: clear task description, relevant blackboard fields, and the exact trigger names to fire.

2. **hitl nodes**:
   - Set a real 'assignee' (e.g. "finance-team", "approvers", "managers")
   - If the description implies a structured form, generate a 'form_schema' JSON Schema
   - Write clear instructions describing what the human must review or decide

3. **wait nodes**: Replace the placeholder condition with a real expr-lang expression,
   or set 'on_event' + 'on_event_match' for event-driven waiting.

4. **Timeouts**: Ensure every node with a 'timeout' field also has a valid 'on_timeout: timeout_occurred' field and a matching transition.

5. **emit_event nodes**: Set a meaningful 'event_name' (e.g. "invoice.approved") and
   include relevant 'payload_fields' from the blackboard schema.

6. **Agents block**: Ensure all agents have appropriate entries. 
   - Set 'model: ""' to designate the use of the platform default.
   - **MCP Servers**: For any agent requiring external tools, add the logical name of the required MCP server to its 'mcp_servers' list (e.g., ` + "`mcp_servers: [\"gleif-server\"]`" + `).
   - Use the "Available MCP Servers" list provided in the system context to find the correct names.

## Node Structure Requirements

**CRITICAL**: Strictly adhere to the following object structures. Do NOT flatten these into strings.

### type: code
Must have a ` + "`code`" + ` object with a nested ` + "`code`" + ` string field.
` + "```yaml" + `
# CORRECT
- name: MY_CODE_NODE
  type: code
  code:
    code: "return {trigger: 'done'};" # Nested key

# WRONG (causes unmarshal error)
- name: MY_CODE_NODE
  type: code
  code: "return {trigger: 'done'};" # DO NOT DO THIS
` + "```" + `

### type: script
Must have a ` + "`script`" + ` object with a nested ` + "`trigger`" + ` (and optionally ` + "`updates`" + `) field.
` + "```yaml" + `
# CORRECT
- name: MY_SCRIPT_NODE
  type: script
  script:
    trigger: "blackboard.amount > 100"

# WRONG
- name: MY_SCRIPT_NODE
  type: script
  script: "blackboard.amount > 100" # DO NOT DO THIS
` + "```" + `

## Rules

- Preserve all code/script node implementations exactly as-is
- Transitions MUST be in the top-level 'transitions:' block, NEVER inside a state
- The 'states' field MUST be a YAML list (sequence), NOT a map.
- **CRITICAL**: ALL 'instructions' prompts MUST be wrapped in double-quotes.
` + "```yaml" + `
states:
  - name: START        # CORRECT
    type: initial
` + "```" + `
`
}
