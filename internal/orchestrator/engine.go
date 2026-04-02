package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/events"
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
	AgentDef   asmtypes.AgentDef
	StateDef   asmtypes.StateDef
	Blackboard map[string]interface{}
}

// TemporalEngineClient abstracts the Temporal client used by the engine.
// Defined here (not in the temporal package) to avoid an import cycle —
// internal/temporal imports internal/orchestrator for EvalGuard and AgentTask.
type TemporalEngineClient interface {
	StartWorkflow(ctx context.Context, run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (temporalID string, err error)
	SendTriggerSignal(ctx context.Context, temporalID, trigger string, payload map[string]interface{}) error
	SendHITLSignal(ctx context.Context, temporalID string, sig asmtypes.HITLSignal) error
	Close()
}

// Engine drives state machine instances.
type Engine struct {
	store    store.Store
	bus      events.Bus
	executor Executor
	temporal TemporalEngineClient // nil in direct mode

	mu   sync.RWMutex
	runs map[string]*runContext // live run contexts (direct mode only)
}

type runContext struct {
	def        *asmtypes.WorkflowDef
	blackboard *Blackboard
	hitlCh     chan string   // receives HITL resolution signals
	wakeupCh   chan struct{} // wakes driveRun when an external Trigger() fires
	cancelFn   context.CancelFunc
}

func NewEngine(s store.Store, bus events.Bus, executor Executor) *Engine {
	return &Engine{
		store:    s,
		bus:      bus,
		executor: executor,
		runs:     make(map[string]*runContext),
	}
}

// UseTemporalClient switches the engine to Temporal mode.
// Must be called before any runs are started.
func (e *Engine) UseTemporalClient(c TemporalEngineClient) {
	e.temporal = c
}

// StartRun creates and starts a new workflow run.
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

	_ = e.bus.Publish(ctx, events.New(events.RunCreated, events.RunCreatedPayload{Run: run}))

	// --- Temporal mode ---
	if e.temporal != nil {
		temporalID, err := e.temporal.StartWorkflow(ctx, run, def)
		if err != nil {
			return nil, fmt.Errorf("start temporal workflow: %w", err)
		}
		run.TemporalID = temporalID
		if err := e.store.UpdateRun(ctx, run); err != nil {
			log.Printf("[engine] failed to persist temporal ID for run %s: %v", run.ID, err)
		}
		return run, nil
	}

	// --- Direct mode ---
	bb := NewBlackboard(def.Blackboard.Schema, run.Blackboard)
	runCtx, cancel := context.WithCancel(context.Background())
	rc := &runContext{
		def:        def,
		blackboard: bb,
		hitlCh:     make(chan string, 1),
		wakeupCh:   make(chan struct{}, 1),
		cancelFn:   cancel,
	}
	e.mu.Lock()
	e.runs[run.ID] = rc
	e.mu.Unlock()

	go e.driveRun(runCtx, run.ID, rc)

	return run, nil
}

