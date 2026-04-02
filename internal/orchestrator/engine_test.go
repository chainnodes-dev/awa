package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// --- mock executor ---

type mockExecutor struct {
	fn func(task AgentTask) (*asmtypes.AgentOutput, error)
}

func (m *mockExecutor) Execute(_ context.Context, task AgentTask) (*asmtypes.AgentOutput, error) {
	return m.fn(task)
}

func fixedOutput(trigger string, updates map[string]interface{}) *mockExecutor {
	return &mockExecutor{fn: func(_ AgentTask) (*asmtypes.AgentOutput, error) {
		return &asmtypes.AgentOutput{Trigger: trigger, BlackboardUpdates: updates}, nil
	}}
}

// --- workflow fixtures ---

// twoStateWorkflow: initial -> terminal, no agent, manual trigger.
const twoStateYAML = `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: two-state
  version: v1
states:
  - name: start
    type: initial
  - name: done
    type: terminal
transitions:
  - from: start
    to: done
    trigger: finish
`

// agentWorkflow: initial -[agent fires "next"]-> review -> terminal.
const agentWorkflowYAML = `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: agent-wf
  version: v1
agents:
  - name: classifier
    model: claude-3-haiku-20240307
states:
  - name: start
    type: initial
    agent: classifier
  - name: review
    type: intermediate
  - name: done
    type: terminal
transitions:
  - from: start
    to: review
    trigger: next
  - from: review
    to: done
    trigger: finish
`

// guardedWorkflow: start -[pass guard]-> approved; start -[fail guard]-> rejected.
const guardedWorkflowYAML = `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: guarded-wf
  version: v1
blackboard:
  schema:
    score:
      type: number
states:
  - name: start
    type: initial
    agent: scorer
  - name: approved
    type: terminal
  - name: rejected
    type: terminal
agents:
  - name: scorer
    model: claude-3-haiku-20240307
transitions:
  - from: start
    to: approved
    trigger: evaluated
    guard: "score >= 80"
  - from: start
    to: rejected
    trigger: evaluated
    guard: "score < 80"
`

// hitlWorkflow: initial -> hitl state (waits for human) -> terminal.
const hitlWorkflowYAML = `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: hitl-wf
  version: v1
states:
  - name: start
    type: initial
  - name: human_review
    type: hitl
    assignee: alice
    timeout: 24h
  - name: approved
    type: terminal
  - name: rejected
    type: terminal
transitions:
  - from: start
    to: human_review
    trigger: submitted
  - from: human_review
    to: approved
    trigger: human_approved
  - from: human_review
    to: rejected
    trigger: human_rejected
`

// helpers

func loadWorkflow(t *testing.T, s store.Store, yamlSrc string) *asmtypes.WorkflowDef {
	t.Helper()
	def, raw, err := asmtypes.LoadFromYAML([]byte(yamlSrc))
	if err != nil {
		t.Fatalf("LoadFromYAML: %v", err)
	}
	if err := s.SaveDefinition(context.Background(), def, raw); err != nil {
		t.Fatalf("SaveDefinition: %v", err)
	}
	return def
}

func newEngine(exec Executor) (*Engine, store.Store) {
	s := store.NewMemoryStore()
	bus := events.NewLocalBus()
	return NewEngine(s, bus, exec), s
}

func waitForStatus(t *testing.T, s store.Store, runID string, want asmtypes.RunStatus, timeout time.Duration) *asmtypes.WorkflowRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := s.GetRun(context.Background(), runID)
		if err == nil && run.Status == want {
			return run
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, _ := s.GetRun(context.Background(), runID)
	t.Fatalf("timed out waiting for status %q (current: %q)", want, run.Status)
	return nil
}

func waitForState(t *testing.T, s store.Store, runID, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := s.GetRun(context.Background(), runID)
		if err == nil && run.CurrentState == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, _ := s.GetRun(context.Background(), runID)
	t.Fatalf("timed out waiting for state %q (current: %q)", want, run.CurrentState)
}

// --- tests ---

