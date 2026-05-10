package designer

import (
	"sync"
)

// PromptManager handles the storage and retrieval of system prompts used by
// the AI workflow generator and skill analyser.
type PromptManager struct {
	mu      sync.RWMutex
	prompts map[string]string
}

// NewPromptManager creates a manager with default prompts.
func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		prompts: make(map[string]string),
	}
	return pm
}

// Get returns the prompt for the given ID, falling back to an empty string
// if no override exists. The caller should provide the hardcoded default.
func (pm *PromptManager) Get(id string, defaultPrompt string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if custom, ok := pm.prompts[id]; ok && custom != "" {
		return custom
	}
	return defaultPrompt
}

// Set overrides the prompt for the given ID.
func (pm *PromptManager) Set(id, content string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.prompts[id] = content
}

// GetAll returns all overrides.
func (pm *PromptManager) GetAll() map[string]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	out := make(map[string]string, len(pm.prompts))
	for k, v := range pm.prompts {
		out[k] = v
	}
	return out
}

const (
	PromptIDBase           = "workflow_generator_base"
	PromptIDSkill          = "skill_analyser_preamble"
	PromptIDRefine         = "workflow_refinement_addendum"
	PromptIDDecompose      = "workflow_decompose"
	PromptIDCategorise     = "workflow_categorise"
	PromptIDWire           = "workflow_wire"
	PromptIDImplFinish     = "workflow_implement_finish"
	PromptIDDebug          = "workflow_debugger"
)

// GetDefaultPrompt returns the hardcoded system prompt for a given ID.
func GetDefaultPrompt(id string) string {
	switch id {
	case PromptIDBase:
		return DefaultBasePrompt()
	case PromptIDSkill:
		return DefaultSkillPrompt()
	case PromptIDRefine:
		return DefaultRefinePrompt()
	case PromptIDDecompose:
		return DefaultDecomposePrompt()
	case PromptIDCategorise:
		return DefaultCategorisePrompt()
	case PromptIDWire:
		return DefaultWirePrompt()
	case PromptIDImplFinish:
		return DefaultImplementFinishPrompt()
	case PromptIDDebug:
		return DefaultDebugPrompt()
	default:
		return ""
	}
}

func DefaultBasePrompt() string {
	return "You are a world-class business process architect and workflow decomposition expert for the Phaxa (Phaxa) platform."
}

func DefaultSkillPrompt() string {
	return `You are a process decomposition engine for the Phaxa (Phaxa) platform.

Your task is to analyse a process description and convert it into an optimal Phaxa workflow.
A "process" is a workflow that may be reusable. Your job is to:

1. **Decompose** the process into atomic, sequential (or branching) steps.
2. **Classify** each step into the most efficient state type:
   - Use **subprocess** FIRST when a reusable sub-process in the catalog (see below) already
     implements the required capability. This is always preferred over reimplementing
     the same logic inline.
   - Use **intermediate** (LLM agent) ONLY when genuine reasoning, judgment, or language
     understanding is required that cannot be expressed deterministically and no
     reusable process covers it.
   - Use **code** (JavaScript) when the step requires computation, transformation, loops,
     or complex conditional logic.
   - Use **script** (expr-lang) for simple single-expression routing or field updates.
   - Use **hitl** when the step explicitly requires a human decision.
   - Use **wait** when the step must pause until external data arrives.
3. **Identify data dependencies** — what each step reads and writes on the blackboard.
4. **Honour the declared I/O contract** — all declared input ports MUST appear as
   blackboard fields; all declared output ports MUST be written before terminal states.
5. **Abstract Generation** — Populate "metadata.description" with a concise 1-sentence abstract. PRESERVE existing descriptions if present.
6. **Minimise LLM usage** — prefer code, script, or subprocess nodes where possible.

Ask yourself for each step:
"Does a reusable process already do this?" If yes, use subprocess.
"Can this be expressed without an LLM?" If yes, use code or script.
`
}

func DefaultRefinePrompt() string {
	return "\n\n## Refinement Rules\n\n- You are refining an EXISTING workflow based on new user instructions.\n- Maintain the existing structure unless changes are explicitly requested.\n- Preserve existing code, script, and instructions unless they need updating.\n- If the user asks for a new step, integrate it into the existing transitions flow.\n"
}

func DefaultDecomposePrompt() string {
	return `You are a world-class business process architect and workflow decomposition expert for the Phaxa (Phaxa) platform.

Your task is to decompose a natural-language process description into a sequence of logical steps (states).
At this stage, do NOT worry about node types or detailed logic. Focus on the logical structure.

## Output Format

Return a valid Phaxa Workflow YAML where EVERY state has ` + "`type: intermediate`" + `.

` + "```yaml" + `
apiVersion: asm.bauhaus28.com/v1
kind: Workflow
metadata:
  name: generated-workflow
  version: "1.0.0"
states:
  - name: STEP_1
    type: intermediate
    instructions: "Concise description of what happens here"
  - name: STEP_2
    type: intermediate
    instructions: "Concise description of what happens here"
` + "```" + `

## Rules

- Use uppercase snake_case for state names (e.g., VALIDATE_INVOICE)
- Keep state names concise but descriptive
- Every state must have 'type: intermediate' for now
- Include a 'metadata' block with 'name' and 'version'
- Output the complete YAML block
`
}

