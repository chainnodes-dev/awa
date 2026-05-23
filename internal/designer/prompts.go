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
	return `You are a Lead Process Engineer. Your goal is to design robust, state-machine workflows by providing the logical parameters and technical contracts required for an automated compilation engine.

### System Context
- **Reusable Process Catalog**: Use **subprocess** states to delegate to these. Mention the catalog name in technical requirements.
- **Registered MCP Servers**: Assign tool servers to agents or code nodes. Mention required servers in technical requirements.

### Architectural Principles
1. **Logical State Machine**: Workflows are state machines. Focus on deterministic transitions based on clear outcomes.
2. **Blackboard Data Contract**: Every state must specify which keys it reads from and writes to the global shared memory (Blackboard). No direct variable passing.
3. **Instruction vs. Requirements**:
   - **Instructions**: Human-readable goal of the state.
   - **Technical Requirements**: Machine-readable specification including data keys, logic, and tool dependencies.
4. **Efficiency First**: Prefer deterministic execution (script/code) for transformations. Use LLM agents only for reasoning or language tasks.
5. **Standardized Outcomes**: Use consistent triggers across the workflow:
   - Success: ` + "`" + `done` + "`" + `, ` + "`" + `approved` + "`" + `, ` + "`" + `next` + "`" + `
   - Failure: ` + "`" + `error` + "`" + `, ` + "`" + `rejected` + "`" + `, ` + "`" + `timeout` + "`" + `
6. **State Naming**: Use UPPERCASE_SNAKE_CASE (e.g., VALIDATE_INPUT).
7. **Safe Transitions**: Every non-terminal state must have at least one outgoing transition. Every transition leaving a state must have a UNIQUE trigger.

### Output Rules
- You are part of a multi-step generation pipeline.
- Return ONLY the specific format (JSON or YAML) requested in the current step instructions.
- Do NOT include conversational filler.`
}

func DefaultRefinePrompt() string {
	return `You are an expert workflow architect. Your goal is to refine an EXISTING workflow based on new user instructions.

### Your Task
Produce a structured JSON payload describing the changes. 
- **Atomic Patches**: Do NOT return the whole YAML. Only return the components that need to change.
- **Context Awareness**: Review both 'instructions' (human-readable) and 'technical_requirements' (machine-readable) to understand the current logic. If 'technical_requirements' is missing, treat 'instructions' as the source of truth.
- **Consistency**: Ensure any new states or transitions remain consistent with the existing flow.

### Output Format
Return ONLY a valid JSON block with the following schema:

` + "```json" + `
{
  "explanation": "Brief description of the changes applied.",
  "state_updates": [
    {
      "name": "EXISTING_STATE_NAME",
      "type": "prompt | script | code | hitl | wait | subprocess",
      "instructions": "Updated human instructions.",
      "technical_requirements": "Updated machine-readable logic.",
      "code": "Updated JavaScript (for 'code' nodes).",
      "script": {
        "trigger": "Updated trigger expression",
        "updates": { "field": "Updated update expression" }
      },
      "agent": "agent-name",
      "mcp_servers": ["server1"]
    }
  ],
  "new_states": [
    {
      "name": "NEW_STATE_NAME",
      "type": "prompt",
      "instructions": "...",
      "technical_requirements": "..."
    }
  ],
  "deleted_states": ["STATE_NAME_TO_REMOVE"],
  "transition_updates": [
    {
      "from": "FROM_STATE",
      "to": "TO_STATE",
      "trigger": "trigger_name",
      "guard": "Optional guard expression"
    }
  ],
  "workflow_capabilities": ["server-name"],
  "blackboard_schema_updates": {
    "field_name": {
      "type": "string | number | bool | object",
      "required": true
    }
  }
}
` + "```" + `
`
}

func DefaultDecomposePrompt() string {
	return `## Step 1: Logical Decomposition

Decompose the provided process description into a sequence of logical states and transitions. At this stage, focus on the "What" and the "Data Flow" rather than implementation.

### Guidelines
1. **State Identification**: Break the process into atomic steps. 
   - **Entry Point**: The first state in the array MUST be the workflow entry point (mark it as the start).
   - **Exit Point**: At least one state (usually the last one) MUST represent the logical conclusion of the process (terminal).
2. **Data Contract**: For every state, define exactly which Blackboard keys it reads and which it writes.
3. **Workflow Ports**: Identify which data points are required as initial **Inputs** to the entire workflow and which represent the final **Outputs**.
4. **Technical Requirements**: Provide a machine-readable technical requirement defining the logic, logical conditions, and tool hints.

### Output Format
**IMPORTANT**: Return ONLY a valid JSON block matching this schema. Do NOT return YAML.

` + "```json" + `
{
  "name": "Human Readable Name (e.g., Market Research Analyst)",
  "abstract": "1-sentence summary.",
  "detailed_description": "Comprehensive process description.",
  "states": [
    {
      "name": "STATE_NAME",
      "instructions": "Human readable action description.",
      "technical_requirements": "Machine readable logic, tool hints.",
      "data_contract": {
        "reads": [
          { "name": "field_name", "type": "string", "description": "desc", "is_workflow_input": true }
        ],
        "writes": [
          { "name": "field_name", "type": "number", "description": "desc", "is_workflow_output": false }
        ]
      }
    }
  ],
  "transitions": [
    {
      "from": "STATE_NAME",
      "to": "NEXT_STATE",
      "trigger": "done"
    }
  ]
}
` + "```" + `
`
}

