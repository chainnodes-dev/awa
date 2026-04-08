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
	StateChanged      = "state.changed"
	BlackboardUpdated = "blackboard.updated"
	AgentThinking     = "agent.thinking"     // LLM token stream
	AgentToolCall     = "agent.tool_call"
	HITLWaiting       = "hitl.waiting"
	HITLResolved      = "hitl.resolved"
)

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
	RunID     string      `json:"run_id"`
	AgentName string      `json:"agent_name"`
	ToolName  string      `json:"tool_name"`
	Input     interface{} `json:"input"`
	Output    interface{} `json:"output,omitempty"`
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
