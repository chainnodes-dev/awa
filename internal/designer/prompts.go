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
	PromptIDBase       = "workflow_generator_base"
	PromptIDSkill      = "skill_analyser_preamble"
	PromptIDRefine     = "workflow_refinement_addendum"
	PromptIDDecompose  = "workflow_decompose"
	PromptIDCategorise = "workflow_categorise"
	PromptIDWire       = "workflow_wire"
	PromptIDImplFinish = "workflow_implement_finish"
	PromptIDDebug      = "workflow_debugger"
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
	return "You are a world-class business process architect and expert in state-machine workflow design."
}

func DefaultSkillPrompt() string {
	return `You are a Lead Process Engineer. Design workflows for a state-machine automation engine.

### System Context
- **Reusable Process Catalog**: A list of existing workflows (subprocesses) with descriptions is provided below. Use **subprocess** states to delegate to these. Do not reimplement logic found here.
- **Registered MCP Servers**: A list of external tool servers with descriptions is provided below. Assign servers to agents via "config.mcp_servers".

### Architectural Rules
1. **State Machine**: Workflows are state machines, not DAGs. Cycles and loops are supported and encouraged for retries/iteration.
2. **Blackboard (BB)**: Global shared memory. States read inputs from and write updates to the blackboard. No direct variable passing between nodes. Information is stored as key/value pairs.
3. **State Structure**: Every state defines: Inputs (BB fields), Processing (LLM/Prompt, Code, Script, HITL, Subprocess), Outputs (BB updates), and Transitions.
4. **Triggers**: Every non-terminal state must emit at least one trigger string. This must match a "trigger" entry in the top-level "transitions" list.
5. **Fallbacks**: MCP tool usage must include fallback paths (e.g., error/failure triggers) to handle exceptions or missing data.
6. **Efficiency**: Use deterministic nodes (code, script) for data transformation. Use LLM nodes only for reasoning or language generation.
7. **Consistency**: Use consistent terminology and formatting throughout the workflow.
8. **State names**: State names must be uppercase snake_case (e.g., VALIDATE_INVOICE)
9. **Timeout**: Every state must have a timeout. Default to 5 minutes and transition to a timeout state node.
10. **Standardized Triggers**: Use consistent trigger names across the workflow for similar actions:
  - **Approvals/Human**: ` + "`" + `approved` + "`" + `, ` + "`" + `rejected` + "`" + `, ` + "`" + `needs_review` + "`" + `
  - **Linear Flow**: ` + "`" + `done` + "`" + `, ` + "`" + `next` + "`" + `, ` + "`" + `error` + "`" + `, ` + "`" + `cancel` + "`" + `
  - **Timeouts**: Always use ` + "`" + `timeout` + "`" + `
11. **Uniqueness**: Every transition leaving a specific state MUST have a unique trigger name. NEVER use the same trigger name for multiple outgoing transitions from the same state.`
}

func DefaultRefinePrompt() string {
	return "\n\n## Refinement Rules\n\n- You are refining an EXISTING workflow based on new user instructions.\n- Maintain the existing structure unless changes are explicitly requested.\n- Preserve existing code, script, and instructions unless they need updating.\n- If the user asks for a new step, integrate it into the existing transitions flow.\n"
}

func DefaultDecomposePrompt() string {
	return `## Step 1: Logical Decomposition

Decompose the provided process description into a sequence of logical states and transitions. At this stage, focus on the "What" and "How it flows" rather than implementation.

### Guidelines
1. **State Identification**: Break the process into atomic steps. Set ` + "`type: prompt`" + ` for EVERY state for now.
2. **Metadata & Contract**:
   - **Inputs**: Identify fields required on the Blackboard to start.
   - **Outputs**: Identify fields representing the final results.
   - **Description**: Provide a 1-sentence abstract of the goal.
3. **Transition Skeleton**: 
   - Define movement between states using architectural triggers (e.g., done, approved, error).
   - Every non-terminal state must have at least one transition.
4. **Subprocess/Tool Alignment**: 
   - If a state matches a subprocess in the Catalog, note it in the instructions.
   - If a state requires an MCP Tool, mention the required server name in the instructions.

### Output Format
**IMPORTANT**: Return the complete, valid Workflow YAML block. Ensure valid indentation and syntax.

` + "```yaml" + `
apiVersion: asm.bauhaus28.com/v1
kind: Workflow
metadata:
  name: uppercase_snake_case_name
  version: "1"
  description: "Summary of workflow goal."
  inputs:
    - name: field_name
      description: "Description"
  outputs:
    - name: field_name
      description: "Description"
states:
  - name: STATE_NAME
    type: prompt
    instructions: "Action description, mentioning subprocesses/MCP servers if needed."
transitions:
  - from: STATE_NAME
    to: NEXT_STATE
    trigger: done
` + "```" + `
`
}