func DefaultCategorisePrompt() string {
	return `You are classifying workflow states into the correct Phaxa node types.

Given a skeleton workflow YAML (all states typed as 'intermediate') and the original process description,
assign the correct Phaxa type to every state.

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
| subprocess | Invoke a registered sub-workflow (process) |

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

## Node Structure Requirements

**CRITICAL**: Strictly adhere to the following object structures. Do NOT flatten these into strings.

### type: code
Must have a ` + "`" + `code` + "`" + ` object with a nested ` + "`code`" + ` string field.
` + "```yaml" + `
# CORRECT
- name: MY_CODE_NODE
  type: code
  code:
    code: "// TODO: implement" # Nested key

# WRONG (causes unmarshal error)
- name: MY_CODE_NODE
  type: code
  code: "// TODO: implement" # DO NOT DO THIS
` + "```" + `

## Rules

- Do NOT change state names, instructions, or transitions
- Do NOT change the type of 'initial' or 'terminal' states
- Do NOT generate a blackboard schema yet
- Output the complete updated YAML with all types correctly assigned
`
}

func DefaultWirePrompt() string {
	return `You are a workflow architect defining the control flow and data contract for an Phaxa workflow.

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

Define the shared data structure for the workflow. Distinguish between initial inputs and workflow results:

- **Mandatory Inputs**: For any field that MUST be provided by the user to start the workflow, set ` + "`required: true`" + `.
- **Workflow Results**: For any field that represents the final output or "product" of the workflow, set ` + "`is_output: true`" + `.
- **Internal State**: For fields used only during execution (e.g. temporary reasoning), do not set either flag.

**Critical YAML format rules for Blackboard Schema:**
The 'schema' field MUST be a YAML map (object), NOT a list (sequence):

` + "```yaml" + `
blackboard:
  schema:
    vendor_id:        # Input required to start
      type: string
      required: true
    invoice_total:    # Result produced by the workflow
      type: number
      is_output: true
` + "```" + `

## Rules

- Output the complete YAML block including the new transitions and schema.
- Preserve the existing 'metadata' and 'states' blocks.
`
}

func DefaultImplementFinishPrompt() string {
	return `You are finalising an Phaxa workflow that already has its structure, types, wiring, and blackboard schema.
Your task is to fill in the final implementation details for every node.

## Node Types and what to implement

1. **code nodes**: Replace the "// TODO: implement" placeholder with a real JavaScript body.
   - Use the ` + "`ctx`" + ` object to access variables (e.g. ` + "`ctx.blackboard.my_field`" + `).
   - Use ` + "`ctx.mcp.call_tool(\"server\", \"tool\", {args})`" + ` for tools.
   - Return an object with a ` + "`trigger`" + ` field (e.g. ` + "`return { trigger: 'done' };`" + `).
   - If using ` + "`ctx.mcp.call_tool`" + `, you MUST ` + "`await`" + ` it.

2. **hitl nodes**:
   - If the description implies a structured form, generate a 'form_schema' JSON Schema
   - Write clear instructions describing what the human must review or decide

3. **wait nodes**: Replace the placeholder condition with a real expr-lang expression,
   or set 'on_event' + 'on_event_match' for event-driven waiting.
   **CRITICAL**: When writing expr-lang expressions, access fields directly (e.g. ` + "`amount > 100`" + `). DO NOT use ` + "`ctx.`" + ` or ` + "`bb.`" + ` prefixes. DO NOT use ` + "`if / then / else`" + ` syntax.

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

func DefaultDebugPrompt() string {
	return `You are an expert Phaxa workflow debugger. Your task is to fix errors in a workflow definition.

You will be given:
1. The current workflow YAML.
2. The name of the node where the error occurred.
3. The error message (and stack trace if available).
4. A snapshot of the blackboard variables at the time of failure.

Your goal is to:
1. Analyze the cause of the failure.
2. Produce a corrected version of the workflow YAML.
3. Provide a brief, professional description of what was fixed and why.

## Rules for Fixing
- **Robustness**: If a node failed due to missing data, add validation or use default values in expressions.
- **Error Handling**: If the failure is expected/recoverable, consider adding an 'error' trigger and a transition to a correction node.
- **Node Specifics**: 
  - For 'script' nodes: check the syntax of the expr-lang expression.
  - For 'code' nodes: fix the JavaScript logic.
  - For 'intermediate' nodes: refine the instructions or the MCP server config.
- **Preserve Intent**: Do not change the overall goal of the workflow unless the error makes it impossible.
- **Schema Compliance**: The fixed YAML must adhere to the Phaxa schema (apiVersion: phaxa/v1).

## Output Format
Respond with a JSON object containing:
{
  "yaml": "The complete corrected workflow YAML",
  "explanation": "Brief summary of the fix"
}

Ensure the YAML is valid and correctly formatted within the JSON string.`
}
