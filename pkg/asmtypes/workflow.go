package asmtypes

import (
	"time"
)

// WorkflowDef is the root of a workflow YAML manifest.
type WorkflowDef struct {
	APIVersion   string           `yaml:"apiVersion"   json:"apiVersion"`
	Kind         string           `yaml:"kind"         json:"kind"`
	Metadata     WorkflowMeta     `yaml:"metadata"     json:"metadata"`
	Triggers     []TriggerDef     `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Inputs       []PortDef        `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs      []PortDef        `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Capabilities []CapabilityDecl `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Blackboard   BlackboardDef    `yaml:"blackboard"   json:"blackboard"`
	States       []StateDef       `yaml:"states"       json:"states"`
	Transitions  []Transition     `yaml:"transitions"  json:"transitions"`
	Agents       []AgentDef       `yaml:"agents"       json:"agents"`
}

type WorkflowMeta struct {
	Name          string            `yaml:"name"                  json:"name"`
	Version       string            `yaml:"version"               json:"version"`
	VersionNumber int               `yaml:"-"                     json:"version_number,omitempty"`
	Description   string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"      json:"labels,omitempty"`
	// Schedule is an optional cron expression that auto-starts the workflow on a
	// fixed cadence.  Uses 6-field format with seconds first (robfig/cron v3):
	//   "0 0 9 * * *"  — daily at 09:00:00
	//   "0 */15 * * * *" — every 15 minutes
	// Leave empty to disable.
	Schedule           string `yaml:"schedule,omitempty"            json:"schedule,omitempty"`
	SystemPrompt       string `yaml:"system_prompt,omitempty"       json:"system_prompt,omitempty"`
	ProcessDescription string `yaml:"process_description,omitempty" json:"process_description,omitempty"`
	// Reusable indicates if this workflow can be invoked as a sub-process (Skill) by other workflows.
	Reusable bool `yaml:"reusable,omitempty" json:"reusable,omitempty"`
}

