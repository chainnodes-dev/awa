package store

import (
	"context"
	"testing"
	"time"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// helpers

func testDef(name, version string) *asmtypes.WorkflowDef {
	return &asmtypes.WorkflowDef{
		APIVersion: "chainnodes/v1",
		Kind:       "Workflow",
		Metadata:   asmtypes.WorkflowMeta{Name: name, Version: version},
		States: []asmtypes.StateDef{
			{Name: "start", Type: asmtypes.StateInitial},
			{Name: "done", Type: asmtypes.StateTerminal},
		},
		Transitions: []asmtypes.Transition{
			{From: "start", To: "done", Trigger: "finish"},
		},
	}
}

func testRun(id, wfName string) *asmtypes.WorkflowRun {
	return &asmtypes.WorkflowRun{
		ID:              id,
		WorkflowName:    wfName,
		WorkflowVersion: "v1",
		Status:          asmtypes.RunRunning,
		CurrentState:    "start",
		Blackboard:      map[string]interface{}{},
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// --- WorkflowStore ---

func TestMemoryStore_SaveAndGetDefinition(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	def := testDef("my-wf", "v1")

	if err := s.SaveDefinition(ctx, def, "raw-yaml"); err != nil {
		t.Fatalf("SaveDefinition: %v", err)
	}

	got, yaml, err := s.GetDefinition(ctx, "my-wf", "v1")
	if err != nil {
		t.Fatalf("GetDefinition: %v", err)
	}
	if got.Metadata.Name != "my-wf" {
		t.Errorf("name: got %q, want %q", got.Metadata.Name, "my-wf")
	}
	if yaml != "raw-yaml" {
		t.Errorf("yaml source: got %q, want %q", yaml, "raw-yaml")
	}
}

func TestMemoryStore_GetDefinition_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, _, err := s.GetDefinition(context.Background(), "missing", "v1")
	if err == nil {
		t.Fatal("expected error for missing definition, got nil")
	}
}

func TestMemoryStore_ListDefinitions(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_ = s.SaveDefinition(ctx, testDef("wf-a", "v1"), "")
	_ = s.SaveDefinition(ctx, testDef("wf-b", "v1"), "")

	defs, err := s.ListDefinitions(ctx, DefinitionFilter{})
	if err != nil {
		t.Fatalf("ListDefinitions: %v", err)
	}
	if len(defs) != 2 {
		t.Errorf("expected 2 definitions, got %d", len(defs))
	}
}

func TestMemoryStore_DeleteDefinition(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.SaveDefinition(ctx, testDef("wf-del", "v1"), "")

	if err := s.DeleteDefinition(ctx, "wf-del", "v1"); err != nil {
		t.Fatalf("DeleteDefinition: %v", err)
	}
	_, _, err := s.GetDefinition(ctx, "wf-del", "v1")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestMemoryStore_SaveDefinition_Overwrite(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.SaveDefinition(ctx, testDef("wf", "v1"), "original")
	_ = s.SaveDefinition(ctx, testDef("wf", "v1"), "updated")

	_, yaml, _ := s.GetDefinition(ctx, "wf", "v1")
	if yaml != "updated" {
		t.Errorf("expected updated yaml source, got %q", yaml)
	}
}

// --- RunStore ---

func TestMemoryStore_CreateAndGetRun(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	run := testRun("run-1", "my-wf")

	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := s.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.ID != "run-1" {
		t.Errorf("id: got %q, want %q", got.ID, "run-1")
	}
	if got.Status != asmtypes.RunRunning {
		t.Errorf("status: got %q, want %q", got.Status, asmtypes.RunRunning)
	}
}

func TestMemoryStore_CreateRun_GeneratesID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	run := testRun("", "my-wf") // empty ID

	if err := s.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.ID == "" {
		t.Error("expected ID to be auto-generated, got empty string")
	}
}

func TestMemoryStore_GetRun_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetRun(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for missing run, got nil")
	}
}

func TestMemoryStore_UpdateRun(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	run := testRun("run-2", "my-wf")
	_ = s.CreateRun(ctx, run)

	run.Status = asmtypes.RunComplete
	run.CurrentState = "done"
	if err := s.UpdateRun(ctx, run); err != nil {
		t.Fatalf("UpdateRun: %v", err)
	}

	got, _ := s.GetRun(ctx, "run-2")
	if got.Status != asmtypes.RunComplete {
		t.Errorf("status: got %q, want %q", got.Status, asmtypes.RunComplete)
	}
	if got.CurrentState != "done" {
		t.Errorf("current_state: got %q, want %q", got.CurrentState, "done")
	}
}

func TestMemoryStore_UpdateRun_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateRun(context.Background(), testRun("no-such", "wf"))
	if err == nil {
		t.Fatal("expected error updating non-existent run, got nil")
	}
}

