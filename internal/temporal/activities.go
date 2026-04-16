package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	temporalactivity "go.temporal.io/sdk/activity"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/code"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Activities holds all dependencies needed by Temporal activity functions.
// Register an instance with the Temporal worker via worker.RegisterActivity(acts).
type Activities struct {
	Store    store.Store
	Bus      events.Bus
	Executor orchestrator.Executor
}

func NewActivities(s store.Store, bus events.Bus, exec orchestrator.Executor) *Activities {
	return &Activities{Store: s, Bus: bus, Executor: exec}
}

// UpdateRun persists run state changes to the store.
func (a *Activities) UpdateRun(ctx context.Context, p UpdateRunParams) error {
	run, err := a.Store.GetRun(ctx, p.RunID)
	if err != nil {
		return err
	}
	run.CurrentState = p.CurrentState
	run.Status = p.Status
	run.Blackboard = p.Blackboard
	run.UpdatedAt = time.Now()
	if p.IsTerminal {
		now := time.Now()
		run.CompletedAt = &now
	}
	return a.Store.UpdateRun(ctx, run)
}

// RecordTransition appends an immutable transition record to the store.
func (a *Activities) RecordTransition(ctx context.Context, p RecordTransitionParams) error {
	if p.Record.ID == "" {
		p.Record.ID = uuid.NewString()
	}
	if p.Record.Timestamp.IsZero() {
		p.Record.Timestamp = time.Now()
	}
	return a.Store.RecordTransition(ctx, p.Record)
}

// PublishEvent publishes a single event to the event bus.
//
// p.Data arrives from Temporal's JSON serialization as map[string]interface{},
// losing the original concrete type. We re-marshal it to json.RawMessage before
// constructing the event so that the payload is embedded as typed JSON bytes —
// consistent with events created directly via events.New in the direct executor.
func (a *Activities) PublishEvent(ctx context.Context, p PublishEventParams) error {
	raw, err := json.Marshal(p.Data)
	if err != nil {
		return err
	}
	return a.Bus.Publish(ctx, events.New(p.EventType, json.RawMessage(raw)))
}

// CreateHITL persists a new HITL request to the store.
func (a *Activities) CreateHITL(ctx context.Context, p CreateHITLParams) error {
	if p.Request.ID == "" {
		p.Request.ID = uuid.NewString()
	}
	return a.Store.CreateHITL(ctx, p.Request)
}

// ExecuteCode runs a user-written JavaScript code node via the goja sandbox
// and returns its structured output. The activity heartbeats every 10 s so
// Temporal can detect worker failures during long-running scripts.
func (a *Activities) ExecuteCode(ctx context.Context, p ExecuteCodeParams) (*asmtypes.AgentOutput, error) {
	timeout := 60 * time.Second
	if p.StateDef.Timeout != "" {
		if d, err := time.ParseDuration(p.StateDef.Timeout); err == nil {
			timeout = d
		}
	}
	heartbeat := func() { temporalactivity.RecordHeartbeat(ctx, "code running") }
	return code.Execute(p.StateDef.Code.Code, p.Blackboard, p.ValidTriggers, timeout, heartbeat)
}

// ExecuteAgent runs an LLM agent and returns its structured output.
// This activity has a longer timeout than the others (10 min vs 10 s).
func (a *Activities) ExecuteAgent(ctx context.Context, p ExecuteAgentParams) (*asmtypes.AgentOutput, error) {
	return a.Executor.Execute(ctx, orchestrator.AgentTask{
		RunID:      p.RunID,
		AgentDef:   p.AgentDef,
		StateDef:   p.StateDef,
		Blackboard: p.Blackboard,
		Def:        p.Def,
	})
}


// LoadWorkflowDef loads the latest version of a WorkflowDef by name.
// Used by skill_call states to resolve the child workflow before launching it.
func (a *Activities) LoadWorkflowDef(ctx context.Context, p LoadWorkflowDefParams) (*asmtypes.WorkflowDef, error) {
	def, _, err := a.Store.GetLatestDefinition(ctx, p.WorkflowName)
	if err != nil {
		return nil, fmt.Errorf("load workflow '%s': %w", p.WorkflowName, err)
	}
	return def, nil
}

// GetRunBlackboard reads the current blackboard snapshot for a completed run.
// Used after a child skill workflow completes to extract its output fields.
func (a *Activities) GetRunBlackboard(ctx context.Context, p GetRunBlackboardParams) (map[string]interface{}, error) {
	run, err := a.Store.GetRun(ctx, p.RunID)
	if err != nil {
		return nil, fmt.Errorf("get run '%s': %w", p.RunID, err)
	}
	return run.Blackboard, nil
}
