package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
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
	Client   client.Client // used by EmitWorkflowEvent to signal waiting workflows
}

func NewActivities(s store.Store, bus events.Bus, exec orchestrator.Executor, c client.Client) *Activities {
	return &Activities{Store: s, Bus: bus, Executor: exec, Client: c}
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

// RegisterEventSubscription persists a wait-on-event subscription so that
// EmitWorkflowEvent can locate this workflow when the named event fires.
func (a *Activities) RegisterEventSubscription(ctx context.Context, p RegisterEventSubscriptionParams) error {
	if err := a.Store.RegisterEventSubscription(ctx, p.Subscription); err != nil {
		return fmt.Errorf("register event subscription: %w", err)
	}
	return nil
}

// UnregisterEventSubscription removes the wait-on-event subscription after
// the event has been received (or the wait state timed out).
func (a *Activities) UnregisterEventSubscription(ctx context.Context, p UnregisterEventSubscriptionParams) error {
	return a.Store.UnregisterEventSubscription(ctx, p.SubscriptionID)
}

// EmitWorkflowEvent publishes a named event to the bus and signals every Temporal
// workflow in the same tenant that is currently waiting for that event name.
func (a *Activities) EmitWorkflowEvent(ctx context.Context, p EmitWorkflowEventParams) error {
	// 1. Publish to the event bus so WebSocket clients and other subscribers see it.
	raw, _ := json.Marshal(map[string]interface{}{
		"run_id":     p.RunID,
		"event_name": p.EventName,
		"payload":    p.Payload,
	})
	_ = a.Bus.Publish(ctx, events.New(events.WorkflowEvent, json.RawMessage(raw)))

	// 2. Find all runs in this tenant waiting for this event name.
	subs, err := a.Store.ListEventSubscriptions(ctx, p.TenantID, p.EventName)
	if err != nil {
		return fmt.Errorf("list event subscriptions for '%s': %w", p.EventName, err)
	}

	// 3. Signal each waiting Temporal workflow and remove the subscription.
	for _, sub := range subs {
		sigErr := a.Client.SignalWorkflow(ctx, sub.TemporalID, "", SignalTrigger, TriggerSignalPayload{
			Trigger: sub.OnMatchTrigger,
			Payload: p.Payload,
		})
		if sigErr != nil {
			// Log but continue — the workflow may have already completed.
			fmt.Printf("warn: signal workflow %s for event %s: %v\n", sub.TemporalID, p.EventName, sigErr)
			continue
		}
		_ = a.Store.UnregisterEventSubscription(ctx, sub.ID)
	}
	return nil
}