// TriggerDef defines an external event source that automatically starts a workflow run.
type TriggerDef struct {
	Name   string                 `yaml:"name"   json:"name"`
	Type   string                 `yaml:"type"   json:"type"` // e.g. "webhook", "telegram", "cron"
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// BlackboardDef defines the typed schema of the shared blackboard.
type BlackboardDef struct {
	Schema map[string]FieldDef `yaml:"schema" json:"schema"`
}

type FieldDef struct {
	Type     string      `yaml:"type"               json:"type"` // string | number | bool | object | file
	Required bool        `yaml:"required,omitempty" json:"required"`
	IsOutput bool        `yaml:"is_output,omitempty" json:"is_output"`
	Default  interface{} `yaml:"default,omitempty"  json:"default,omitempty"`
}

// PortDef describes a single typed input or output port of a reusable workflow.
type PortDef struct {
	Name        string `yaml:"name"                  json:"name"`
	Type        string `yaml:"type"                  json:"type"` // string | number | bool | object
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"    json:"required,omitempty"`
}

// CapabilityDecl declares a dependency on a named MCP server.
type CapabilityDecl struct {
	MCPServer string `yaml:"mcp_server" json:"mcp_server"`
}

// StateType classifies the role of a state in the machine.
type StateType string

const (
	StateInitial      StateType = "initial"
	StatePrompt       StateType = "prompt"
	StateHITL         StateType = "hitl"      // Human-in-the-loop pause
	StateTerminal     StateType = "terminal"
	StateScript       StateType = "script"     // Deterministic expr-lang evaluation; no LLM
	StateWait         StateType = "wait"       // Pauses until a condition is met or timeout
	StateCode         StateType = "code"       // User-defined JavaScript executed via goja
	StateSubProcess   StateType = "subprocess" // Invokes another process as a child workflow
	StateEmitEvent    StateType = "emit_event" // Fires a named platform event to the bus
	StateTelegramOutput StateType = "telegram_output"
	StateDiscordOutput  StateType = "discord_output"
)

type StateDef struct {
	Name                  string     `yaml:"name"                   json:"name"`
	Type                  StateType  `yaml:"type"                   json:"type"`
	Instructions          string     `yaml:"instructions,omitempty" json:"instructions,omitempty"` // Plain-language context for the LLM agent
	TechnicalRequirements string     `yaml:"technical_requirements,omitempty" json:"technical_requirements,omitempty"` // Machine-readable logic
	Agent                 string     `yaml:"agent,omitempty"        json:"agent,omitempty"`
	Script       *ScriptDef `yaml:"script,omitempty"       json:"script,omitempty"`        // Set when type == "script"
	Timeout      string     `yaml:"timeout,omitempty"      json:"timeout,omitempty"`
	OnTimeout    string     `yaml:"on_timeout,omitempty"   json:"on_timeout,omitempty"`
	OnEnter      []string   `yaml:"on_enter,omitempty"     json:"on_enter,omitempty"`
	Assignee     string     `yaml:"assignee,omitempty"     json:"assignee,omitempty"` // HITL
	Compensation string     `yaml:"compensation,omitempty" json:"compensation,omitempty"`
	// Condition is an expr-lang expression used by StateWait nodes.
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
	// OnCondition is the trigger fired when Condition evaluates to true.
	OnCondition string `yaml:"on_condition,omitempty" json:"on_condition,omitempty"`
	// Code is set when type == "code" (JavaScript executed via goja).
	Code *CodeDef `yaml:"code,omitempty" json:"code,omitempty"`
	// SubProcess is set when type == "subprocess".
	SubProcess *SubProcessDef `yaml:"subprocess,omitempty" json:"subprocess,omitempty"`
	// EmitEvent is set when type == "emit_event".
	EmitEvent *EmitEventDef `yaml:"emit_event,omitempty" json:"emit_event,omitempty"`
	// TelegramOutput is set when type == "telegram_output".
	TelegramOutput *TelegramOutputDef `yaml:"telegram_output,omitempty" json:"telegram_output,omitempty"`
	// DiscordOutput is set when type == "discord_output".
	DiscordOutput *DiscordOutputDef `yaml:"discord_output,omitempty" json:"discord_output,omitempty"`
	// OnEvent names a platform event that will wake this wait state.
	// Only effective when type == "wait". When the named event fires in
	// the same tenant, the run receives the on_event_match trigger.
	OnEvent      string `yaml:"on_event,omitempty"       json:"on_event,omitempty"`
	// OnEventMatch is the trigger fired when OnEvent arrives (default: "event_received").
	OnEventMatch string `yaml:"on_event_match,omitempty" json:"on_event_match,omitempty"`
	// Layout hint for the designer canvas — not required at runtime.
	Position *Position `yaml:"position,omitempty" json:"position,omitempty"`
	// FormSchema is an optional JSON Schema used to render a custom input form
	// in the dashboard when type == "hitl".
	FormSchema map[string]interface{} `yaml:"form_schema,omitempty" json:"form_schema,omitempty"`

	// Shorthand transitions (synthesized into formal Transitions at load time)
	To          string       `yaml:"to,omitempty"            json:"to,omitempty"`
	ToNodes     []string     `yaml:"to_nodes,omitempty"      json:"to_nodes,omitempty"`
	ElseToNodes []string     `yaml:"else_to_nodes,omitempty" json:"else_to_nodes,omitempty"`
	Transitions []Transition `yaml:"transitions,omitempty"    json:"transitions,omitempty"`
}

// EmitEventDef is the configuration for a state of type "emit_event".
// The state fires a named platform event, optionally carrying a subset of the
// blackboard as payload, then immediately advances via CompletionTrigger.
type EmitEventDef struct {
	// EventName is the logical name of the event (e.g. "document.ready").
	// Other runs waiting with on_event: document.ready in the same tenant will wake.
	EventName string `yaml:"event_name" json:"event_name"`
	// PayloadFields is an optional list of blackboard field names to include
	// in the event payload. Omit to emit an event with no payload.
	PayloadFields []string `yaml:"payload_fields,omitempty" json:"payload_fields,omitempty"`
	// CompletionTrigger is the transition trigger fired after the event is emitted.
	// Defaults to "emitted".
	CompletionTrigger string `yaml:"completion_trigger,omitempty" json:"completion_trigger,omitempty"`
}

type TelegramOutputDef struct {
	ChatID            string `yaml:"chat_id" json:"chat_id"`
	MessageText       string `yaml:"message_text" json:"message_text"`
	CompletionTrigger string `yaml:"completion_trigger,omitempty" json:"completion_trigger,omitempty"`
}

type DiscordOutputDef struct {
	ChannelID         string `yaml:"channel_id" json:"channel_id"`
	MessageText       string `yaml:"message_text" json:"message_text"`
	CompletionTrigger string `yaml:"completion_trigger,omitempty" json:"completion_trigger,omitempty"`
}

// SubProcessDef describes a delegation to another reusable workflow.
type SubProcessDef struct {
	// ProcessRef is the name of the workflow/process to invoke.
	ProcessRef string `yaml:"process_ref"                     json:"process_ref"`
	// ProcessVersion is an optional specific version to invoke (default: "latest").
	ProcessVersion string `yaml:"process_version,omitempty"       json:"process_version,omitempty"`
	// CompletionTrigger is fired when the sub-process completes successfully.
	CompletionTrigger string `yaml:"completion_trigger,omitempty"  json:"completion_trigger,omitempty"`
	// FailureTrigger is fired if the sub-process fails or is terminated.
	FailureTrigger string `yaml:"failure_trigger,omitempty"     json:"failure_trigger,omitempty"`
	// InputMappings maps parent blackboard fields to sub-process input ports.
	InputMappings map[string]string `yaml:"input_mappings,omitempty"      json:"input_mappings,omitempty"`
	// OutputMappings maps sub-process output ports back to parent blackboard fields.
	OutputMappings map[string]string `yaml:"output_mappings,omitempty"     json:"output_mappings,omitempty"`
}

// ScriptDef defines deterministic inline logic for a state of type "script".
// Both fields are evaluated using the expr-lang expression language.
// The blackboard fields are available as top-level variables.
type ScriptDef struct {
	// Trigger is an expr-lang expression that evaluates to a string — the name of
	// the transition trigger to fire.  Example: `amount > 1000 ? "needs_review" : "auto_approve"`
	Trigger string `yaml:"trigger" json:"trigger"`
	// Updates maps blackboard field names to expr-lang expressions that compute the
	// new value.  Example: { "approved": "amount < 1000", "vat": "amount * 0.2" }
	Updates map[string]string `yaml:"updates,omitempty" json:"updates,omitempty"`
}

// CodeDef defines user-written JavaScript for a state of type "code".
// The script has access to a mutable `bb` object (the current blackboard) and
// must either return { trigger, blackboard_updates?, reasoning? } or call the
// injected trigger("name") function for early exit.
// Execution is sandboxed: no network, no filesystem access.
type CodeDef struct {
	// Language is currently always "javascript". Reserved for future Starlark support.
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
	// Code is the JavaScript source to execute.
	Code string `yaml:"code" json:"code"`
}

type Position struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

type Transition struct {
	From    string   `yaml:"from"            json:"from"`
	To      string   `yaml:"to"              json:"to"`                // Primary target (backward-compat)
	ToNodes []string `yaml:"to_nodes"        json:"to_nodes,omitempty"` // Multiple targets for parallel dispatch
	Trigger string   `yaml:"trigger"         json:"trigger"`
	Guard   string   `yaml:"guard,omitempty" json:"guard,omitempty"` // expr-lang expression
}

type AgentDef struct {
	Name           string            `yaml:"name"                      json:"name"`
	Model          string            `yaml:"model,omitempty"           json:"model,omitempty"`
	// TaskQueue overrides the default Temporal task queue for this agent.
	// When set, the ASM workflow schedules this agent's activity on the named
	// queue so that a specialist worker (not the LLM worker) picks it up.
	// Leave empty to use the platform default queue (asm-workers).
	TaskQueue      string            `yaml:"task_queue,omitempty"      json:"task_queue,omitempty"`
	Tools          []string          `yaml:"tools,omitempty"           json:"tools,omitempty"`
	PromptTemplate string            `yaml:"prompt_template,omitempty" json:"prompt_template,omitempty"`
	Rules          []string          `yaml:"rules,omitempty"           json:"rules,omitempty"`
	Config         map[string]string `yaml:"config,omitempty"          json:"config,omitempty"`
}

// Helpers

func (w *WorkflowDef) StateByName(name string) *StateDef {
	for i := range w.States {
		if w.States[i].Name == name {
			return &w.States[i]
		}
	}
	return nil
}

func (w *WorkflowDef) AgentByName(name string) *AgentDef {
	for i := range w.Agents {
		if w.Agents[i].Name == name {
			return &w.Agents[i]
		}
	}
	return nil
}

func (w *WorkflowDef) InitialState() *StateDef {
	for i := range w.States {
		if w.States[i].Type == StateInitial {
			return &w.States[i]
		}
	}
	return nil
}

// TransitionsFrom returns all transitions whose source is `from`.
func (w *WorkflowDef) TransitionsFrom(from string) []Transition {
	var out []Transition
	for _, t := range w.Transitions {
		if t.From == from {
			out = append(out, t)
		}
	}
	return out
}

// MCPAuditLog represents a single communication event with an MCP server.
type MCPAuditLog struct {
	ID         string      `json:"id"`
	RunID      string      `json:"run_id,omitempty"`
	StateName  string      `json:"state_name,omitempty"`
	AgentName  string      `json:"agent_name,omitempty"`
	ServerURL  string      `json:"server_url"`
	Method     string      `json:"method"`
	ToolName   string      `json:"tool_name,omitempty"`
	Input      interface{} `json:"input"`
	Output     interface{} `json:"output"`
	IsError    bool        `json:"is_error"`
	ErrorMsg   string      `json:"error_msg,omitempty"`
	DurationMs int         `json:"duration_ms"`
	CreatedAt  time.Time   `json:"created_at"`
}