func TestMemoryStore_ListRuns_NoFilter(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.CreateRun(ctx, testRun("r1", "wf-a"))
	_ = s.CreateRun(ctx, testRun("r2", "wf-b"))

	runs, err := s.ListRuns(ctx, RunFilter{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestMemoryStore_ListRuns_FilterByName(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.CreateRun(ctx, testRun("r1", "wf-a"))
	_ = s.CreateRun(ctx, testRun("r2", "wf-b"))
	_ = s.CreateRun(ctx, testRun("r3", "wf-a"))

	runs, _ := s.ListRuns(ctx, RunFilter{WorkflowName: "wf-a"})
	if len(runs) != 2 {
		t.Errorf("expected 2 runs for wf-a, got %d", len(runs))
	}
}

func TestMemoryStore_ListRuns_FilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	r1 := testRun("r1", "wf")
	r1.Status = asmtypes.RunComplete
	r2 := testRun("r2", "wf")
	r2.Status = asmtypes.RunFailed
	_ = s.CreateRun(ctx, r1)
	_ = s.CreateRun(ctx, r2)

	runs, _ := s.ListRuns(ctx, RunFilter{Status: asmtypes.RunComplete})
	if len(runs) != 1 {
		t.Errorf("expected 1 complete run, got %d", len(runs))
	}
}

func TestMemoryStore_GetRun_IsolatedFromMutations(t *testing.T) {
	// GetRun must return a deep copy — mutating the result must not affect the stored run.
	s := NewMemoryStore()
	ctx := context.Background()
	run := testRun("r1", "wf")
	run.Blackboard = map[string]interface{}{"key": "original"}
	_ = s.CreateRun(ctx, run)

	got, _ := s.GetRun(ctx, "r1")
	got.Blackboard["key"] = "mutated"

	got2, _ := s.GetRun(ctx, "r1")
	if got2.Blackboard["key"] != "original" {
		t.Errorf("deep copy violated: stored blackboard was mutated")
	}
}

// --- TransitionStore ---

func TestMemoryStore_RecordAndListTransitions(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rec := &asmtypes.TransitionRecord{
		RunID:     "run-x",
		FromState: "start",
		ToState:   "review",
		Trigger:   "submitted",
	}
	if err := s.RecordTransition(ctx, rec); err != nil {
		t.Fatalf("RecordTransition: %v", err)
	}
	if rec.ID == "" {
		t.Error("expected auto-generated transition ID")
	}
	if rec.Timestamp.IsZero() {
		t.Error("expected auto-generated timestamp")
	}

	transitions, err := s.ListTransitions(ctx, "run-x")
	if err != nil {
		t.Fatalf("ListTransitions: %v", err)
	}
	if len(transitions) != 1 {
		t.Errorf("expected 1 transition, got %d", len(transitions))
	}
	if transitions[0].ToState != "review" {
		t.Errorf("to_state: got %q, want %q", transitions[0].ToState, "review")
	}
}

func TestMemoryStore_ListTransitions_EmptyForUnknownRun(t *testing.T) {
	s := NewMemoryStore()
	transitions, err := s.ListTransitions(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions for unknown run, got %d", len(transitions))
	}
}

// --- HITLStore ---

func TestMemoryStore_CreateAndGetHITL(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	req := &asmtypes.HITLRequest{
		RunID:     "run-hitl",
		StateName: "approve",
		Assignee:  "alice",
		CreatedAt: time.Now(),
	}
	if err := s.CreateHITL(ctx, req); err != nil {
		t.Fatalf("CreateHITL: %v", err)
	}
	if req.ID == "" {
		t.Error("expected auto-generated HITL ID")
	}

	got, err := s.GetHITL(ctx, "run-hitl")
	if err != nil {
		t.Fatalf("GetHITL: %v", err)
	}
	if got.Assignee != "alice" {
		t.Errorf("assignee: got %q, want %q", got.Assignee, "alice")
	}
}

func TestMemoryStore_GetHITL_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetHITL(context.Background(), "no-run")
	if err == nil {
		t.Fatal("expected error for missing HITL, got nil")
	}
}

func TestMemoryStore_ResolveHITL(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	req := &asmtypes.HITLRequest{RunID: "run-h", StateName: "approve", CreatedAt: time.Now()}
	_ = s.CreateHITL(ctx, req)

	if err := s.ResolveHITL(ctx, "run-h", "approved", "bob"); err != nil {
		t.Fatalf("ResolveHITL: %v", err)
	}

	got, _ := s.GetHITL(ctx, "run-h")
	if !got.Resolved {
		t.Error("expected Resolved = true after ResolveHITL")
	}
	if got.Resolution != "approved" {
		t.Errorf("resolution: got %q, want %q", got.Resolution, "approved")
	}
	if got.Resolver != "bob" {
		t.Errorf("resolver: got %q, want %q", got.Resolver, "bob")
	}
	if got.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
}

func TestMemoryStore_ResolveHITL_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.ResolveHITL(context.Background(), "ghost", "approved", "alice")
	if err == nil {
		t.Fatal("expected error resolving non-existent HITL, got nil")
	}
}

func TestMemoryStore_ListHITLs(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_ = s.CreateHITL(ctx, &asmtypes.HITLRequest{RunID: "r1", StateName: "approve", CreatedAt: time.Now()})
	_ = s.CreateHITL(ctx, &asmtypes.HITLRequest{RunID: "r2", StateName: "approve", Assignee: "alice", CreatedAt: time.Now()})
	_ = s.ResolveHITL(ctx, "r2", "approved", "alice")

	resolved := false
	pending, err := s.ListHITLs(ctx, HITLFilter{Resolved: &resolved})
	if err != nil {
		t.Fatalf("ListHITLs: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending HITL, got %d", len(pending))
	}
	if pending[0].RunID != "r1" {
		t.Errorf("expected pending run r1, got %q", pending[0].RunID)
	}

	// Filter by assignee
	alicePending, _ := s.ListHITLs(ctx, HITLFilter{Assignee: "alice", Resolved: &resolved})
	if len(alicePending) != 0 {
		t.Errorf("expected 0 pending for alice, got %d", len(alicePending))
	}
}
