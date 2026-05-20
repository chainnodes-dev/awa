package temporal

import (
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Signal name constants used by both the workflow and the client.
const (
	SignalTrigger        = "trigger"
	SignalHITLResolution = "hitl_resolution"
	SignalChat           = "chat"
)

// WorkflowParams is the single input argument to ASMWorkflow.
// The full WorkflowDef is passed in (not loaded inside the workflow) so the
// workflow function stays deterministic across replays.
type WorkflowParams struct {
	RunID      string
	TenantID   string // required for event subscription activities
	Def        *asmtypes.WorkflowDef
	Blackboard map[string]interface{}
}

// TriggerSignalPayload is the body of the "trigger" signal sent by Engine.Trigger.
type TriggerSignalPayload struct {
	Trigger string
	Payload map[string]interface{}
}

// ChatSignalPayload is sent by a human to talk to an agent during HITL.
type ChatSignalPayload struct {
	Message string
	Sender  string
}

// HITLResolutionPayload is the body of the "hitl_resolution" signal sent by Engine.SignalHITL.
type HITLResolutionPayload struct {
	Resolution string // "approved" | "rejected"
	Resolver   string
	Comment    string
	Payload    map[string]interface{} // Data collected from dynamic forms
}

// UpdateRunParams is the activity input for updating a run's persisted state.
type UpdateRunParams struct {
	RunID           string
	TenantID        string // for metrics
	WorkflowName    string // for metrics
	WorkflowVersion string // for metrics
	CurrentState    string
	Status          asmtypes.RunStatus
	Blackboard      map[string]interface{}
	FailureReason   string
	IsTerminal      bool // set CompletedAt when true
}

// RecordTransitionParams is the activity input for appending a transition record.
type RecordTransitionParams struct {
	TenantID        string // for metrics
	WorkflowName    string // for metrics
	WorkflowVersion string // for metrics
	Record          *asmtypes.TransitionRecord
}

// PublishEventParams is the activity input for publishing a bus event.
type PublishEventParams struct {
	EventType string
	Data      interface{}
}

// CreateHITLParams is the activity input for creating a HITL request.
type CreateHITLParams struct {
	Request *asmtypes.HITLRequest
}

// ExecuteAgentParams is the activity input for running an LLM agent.
type ExecuteAgentParams struct {
	RunID           string
	TenantID        string
	WorkflowName    string // for metrics
	WorkflowVersion string // for metrics
	AgentDef        asmtypes.AgentDef
	StateDef        asmtypes.StateDef
	Blackboard      map[string]interface{}
	Def             *asmtypes.WorkflowDef
}

// ExecuteCodeParams is the activity input for running a user JavaScript code node.
type ExecuteCodeParams struct {
	RunID           string
	TenantID        string
	WorkflowName    string // for metrics
	WorkflowVersion string // for metrics
	StateDef        asmtypes.StateDef
	Blackboard      map[string]interface{}
	ValidTriggers   []string
}

// LoadWorkflowDefParams is the activity input for loading a WorkflowDef by name and version.
// Used by subprocess states to resolve the child workflow before launching it.
type LoadWorkflowDefParams struct {
	TenantID     string
	WorkflowName string
	Version      string // optional; empty string triggers latest lookup
}

// GetRunBlackboardParams is the activity input for reading the terminal blackboard of a run.
// Used after a child skill workflow completes to extract its output fields.
type GetRunBlackboardParams struct {
	TenantID string
	RunID    string
}

// RegisterEventSubscriptionParams is the activity input for registering a wait-on-event subscription.
type RegisterEventSubscriptionParams struct {
	Subscription *store.EventSubscription
}

// UnregisterEventSubscriptionParams is the activity input for removing a wait-on-event subscription.
type UnregisterEventSubscriptionParams struct {
	SubscriptionID string
}

// EmitWorkflowEventParams is the activity input for emitting a named platform event
// and signalling all waiting Temporal workflows in the same tenant.
type EmitWorkflowEventParams struct {
	TenantID  string
	RunID     string
	EventName string
	Payload   map[string]interface{}
}

// SendTelegramMessageParams is the activity input for sending a Telegram message.
type SendTelegramMessageParams struct {
	TenantID    string
	ChatID      string
	MessageText string
}

// SendDiscordMessageParams is the activity input for sending a Discord message.
type SendDiscordMessageParams struct {
	TenantID    string
	ChannelID   string
	MessageText string
}

