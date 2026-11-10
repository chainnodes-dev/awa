package temporal

import (
	"testing"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestMatchTrigger(t *testing.T) {
	tests := []struct {
		candidate string
		requested string
		expected  bool
	}{
		{"approve", "approve", true},
		{"Approve", "approve", true},
		{"APPROVED", "approve", true},
		{"submit", "approve", true},
		{"accept", "approve", true},
		{"approved", "submit", true},
		{"reject", "reject", true},
		{"rejected", "reject", true},
		{"deny", "reject", true},
		{"cancel", "reject", true},
		{"approve", "reject", false},
		{"custom_action", "custom_action", true},
		{"custom_action", "other_action", false},
	}

	for _, tc := range tests {
		got := matchTrigger(tc.candidate, tc.requested)
		if got != tc.expected {
			t.Errorf("matchTrigger(%q, %q) = %v; want %v", tc.candidate, tc.requested, got, tc.expected)
		}
	}
}

func TestApplyTransition_HITL_SynonymsAndFallbacks(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		Metadata: asmtypes.WorkflowMeta{Name: "test-wf", Version: "v1"},
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial},
			{Name: "REVIEW", Type: asmtypes.StateHITL},
			{Name: "VERIFY", Type: asmtypes.StateCode},
			{Name: "REJECTED", Type: asmtypes.StateTerminal},
		},
		Transitions: []asmtypes.Transition{
			{From: "REVIEW", To: "VERIFY", Trigger: "submit"},
			{From: "REVIEW", To: "REJECTED", Trigger: "reject"},
		},
	}

	bb := make(map[string]interface{})

	// 1. Exact match "submit"
	next, err := applyTransition(def, "REVIEW", "submit", bb)
	if err != nil || len(next) != 1 || next[0] != "VERIFY" {
		t.Fatalf("expected VERIFY on 'submit', got %v, err: %v", next, err)
	}

	// 2. Synonym match "approve" -> "submit"
	next, err = applyTransition(def, "REVIEW", "approve", bb)
	if err != nil || len(next) != 1 || next[0] != "VERIFY" {
		t.Fatalf("expected VERIFY on 'approve' (synonym for 'submit'), got %v, err: %v", next, err)
	}

	// 3. Synonym match "accept" -> "submit"
	next, err = applyTransition(def, "REVIEW", "accept", bb)
	if err != nil || len(next) != 1 || next[0] != "VERIFY" {
		t.Fatalf("expected VERIFY on 'accept' (synonym for 'submit'), got %v, err: %v", next, err)
	}

	// 4. Synonym match "rejected" -> "reject"
	next, err = applyTransition(def, "REVIEW", "rejected", bb)
	if err != nil || len(next) != 1 || next[0] != "REJECTED" {
		t.Fatalf("expected REJECTED on 'rejected', got %v, err: %v", next, err)
	}
}

func TestApplyTransition_HITL_DefaultEmptyTrigger(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		Metadata: asmtypes.WorkflowMeta{Name: "test-wf", Version: "v1"},
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial},
			{Name: "REVIEW", Type: asmtypes.StateHITL},
			{Name: "NEXT_STEP", Type: asmtypes.StateCode},
			{Name: "REJECTED", Type: asmtypes.StateTerminal},
		},
		Transitions: []asmtypes.Transition{
			{From: "REVIEW", To: "NEXT_STEP", Trigger: ""}, // default transition without trigger
			{From: "REVIEW", To: "REJECTED", Trigger: "reject"},
		},
	}

	bb := make(map[string]interface{})

	// When user approves, it should take the default forward transition
	next, err := applyTransition(def, "REVIEW", "approve", bb)
	if err != nil || len(next) != 1 || next[0] != "NEXT_STEP" {
		t.Fatalf("expected NEXT_STEP on 'approve' with default trigger '', got %v, err: %v", next, err)
	}
}

func TestApplyTransition_HITL_SingleForwardFallback(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		Metadata: asmtypes.WorkflowMeta{Name: "test-wf", Version: "v1"},
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial},
			{Name: "REVIEW", Type: asmtypes.StateHITL},
			{Name: "PROCESS", Type: asmtypes.StateCode},
			{Name: "REJECTED", Type: asmtypes.StateTerminal},
		},
		Transitions: []asmtypes.Transition{
			{From: "REVIEW", To: "PROCESS", Trigger: "forward_data"}, // custom forward trigger
			{From: "REVIEW", To: "REJECTED", Trigger: "reject"},
		},
	}

	bb := make(map[string]interface{})

	// When user clicks 'approve', since 'PROCESS' is the only non-rejection path, it should fallback to it
	next, err := applyTransition(def, "REVIEW", "approve", bb)
	if err != nil || len(next) != 1 || next[0] != "PROCESS" {
		t.Fatalf("expected PROCESS on 'approve' fallback, got %v, err: %v", next, err)
	}
}