func TestEngine_StartRun_InitialState(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, err := eng.StartRun(context.Background(), "two-state", "v1", nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if run.CurrentState != "start" {
		t.Errorf("current_state: got %q, want %q", run.CurrentState, "start")
	}
	if run.Status != asmtypes.RunRunning {
		t.Errorf("status: got %q, want %q", run.Status, asmtypes.RunRunning)
	}
	if run.ID == "" {
		t.Error("expected non-empty run ID")
	}
}

func TestEngine_StartRun_UnknownWorkflow(t *testing.T) {
	eng, _ := newEngine(fixedOutput("", nil))
	_, err := eng.StartRun(context.Background(), "no-such", "v1", nil)
	if err == nil {
		t.Fatal("expected error for unknown workflow, got nil")
	}
}

func TestEngine_ExternalTrigger_AdvancesToTerminal(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, _ := eng.StartRun(context.Background(), "two-state", "v1", nil)

	// The engine is parked at "start" (no agent), waiting for an external trigger.
	err := eng.Trigger(context.Background(), asmtypes.TriggerRequest{
		RunID:   run.ID,
		Trigger: "finish",
	})
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	final, _ := s.GetRun(context.Background(), run.ID)
	if final.CurrentState != "done" {
		t.Errorf("current_state: got %q, want %q", final.CurrentState, "done")
	}
}

func TestEngine_ExternalTrigger_WithPayload(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, _ := eng.StartRun(context.Background(), "two-state", "v1", nil)

	err := eng.Trigger(context.Background(), asmtypes.TriggerRequest{
		RunID:   run.ID,
		Trigger: "finish",
		Payload: map[string]interface{}{"note": "manual approval"},
	})
	if err != nil {
		t.Fatalf("Trigger with payload: %v", err)
	}

	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)
}

func TestEngine_ExternalTrigger_UnknownTrigger(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, _ := eng.StartRun(context.Background(), "two-state", "v1", nil)
	err := eng.Trigger(context.Background(), asmtypes.TriggerRequest{
		RunID:   run.ID,
		Trigger: "no-such-trigger",
	})
	if err == nil {
		t.Fatal("expected error for unknown trigger, got nil")
	}
}

func TestEngine_Trigger_InactiveRun(t *testing.T) {
	eng, _ := newEngine(fixedOutput("", nil))
	err := eng.Trigger(context.Background(), asmtypes.TriggerRequest{
		RunID:   "ghost-run",
		Trigger: "finish",
	})
	if err == nil {
		t.Fatal("expected error for inactive run, got nil")
	}
}

