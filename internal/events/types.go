package events

import (
	"encoding/json"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Event types emitted to WebSocket clients and internal subscribers.
const (
	RunCreated        = "run.created"
	RunCompleted      = "run.completed"
	RunFailed         = "run.failed"
	RunCancelled      = "run.cancelled"
	StateChanged      = "state.changed"
	BlackboardUpdated = "blackboard.updated"
	AgentThinking     = "agent.thinking"     // LLM token stream
	AgentToolCall     = "agent.tool_call"
	AgentPrompt       = "agent.prompt"       // full request sent to the LLM (debug)
	AgentResponse     = "agent.response"     // full response received from the LLM (debug)
	AgentChat         = "agent.chat"         // interactive message between agent and human
	HITLWaiting       = "hitl.waiting"
	HITLResolved      = "hitl.resolved"
	WorkflowEvent     = "workflow.event" // named event emitted by an emit_event state
	EmailInbound      = "email.inbound"  // inbound email received via webhook or IMAP
)

// InboundEmailPayload is emitted when a new email is ingested by the platform.
type InboundEmailPayload struct {
	TenantID    string            `json:"tenant_id"`
	MessageID   string            `json:"message_id"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Subject     string            `json:"subject"`
	Date        time.Time         `json:"date"`
	TextBody    string            `json:"text_body"`
	HTMLBody    string            `json:"html_body,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Source      string            `json:"source"` // "webhook" | "imap"
}

// EmailAttachment describes a file attachment parsed from an inbound email.
type EmailAttachment struct {
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
}

// Event is the envelope for all platform events.
//
// Data is stored as json.RawMessage so that the payload type is preserved
// through Redis round-trips. When Event is deserialized from JSON (e.g. after
// being carried through RedisBus), Data stays as raw bytes rather than being
// converted to map[string]interface{} — callers can unmarshal into the
// concrete payload struct for their event type.
type Event struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// Payloads

type RunCreatedPayload struct {
	Run *asmtypes.WorkflowRun `json:"run"`
}

type RunCancelledPayload struct {
	Run *asmtypes.WorkflowRun `json:"run"`
}

type StateChangedPayload struct {
	RunID      string                 `json:"run_id"`
	FromState  string                 `json:"from_state"`
	ToState    string                 `json:"to_state"`
	Trigger    string                 `json:"trigger"`
	Blackboard map[string]interface{} `json:"blackboard"`
}

type BlackboardUpdatedPayload struct {
	RunID string      `json:"run_id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type AgentThinkingPayload struct {
	RunID     string `json:"run_id"`
	AgentName string `json:"agent_name"`
	Token     string `json:"token"`
}

type AgentToolCallPayload struct {
	RunID      string      `json:"run_id"`
	StateName  string      `json:"state_name"`
	AgentName  string      `json:"agent_name"`
	ToolName   string      `json:"tool_name"`
	Input      interface{} `json:"input"`
	Output     interface{} `json:"output,omitempty"`
	DurationMs int         `json:"duration_ms,omitempty"`
	IsError    bool        `json:"is_error,omitempty"`
	ErrorMsg   string      `json:"error_msg,omitempty"`
}

// AgentPromptPayload carries the exact request sent to the LLM for one agent
// invocation.  Only emitted when a run is active; intended for admin debug UIs.
type AgentPromptPayload struct {
	RunID      string        `json:"run_id"`
	StateName  string        `json:"state_name"`
	AgentName  string        `json:"agent_name"`
	System     string        `json:"system"`
	Messages   []interface{} `json:"messages"` // []llm.Message serialised as-is
}

// AgentResponsePayload carries the final (post-tool-loop) LLM response for one
// agent invocation together with the parsed trigger and reasoning.
type AgentResponsePayload struct {
	RunID      string `json:"run_id"`
	StateName  string `json:"state_name"`
	AgentName  string `json:"agent_name"`
	Content    string `json:"content"`    // raw text from the model
	Trigger    string `json:"trigger"`
	Reasoning  string `json:"reasoning"`
}

type HITLWaitingPayload struct {
	RunID     string `json:"run_id"`
	StateName string `json:"state_name"`
	Assignee  string `json:"assignee,omitempty"`
}

// New creates a typed Event, marshalling data to json.RawMessage immediately.
// Using json.RawMessage for Data means the payload type survives JSON
// round-trips (e.g. through Redis) without degrading to map[string]interface{}.
func New(eventType string, data interface{}) Event {
	raw, _ := json.Marshal(data)
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      json.RawMessage(raw),
	}
}

// Decode unmarshals the event's Data into the provided value.
// Use this instead of a type assertion when the Event may have come
// from a Redis subscriber (where concrete types are not preserved).
//
//	var p events.HITLWaitingPayload
//	if err := event.Decode(&p); err != nil { ... }
func (e Event) Decode(v interface{}) error {
	return json.Unmarshal(e.Data, v)
}
