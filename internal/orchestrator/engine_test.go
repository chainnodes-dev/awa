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
    trigger: approved
  - from: human_review
    to: rejected
    trigger: rejected
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

func newEngine() (*Engine, store.Store, *MockTemporalClient) {
	s := store.NewMemoryStore()
	bus := events.NewLocalBus()
	mock := &MockTemporalClient{}
	return NewEngine(s, bus, mock), s, mock
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

func TestEngine_StartRun(t *testing.T) {
	eng, s, mock := newEngine()
	loadWorkflow(t, s, twoStateYAML)

	var startCalled bool
	mock.OnStartWorkflow = func(run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (string, error) {
		startCalled = true
		return "temporal-id-123", nil
	}

	run, err := eng.StartRun(context.Background(), "two-state", "v1", nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	if !startCalled {
		t.Errorf("StartWorkflow was not called")
	}
	if run.TemporalID != "temporal-id-123" {
		t.Errorf("TemporalID: got %q, want %q", run.TemporalID, "temporal-id-123")
	}
}

func TestEngine_StartRun_UnknownWorkflow(t *testing.T) {
	eng, _, _ := newEngine()
	_, err := eng.StartRun(context.Background(), "no-such", "v1", nil)
	if err == nil {
		t.Fatal("expected error for unknown workflow, got nil")
	}
}

func TestEngine_Trigger(t *testing.T) {
	eng, s, mock := newEngine()
	loadWorkflow(t, s, twoStateYAML)

	// Mock run with Temporal ID
	runID := "run-123"
	_ = s.CreateRun(context.Background(), &asmtypes.WorkflowRun{
		ID:         runID,
		TemporalID: "temporal-123",
		Status:     asmtypes.RunRunning,
	})

	var triggerCalled bool
	mock.OnSendTriggerSignal = func(temporalID, trigger string, payload map[string]interface{}) error {
		triggerCalled = true
		if temporalID != "temporal-123" {
			t.Errorf("temporalID: got %q, want %q", temporalID, "temporal-123")
		}
		if trigger != "finish" {
			t.Errorf("trigger: got %q, want %q", trigger, "finish")
		}
		return nil
	}

	err := eng.Trigger(context.Background(), asmtypes.TriggerRequest{
		RunID:   runID,
		Trigger: "finish",
	})
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	if !triggerCalled {
		t.Errorf("SendTriggerSignal was not called")
	}
}

func TestEngine_SignalHITL(t *testing.T) {
	eng, s, mock := newEngine()
	loadWorkflow(t, s, hitlWorkflowYAML)

	runID := "run-hitl"
	_ = s.CreateRun(context.Background(), &asmtypes.WorkflowRun{
		ID:         runID,
		TemporalID: "temporal-hitl",
		Status:     asmtypes.RunWaiting,
	})
	_ = s.CreateHITL(context.Background(), &asmtypes.HITLRequest{
		RunID: runID, StateName: "human_review",
	})

	var signalCalled bool
	mock.OnSendHITLSignal = func(temporalID string, sig asmtypes.HITLSignal) error {
		signalCalled = true
		return nil
	}

	err := eng.SignalHITL(context.Background(), asmtypes.HITLSignal{
		RunID:      runID,
		Resolution: "approved",
	})
	if err != nil {
		t.Fatalf("SignalHITL: %v", err)
	}

	if !signalCalled {
		t.Errorf("SendHITLSignal was not called")
	}
}