func TestEngine_AgentState_FiresTrigger(t *testing.T) {
	exec := fixedOutput("next", map[string]interface{}{"classified": true})
	eng, s := newEngine(exec)
	loadWorkflow(t, s, agentWorkflowYAML)

	run, err := eng.StartRun(context.Background(), "agent-wf", "v1", nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	// Agent should run automatically and advance to "review".
	waitForState(t, s, run.ID, "review", 2*time.Second)

	updated, _ := s.GetRun(context.Background(), run.ID)
	if updated.CurrentState != "review" {
		t.Errorf("current_state: got %q, want %q", updated.CurrentState, "review")
	}
	// Blackboard update from agent should be persisted.
	if updated.Blackboard["classified"] != true {
		t.Errorf("blackboard[classified]: got %v, want true", updated.Blackboard["classified"])
	}
}

func TestEngine_AgentState_BlackboardUpdated(t *testing.T) {
	exec := fixedOutput("next", map[string]interface{}{"result": "ok", "confidence": float64(0.95)})
	eng, s := newEngine(exec)
	loadWorkflow(t, s, agentWorkflowYAML)

	run, _ := eng.StartRun(context.Background(), "agent-wf", "v1", nil)
	waitForState(t, s, run.ID, "review", 2*time.Second)

	updated, _ := s.GetRun(context.Background(), run.ID)
	if updated.Blackboard["result"] != "ok" {
		t.Errorf("blackboard[result]: got %v, want 'ok'", updated.Blackboard["result"])
	}
}

func TestEngine_TransitionRecorded(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, _ := eng.StartRun(context.Background(), "two-state", "v1", nil)
	_ = eng.Trigger(context.Background(), asmtypes.TriggerRequest{RunID: run.ID, Trigger: "finish"})
	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	transitions, err := s.ListTransitions(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ListTransitions: %v", err)
	}
	if len(transitions) == 0 {
		t.Fatal("expected at least one transition record")
	}
	last := transitions[len(transitions)-1]
	if last.ToState != "done" {
		t.Errorf("last transition to_state: got %q, want %q", last.ToState, "done")
	}
	if last.Trigger != "finish" {
		t.Errorf("last transition trigger: got %q, want %q", last.Trigger, "finish")
	}
}

func TestEngine_GuardedTransition_Pass(t *testing.T) {
	exec := fixedOutput("evaluated", map[string]interface{}{"score": float64(90)})
	eng, s := newEngine(exec)
	loadWorkflow(t, s, guardedWorkflowYAML)

	run, _ := eng.StartRun(context.Background(), "guarded-wf", "v1", map[string]interface{}{"score": float64(0)})
	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	final, _ := s.GetRun(context.Background(), run.ID)
	if final.CurrentState != "approved" {
		t.Errorf("expected 'approved', got %q", final.CurrentState)
	}
}

func TestEngine_GuardedTransition_Fail(t *testing.T) {
	exec := fixedOutput("evaluated", map[string]interface{}{"score": float64(50)})
	eng, s := newEngine(exec)
	loadWorkflow(t, s, guardedWorkflowYAML)

	run, _ := eng.StartRun(context.Background(), "guarded-wf", "v1", map[string]interface{}{"score": float64(0)})
	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	final, _ := s.GetRun(context.Background(), run.ID)
	if final.CurrentState != "rejected" {
		t.Errorf("expected 'rejected', got %q", final.CurrentState)
	}
}

func TestEngine_HITL_ApprovesAndCompletes(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, hitlWorkflowYAML)

	run, _ := eng.StartRun(context.Background(), "hitl-wf", "v1", nil)

	// Advance to HITL state.
	_ = eng.Trigger(context.Background(), asmtypes.TriggerRequest{RunID: run.ID, Trigger: "submitted"})
	waitForStatus(t, s, run.ID, asmtypes.RunWaiting, 2*time.Second)

	// Verify HITL request was created.
	pending, _ := s.ListPendingHITL(context.Background())
	if len(pending) == 0 {
		t.Fatal("expected a pending HITL request")
	}

	// Signal approval.
	err := eng.SignalHITL(context.Background(), asmtypes.HITLSignal{
		RunID:      run.ID,
		Resolution: "approved",
		Resolver:   "alice",
	})
	if err != nil {
		t.Fatalf("SignalHITL: %v", err)
	}

	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	final, _ := s.GetRun(context.Background(), run.ID)
	if final.CurrentState != "approved" {
		t.Errorf("current_state: got %q, want %q", final.CurrentState, "approved")
	}
}

func TestEngine_HITL_RejectsAndCompletes(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, hitlWorkflowYAML)

	run, _ := eng.StartRun(context.Background(), "hitl-wf", "v1", nil)
	_ = eng.Trigger(context.Background(), asmtypes.TriggerRequest{RunID: run.ID, Trigger: "submitted"})
	waitForStatus(t, s, run.ID, asmtypes.RunWaiting, 2*time.Second)

	_ = eng.SignalHITL(context.Background(), asmtypes.HITLSignal{
		RunID:      run.ID,
		Resolution: "rejected",
		Resolver:   "bob",
	})

	waitForStatus(t, s, run.ID, asmtypes.RunComplete, 2*time.Second)

	final, _ := s.GetRun(context.Background(), run.ID)
	if final.CurrentState != "rejected" {
		t.Errorf("current_state: got %q, want %q", final.CurrentState, "rejected")
	}
}

func TestEngine_HITL_SignalInactiveRun(t *testing.T) {
	eng, _ := newEngine(fixedOutput("", nil))
	err := eng.SignalHITL(context.Background(), asmtypes.HITLSignal{
		RunID:      "ghost",
		Resolution: "approved",
		Resolver:   "alice",
	})
	if err == nil {
		t.Fatal("expected error signalling inactive run, got nil")
	}
}

func TestEngine_InputPropagatedToBlackboard(t *testing.T) {
	eng, s := newEngine(fixedOutput("", nil))
	loadWorkflow(t, s, twoStateYAML)

	run, _ := eng.StartRun(context.Background(), "two-state", "v1", map[string]interface{}{
		"invoice_id": "INV-123",
	})

	stored, _ := s.GetRun(context.Background(), run.ID)
	if stored.Blackboard["invoice_id"] != "INV-123" {
		t.Errorf("blackboard[invoice_id]: got %v, want 'INV-123'", stored.Blackboard["invoice_id"])
	}
}