// driveRun is the goroutine that advances a run's state machine.
func (e *Engine) driveRun(ctx context.Context, runID string, rc *runContext) {
	defer func() {
		e.mu.Lock()
		delete(e.runs, runID)
		e.mu.Unlock()
	}()

	for {
		run, err := e.store.GetRun(ctx, runID)
		if err != nil {
			log.Printf("[engine] cannot load run %s: %v", runID, err)
			return
		}

		state := rc.def.StateByName(run.CurrentState)
		if state == nil {
			log.Printf("[engine] unknown state '%s' in run %s", run.CurrentState, runID)
			return
		}

		// Terminal state — we're done.
		if state.Type == asmtypes.StateTerminal {
			e.completeRun(ctx, run, rc)
			return
		}

		// HITL — park and wait for a signal.
		if state.Type == asmtypes.StateHITL {
			if err := e.handleHITL(ctx, run, state, rc); err != nil {
				log.Printf("[engine] HITL error for run %s: %v", runID, err)
				return
			}
			continue
		}

		// Script execution — runs when a Script block is present, regardless of
		// the state's display type (an initial or intermediate state may also carry
		// a script to auto-advance without an LLM call).
		if state.Script != nil {
			output, err := e.evalScript(ctx, state, rc.blackboard.Read())
			if err != nil {
				log.Printf("[engine] script error in run %s state '%s': %v", runID, state.Name, err)
				e.failRun(ctx, run, rc, err)
				return
			}
			if len(output.BlackboardUpdates) > 0 {
				if err := rc.blackboard.Write(output.BlackboardUpdates); err != nil {
					log.Printf("[engine] blackboard write error: %v", err)
				}
				for k, v := range output.BlackboardUpdates {
					_ = e.bus.Publish(ctx, events.New(events.BlackboardUpdated, events.BlackboardUpdatedPayload{
						RunID: runID, Key: k, Value: v,
					}))
				}
			}
			if output.Trigger != "" {
				if err := e.applyTrigger(ctx, run, rc, output.Trigger, output); err != nil {
					log.Printf("[engine] trigger error: %v", err)
					e.failRun(ctx, run, rc, err)
					return
				}
			}
			continue
		}

		// If state has an agent, dispatch it.
		if state.Agent != "" {
			agentDef := rc.def.AgentByName(state.Agent)
			if agentDef == nil {
				log.Printf("[engine] agent '%s' not found in workflow", state.Agent)
				e.failRun(ctx, run, rc, fmt.Errorf("agent '%s' not defined", state.Agent))
				return
			}

			output, err := e.executor.Execute(ctx, AgentTask{
				RunID:      runID,
				AgentDef:   *agentDef,
				StateDef:   *state,
				Blackboard: rc.blackboard.Read(),
			})
			if err != nil {
				log.Printf("[engine] executor error in run %s: %v", runID, err)
				e.failRun(ctx, run, rc, err)
				return
			}

			// Apply blackboard updates from agent output.
			if len(output.BlackboardUpdates) > 0 {
				if err := rc.blackboard.Write(output.BlackboardUpdates); err != nil {
					log.Printf("[engine] blackboard write error: %v", err)
				}
				for k, v := range output.BlackboardUpdates {
					_ = e.bus.Publish(ctx, events.New(events.BlackboardUpdated, events.BlackboardUpdatedPayload{
						RunID: runID, Key: k, Value: v,
					}))
				}
			}

			// Fire the trigger returned by the agent.
			if output.Trigger != "" {
				if err := e.applyTrigger(ctx, run, rc, output.Trigger, output); err != nil {
					log.Printf("[engine] trigger error: %v", err)
					e.failRun(ctx, run, rc, err)
					return
				}
			}
			continue
		}

		// No agent on this state — wait for an external Trigger() call.
		// wakeupCh is signalled by transitionTo after the trigger is applied.
		select {
		case <-rc.wakeupCh:
			// Trigger() has already updated the store; loop to re-read state.
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute): // idle timeout guard
			return
		}
	}
}

// Trigger fires a named trigger on a running workflow from an external caller.
func (e *Engine) Trigger(ctx context.Context, req asmtypes.TriggerRequest) error {
	// --- Temporal mode ---
	if e.temporal != nil {
		run, err := e.store.GetRun(ctx, req.RunID)
		if err != nil {
			return fmt.Errorf("load run: %w", err)
		}
		if run.TemporalID == "" {
			return fmt.Errorf("run '%s' has no temporal workflow ID", req.RunID)
		}
		return e.temporal.SendTriggerSignal(ctx, run.TemporalID, req.Trigger, req.Payload)
	}

	// --- Direct mode ---
	e.mu.RLock()
	rc, ok := e.runs[req.RunID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("run '%s' not active", req.RunID)
	}

	run, err := e.store.GetRun(ctx, req.RunID)
	if err != nil {
		return err
	}

	if len(req.Payload) > 0 {
		_ = rc.blackboard.Write(req.Payload)
	}

	return e.applyTrigger(ctx, run, rc, req.Trigger, nil)
}

