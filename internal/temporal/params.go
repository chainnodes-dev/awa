package temporal

import (
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Signal name constants used by both the workflow and the client.
const (
	SignalTrigger        = "trigger"
	SignalHITLResolution = "hitl_resolution"
)

// WorkflowParams is the single input argument to ASMWorkflow.
// The full WorkflowDef is passed in (not loaded inside the workflow) so the
// workflow function stays deterministic across replays.
type WorkflowParams struct {
	RunID      string
	Def        *asmtypes.WorkflowDef
	Blackboard map[string]interface{}
}

// TriggerSignalPayload is the body of the "trigger" signal sent by Engine.Trigger.
type TriggerSignalPayload struct {
	Trigger string
	Payload map[string]interface{}
}

// HITLResolutionPayload is the body of the "hitl_resolution" signal sent by Engine.SignalHITL.
type HITLResolutionPayload struct {
	Resolution string // "approved" | "rejected"
	Resolver   string
	Comment    string
}

// UpdateRunParams is the activity input for updating a run's persisted state.
type UpdateRunParams struct {
	RunID        string
	CurrentState string
	Status       asmtypes.RunStatus
	Blackboard   map[string]interface{}
	IsTerminal   bool // set CompletedAt when true
}

// RecordTransitionParams is the activity input for appending a transition record.
type RecordTransitionParams struct {
	Record *asmtypes.TransitionRecord
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
	RunID      string
	AgentDef   asmtypes.AgentDef
	StateDef   asmtypes.StateDef
	Blackboard map[string]interface{}
}
