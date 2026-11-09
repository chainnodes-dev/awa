package email

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestEmailService_IngestEmail(t *testing.T) {
	memStore := store.NewMemoryStore()
	bus := events.NewLocalBus()
	ctx := store.WithTenantID(context.Background(), store.DefaultTenantID)

	var startedWorkflow string
	var startedInputs map[string]interface{}

	starter := func(c context.Context, tenantID, workflowName, version string, inputs map[string]interface{}) (*asmtypes.WorkflowRun, error) {
		startedWorkflow = workflowName
		startedInputs = inputs
		return &asmtypes.WorkflowRun{
			ID:           uuid.NewString(),
			TenantID:     tenantID,
			WorkflowName: workflowName,
			Status:       asmtypes.RunRunning,
		}, nil
	}

	svc := NewService(memStore, bus, starter)

	// 1. Subscribe to EmailInbound event
	subCh, cancel := bus.Subscribe(ctx)
	defer cancel()

	emailMsg := &ParsedEmail{
		MessageID: "msg-1234",
		From:      "vendor@acme.com",
		To:        []string{"invoices@myorg.com"},
		Subject:   "Invoice #9901 for review",
		Date:      time.Now(),
		TextBody:  "Invoice details enclosed.",
	}

	// Save target workflow definition
	def := &asmtypes.WorkflowDef{
		APIVersion: "chainnodes/v1",
		Kind:       "Workflow",
		Metadata: asmtypes.WorkflowMeta{
			Name:    "invoice-processing",
			Version: "v1",
		},
		Triggers: []asmtypes.TriggerDef{
			{
				Name: "on_invoice",
				Type: "email",
				Config: map[string]interface{}{
					"subject_contains": "Invoice",
				},
			},
		},
		States: []asmtypes.StateDef{
			{Name: "start", Type: "initial"},
			{Name: "done", Type: "terminal"},
		},
	}
	_ = memStore.SaveDefinition(ctx, def, "")

	run, err := svc.IngestEmail(ctx, store.DefaultTenantID, emailMsg, "webhook", "")
	if err != nil {
		t.Fatalf("IngestEmail failed: %v", err)
	}

	if run == nil {
		t.Fatalf("expected workflow to be triggered by filter match")
	}
	if startedWorkflow != "invoice-processing" {
		t.Errorf("expected workflow invoice-processing, got %s", startedWorkflow)
	}
	if startedInputs["email_subject"] != "Invoice #9901 for review" {
		t.Errorf("inputs mismatch: %v", startedInputs)
	}

	// Verify event was emitted
	select {
	case evt := <-subCh:
		if evt.Type != events.EmailInbound {
			t.Errorf("expected event %s, got %s", events.EmailInbound, evt.Type)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("timeout waiting for email.inbound event")
	}
}

func TestMatchTriggerFilter(t *testing.T) {
	email := &ParsedEmail{
		From:    "billing@hospital.org",
		Subject: "Urgent: Patient Records",
	}

	// Match subject
	if !matchTriggerFilter(map[string]interface{}{"subject_contains": "patient"}, email) {
		t.Errorf("expected subject match")
	}

	// Match from
	if !matchTriggerFilter(map[string]interface{}{"from_contains": "hospital"}, email) {
		t.Errorf("expected from match")
	}

	// No match
	if matchTriggerFilter(map[string]interface{}{"subject_contains": "invoice"}, email) {
		t.Errorf("expected no match for invoice")
	}
}