// SignalHITL resolves a human-in-the-loop state.
func (e *Engine) SignalHITL(ctx context.Context, sig asmtypes.HITLSignal) error {
	if err := e.store.ResolveHITL(ctx, sig.RunID, sig.Resolution, sig.Resolver); err != nil {
		return err
	}

	_ = e.bus.Publish(ctx, events.New(events.HITLResolved, map[string]string{
		"run_id": sig.RunID, "resolution": sig.Resolution, "resolver": sig.Resolver,
	}))

	// --- Temporal mode ---
	if e.temporal != nil {
		run, err := e.store.GetRun(ctx, sig.RunID)
		if err != nil {
			return fmt.Errorf("load run: %w", err)
		}
		if run.TemporalID == "" {
			return fmt.Errorf("run '%s' has no temporal workflow ID", sig.RunID)
		}
		return e.temporal.SendHITLSignal(ctx, run.TemporalID, sig)
	}

	// --- Direct mode ---
	e.mu.RLock()
	rc, ok := e.runs[sig.RunID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("run '%s' not active", sig.RunID)
	}

	rc.hitlCh <- sig.Resolution
	return nil
}

// applyTrigger finds a valid transition for the given trigger and transitions the state.
func (e *Engine) applyTrigger(ctx context.Context, run *asmtypes.WorkflowRun, rc *runContext, trigger string, output *asmtypes.AgentOutput) error {
	transitions := rc.def.TransitionsFrom(run.CurrentState)
	bb := rc.blackboard.Read()

	for _, t := range transitions {
		if t.Trigger != trigger {
			continue
		}
		if t.Guard != "" {
			ok, err := EvalGuard(t.Guard, bb)
			if err != nil {
				return fmt.Errorf("guard eval error: %w", err)
			}
			if !ok {
				continue
			}
		}
		return e.transitionTo(ctx, run, rc, t.To, trigger, output)
	}
	return fmt.Errorf("no valid transition from '%s' for trigger '%s'", run.CurrentState, trigger)
}

func (e *Engine) transitionTo(ctx context.Context, run *asmtypes.WorkflowRun, rc *runContext, toState, trigger string, output *asmtypes.AgentOutput) error {
	fromState := run.CurrentState
	run.CurrentState = toState
	run.UpdatedAt = time.Now()
	run.Blackboard = rc.blackboard.Read()

	state := rc.def.StateByName(toState)
	if state != nil && state.Type == asmtypes.StateTerminal {
		now := time.Now()
		run.Status = asmtypes.RunComplete
		run.CompletedAt = &now
	}

	if err := e.store.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("persist run update: %w", err)
	}

	// Wake driveRun if it's parked waiting for an external trigger.
	if rc != nil {
		select {
		case rc.wakeupCh <- struct{}{}:
		default:
		}
	}

	_ = e.store.RecordTransition(ctx, &asmtypes.TransitionRecord{
		RunID:              run.ID,
		FromState:          fromState,
		ToState:            toState,
		Trigger:            trigger,
		BlackboardSnapshot: run.Blackboard,
		AgentOutput:        output,
		Timestamp:          time.Now(),
	})

	_ = e.bus.Publish(ctx, events.New(events.StateChanged, events.StateChangedPayload{
		RunID:      run.ID,
		FromState:  fromState,
		ToState:    toState,
		Trigger:    trigger,
		Blackboard: run.Blackboard,
	}))

	return nil
}

