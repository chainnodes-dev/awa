package events

import (
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
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
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

// New creates a typed Event.
func New(eventType string, data interface{}) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}