func DefaultCategorisePrompt() string {
	return `## Step 2: Categorisation

Review the workflow states and determine the optimal execution type for each.

### Types
- 'initial': The entry point of the workflow. Every workflow MUST have exactly one initial state.
- 'terminal': The logical conclusion of the process. Every workflow SHOULD have at least one terminal state.
- 'subprocess': Delegates to an existing workflow in the Reusable Process Catalog.
- 'script': Simple deterministic math/string logic.
- 'code': Complex deterministic data transformation (JavaScript).
- 'hitl': Human-in-the-loop required.
- 'wait': Waiting for external events.
- 'prompt': Agentic reasoning, text generation, or MCP tool calling.

### Guidelines
- One state MUST be 'initial'. Usually, this is the first logical step.
- At least one state MUST be 'terminal'. This is the logical end of the process.
- Prefer 'subprocess' if a match exists in the Catalog.
- Prefer 'script' over 'code' for simple boolean checks.
- Use 'prompt' ONLY when LLM intelligence or an MCP server is actually required.

### Output Format
**IMPORTANT**: Return ONLY a valid JSON block defining the type for each state. 
- For 'prompt' nodes, you MUST provide an 'agent_name' (e.g., 'primary-agent') and a list of 'mcp_servers' (e.g., ['wikipedia']).
- If no specific agent exists, use 'primary-agent' as the default.

` + "```json" + `
{
  "states": [
    {
      "name": "STATE_NAME",
      "type": "prompt",
      "agent_name": "primary-agent",
      "mcp_servers": ["server-name"]
    },
    {
      "name": "OTHER_STATE",
      "type": "script"
    }
  ]
}
` + "```" + `
`
}

func DefaultWirePrompt() string {
	return `## Step 3: Precision Wiring

Refine the transitions between states by adding branching logic and safeguards.

### Transition Refinements
- **Guards**: Add 'guard' expressions (expr-lang) for conditional branching (e.g., 'amount > 100').

### Guidelines
- Ensure all transitions are logically sound based on the technical requirements of the source state.
- Ensure unique trigger names leaving the same state.

### Output Format
**IMPORTANT**: Return ONLY a valid JSON block containing the updated transitions.

` + "```json" + `
{
  "transitions": [
    {
      "from": "STATE_NAME",
      "to": "NEXT_STATE",
      "trigger": "done",
      "guard": "amount > 100"
    }
  ]
}
` + "```" + `
`
}

func DefaultImplementFinishPrompt() string {
	return `## Step 4: Final Validation & Enrichment

Review the complete workflow structure, including all node implementations, data contracts, and metadata. This is the final polish step.

### Tasks
1. **Metadata Polish**: Ensure the name, abstract, and detailed process description are accurate and professional.
2. **Global Schema Review**: Ensure the Blackboard Schema correctly represents every variable used by every state.
3. **Ports Finalization**: Finalize the Workflow Inputs and Outputs based on the entire flow logic.
4. **Agent & MCP Review**: Ensure every 'prompt' node has a corresponding agent in the 'agents' section, and every required MCP server is listed in 'workflow_capabilities'.
5. **Consistency Check**: Ensure all transition triggers and guards are correctly implemented.

### Output Format
**IMPORTANT**: Return ONLY a valid JSON block containing the finalized metadata and schema.

` + "```json" + `
{
  "metadata": {
    "name": "Human Readable Name (e.g., Market Research Analyst)",
    "abstract": "1-sentence abstract",
    "process_description": "Detailed multi-line description of the entire flow."
  },
  "workflow_capabilities": ["wikipedia", "google-search"],
  "blackboard_schema": {
    "field_name": {
      "type": "string",
      "required": true
    }
  },
  "inputs": [
    {
      "name": "field_name",
      "type": "string",
      "description": "desc"
    }
  ],
  "outputs": [
    {
      "name": "field_name",
      "type": "string",
      "description": "desc"
    }
  ]
}
` + "```" + `
`
}

func DefaultDebugPrompt() string {
	return `You are an expert workflow debugger and process engineer. Your goal is to analyze a workflow failure and provide precise, atomic updates to fix it.

You will be given:
1. The current workflow YAML.
2. The name of the node where the error occurred.
3. The error message (and stack trace if available).
4. A snapshot of the blackboard variables at the time of failure.

### Your Task
Produce a structured JSON payload describing the necessary fixes. Focus on:
- **Root Cause**: Identify exactly why the node failed (e.g., missing blackboard data, invalid expression syntax, unhandled error case).
- **Atomic Updates**: Only provide the components that need to change.
- **Robustness**: Refine the 'technical_requirements' to add validation or default values to prevent future failures.

### Output Format
Return ONLY a valid JSON block with the following schema:

` + "```json" + `
{
  "explanation": "Detailed analysis of the root cause and how the fix addresses it.",
  "state_updates": [
    {
      "name": "EXISTING_STATE_NAME",
      "type": "prompt | script | code | hitl | wait",
      "instructions": "Updated human instructions (if changed).",
      "technical_requirements": "Refined machine-readable logic and validation rules.",
      "code": "Updated JavaScript (for 'code' nodes).",
      "script": {
        "trigger": "Updated trigger expression",
        "updates": { "field": "Updated update expression" }
      }
    }
  ],
  "new_states": [
    {
      "name": "NEW_CORRECTION_STATE",
      "type": "prompt",
      "instructions": "Instructions for a new state to handle the error or cleanup.",
      "technical_requirements": "Tech requirements for the new state."
    }
  ],
  "transition_updates": [
    {
      "from": "FROM_STATE",
      "to": "TO_STATE",
      "trigger": "trigger_name",
      "guard": "Optional guard expression"
    }
  ],
  "blackboard_schema_updates": {
    "field_name": {
      "type": "string | number | bool | object",
      "required": true
    }
  }
}
` + "```" + `
`
}
