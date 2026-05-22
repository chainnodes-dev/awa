package orchestrator

import (
	"context"
	"testing"

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

// --- workflow fixtures ---

// twoStateWorkflow: initial -> terminal, no agent, manual trigger.
const twoStateYAML = `
apiVersion: chainnodes/v1
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

// hitlWorkflow: initial -> hitl state (waits for human) -> terminal.
const hitlWorkflowYAML = `
apiVersion: chainnodes/v1
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

func TestEvalGuard_AllowUndefined(t *testing.T) {
	bb := map[string]interface{}{"known": 100}
	// Referencing "unknown" which is not in bb.
	// Without AllowUndefinedVariables, this would fail to compile.
	pass, err := EvalGuard("known > 50 && unknown == nil", bb)
	if err != nil {
		t.Fatalf("EvalGuard failed: %v", err)
	}
	if !pass {
		t.Errorf("Expected guard to pass, got false")
	}
}

func TestEvalScript_AllowUndefined(t *testing.T) {
	bb := map[string]interface{}{"known": "value"}
	state := &asmtypes.StateDef{
		Name: "test",
		Script: &asmtypes.ScriptDef{
			Trigger: "unknown == nil ? 'next' : 'error'",
			Updates: map[string]string{
				"new_field": "unknown",
			},
		},
	}
	out, err := EvalScript(state, bb)
	if err != nil {
		t.Fatalf("EvalScript failed: %v", err)
	}
	if out.Trigger != "next" {
		t.Errorf("Trigger: got %q, want %q", out.Trigger, "next")
	}
	if val, ok := out.BlackboardUpdates["new_field"]; !ok || val != nil {
		t.Errorf("Update 'new_field': got %v, want nil", val)
	}
}
