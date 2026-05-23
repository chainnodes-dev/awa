package asmtypes

import "time"

// RunStatus represents the lifecycle status of a workflow run.
type RunStatus string

const (
	RunPending  RunStatus = "pending"
	RunRunning  RunStatus = "running"
	RunWaiting  RunStatus = "waiting"   // paused at a HITL state
	RunComplete RunStatus = "complete"
	RunFailed   RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

// WorkflowRun is a live instance of a WorkflowDef.
type WorkflowRun struct {
	TenantID        string                 `json:"tenant_id"`
	ID              string                 `json:"id"`
	WorkflowName    string                 `json:"workflow_name"`
	WorkflowVersion string                 `json:"workflow_version"`
	Status          RunStatus              `json:"status"`
	CurrentState    string                 `json:"current_state"`
	Blackboard      map[string]interface{} `json:"blackboard"`
	TemporalID      string                 `json:"temporal_id,omitempty"`
	// FailureReason is set when Status == RunFailed; contains the error message
	// from the executor or engine so it is visible in the UI without digging in logs.
	FailureReason   string                 `json:"failure_reason,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
}

// TransitionRecord is an immutable history entry.
type TransitionRecord struct {
	ID                 string                 `json:"id"`
	RunID              string                 `json:"run_id"`
	FromState          string                 `json:"from_state"`
	ToState            string                 `json:"to_state"`
	Trigger            string                 `json:"trigger"`
	BlackboardSnapshot map[string]interface{} `json:"blackboard_snapshot"`
	AgentOutput        *AgentOutput           `json:"agent_output,omitempty"`
	Timestamp          time.Time              `json:"timestamp"`
}

// HITLRequest tracks a pending human-in-the-loop action.
type HITLRequest struct {
	ID         string                 `json:"id"`
	RunID      string                 `json:"run_id"`
	StateName  string                 `json:"state_name"`
	Assignee   string                 `json:"assignee,omitempty"`
	TimeoutAt  *time.Time             `json:"timeout_at,omitempty"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Resolution string                 `json:"resolution,omitempty"` // "approved" | "rejected"
	Resolver   string                 `json:"resolver,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	FormSchema       map[string]interface{} `json:"form_schema,omitempty"`
	Blackboard       map[string]interface{} `json:"blackboard,omitempty"`
	BlackboardSchema map[string]FieldDef    `json:"blackboard_schema,omitempty"`
	WorkflowName     string                 `json:"workflow_name,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// AgentOutput is the structured JSON an agent returns after execution.
type AgentOutput struct {
	BlackboardUpdates map[string]interface{} `json:"blackboard_updates"`
	Trigger           string                 `json:"trigger"`
	Content           string                 `json:"content,omitempty"` // raw model output
	Reasoning         string                 `json:"reasoning,omitempty"`
	Error             string                 `json:"error,omitempty"`
	StackTrace        string                 `json:"stack_trace,omitempty"`
	LLMCalls          []LLMCallLog           `json:"llm_calls,omitempty"`
}

// LLMCallResponse captures the final decision from the LLM for the observability panel.
type LLMCallResponse struct {
	Content   string `json:"content"`
	Trigger   string `json:"trigger"`
	Reasoning string `json:"reasoning"`
}

// LLMCallLog records the raw LLM prompt and response for observability.
type LLMCallLog struct {
	StateName string           `json:"stateName"`
	AgentName string           `json:"agentName"`
	System    string           `json:"system"`
	Messages  []interface{}    `json:"messages"`
	Response  *LLMCallResponse `json:"response,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// TriggerRequest asks the engine to fire a named trigger on a run.
type TriggerRequest struct {
	RunID   string                 `json:"run_id"`
	Trigger string                 `json:"trigger"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// HITLSignal is sent by a human to resolve a HITL state.
type HITLSignal struct {
	RunID      string                 `json:"run_id"`
	Resolution string                 `json:"resolution"` // "approved" | "rejected"
	Resolver   string                 `json:"resolver"`
	Comment    string                 `json:"comment,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"` // Data collected from dynamic forms
}
