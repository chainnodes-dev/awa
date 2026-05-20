package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/metrics"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Executor is the interface the engine calls to dispatch agent work.
type Executor interface {
	Execute(ctx context.Context, task AgentTask) (*asmtypes.AgentOutput, error)
}

// AgentTask carries everything an executor needs to run an agent.
type AgentTask struct {
	RunID      string
	TenantID   string
	AgentDef   asmtypes.AgentDef
	StateDef   asmtypes.StateDef
	Blackboard map[string]interface{}
	Def        *asmtypes.WorkflowDef
}

// TemporalEngineClient abstracts the Temporal client used by the engine.
// Defined here (not in the temporal package) to avoid an import cycle —
// internal/temporal imports internal/orchestrator for EvalGuard and AgentTask.
type TemporalEngineClient interface {
	StartWorkflow(ctx context.Context, run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (temporalID string, err error)
	SendTriggerSignal(ctx context.Context, temporalID, trigger string, payload map[string]interface{}) error
	SendHITLSignal(ctx context.Context, temporalID string, sig asmtypes.HITLSignal) error
	SendChatSignal(ctx context.Context, temporalID, message, sender string) error
	TerminateWorkflow(ctx context.Context, temporalID string) error
	AwaitWorkflowCompletion(ctx context.Context, temporalID string) error
	Close()
}

// Engine drives workflow runs via a Temporal client.
type Engine struct {
	store    store.Store
	bus      events.Bus
	temporal TemporalEngineClient
}

func NewEngine(s store.Store, bus events.Bus, temporal TemporalEngineClient) *Engine {
	return &Engine{
		store:    s,
		bus:      bus,
		temporal: temporal,
	}
}

// StartRun creates and starts a new workflow run in Temporal.
func (e *Engine) StartRun(ctx context.Context, workflowName, version string, input map[string]interface{}) (*asmtypes.WorkflowRun, error) {
	def, _, err := e.store.GetDefinition(ctx, workflowName, version)
	if err != nil {
		return nil, fmt.Errorf("load workflow definition: %w", err)
	}

	initial := def.InitialState()
	if initial == nil {
		return nil, fmt.Errorf("workflow '%s' has no initial state", workflowName)
	}

	run := &asmtypes.WorkflowRun{
		ID:              uuid.NewString(),
		TenantID:        store.TenantIDFromContext(ctx),
		WorkflowName:    workflowName,
		WorkflowVersion: version,
		Status:          asmtypes.RunRunning,
		CurrentState:    initial.Name,
		Blackboard:      input,
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if run.Blackboard == nil {
		run.Blackboard = make(map[string]interface{})
	}

	if err := e.store.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("persist run: %w", err)
	}

	metrics.RunsStartedTotal.WithLabelValues(workflowName, version, run.TenantID).Inc()
	metrics.RunsActive.WithLabelValues(workflowName, run.TenantID).Inc()

	_ = e.bus.Publish(ctx, events.New(events.RunCreated, events.RunCreatedPayload{Run: run}))

	temporalID, err := e.temporal.StartWorkflow(ctx, run, def)
	if err != nil {
		return nil, fmt.Errorf("start temporal workflow: %w", err)
	}

	run.TemporalID = temporalID
	if err := e.store.UpdateRun(ctx, run); err != nil {
		slog.Warn("Failed to persist temporal ID", "run_id", run.ID, "error", err)
	}

	return run, nil
}

// AwaitWorkflowCompletion blocks until the specified temporal workflow completes.
func (e *Engine) AwaitWorkflowCompletion(ctx context.Context, temporalID string) error {
	return e.temporal.AwaitWorkflowCompletion(ctx, temporalID)
}

// Trigger fires a named trigger on a running workflow in Temporal.
func (e *Engine) Trigger(ctx context.Context, req asmtypes.TriggerRequest) error {
	run, err := e.store.GetRun(ctx, req.RunID)
	if err != nil {
		return fmt.Errorf("load run: %w", err)
	}
	if run.TemporalID == "" {
		return fmt.Errorf("run '%s' has no temporal workflow ID", req.RunID)
	}

	// Any payload is persisted via the store if needed/supported (Temporal handles its own blackboard state)
	// but currently the store.UpdateRun is mostly used for status tracking.
	return e.temporal.SendTriggerSignal(ctx, run.TemporalID, req.Trigger, req.Payload)
}

// SignalHITL resolves a human-in-the-loop state in Temporal.
func (e *Engine) SignalHITL(ctx context.Context, sig asmtypes.HITLSignal) error {
	if err := e.store.ResolveHITL(ctx, sig.RunID, sig.Resolution, sig.Resolver); err != nil {
		return err
	}

	_ = e.bus.Publish(ctx, events.New(events.HITLResolved, map[string]string{
		"run_id": sig.RunID, "resolution": sig.Resolution, "resolver": sig.Resolver,
	}))

	run, err := e.store.GetRun(ctx, sig.RunID)
	if err != nil {
		return fmt.Errorf("load run: %w", err)
	}

	if run.TemporalID == "" {
		return fmt.Errorf("run '%s' has no temporal workflow ID", sig.RunID)
	}
	return e.temporal.SendHITLSignal(ctx, run.TemporalID, sig)
}

// SendChat sends a chat message to a waiting agent in Temporal.
func (e *Engine) SendChat(ctx context.Context, runID, message, sender string) error {
	run, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("load run: %w", err)
	}
	if run.TemporalID == "" {
		return fmt.Errorf("run '%s' has no temporal workflow ID", runID)
	}

	// Record the chat event in the bus so UI can show it immediately.
	_ = e.bus.Publish(ctx, events.New(events.AgentChat, map[string]string{
		"run_id": runID, "message": message, "sender": sender, "role": "human",
	}))

	return e.temporal.SendChatSignal(ctx, run.TemporalID, message, sender)
}