func (e *Engine) handleHITL(ctx context.Context, run *asmtypes.WorkflowRun, state *asmtypes.StateDef, rc *runContext) error {
	run.Status = asmtypes.RunWaiting
	run.UpdatedAt = time.Now()
	_ = e.store.UpdateRun(ctx, run)

	hitlReq := &asmtypes.HITLRequest{
		RunID:     run.ID,
		StateName: state.Name,
		Assignee:  state.Assignee,
		CreatedAt: time.Now(),
	}
	if state.Timeout != "" {
		d, err := time.ParseDuration(state.Timeout)
		if err == nil {
			t := time.Now().Add(d)
			hitlReq.TimeoutAt = &t
		}
	}
	_ = e.store.CreateHITL(ctx, hitlReq)

	_ = e.bus.Publish(ctx, events.New(events.HITLWaiting, events.HITLWaitingPayload{
		RunID:     run.ID,
		StateName: state.Name,
		Assignee:  state.Assignee,
	}))

	// Determine wait duration
	waitDur := 24 * time.Hour
	if state.Timeout != "" {
		if d, err := time.ParseDuration(state.Timeout); err == nil {
			waitDur = d
		}
	}

	select {
	case trigger := <-rc.hitlCh:
		run.Status = asmtypes.RunRunning
		return e.applyTrigger(ctx, run, rc, trigger, nil)
	case <-time.After(waitDur):
		if state.OnTimeout != "" {
			return e.applyTrigger(ctx, run, rc, "timeout", nil)
		}
		return e.applyTrigger(ctx, run, rc, "timeout", nil)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) completeRun(ctx context.Context, run *asmtypes.WorkflowRun, _ *runContext) {
	now := time.Now()
	run.Status = asmtypes.RunComplete
	run.CompletedAt = &now
	run.UpdatedAt = now
	_ = e.store.UpdateRun(ctx, run)
	_ = e.bus.Publish(ctx, events.New(events.RunCompleted, events.RunCreatedPayload{Run: run}))
}

func (e *Engine) failRun(ctx context.Context, run *asmtypes.WorkflowRun, _ *runContext, cause error) {
	now := time.Now()
	run.Status = asmtypes.RunFailed
	run.CompletedAt = &now
	run.UpdatedAt = now
	if cause != nil {
		run.FailureReason = cause.Error()
	}
	_ = e.store.UpdateRun(ctx, run)
	_ = e.bus.Publish(ctx, events.New(events.RunFailed, events.RunCreatedPayload{Run: run}))
}

// GetRun returns a run from the store.
func (e *Engine) GetRun(ctx context.Context, id string) (*asmtypes.WorkflowRun, error) {
	return e.store.GetRun(ctx, id)
}

// ListRuns returns runs matching the filter.
func (e *Engine) ListRuns(ctx context.Context, filter store.RunFilter) ([]*asmtypes.WorkflowRun, error) {
	return e.store.ListRuns(ctx, filter)
}

// EvalGuard evaluates an expr-lang expression against blackboard data.
// Exported so the Temporal workflow package can reuse it without duplication.
// evalScript evaluates a ScriptDef against the current blackboard and returns
// an AgentOutput containing the computed trigger and blackboard updates.
// Uses the same expr-lang evaluator as EvalGuard.
func (e *Engine) evalScript(ctx context.Context, state *asmtypes.StateDef, bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
	if state.Script == nil {
		return nil, fmt.Errorf("state '%s' has type 'script' but no script definition", state.Name)
	}
	script := state.Script

	// Evaluate the trigger expression → must produce a string.
	triggerProg, err := expr.Compile(script.Trigger, expr.Env(bb))
	if err != nil {
		return nil, fmt.Errorf("compile trigger expr '%s': %w", script.Trigger, err)
	}
	triggerResult, err := expr.Run(triggerProg, bb)
	if err != nil {
		return nil, fmt.Errorf("run trigger expr '%s': %w", script.Trigger, err)
	}
	trigger, ok := triggerResult.(string)
	if !ok {
		return nil, fmt.Errorf("trigger expr '%s' must evaluate to a string, got %T", script.Trigger, triggerResult)
	}

	// Evaluate each blackboard update expression.
	updates := make(map[string]interface{})
	for field, exprStr := range script.Updates {
		prog, err := expr.Compile(exprStr, expr.Env(bb))
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
	program, err := expr.Compile(expression, expr.Env(bb), expr.AsBool())
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
