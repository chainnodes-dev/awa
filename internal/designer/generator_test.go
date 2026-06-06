package designer

import (
	"testing"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestEnsureSingleUnconditionalInitialTransition_NoAdjustment(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: ""},
		},
	}

	EnsureSingleUnconditionalInitialTransition(def)

	if len(def.States) != 2 {
		t.Errorf("Expected 2 states, got %d", len(def.States))
	}
	if def.States[0].Name != "START" || def.States[0].Type != asmtypes.StateInitial {
		t.Errorf("Initial state should not have changed")
	}
}

func TestEnsureSingleUnconditionalInitialTransition_WithGuard(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial, Position: &asmtypes.Position{X: 200, Y: 100}},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: "", Guard: "amount > 100"},
		},
	}

	EnsureSingleUnconditionalInitialTransition(def)

	// A new initial state should be prepended
	if len(def.States) != 3 {
		t.Fatalf("Expected 3 states, got %d", len(def.States))
	}
	if def.States[0].Name != "START_NODE" || def.States[0].Type != asmtypes.StateInitial {
		t.Errorf("Expected first state to be 'START_NODE' and initial, got %v", def.States[0])
	}
	if def.States[1].Name != "START" || def.States[1].Type != asmtypes.StatePrompt {
		t.Errorf("Expected original START state to be demoted to prompt, got %v", def.States[1])
	}

	// Verify transition from START_NODE to START is added
	foundNewTransition := false
	for _, tr := range def.Transitions {
		if tr.From == "START_NODE" && tr.To == "START" && tr.Trigger == "done" && tr.Guard == "" {
			foundNewTransition = true
			break
		}
	}
	if !foundNewTransition {
		t.Errorf("Expected unconditional transition from START_NODE to START with trigger 'done'")
	}
}

func TestEnsureSingleUnconditionalInitialTransition_MultipleTransitions(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{Name: "START", Type: asmtypes.StateInitial},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
			{Name: "STEP2", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: "done", Guard: "amount > 100"},
			{From: "START", To: "STEP2", Trigger: "done", Guard: "amount <= 100"},
		},
	}

	EnsureSingleUnconditionalInitialTransition(def)

	if len(def.States) != 4 {
		t.Fatalf("Expected 4 states, got %d", len(def.States))
	}
	if def.States[0].Name != "START_NODE" || def.States[0].Type != asmtypes.StateInitial {
		t.Errorf("Expected first state to be 'START_NODE' and initial, got %v", def.States[0])
	}

	// Verify transition from START_NODE to START is added
	foundNewTransition := false
	for _, tr := range def.Transitions {
		if tr.From == "START_NODE" && tr.To == "START" && tr.Trigger == "done" && tr.Guard == "" {
			foundNewTransition = true
			break
		}
	}
	if !foundNewTransition {
		t.Errorf("Expected unconditional transition from START_NODE to START with trigger 'done'")
	}
}

func TestEnsureTimeoutNodes_Injected(t *testing.T) {
	def := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{Name: "STEP1", Type: asmtypes.StatePrompt, Timeout: "2m", OnTimeout: "timeout"},
		},
	}

	EnsureTimeoutNodes(def)

	if len(def.States) != 2 {
		t.Fatalf("Expected 2 states, got %d", len(def.States))
	}
	if def.States[1].Name != "timeout" || def.States[1].Type != asmtypes.StateTerminal || !def.States[1].IsTimeoutNode {
		t.Errorf("Expected timeout node to be injected, got %v", def.States[1])
	}

	// Check transition
	if len(def.Transitions) != 1 {
		t.Fatalf("Expected 1 transition, got %d", len(def.Transitions))
	}
	t1 := def.Transitions[0]
	if t1.From != "STEP1" || t1.To != "timeout" || t1.Trigger != "timeout" {
		t.Errorf("Expected transition from STEP1 to timeout with trigger timeout, got %v", t1)
	}
}

