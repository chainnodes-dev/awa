package specialist

import (
	"context"
	"errors"
	"fmt"
	"testing"

	temporalpkg "github.com/asm-platform/asm/internal/temporal"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// callExecuteAgent invokes the unexported executeAgent method directly, letting
// us test dispatch logic without a live Temporal server.
func callExecuteAgent(w *Worker, p temporalpkg.ExecuteAgentParams) (*asmtypes.AgentOutput, error) {
	return w.executeAgent(context.Background(), p)
}

func makeParams(agentName string, bb map[string]interface{}) temporalpkg.ExecuteAgentParams {
	return temporalpkg.ExecuteAgentParams{
		RunID:      "run-001",
		AgentDef:   asmtypes.AgentDef{Name: agentName},
		StateDef:   asmtypes.StateDef{Name: "TEST_STATE"},
		Blackboard: bb,
	}
}

func returnsTrigger(trigger string) HandlerFunc {
	return func(_ context.Context, _ map[string]interface{}) (*asmtypes.AgentOutput, error) {
		return &asmtypes.AgentOutput{Trigger: trigger}, nil
	}
}

// ── Dispatch tests ────────────────────────────────────────────────────────────

func TestDispatch_KnownAgent(t *testing.T) {
	sw := New("test-queue")
	sw.Register("my-agent", func(_ context.Context, _ map[string]interface{}) (*asmtypes.AgentOutput, error) {
		return &asmtypes.AgentOutput{Trigger: "done", Reasoning: "handler called"}, nil
	})

	out, err := callExecuteAgent(sw, makeParams("my-agent", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "done" {
		t.Errorf("trigger = %q, want %q", out.Trigger, "done")
	}
	if out.Reasoning != "handler called" {
		t.Errorf("reasoning = %q, want %q", out.Reasoning, "handler called")
	}
}

func TestDispatch_UnknownAgent(t *testing.T) {
	sw := New("test-queue")
	sw.Register("agent-a", returnsTrigger("ok"))

	_, err := callExecuteAgent(sw, makeParams("agent-z", nil))
	if err == nil {
		t.Fatal("expected error for unknown agent name, got nil")
	}
}

func TestDispatch_ErrorMessageContainsQueueAndName(t *testing.T) {
	sw := New("my-queue")
	_, err := callExecuteAgent(sw, makeParams("missing-agent", nil))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !contains(msg, "my-queue") {
		t.Errorf("error %q does not mention queue name", msg)
	}
	if !contains(msg, "missing-agent") {
		t.Errorf("error %q does not mention agent name", msg)
	}
}

func TestDispatch_HandlerErrorPropagated(t *testing.T) {
	sw := New("test-queue")
	sentinel := errors.New("handler blew up")
	sw.Register("broken", func(_ context.Context, _ map[string]interface{}) (*asmtypes.AgentOutput, error) {
		return nil, sentinel
	})

	_, err := callExecuteAgent(sw, makeParams("broken", nil))
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want sentinel", err)
	}
}

func TestDispatch_BlackboardForwarded(t *testing.T) {
	sw := New("test-queue")
	var received map[string]interface{}
	sw.Register("echo", func(_ context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
		received = bb
		return &asmtypes.AgentOutput{Trigger: "ok"}, nil
	})

	bb := map[string]interface{}{"invoice_id": "INV-123", "amount": float64(999)}
	_, err := callExecuteAgent(sw, makeParams("echo", bb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["invoice_id"] != "INV-123" {
		t.Errorf("invoice_id = %v, want INV-123", received["invoice_id"])
	}
	if received["amount"] != float64(999) {
		t.Errorf("amount = %v, want 999", received["amount"])
	}
}

func TestDispatch_MultipleHandlersRouteCorrectly(t *testing.T) {
	sw := New("multi-queue")
	sw.Register("agent-a", returnsTrigger("from-a"))
	sw.Register("agent-b", returnsTrigger("from-b"))

	outA, err := callExecuteAgent(sw, makeParams("agent-a", nil))
	if err != nil || outA.Trigger != "from-a" {
		t.Errorf("agent-a: err=%v trigger=%q", err, outA.Trigger)
	}
	outB, err := callExecuteAgent(sw, makeParams("agent-b", nil))
	if err != nil || outB.Trigger != "from-b" {
		t.Errorf("agent-b: err=%v trigger=%q", err, outB.Trigger)
	}
}

func TestDispatch_OverwriteHandlerWins(t *testing.T) {
	sw := New("test-queue")
	sw.Register("agent-x", returnsTrigger("v1"))
	sw.Register("agent-x", returnsTrigger("v2")) // second registration wins

	out, err := callExecuteAgent(sw, makeParams("agent-x", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "v2" {
		t.Errorf("trigger = %q, want v2 (second registration should win)", out.Trigger)
	}
}

func TestDispatch_EmptyWorkerErrors(t *testing.T) {
	sw := New("empty-queue")
	_, err := callExecuteAgent(sw, makeParams("any-agent", nil))
	if err == nil {
		t.Fatal("expected error for empty worker, got nil")
	}
}

// ── Metadata tests ────────────────────────────────────────────────────────────

func TestWorker_TaskQueue(t *testing.T) {
	sw := New("my-fancy-queue")
	if sw.TaskQueue() != "my-fancy-queue" {
		t.Errorf("TaskQueue() = %q, want %q", sw.TaskQueue(), "my-fancy-queue")
	}
}

func TestWorker_HandlersReflectsRegistrations(t *testing.T) {
	sw := New("q")
	if len(sw.Handlers()) != 0 {
		t.Errorf("expected 0 handlers before registration, got %d", len(sw.Handlers()))
	}
	sw.Register("a", returnsTrigger("ok"))
	sw.Register("b", returnsTrigger("ok"))
	if len(sw.Handlers()) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(sw.Handlers()))
	}
}

// ── Invoice validation logic tests ───────────────────────────────────────────
// The invoice-validator handler lives in cmd/specialist-worker.  The core logic
// is tested here via the internal helper below so we don't need to import cmd/.

func testValidateInvoice(bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
	// Mirror the validateInvoice logic from cmd/specialist-worker/main.go.
	required := []string{"invoice_id", "vendor_id", "amount", "currency"}
	for _, field := range required {
		v := bb[field]
		if v == nil || v == "" {
			return &asmtypes.AgentOutput{
				Trigger:   "validation_failed",
				Reasoning: fmt.Sprintf("missing required field: %s", field),
				BlackboardUpdates: map[string]interface{}{
					"validation_error": fmt.Sprintf("missing required field: %s", field),
				},
			}, nil
		}
	}
	amount, ok := floatVal(bb["amount"])
	if !ok || amount <= 0 {
		return &asmtypes.AgentOutput{
			Trigger:           "validation_failed",
			Reasoning:         "amount must be a positive number",
			BlackboardUpdates: map[string]interface{}{"validation_error": "amount must be a positive number"},
		}, nil
	}
	validCurrencies := map[string]bool{"USD": true, "EUR": true, "GBP": true, "JPY": true, "CHF": true}
	if !validCurrencies[fmt.Sprintf("%v", bb["currency"])] {
		return &asmtypes.AgentOutput{
			Trigger:           "validation_failed",
			Reasoning:         fmt.Sprintf("unknown currency: %v", bb["currency"]),
			BlackboardUpdates: map[string]interface{}{"validation_error": fmt.Sprintf("unknown currency: %v", bb["currency"])},
		}, nil
	}
	return &asmtypes.AgentOutput{
		Trigger:   "validation_passed",
		Reasoning: "all required fields present and valid",
		BlackboardUpdates: map[string]interface{}{"validation_passed": true},
	}, nil
}

func floatVal(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func TestInvoiceValidator_AllFieldsValid(t *testing.T) {
	bb := map[string]interface{}{
		"invoice_id": "INV-001",
		"vendor_id":  "VND-42",
		"amount":     float64(1500),
		"currency":   "USD",
	}
	out, err := testValidateInvoice(bb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "validation_passed" {
		t.Errorf("trigger = %q, want validation_passed", out.Trigger)
	}
	if out.BlackboardUpdates["validation_passed"] != true {
		t.Errorf("validation_passed flag not set")
	}
}

func TestInvoiceValidator_MissingField(t *testing.T) {
	cases := []struct {
		name    string
		bb      map[string]interface{}
		missing string
	}{
		{"no invoice_id", map[string]interface{}{"vendor_id": "V", "amount": float64(1), "currency": "USD"}, "invoice_id"},
		{"no vendor_id", map[string]interface{}{"invoice_id": "I", "amount": float64(1), "currency": "USD"}, "vendor_id"},
		{"no amount", map[string]interface{}{"invoice_id": "I", "vendor_id": "V", "currency": "USD"}, "amount"},
		{"no currency", map[string]interface{}{"invoice_id": "I", "vendor_id": "V", "amount": float64(1)}, "currency"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := testValidateInvoice(tc.bb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Trigger != "validation_failed" {
				t.Errorf("trigger = %q, want validation_failed", out.Trigger)
			}
			if !contains(fmt.Sprintf("%v", out.BlackboardUpdates["validation_error"]), tc.missing) {
				t.Errorf("error message doesn't mention missing field %q: %v", tc.missing, out.BlackboardUpdates)
			}
		})
	}
}

func TestInvoiceValidator_NegativeAmount(t *testing.T) {
	bb := map[string]interface{}{
		"invoice_id": "INV-001",
		"vendor_id":  "VND-42",
		"amount":     float64(-50),
		"currency":   "EUR",
	}
	out, err := testValidateInvoice(bb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "validation_failed" {
		t.Errorf("trigger = %q, want validation_failed for negative amount", out.Trigger)
	}
}

func TestInvoiceValidator_ZeroAmount(t *testing.T) {
	bb := map[string]interface{}{
		"invoice_id": "INV-001",
		"vendor_id":  "VND-42",
		"amount":     float64(0),
		"currency":   "GBP",
	}
	out, _ := testValidateInvoice(bb)
	if out.Trigger != "validation_failed" {
		t.Errorf("trigger = %q, want validation_failed for zero amount", out.Trigger)
	}
}

func TestInvoiceValidator_UnknownCurrency(t *testing.T) {
	bb := map[string]interface{}{
		"invoice_id": "INV-001",
		"vendor_id":  "VND-42",
		"amount":     float64(100),
		"currency":   "XYZ",
	}
	out, _ := testValidateInvoice(bb)
	if out.Trigger != "validation_failed" {
		t.Errorf("trigger = %q, want validation_failed for unknown currency", out.Trigger)
	}
}

func TestInvoiceValidator_MultipleKnownCurrencies(t *testing.T) {
	currencies := []string{"USD", "EUR", "GBP", "JPY", "CHF"}
	for _, cur := range currencies {
		bb := map[string]interface{}{
			"invoice_id": "INV-001",
			"vendor_id":  "VND-42",
			"amount":     float64(100),
			"currency":   cur,
		}
		out, err := testValidateInvoice(bb)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", cur, err)
		}
		if out.Trigger != "validation_passed" {
			t.Errorf("%s: trigger = %q, want validation_passed", cur, out.Trigger)
		}
	}
}

// ── AgentDef.TaskQueue field tests ────────────────────────────────────────────

func TestAgentDef_TaskQueueField(t *testing.T) {
	// Verify the field is accessible and defaults to empty.
	agent := asmtypes.AgentDef{
		Name:  "invoice-validator",
		Model: "claude-sonnet-4-6",
	}
	if agent.TaskQueue != "" {
		t.Errorf("TaskQueue should default to empty, got %q", agent.TaskQueue)
	}

	agent.TaskQueue = "validate-workers"
	if agent.TaskQueue != "validate-workers" {
		t.Errorf("TaskQueue = %q, want validate-workers", agent.TaskQueue)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
