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

	// Activity options: short timeout for store/bus operations.
	shortOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporalsdk.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
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

	// Signal channels.
	triggerCh := workflow.GetSignalChannel(ctx, SignalTrigger)
	hitlCh := workflow.GetSignalChannel(ctx, SignalHITLResolution)

	var (
		wg             = workflow.NewWaitGroup(ctx)
		processedJoins = make(map[string]bool)
		workflowErr    error
	)

	var execBranch func(string)
	execBranch = func(currentState string) {
		wg.Add(1)
		defer wg.Done()

		for {
			state := p.Def.StateByName(currentState)
			if state == nil {
				workflowErr = fmt.Errorf("unknown state '%s'", currentState)
				return
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
				return
			}

			// ── Wait Node (Join) ────────────────────────────────────────────────
			if state.Type == asmtypes.StateWait {
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
					RunID:        p.RunID,
					CurrentState: currentState,
					Status:       asmtypes.RunWaiting,
					Blackboard:   bb,
				}).Get(ctx, nil)

				waitDur := 24 * time.Hour
				if state.Timeout != "" {
					if d, err := time.ParseDuration(state.Timeout); err == nil {
						waitDur = d
					}
				}

				var (
					trigger      string
					timeoutFired bool
					conditionMet bool
					conditionCh  = workflow.NewChannel(ctx)
				)

				// Helper goroutine to wait for the condition.
				workflow.Go(ctx, func(gCtx workflow.Context) {
					_ = workflow.Await(gCtx, func() bool {
						ok, _ := orchestrator.EvalGuard(state.Condition, bb)
						return ok
					})
					conditionCh.Send(gCtx, true)
				})

				selector := workflow.NewSelector(ctx)
				selector.AddReceive(conditionCh, func(c workflow.ReceiveChannel, _ bool) {
					c.Receive(ctx, nil)
					conditionMet = true
				})
				selector.AddReceive(triggerCh, func(c workflow.ReceiveChannel, _ bool) {
					var sig TriggerSignalPayload
					c.Receive(ctx, &sig)
					trigger = sig.Trigger
					for k, v := range sig.Payload {
						bb[k] = v
					}
				})
				selector.AddFuture(workflow.NewTimer(ctx, waitDur), func(f workflow.Future) {
					timeoutFired = true
				})

				selector.Select(ctx)

				if processedJoins[state.Name] {
					return
				}

				if conditionMet {
					processedJoins[state.Name] = true
				} else if timeoutFired && trigger == "" {
					trigger = state.OnTimeout
					if trigger == "" {
						trigger = "timeout"
					}
				}

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) > 0 {
					next := nextStates[0]
					if err := persistTransition(shortOpts, ctx, p.RunID, currentState, next, trigger, bb, nil); err != nil {
						workflowErr = err
						return
					}
					currentState = next
					continue
				}
				return
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

				var trigger string
				selector := workflow.NewSelector(ctx)
				selector.AddReceive(hitlCh, func(c workflow.ReceiveChannel, _ bool) {
					var sig HITLResolutionPayload
					c.Receive(ctx, &sig)
					trigger = sig.Resolution
				})
				selector.AddFuture(workflow.NewTimer(ctx, waitDur), func(f workflow.Future) {
					trigger = "timeout"
				})
				selector.Select(ctx)

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(ctx, func(gCtx workflow.Context) { execBranch(target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Agent state ───────────────────────────────────────────────────────
			if state.Agent != "" {
				agentDef := p.Def.AgentByName(state.Agent)
				if agentDef == nil {
					workflowErr = fmt.Errorf("agent '%s' not defined", state.Agent)
					return
				}

				var output *asmtypes.AgentOutput
				err := workflow.ExecuteActivity(agentOptsFor(agentDef), (*Activities).ExecuteAgent, ExecuteAgentParams{
					RunID:      p.RunID,
					AgentDef:   *agentDef,
					StateDef:   *state,
					Blackboard: bb,
					Def:        p.Def,
				}).Get(ctx, &output)
				if err != nil {
					workflowErr = fmt.Errorf("agent execution failed: %w", err)
					return
				}

				for k, v := range output.BlackboardUpdates {
					bb[k] = v
				}

				nextStates, err := applyTransition(p.Def, currentState, output.Trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextStates[0], output.Trigger, bb, output); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(ctx, func(gCtx workflow.Context) { execBranch(target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Script state ─────────────────────────────────────────────────────
			if state.Script != nil {
				output, err := orchestrator.EvalScript(state, bb)
				if err != nil {
					workflowErr = fmt.Errorf("script eval failed: %w", err)
					return
				}
				for k, v := range output.BlackboardUpdates {
					bb[k] = v
				}

				nextStates, err := applyTransition(p.Def, currentState, output.Trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextStates[0], output.Trigger, bb, output); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(ctx, func(gCtx workflow.Context) { execBranch(target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── External trigger state ────────────────────────────────────────────
			var (
				sig     TriggerSignalPayload
				trigger string
			)

			waitDur := 0 * time.Second
			if state.Timeout != "" {
				if d, err := time.ParseDuration(state.Timeout); err == nil {
					waitDur = d
				}
			}

			if waitDur > 0 {
				selector := workflow.NewSelector(ctx)
				selector.AddReceive(triggerCh, func(c workflow.ReceiveChannel, _ bool) {
					c.Receive(ctx, &sig)
					trigger = sig.Trigger
				})
				selector.AddFuture(workflow.NewTimer(ctx, waitDur), func(f workflow.Future) {
					trigger = state.OnTimeout
					if trigger == "" {
						trigger = "timeout"
					}
				})
				selector.Select(ctx)
			} else {
				triggerCh.Receive(ctx, &sig)
				trigger = sig.Trigger
			}

			for k, v := range sig.Payload {
				bb[k] = v
			}

			nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
			if err != nil {
				workflowErr = err
				return
			}
			if len(nextStates) == 0 {
				return
			}
			if err := persistTransition(shortOpts, ctx, p.RunID, currentState, nextStates[0], trigger, bb, nil); err != nil {
				workflowErr = err
				return
			}
			for i := 1; i < len(nextStates); i++ {
				target := nextStates[i]
				workflow.Go(ctx, func(gCtx workflow.Context) { execBranch(target) })
			}
			currentState = nextStates[0]
		}
	}

	execBranch(p.Def.InitialState().Name)
	wg.Wait(ctx)

	if workflowErr != nil {
		return failWorkflow(ctx, shortOpts, p, workflowErr)
	}

	return nil
}

func applyTransition(def *asmtypes.WorkflowDef, fromState, trigger string, bb map[string]interface{}) ([]string, error) {
	for _, t := range def.TransitionsFrom(fromState) {
		if t.Trigger != trigger {
			continue
		}
		if t.Guard != "" {
			ok, err := orchestrator.EvalGuard(t.Guard, bb)
			if err != nil {
				return nil, fmt.Errorf("guard eval error: %w", err)
			}
			if !ok {
				continue
			}
		}
		if len(t.ToNodes) > 0 {
			return t.ToNodes, nil
		}
		return []string{t.To}, nil
	}
	return nil, fmt.Errorf("no valid transition from '%s' for trigger '%s'", fromState, trigger)
}

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