func TestEnsureSingleUnconditionalInitialTransition_DemotedTypesAndAgent(t *testing.T) {
	// Case 1: Initial state with script -> should become StateScript
	def1 := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{
				Name: "START",
				Type: asmtypes.StateInitial,
				Script: &asmtypes.ScriptDef{
					Trigger: "'done'",
				},
			},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: "", Guard: "amount > 100"},
		},
	}
	EnsureSingleUnconditionalInitialTransition(def1)
	if def1.States[1].Name != "START" || def1.States[1].Type != asmtypes.StateScript {
		t.Errorf("Expected script state type, got %v", def1.States[1].Type)
	}

	// Case 2: Initial state with no agent, code, or script -> should become StatePrompt and get default agent
	def2 := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{
				Name: "START",
				Type: asmtypes.StateInitial,
			},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: "", Guard: "amount > 100"},
		},
		Agents: []asmtypes.AgentDef{
			{Name: "custom-agent"},
		},
	}
	EnsureSingleUnconditionalInitialTransition(def2)
	demoted := def2.States[1]
	if demoted.Name != "START" || demoted.Type != asmtypes.StatePrompt {
		t.Errorf("Expected prompt state type, got %v", demoted.Type)
	}
	if demoted.Agent != "custom-agent" {
		t.Errorf("Expected agent to be 'custom-agent', got %q", demoted.Agent)
	}
	// Case 3: Initial state with empty agents slice -> should become StatePrompt, get default agent 'primary-agent', and declare it in Agents list
	def3 := &asmtypes.WorkflowDef{
		States: []asmtypes.StateDef{
			{
				Name: "START",
				Type: asmtypes.StateInitial,
			},
			{Name: "STEP1", Type: asmtypes.StatePrompt},
		},
		Transitions: []asmtypes.Transition{
			{From: "START", To: "STEP1", Trigger: "", Guard: "amount > 100"},
		},
		Agents: []asmtypes.AgentDef{},
	}
	EnsureSingleUnconditionalInitialTransition(def3)
	demoted3 := def3.States[1]
	if demoted3.Name != "START" || demoted3.Type != asmtypes.StatePrompt {
		t.Errorf("Expected prompt state type, got %v", demoted3.Type)
	}
	if demoted3.Agent != "primary-agent" {
		t.Errorf("Expected agent to be 'primary-agent', got %q", demoted3.Agent)
	}
	if len(def3.Agents) != 1 || def3.Agents[0].Name != "primary-agent" {
		t.Errorf("Expected 'primary-agent' to be declared in Agents slice, got %+v", def3.Agents)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Invoice Enrichment & Customer Default Risk Assessment", "invoice-enrichment-customer-default-risk-assessment"},
		{"!!!", "generated-workflow"},
		{"a---b", "a-b"},
		{"", "generated-workflow"},
		{"This-is-a-very-long-name-that-exceeds-sixty-three-characters-limit-indeed", "this-is-a-very-long-name-that-exceeds-sixty-three-characters-li"},
	}

	for _, tc := range tests {
		got := slugify(tc.input)
		if got != tc.expected {
			t.Errorf("slugify(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestWorkflowEditorSetMetadataSlugification(t *testing.T) {
	editor := NewWorkflowEditor(nil)
	editor.SetMetadata("My Workflow Name & Details", "Abstract info", "Detailed desc")
	if editor.Def.Metadata.Name != "my-workflow-name-details" {
		t.Errorf("Expected slugified metadata name, got %q", editor.Def.Metadata.Name)
	}
}

func TestWorkflowEditorAssigneeAndFormSchema(t *testing.T) {
	editor := NewWorkflowEditor(nil)
	editor.AddState("MY_HITL_STATE", asmtypes.StateHITL, "Review details", "Ask compliance officer")
	
	err := editor.SetStateAssignee("MY_HITL_STATE", "compliance-officer")
	if err != nil {
		t.Fatalf("SetStateAssignee failed: %v", err)
	}
	
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"approved": map[string]interface{}{
				"type": "boolean",
			},
		},
	}
	err = editor.SetStateFormSchema("MY_HITL_STATE", schema)
	if err != nil {
		t.Fatalf("SetStateFormSchema failed: %v", err)
	}
	
	state := editor.Def.StateByName("MY_HITL_STATE")
	if state == nil {
		t.Fatalf("State not found")
	}
	if state.Assignee != "compliance-officer" {
		t.Errorf("Expected assignee compliance-officer, got %q", state.Assignee)
	}
	if state.FormSchema == nil || state.FormSchema["type"] != "object" {
		t.Errorf("Expected form schema type object, got %v", state.FormSchema)
	}
}
