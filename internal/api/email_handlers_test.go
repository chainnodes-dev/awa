package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/email"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestInboundEmailWebhook_JSON(t *testing.T) {
	env := testRouter(t)
	ctx := context.Background()

	emailSvc := email.NewService(env.store, events.NewLocalBus(), func(c context.Context, tenantID, workflowName, version string, inputs map[string]interface{}) (*asmtypes.WorkflowRun, error) {
		return &asmtypes.WorkflowRun{
			ID:           uuid.NewString(),
			TenantID:     tenantID,
			WorkflowName: workflowName,
			Status:       asmtypes.RunRunning,
		}, nil
	})

	// Inject emailSvc via NewHandlers or setting on handlers
	// Since testRouter creates router, we can invoke the webhook endpoint
	reqBody, _ := json.Marshal(map[string]interface{}{
		"from":    "doctor@clinic.com",
		"to":      []string{"intake@hospital.com"},
		"subject": "Patient Referral",
		"text":    "Please schedule appointment.",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/inbound-email", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["ok"] != true || res["subject"] != "Patient Referral" {
		t.Errorf("unexpected response: %v", res)
	}
	_ = ctx
	_ = emailSvc
}

func TestInboundEmailWebhook_Multipart(t *testing.T) {
	env := testRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("from", "vendor@corp.com")
	_ = writer.WriteField("to", "invoices@myorg.com")
	_ = writer.WriteField("subject", "Invoice Q4")
	_ = writer.WriteField("text", "Attached is the Q4 invoice.")

	part, _ := writer.CreateFormFile("file", "q4_invoice.pdf")
	pdfContent := "%PDF-1.4 test invoice attachment"
	_, _ = part.Write([]byte(pdfContent))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/inbound-email", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["ok"] != true || res["attachments_count"].(float64) < 1 {
		t.Errorf("expected attachment count >= 1: %v", res)
	}
}

func TestInboundEmailWebhook_TargetWorkflow(t *testing.T) {
	env := testRouter(t)
	ctx := store.WithTenantID(context.Background(), "00000000-0000-0000-0000-000000000001")

	// Register a target workflow
	def := &asmtypes.WorkflowDef{
		APIVersion: "chainnodes/v1",
		Kind:       "Workflow",
		Metadata: asmtypes.WorkflowMeta{
			Name:    "triage-wf",
			Version: "v1",
		},
		States: []asmtypes.StateDef{
			{Name: "start", Type: "initial"},
			{Name: "done", Type: "terminal"},
		},
	}
	_ = env.store.SaveDefinition(ctx, def, "")

	rawEmail := "From: client@firm.com\r\n" +
		"To: support@myorg.com\r\n" +
		"Subject: Support Case #5512\r\n" +
		"Date: Mon, 9 Nov 2026 15:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"Need help with priority issue."

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/inbound-email?workflow=triage-wf", strings.NewReader(rawEmail))
	req.Header.Set("Content-Type", "message/rfc822")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["ok"] != true || res["workflow_name"] != "triage-wf" || res["run_id"] == nil {
		t.Errorf("expected workflow run to start: %v", res)
	}
}