// ListRuns returns runs matching the filter.
func (e *Engine) ListRuns(ctx context.Context, filter store.RunFilter) ([]*asmtypes.WorkflowRun, error) {
	return e.store.ListRuns(ctx, filter)
}

// GetRun returns a run from the store.
func (e *Engine) GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error) {
	return e.store.GetRun(ctx, id)
}

// DeleteRun removes a run and its associated transitions and HITL requests.
func (e *Engine) DeleteRun(ctx context.Context, id string) error {
	return e.store.DeleteRun(ctx, id)
}

// TerminateRun stops a running workflow in Temporal and updates its status to cancelled.
func (e *Engine) TerminateRun(ctx context.Context, id string) error {
	run, err := e.store.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("load run: %w", err)
	}

	// Immediate termination in Temporal if it's currently active.
	if run.TemporalID != "" && (run.Status == asmtypes.RunRunning || run.Status == asmtypes.RunWaiting || run.Status == asmtypes.RunPending) {
		if err := e.temporal.TerminateWorkflow(ctx, run.TemporalID); err != nil {
			slog.Warn("Failed to terminate temporal workflow", "run_id", id, "temporal_id", run.TemporalID, "error", err)
			// Continue to update DB status anyway to reflect user intent.
		}
	}

	now := time.Now()
	run.Status = asmtypes.RunCancelled
	run.UpdatedAt = now
	run.CompletedAt = &now

	if err := e.store.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("update run status: %w", err)
	}

	_ = e.bus.Publish(ctx, events.New(events.RunCancelled, events.RunCancelledPayload{Run: run}))
	metrics.RunsActive.WithLabelValues(run.WorkflowName, run.TenantID).Dec()

	return nil
}


// EvalGuard evaluates an expr-lang expression against blackboard data.
// EvalScript evaluates a state's ScriptDef against the blackboard.
// It is a pure deterministic function; safe to call inline in Temporal workflows.
func EvalScript(state *asmtypes.StateDef, bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
	if state.Script == nil {
		return nil, fmt.Errorf("state '%s' has no script definition", state.Name)
	}
	script := state.Script

	triggerProg, err := expr.Compile(script.Trigger, expr.Env(bb), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("compile trigger expr '%s': %w", script.Trigger, err)
	}
	triggerResult, err := expr.Run(triggerProg, bb)
	if err != nil {
		return nil, fmt.Errorf("run trigger expr '%s': %w", script.Trigger, err)
	}
	trigger, ok := triggerResult.(string)
	if !ok {
		return nil, fmt.Errorf("trigger expr '%s' must evaluate to string, got %T", script.Trigger, triggerResult)
	}

	updates := make(map[string]interface{})
	for field, exprStr := range script.Updates {
		prog, err := expr.Compile(exprStr, expr.Env(bb), expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("compile update expr for '%s': %w", field, err)
		}
		val, err := expr.Run(prog, bb)
		if err != nil {
			return nil, fmt.Errorf("run update expr for '%s': %w", field, err)
		}
		updates[field] = val
	}

	return &asmtypes.AgentOutput{
		Trigger:           trigger,
		BlackboardUpdates: updates,
		Reasoning:         fmt.Sprintf("script: %s", script.Trigger),
	}, nil
}

func EvalGuard(expression string, bb map[string]interface{}) (bool, error) {
	program, err := expr.Compile(expression, expr.Env(bb), expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return false, fmt.Errorf("compile guard '%s': %w", expression, err)
	}
	result, err := expr.Run(program, bb)
	if err != nil {
		return false, fmt.Errorf("run guard '%s': %w", expression, err)
	}
	b, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("guard '%s' did not return bool", expression)
	}
	return b, nil
}
