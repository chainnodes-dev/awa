package temporal

import (
	"fmt"
	"time"

	temporalsdk "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ASMWorkflow is the Temporal workflow function that drives a single ASM run.
// It mirrors the Engine.driveRun loop but uses Temporal primitives for
// durability: all I/O goes through activities, all state lives in workflow memory.
func ASMWorkflow(ctx workflow.Context, p WorkflowParams) error {
	// Apply schema defaults to the initial blackboard.
	bb := applyDefaults(p.Def, p.Blackboard)

	// Start from the initial state.
	currentState := p.Def.InitialState().Name

	// Activity options: short timeout for store/bus operations.
	shortOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalsdk.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	})

	// Activity options: long timeout for LLM agent calls (default task queue).
	agentOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporalsdk.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    60 * time.Second,
			MaximumAttempts:    3,
		},
	})

	// agentOptsFor returns activity options for a given AgentDef.
	// When the agent declares a TaskQueue, a new context with that queue is
	// returned so Temporal routes the activity to a specialist worker.
	agentOptsFor := func(agentDef *asmtypes.AgentDef) workflow.Context {
		if agentDef.TaskQueue == "" {
			return agentOpts
		}
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			TaskQueue:           agentDef.TaskQueue,
			StartToCloseTimeout: 10 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
			RetryPolicy: &temporalsdk.RetryPolicy{
				InitialInterval:    2 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    60 * time.Second,
				MaximumAttempts:    3,
			},
		})
	}

	// Signal channels (set up once, reused across loop iterations).
	triggerCh := workflow.GetSignalChannel(ctx, SignalTrigger)
	hitlCh := workflow.GetSignalChannel(ctx, SignalHITLResolution)

	for {
		state := p.Def.StateByName(currentState)
		if state == nil {
			return fmt.Errorf("unknown state '%s'", currentState)
		}

		// ── Terminal ─────────────────────────────────────────────────────────
		if state.Type == asmtypes.StateTerminal {
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
				RunID:        p.RunID,
				CurrentState: currentState,
				Status:       asmtypes.RunComplete,
				Blackboard:   bb,
				IsTerminal:   true,
			}).Get(ctx, nil)
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
				EventType: events.RunCompleted,
				Data:      events.RunCreatedPayload{},
			}).Get(ctx, nil)
			return nil
		}

		// ── HITL ─────────────────────────────────────────────────────────────
		if state.Type == asmtypes.StateHITL {
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
				RunID:        p.RunID,
				CurrentState: currentState,
				Status:       asmtypes.RunWaiting,
				Blackboard:   bb,
			}).Get(ctx, nil)

			hitlReq := &asmtypes.HITLRequest{
				RunID:     p.RunID,
				StateName: state.Name,
				Assignee:  state.Assignee,
				CreatedAt: workflow.Now(ctx),
			}
			waitDur := 24 * time.Hour
			if state.Timeout != "" {
				if d, err := time.ParseDuration(state.Timeout); err == nil {
					waitDur = d
					t := workflow.Now(ctx).Add(d)
					hitlReq.TimeoutAt = &t
				}
			}
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).CreateHITL, CreateHITLParams{
				Request: hitlReq,
			}).Get(ctx, nil)
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
				EventType: events.HITLWaiting,
				Data: events.HITLWaitingPayload{
					RunID: p.RunID, StateName: state.Name, Assignee: state.Assignee,
				},
			}).Get(ctx, nil)

			// Park: wait for HITL resolution or timeout.
			var trigger string
			timerFired := false

			workflow.NewSelector(ctx).
				AddReceive(hitlCh, func(c workflow.ReceiveChannel, _ bool) {
					var sig HITLResolutionPayload
					c.Receive(ctx, &sig)
					trigger = "human_approved"
					if sig.Resolution == "rejected" {
						trigger = "human_rejected"
					}
				}).
				AddFuture(workflow.NewTimer(ctx, waitDur), func(_ workflow.Future) {
					timerFired = true
					trigger = "timeout"
				}).
				Select(ctx)

			_ = timerFired // suppress unused warning

			nextState, err := applyTransition(p.Def, currentState, trigger, bb)
			if err != nil {
				return failWorkflow(ctx, shortOpts, p, err)
			}
			if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextState, trigger, bb, nil); err != nil {
				return err
			}
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
				RunID:        p.RunID,
				CurrentState: nextState,
				Status:       asmtypes.RunRunning,
				Blackboard:   bb,
				IsTerminal:   p.Def.StateByName(nextState) != nil && p.Def.StateByName(nextState).Type == asmtypes.StateTerminal,
			}).Get(ctx, nil)
			currentState = nextState
			continue
		}

		// ── Agent state ───────────────────────────────────────────────────────
		if state.Agent != "" {
			agentDef := p.Def.AgentByName(state.Agent)
			if agentDef == nil {
				return failWorkflow(ctx, shortOpts, p, fmt.Errorf("agent '%s' not defined", state.Agent))
			}

			var output *asmtypes.AgentOutput
			err := workflow.ExecuteActivity(agentOptsFor(agentDef), (*Activities).ExecuteAgent, ExecuteAgentParams{
				RunID:      p.RunID,
				AgentDef:   *agentDef,
				StateDef:   *state,
				Blackboard: bb,
			}).Get(ctx, &output)
			if err != nil {
				return failWorkflow(ctx, shortOpts, p, fmt.Errorf("agent execution failed: %w", err))
			}

			// Apply blackboard updates.
			for k, v := range output.BlackboardUpdates {
				bb[k] = v
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
					EventType: events.BlackboardUpdated,
					Data:      events.BlackboardUpdatedPayload{RunID: p.RunID, Key: k, Value: v},
				}).Get(ctx, nil)
			}

			nextState, err := applyTransition(p.Def, currentState, output.Trigger, bb)
			if err != nil {
				return failWorkflow(ctx, shortOpts, p, err)
			}
			if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextState, output.Trigger, bb, output); err != nil {
				return err
			}
			_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
				RunID:        p.RunID,
				CurrentState: nextState,
				Status:       asmtypes.RunRunning,
				Blackboard:   bb,
				IsTerminal:   p.Def.StateByName(nextState) != nil && p.Def.StateByName(nextState).Type == asmtypes.StateTerminal,
			}).Get(ctx, nil)
			currentState = nextState
			continue
		}

		// ── External trigger state ────────────────────────────────────────────
		// No agent: park and wait for a trigger signal from the API.
		var sig TriggerSignalPayload
		triggerCh.Receive(ctx, &sig)

		for k, v := range sig.Payload {
			bb[k] = v
		}

		nextState, err := applyTransition(p.Def, currentState, sig.Trigger, bb)
		if err != nil {
			return failWorkflow(ctx, shortOpts, p, err)
		}
		if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextState, sig.Trigger, bb, nil); err != nil {
			return err
		}
		_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
			RunID:        p.RunID,
			CurrentState: nextState,
			Status:       asmtypes.RunRunning,
			Blackboard:   bb,
			IsTerminal:   p.Def.StateByName(nextState) != nil && p.Def.StateByName(nextState).Type == asmtypes.StateTerminal,
		}).Get(ctx, nil)
		currentState = nextState
	}
}

