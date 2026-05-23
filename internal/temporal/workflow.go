package temporal

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	temporalsdk "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ASMWorkflow is the Temporal workflow function that drives a single Phaxa run.
// It mirrors the Engine.driveRun loop but uses Temporal primitives for
// durability: all I/O goes through activities, all state lives in workflow memory.
func ASMWorkflow(ctx workflow.Context, p WorkflowParams) error {
	// Apply schema defaults to the initial blackboard.
	bb := applyDefaults(p.Def, p.Blackboard)

	// Signal channels are workflow-level (not coroutine-level) — safe to share.
	triggerCh := workflow.GetSignalChannel(ctx, SignalTrigger)
	hitlCh := workflow.GetSignalChannel(ctx, SignalHITLResolution)

	var (
		wg             = workflow.NewWaitGroup(ctx)
		processedJoins = make(map[string]bool)
		workflowErr    error
	)

	// makeOpts builds branch-local activity options derived from the branch's own
	// context. This is required because Temporal coroutines must only block on
	// contexts that belong to the current coroutine — using a parent context inside
	// a workflow.Go goroutine causes the "wrong Context" panic.
	makeShortOpts := func(bCtx workflow.Context) workflow.Context {
		return workflow.WithActivityOptions(bCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporalsdk.RetryPolicy{
				InitialInterval:    2 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    30 * time.Second,
				MaximumAttempts:    3,
			},
		})
	}
	makeAgentOpts := func(bCtx workflow.Context, taskQueue string) workflow.Context {
		return workflow.WithActivityOptions(bCtx, workflow.ActivityOptions{
			TaskQueue:           taskQueue, // empty string → default queue
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

	// execBranch drives one branch of the workflow starting from currentState.
	// bCtx must be the context of the current coroutine (the root ctx for the
	// initial branch; the gCtx passed by workflow.Go for parallel branches).
	var execBranch func(bCtx workflow.Context, currentState string)
	execBranch = func(bCtx workflow.Context, currentState string) {
		wg.Add(1)
		defer wg.Done()

		// Build branch-local activity options from THIS coroutine's context.
		shortOpts := makeShortOpts(bCtx)

		for {
			state := p.Def.StateByName(currentState)
			if state == nil {
				workflowErr = fmt.Errorf("unknown state '%s'", currentState)
				return
			}

			// ── Initial (no agent / no script) ───────────────────────────────────
			// The act of starting a run is itself the trigger for the initial state.
			// If no agent or script is assigned we auto-advance immediately using the
			// first outgoing transition whose guard passes — no external signal needed.
			if state.Type == asmtypes.StateInitial && state.Agent == "" && state.Script == nil {
				transitions := p.Def.TransitionsFrom(currentState)
				if len(transitions) == 0 {
					workflowErr = fmt.Errorf("initial state '%s' has no outgoing transitions", currentState)
					return
				}
				var autoTrigger string
				for _, t := range transitions {
					if t.Guard == "" {
						autoTrigger = t.Trigger
						break
					}
					if ok, _ := orchestrator.EvalGuard(t.Guard, bb); ok {
						autoTrigger = t.Trigger
						break
					}
				}
				if autoTrigger == "" {
					workflowErr = fmt.Errorf("initial state '%s': no outgoing transition guard passed", currentState)
					return
				}
				nextStates, err := applyTransition(p.Def, currentState, autoTrigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], autoTrigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Terminal ─────────────────────────────────────────────────────────
			if state.Type == asmtypes.StateTerminal {
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
					RunID:           p.RunID,
					TenantID:        p.TenantID,
					WorkflowName:    p.Def.Metadata.Name,
					WorkflowVersion: p.Def.Metadata.Version,
					CurrentState:    currentState,
					Status:          asmtypes.RunComplete,
					Blackboard:      bb,
					IsTerminal:      true,
				}).Get(bCtx, nil)
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).PublishEvent, PublishEventParams{
					EventType: events.RunCompleted,
					Data:      events.RunCreatedPayload{},
				}).Get(bCtx, nil)
				return
			}

			// ── Wait Node (Join / wait-on-event) ────────────────────────────────
			if state.Type == asmtypes.StateWait {
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
					RunID:        p.RunID,
					CurrentState: currentState,
					Status:       asmtypes.RunWaiting,
					Blackboard:   bb,
				}).Get(bCtx, nil)

				waitDur := 24 * time.Hour
				if state.Timeout != "" {
					if d, err := time.ParseDuration(state.Timeout); err == nil {
						waitDur = d
					}
				}

				// If this wait state listens for a platform event, register a subscription
				// so EmitWorkflowEvent can locate and signal this workflow.
				var eventSubID string
				if state.OnEvent != "" {
					wfID := workflow.GetInfo(bCtx).WorkflowExecution.ID
					eventSubID = wfID + "__" + state.Name
					onMatchTrigger := state.OnEventMatch
					if onMatchTrigger == "" {
						onMatchTrigger = "event_received"
					}
					_ = workflow.ExecuteActivity(shortOpts, (*Activities).RegisterEventSubscription,
						RegisterEventSubscriptionParams{
							Subscription: &store.EventSubscription{
								ID:             eventSubID,
								TenantID:       p.TenantID,
								RunID:          p.RunID,
								TemporalID:     wfID,
								EventName:      state.OnEvent,
								OnMatchTrigger: onMatchTrigger,
							},
						},
					).Get(bCtx, nil)
				}

				var (
					trigger      string
					timeoutFired bool
					conditionMet bool
					conditionCh  = workflow.NewChannel(bCtx)
				)

				// Helper goroutine to wait for the condition.
				workflow.Go(bCtx, func(gCtx workflow.Context) {
					_ = workflow.Await(gCtx, func() bool {
						ok, _ := orchestrator.EvalGuard(state.Condition, bb)
						return ok
					})
					conditionCh.Send(gCtx, true)
				})

				selector := workflow.NewSelector(bCtx)
				selector.AddReceive(conditionCh, func(c workflow.ReceiveChannel, _ bool) {
					c.Receive(bCtx, nil)
					conditionMet = true
				})
				selector.AddReceive(triggerCh, func(c workflow.ReceiveChannel, _ bool) {
					var sig TriggerSignalPayload
					c.Receive(bCtx, &sig)
					trigger = sig.Trigger
					for k, v := range sig.Payload {
						bb[k] = v
					}
				})
				selector.AddFuture(workflow.NewTimer(bCtx, waitDur), func(f workflow.Future) {
					timeoutFired = true
				})

				selector.Select(bCtx)

				// Clean up event subscription regardless of how the wait resolved.
				if eventSubID != "" {
					_ = workflow.ExecuteActivity(shortOpts, (*Activities).UnregisterEventSubscription,
						UnregisterEventSubscriptionParams{SubscriptionID: eventSubID},
					).Get(bCtx, nil)
				}

				if processedJoins[state.Name] {
					return
				}

				if conditionMet {
					processedJoins[state.Name] = true
					trigger = state.OnCondition
					if trigger == "" {
						trigger = "condition_met"
					}
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
					if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, next, trigger, bb, nil); err != nil {
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
				}).Get(bCtx, nil)

				hitlReq := &asmtypes.HITLRequest{
					RunID:            p.RunID,
					StateName:        state.Name,
					Assignee:         state.Assignee,
					CreatedAt:        workflow.Now(bCtx),
					FormSchema:       state.FormSchema,
					Blackboard:       bb,
					BlackboardSchema: p.Def.Blackboard.Schema,
					WorkflowName:     p.Def.Metadata.Name,
				}
				waitDur := 24 * time.Hour
				if state.Timeout != "" {
					if d, err := time.ParseDuration(state.Timeout); err == nil {
						waitDur = d
						t := workflow.Now(bCtx).Add(d)
						hitlReq.TimeoutAt = &t
					}
				}
				_ = workflow.ExecuteActivity(shortOpts, (*Activities).CreateHITL, CreateHITLParams{
					Request: hitlReq,
				}).Get(bCtx, nil)

				var trigger string
				selector := workflow.NewSelector(bCtx)
				selector.AddReceive(hitlCh, func(c workflow.ReceiveChannel, _ bool) {
					var sig HITLResolutionPayload
					c.Receive(bCtx, &sig)
					trigger = sig.Resolution
					for k, v := range sig.Payload {
						bb[k] = v
					}
				})
				selector.AddReceive(workflow.GetSignalChannel(bCtx, SignalChat), func(c workflow.ReceiveChannel, _ bool) {
					var sig ChatSignalPayload
					c.Receive(bCtx, &sig)
					
					// Update blackboard with the user's message and re-run the agent to respond.
					bb["_last_user_message"] = sig.Message
					
					if state.Agent != "" {
						agentDef := p.Def.AgentByName(state.Agent)
						if agentDef != nil {
							agentOpts := makeAgentOpts(bCtx, agentDef.TaskQueue)
							var output *asmtypes.AgentOutput
							_ = workflow.ExecuteActivity(agentOpts, (*Activities).ExecuteAgent, ExecuteAgentParams{
								RunID:           p.RunID,
								TenantID:        p.TenantID,
								WorkflowName:    p.Def.Metadata.Name,
								WorkflowVersion: p.Def.Metadata.Version,
								AgentDef:        *agentDef,
								StateDef:        *state,
								Blackboard:      bb,
								Def:             p.Def,
							}).Get(bCtx, &output)
							
							if output != nil {
								for k, v := range output.BlackboardUpdates {
									bb[k] = v
								}

								// Publish agent's response as a chat message.
								_ = workflow.ExecuteActivity(agentOpts, (*Activities).PublishEvent, PublishEventParams{
									EventType: events.AgentChat,
									Data: map[string]string{
										"run_id":  p.RunID,
										"message": output.Content,
										"sender":  agentDef.Name,
										"role":    "agent",
									},
								})
							}
						}
					}
				})
				selector.AddFuture(workflow.NewTimer(bCtx, waitDur), func(f workflow.Future) {
					trigger = "timeout"
				})
				selector.Select(bCtx)

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
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

				agentOpts := makeAgentOpts(bCtx, agentDef.TaskQueue)
				var output *asmtypes.AgentOutput
				err := workflow.ExecuteActivity(agentOpts, (*Activities).ExecuteAgent, ExecuteAgentParams{
					RunID:           p.RunID,
					TenantID:        p.TenantID,
					WorkflowName:    p.Def.Metadata.Name,
					WorkflowVersion: p.Def.Metadata.Version,
					AgentDef:        *agentDef,
					StateDef:        *state,
					Blackboard:      bb,
					Def:             p.Def,
				}).Get(bCtx, &output)
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
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], output.Trigger, bb, output); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
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
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], output.Trigger, bb, output); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Code state (JavaScript via goja) ─────────────────────────────────
			if state.Type == asmtypes.StateCode && state.Code != nil {
				outgoing := p.Def.TransitionsFrom(currentState)
				validTriggers := make([]string, 0, len(outgoing))
				for _, t := range outgoing {
					validTriggers = append(validTriggers, t.Trigger)
				}

				codeOpts := makeAgentOpts(bCtx, "") // same long-timeout / heartbeat as agent activities
				var output *asmtypes.AgentOutput
				err := workflow.ExecuteActivity(codeOpts, (*Activities).ExecuteCode, ExecuteCodeParams{
					RunID:           p.RunID,
					TenantID:        p.TenantID,
					WorkflowName:    p.Def.Metadata.Name,
					WorkflowVersion: p.Def.Metadata.Version,
					StateDef:        *state,
					Blackboard:      bb,
					ValidTriggers:   validTriggers,
				}).Get(bCtx, &output)
				if err != nil {
					workflowErr = fmt.Errorf("code execution failed in state '%s': %w", currentState, err)
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
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], output.Trigger, bb, output); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Emit Event ──────────────────────────────────────────────────────
			if state.Type == asmtypes.StateEmitEvent && state.EmitEvent != nil {
				ee := state.EmitEvent

				// Build payload: include only named fields, or the full blackboard.
				payload := make(map[string]interface{})
				if len(ee.PayloadFields) > 0 {
					for _, f := range ee.PayloadFields {
						if v, ok := bb[f]; ok {
							payload[f] = v
						}
					}
				} else {
					for k, v := range bb {
						payload[k] = v
					}
				}

				_ = workflow.ExecuteActivity(shortOpts, (*Activities).EmitWorkflowEvent, EmitWorkflowEventParams{
					TenantID:  p.TenantID,
					RunID:     p.RunID,
					EventName: ee.EventName,
					Payload:   payload,
				}).Get(bCtx, nil)

				trigger := ee.CompletionTrigger
				if trigger == "" {
					trigger = "event_emitted"
				}
				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Telegram Output ───────────────────────────────────────────────────
			if state.Type == asmtypes.StateTelegramOutput && state.TelegramOutput != nil {
				to := state.TelegramOutput
				chatID := resolveTemplate(to.ChatID, bb)
				msgText := resolveTemplate(to.MessageText, bb)

				var trigger string
				err := workflow.ExecuteActivity(shortOpts, (*Activities).SendTelegramMessage, SendTelegramMessageParams{
					TenantID:    p.TenantID,
					ChatID:      chatID,
					MessageText: msgText,
				}).Get(bCtx, &trigger)
				if err != nil {
					workflowErr = fmt.Errorf("failed to send Telegram message in state '%s': %w", currentState, err)
					return
				}

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── Discord Output ────────────────────────────────────────────────────
			if state.Type == asmtypes.StateDiscordOutput && state.DiscordOutput != nil {
				do := state.DiscordOutput
				channelID := resolveTemplate(do.ChannelID, bb)
				msgText := resolveTemplate(do.MessageText, bb)

				var trigger string
				err := workflow.ExecuteActivity(shortOpts, (*Activities).SendDiscordMessage, SendDiscordMessageParams{
					TenantID:    p.TenantID,
					ChannelID:   channelID,
					MessageText: msgText,
				}).Get(bCtx, &trigger)
				if err != nil {
					workflowErr = fmt.Errorf("failed to send Discord message in state '%s': %w", currentState, err)
					return
				}

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
				}
				currentState = nextStates[0]
				continue
			}

			// ── SubProcess Call ──────────────────────────────────────────────────────
			if state.Type == asmtypes.StateSubProcess && state.SubProcess != nil {
				sc := state.SubProcess
				skillOpts := makeShortOpts(bCtx)

				// 1. Verify license for sub-processes (Enterprise Feature).
				var claims *enterprise.LicenseClaims
				err := workflow.ExecuteActivity(skillOpts, (*Activities).GetLicenseClaims, p.TenantID).Get(bCtx, &claims)
				if err != nil {
					workflowErr = fmt.Errorf("license verification failed: %w", err)
					return
				}
				if claims.Tier == enterprise.TierFree {
					workflowErr = fmt.Errorf("%w: subprocesses require a Pro or Enterprise license", enterprise.ErrTierNotMet)
					return
				}

				// 2. Load the child WorkflowDef by name and version.
				var childDef *asmtypes.WorkflowDef
				err = workflow.ExecuteActivity(skillOpts, (*Activities).LoadWorkflowDef,
					LoadWorkflowDefParams{
						TenantID:     p.TenantID,
						WorkflowName: sc.ProcessRef,
						Version:      sc.ProcessVersion,
					},
				).Get(bCtx, &childDef)
				if err != nil {
					workflowErr = fmt.Errorf("subprocess: load workflow '%s' (version: %s): %w", sc.ProcessRef, sc.ProcessVersion, err)
					return
				}

				// 2. Build the child blackboard from input mappings.
				childBB := make(map[string]interface{})
				for childField, parentField := range sc.InputMappings {
					if v, ok := bb[parentField]; ok {
						childBB[childField] = v
					}
				}

				// 3. Launch a Temporal child workflow.
				childRunID := workflow.GetInfo(bCtx).WorkflowExecution.ID + "__sub__" + currentState
				cwo := workflow.ChildWorkflowOptions{
					WorkflowID: childRunID,
				}

				// Apply state-level timeout to the sub-process if provided.
				if state.Timeout != "" {
					if d, err := time.ParseDuration(state.Timeout); err == nil {
						cwo.WorkflowExecutionTimeout = d
						cwo.WorkflowRunTimeout = d
					}
				}

				childCtx := workflow.WithChildOptions(bCtx, cwo)
				skillErr := workflow.ExecuteChildWorkflow(childCtx, ASMWorkflow, WorkflowParams{
					RunID:      childRunID,
					Def:        childDef,
					Blackboard: childBB,
				}).Get(bCtx, nil)

				// 4. Determine trigger and apply output mappings.
				trigger := sc.CompletionTrigger
				if trigger == "" {
					trigger = "done" // default for unified process model
				}
				if skillErr != nil {
					trigger = sc.FailureTrigger
					if trigger == "" {
						trigger = "subprocess_failed"
					}
				} else {
					// Read the child run's terminal blackboard and map outputs back.
					var childFinalBB map[string]interface{}
					_ = workflow.ExecuteActivity(skillOpts, (*Activities).GetRunBlackboard,
						GetRunBlackboardParams{TenantID: p.TenantID, RunID: childRunID},
					).Get(bCtx, &childFinalBB)
					for childField, parentField := range sc.OutputMappings {
						if v, ok := childFinalBB[childField]; ok {
							bb[parentField] = v
						}
					}
				}

				nextStates, err := applyTransition(p.Def, currentState, trigger, bb)
				if err != nil {
					workflowErr = err
					return
				}
				if len(nextStates) == 0 {
					return
				}
				if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
					workflowErr = err
					return
				}
				for i := 1; i < len(nextStates); i++ {
					target := nextStates[i]
					workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
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
				selector := workflow.NewSelector(bCtx)
				selector.AddReceive(triggerCh, func(c workflow.ReceiveChannel, _ bool) {
					c.Receive(bCtx, &sig)
					trigger = sig.Trigger
				})
				selector.AddFuture(workflow.NewTimer(bCtx, waitDur), func(f workflow.Future) {
					trigger = state.OnTimeout
					if trigger == "" {
						trigger = "timeout"
					}
				})
				selector.Select(bCtx)
			} else {
				triggerCh.Receive(bCtx, &sig)
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
			if err := persistTransition(shortOpts, bCtx, p.RunID, p.Def, p.TenantID, currentState, nextStates[0], trigger, bb, nil); err != nil {
				workflowErr = err
				return
			}
			for i := 1; i < len(nextStates); i++ {
				target := nextStates[i]
				workflow.Go(bCtx, func(gCtx workflow.Context) { execBranch(gCtx, target) })
			}
			currentState = nextStates[0]
		}
	}

	execBranch(ctx, p.Def.InitialState().Name)
	wg.Wait(ctx)

	if workflowErr != nil {
		return failWorkflow(ctx, makeShortOpts(ctx), p, workflowErr)
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

func persistTransition(shortOpts, ctx workflow.Context, runID string, def *asmtypes.WorkflowDef, tenantID, fromState, toState, trigger string, bb map[string]interface{}, output *asmtypes.AgentOutput) error {
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
		TenantID:        tenantID,
		WorkflowName:    def.Metadata.Name,
		WorkflowVersion: def.Metadata.Version,
		Record:          rec,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Keep workflow_runs.current_state in sync so GET /runs/{id} always reflects
	// the live state without relying solely on WebSocket delivery.
	status := asmtypes.RunRunning
	if target := def.StateByName(toState); target != nil {
		if target.Type == asmtypes.StateHITL || target.Type == asmtypes.StateWait {
			status = asmtypes.RunWaiting
		}
	}

	if err := workflow.ExecuteActivity(shortOpts, (*Activities).UpdateRun, UpdateRunParams{
		RunID:           runID,
		TenantID:        tenantID,
		WorkflowName:    def.Metadata.Name,
		WorkflowVersion: def.Metadata.Version,
		CurrentState:    toState,
		Status:          status,
		Blackboard:      bb,
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
		RunID:           p.RunID,
		TenantID:        p.TenantID,
		WorkflowName:    p.Def.Metadata.Name,
		WorkflowVersion: p.Def.Metadata.Version,
		Status:          asmtypes.RunFailed,
		FailureReason:   cause.Error(),
		IsTerminal:      true,
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

var templateRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_\.]+)\s*\}\}`)

func resolveTemplate(tmpl string, bb map[string]interface{}) string {
	return templateRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		sub := templateRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		path := sub[1]
		// Trim optional "bb." or "blackboard." prefixes
		path = strings.TrimPrefix(path, "bb.")
		path = strings.TrimPrefix(path, "blackboard.")

		val, ok := bb[path]
		if !ok {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