func DefaultCategorisePrompt() string {
	return `## Step 2: Node Categorisation

Assign the most efficient and correct ` + "`type`" + ` to every state in the provided skeleton YAML. 

### Classification Strategy
1. **Subprocess First**: If a state matches an entry in the **Reusable Process Catalog**, you MUST use ` + "`type: subprocess`" + `.
2. **Deterministic Logic**: If a state performs data transformation, math, or logical branching without needing an LLM, use ` + "`type: code`" + ` (JavaScript) or ` + "`type: script`" + ` (expr-lang).
3. **Human/External**: Use ` + "`type: hitl`" + ` for human actions and ` + "`type: wait`" + ` for external synchronization.
4. **LLM/Agent (type: prompt)**: Use ` + "`type: prompt`" + ` ONLY for:
   - Non-deterministic reasoning or language generation.
   - Tasks requiring **MCP Tools** (these are handled by an agent within a prompt node).
5. **Efficiency**: Never use ` + "`prompt`" + ` for logic that can be expressed deterministically.

### Mandatory Placeholder Structures
For non-prompt types, you MUST include these minimal valid structures:
- **code**: ` + "`code: { code: \"// TODO: implement\" }`" + `
- **script**: ` + "`script: { trigger: \"'done'\" }`" + `
- **hitl**: ` + "`assignee: \"placeholder\"`" + `
- **wait**: ` + "`condition: \"true\"`" + `
- **subprocess**: ` + "`subprocess: { name: \"catalog-name\" }`" + `

### Rules
- **Preserve Structure**: Do NOT change state names, metadata, or the logical flow defined in Step 1.
- **Complete Output**: Return the **complete, valid Workflow YAML block**.
`
}

func DefaultWirePrompt() string {
	return `## Step 3: Workflow Wiring & Data Contract

Define the shared data structure (Blackboard Schema) and finalize the control flow details.

### 1. Blackboard Schema
Define every field required for the workflow's operation.
- **Contract Fulfillment**: Ensure all ` + "`metadata.inputs`" + ` and ` + "`metadata.outputs`" + ` from Step 1 are explicitly defined in the ` + "`blackboard.schema`" + `.
- **Attributes**: Use ` + "`type`" + `, ` + "`required: true`" + ` (for inputs), and ` + "`is_output: true`" + ` (for final results).

### 2. HITL Data Contract (form_schema)
For every ` + "`hitl`" + ` state, you MUST define a ` + "`form_schema`" + ` (JSON Schema) to collect required data from the user.
- **Dependency Check**: The schema must collect all fields that subsequent state logic or transitions depend on.
- **UI Requirements**: Use standard JSON Schema types (string, number, boolean, object) and ` + "`enum`" + ` for dropdowns.
- **Format**: 
  ` + "```yaml" + `
  form_schema:
    type: object
    properties:
      is_approved:
        type: boolean
        title: "Approve this request?"
      reason:
        type: string
        title: "Reason for decision"
    required: [is_approved]
  ` + "```" + `

### 3. Finalize Transitions
- **Guards**: Add ` + "`guard`" + ` expressions (expr-lang) for conditional branching (e.g., ` + "`amount > 100`" + `).
- **Timeouts**: Ensure ` + "`timeout: \"5m\"`" + ` and ` + "`on_timeout: timeout`" + ` are set on states that need limits, with matching transitions.
- **Trigger Integrity**: Ensure every state has transitions for all possible outcomes.

### Rules
- **Structure**: Transitions belong in the top-level ` + "`transitions:`" + ` block.
- **Preserve**: Do NOT change state names or node types from Step 2.
- **Complete Output**: Return the **complete, valid Workflow YAML block**.
`
}

func DefaultImplementFinishPrompt() string {
	return `## Step 4: Final Implementation

Fill in the functional logic for every state. This is the final step to make the workflow runnable.

### Implementation Rules by Node Type

1. **code (JavaScript)**:
   - **Data Access**: Use ` + "`ctx.blackboard.field_name`" + ` to read and write.
   - **MCP Tools**: Use ` + "`await ctx.mcp.call_tool(\"server\", \"tool\", {args})`" + `. You MUST use ` + "`await`" + `.
   - **Return**: Every code node MUST return a trigger object (e.g., ` + "`return { trigger: 'done' };`" + `).
   - **Format**: Use the literal block scalar ` + "`|`" + ` for multiline code to ensure YAML validity.

2. **script (expr-lang)**:
   - **Syntax**: Access fields directly (e.g., ` + "`amount > 100`" + `).
   - **Constraint**: DO NOT use ` + "`ctx.`" + ` or ` + "`bb.`" + ` prefixes.

3. **prompt (LLM Agent)**:
   - **Agent Connection**: Ensure the ` + "`agent`" + ` field in the state matches a name in the top-level ` + "`agents:`" + ` block.
   - **MCP Assignment**: If tools are required, add the server name to the agent's ` + "`config.mcp_servers`" + ` list.
   - **Instructions**: Provide detailed, goal-oriented instructions. Explicitly list which blackboard fields to read and which to update.

4. **wait / emit_event**:
   - Implement real ` + "`condition`" + ` logic or ` + "`event_name`" + ` details as required by the process description.

### Rules
- **Blueprint Adherence**: Do NOT change state names, node types, schema, or transitions from previous steps.
- **Complete Output**: Return the **complete, valid Workflow YAML block**.
`
}

func DefaultDebugPrompt() string {
	return `You are an expert workflow debugger. Fix errors in the workflow definition.

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
  - For 'prompt' nodes: refine the instructions or the MCP server config.
- **Preserve Intent**: Do not change the overall goal of the workflow unless the error makes it impossible.
- **Schema Compliance**: The fixed YAML must adhere to the engine schema (apiVersion: asm.bauhaus28.com/v1).

## Output Format
Respond with a JSON object containing:
{
  "yaml": "The complete corrected workflow YAML",
  "explanation": "Brief summary of the fix"
}

Ensure the YAML is valid and correctly formatted within the JSON string.`
}