// applyTransition finds the next state for a given trigger, evaluating guards.
// Pure function — safe to call in workflow context (no I/O, deterministic).
func applyTransition(def *asmtypes.WorkflowDef, fromState, trigger string, bb map[string]interface{}) (string, error) {
	for _, t := range def.TransitionsFrom(fromState) {
		if t.Trigger != trigger {
			continue
		}
		if t.Guard != "" {
			ok, err := orchestrator.EvalGuard(t.Guard, bb)
			if err != nil {
				return "", fmt.Errorf("guard eval error: %w", err)
			}
			if !ok {
				continue
			}
		}
		return t.To, nil
	}
	return "", fmt.Errorf("no valid transition from '%s' for trigger '%s'", fromState, trigger)
}

// persistTransition fires the RecordTransition and StateChanged activities.
func persistTransition(shortOpts, ctx workflow.Context, runID, fromState, toState, trigger string, bb map[string]interface{}, output *asmtypes.AgentOutput) error {
	rec := &asmtypes.TransitionRecord{
		RunID:              runID,
		FromState:          fromState,
		ToState:            toState,
		Trigger:            trigger,
		BlackboardSnapshot: bb,
		AgentOutput:        output,
		Timestamp:          workflow.Now(ctx),
	}
	if err := workflow.ExecuteActivity(shortOpts, (*Activities).RecordTransition, RecordTransitionParams{
		Record: rec,
	}).Get(ctx, nil); err != nil {
		return err
	}
	_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
		EventType: events.StateChanged,
		Data: events.StateChangedPayload{
			RunID:      runID,
			FromState:  fromState,
			ToState:    toState,
			Trigger:    trigger,
			Blackboard: bb,
		},
	}).Get(ctx, nil)
	return nil
}

// failWorkflow marks the run as failed in the store and returns the cause.
func failWorkflow(ctx, shortOpts workflow.Context, p WorkflowParams, cause error) error {
	_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
		RunID:      p.RunID,
		Status:     asmtypes.RunFailed,
		IsTerminal: true,
	}).Get(ctx, nil)
	_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
		EventType: events.RunFailed,
		Data:      map[string]string{"run_id": p.RunID, "error": cause.Error()},
	}).Get(ctx, nil)
	return cause
}

// applyDefaults seeds the blackboard with schema default values.
func applyDefaults(def *asmtypes.WorkflowDef, initial map[string]interface{}) map[string]interface{} {
	bb := make(map[string]interface{})
	for k, field := range def.Blackboard.Schema {
		if field.Default != nil {
			bb[k] = field.Default
		}
	}
	for k, v := range initial {
		bb[k] = v
	}
	return bb
}
