package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestPortalAPI_GetForm(t *testing.T) {
	env := testRouter(t)
	ctx := context.Background()

	runID := uuid.NewString()
	run := &asmtypes.WorkflowRun{
		ID:              runID,
		TenantID:        store.DefaultTenantID,
		WorkflowName:    "onboarding-wf",
		WorkflowVersion: "v1",
		Status:          asmtypes.RunWaiting,
		CurrentState:    "guest_intake",
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := env.store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	rawToken, tokenHash, err := store.GenerateGuestToken()
	if err != nil {
		t.Fatalf("GenerateGuestToken failed: %v", err)
	}

	gt := &store.GuestToken{
		ID:          uuid.NewString(),
		TenantID:    store.DefaultTenantID,
		RunID:       runID,
		StateName:   "guest_intake",
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(48 * time.Hour),
		Status:      "active",
		Title:       "Vendor Registration Form",
		Description: "Please complete your vendor details.",
		FormSchema: map[string]interface{}{
			"type": "object",
			"required": []interface{}{"vendorName", "taxId"},
			"properties": map[string]interface{}{
				"vendorName": map[string]interface{}{"type": "string", "title": "Vendor Name"},
				"taxId":      map[string]interface{}{"type": "string", "title": "Tax ID / EIN"},
			},
		},
	}
	if err := env.store.CreateGuestToken(ctx, gt); err != nil {
		t.Fatalf("CreateGuestToken failed: %v", err)
	}

	// 1. Success case: Active token
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/portal/%s/%s", runID, rawToken), nil)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["status"] != "active" || res["title"] != "Vendor Registration Form" {
		t.Errorf("unexpected response body: %v", res)
	}

	// 2. Not found case: Invalid token
	badReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/portal/%s/invalid-token-123", runID), nil)
	badRec := httptest.NewRecorder()
	env.router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid token, got %d", badRec.Code)
	}

	// 3. Not found case: Mismatched run ID
	mismatchReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/portal/%s/%s", uuid.NewString(), rawToken), nil)
	mismatchRec := httptest.NewRecorder()
	env.router.ServeHTTP(mismatchRec, mismatchReq)
	if mismatchRec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for mismatched run_id, got %d", mismatchRec.Code)
	}

	// 4. Expired token case
	expRaw, expHash, _ := store.GenerateGuestToken()
	expGT := &store.GuestToken{
		ID:        uuid.NewString(),
		TenantID:  store.DefaultTenantID,
		RunID:     runID,
		StateName: "guest_intake",
		TokenHash: expHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		Status:    "active",
		Title:     "Expired Form",
	}
	_ = env.store.CreateGuestToken(ctx, expGT)

	expReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/portal/%s/%s", runID, expRaw), nil)
	expRec := httptest.NewRecorder()
	env.router.ServeHTTP(expRec, expReq)
	if expRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for expired form status, got %d", expRec.Code)
	}
	var expRes map[string]interface{}
	_ = json.Unmarshal(expRec.Body.Bytes(), &expRes)
	if expRes["status"] != "expired" {
		t.Errorf("expected status 'expired', got %v", expRes["status"])
	}
}

func TestPortalAPI_UploadAndDownload(t *testing.T) {
	env := testRouter(t)
	ctx := context.Background()

	runID := uuid.NewString()
	rawToken, tokenHash, _ := store.GenerateGuestToken()
	gt := &store.GuestToken{
		ID:        uuid.NewString(),
		TenantID:  store.DefaultTenantID,
		RunID:     runID,
		StateName: "doc_upload",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    "active",
	}
	_ = env.store.CreateGuestToken(ctx, gt)

	// 1. Multipart file upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "contract_sample.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	testPdfContent := "%PDF-1.4 test document content for public portal"
	_, _ = part.Write([]byte(testPdfContent))
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/%s/%s/upload", runID, rawToken), body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	env.router.ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload failed with code %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	var uploadRes struct {
		FileID   string `json:"file_id"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		MimeType string `json:"mime_type"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadRes); err != nil {
		t.Fatalf("unmarshal upload response failed: %v", err)
	}
	if uploadRes.Name != "contract_sample.pdf" || uploadRes.FileID == "" {
		t.Errorf("unexpected upload response: %+v", uploadRes)
	}

	defer func() {
		_ = os.RemoveAll(filepath.Join("data", "uploads", uploadRes.FileID))
	}()

	// 2. Download via portal token
	downloadReq := httptest.NewRequest(http.MethodGet, uploadRes.URL, nil)
	downloadRec := httptest.NewRecorder()
	env.router.ServeHTTP(downloadRec, downloadReq)

	if downloadRec.Code != http.StatusOK {
		t.Fatalf("download failed with code %d: %s", downloadRec.Code, downloadRec.Body.String())
	}
	if downloadRec.Body.String() != testPdfContent {
		t.Errorf("downloaded content mismatch: got %q, want %q", downloadRec.Body.String(), testPdfContent)
	}
}

func TestPortalAPI_Submit(t *testing.T) {
	env := testRouter(t)
	ctx := context.Background()

	runID := uuid.NewString()
	run := &asmtypes.WorkflowRun{
		ID:              runID,
		TenantID:        store.DefaultTenantID,
		WorkflowName:    "intake-wf",
		WorkflowVersion: "v1",
		TemporalID:      "temporal-" + runID,
		Status:          asmtypes.RunWaiting,
		CurrentState:    "patient_form",
		Blackboard:      make(map[string]interface{}),
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_ = env.store.CreateRun(ctx, run)

	hitlReq := &asmtypes.HITLRequest{
		ID:           uuid.NewString(),
		RunID:        runID,
		StateName:    "patient_form",
		Resolved:     false,
		WorkflowName: "intake-wf",
		FormSchema: map[string]interface{}{
			"type": "object",
			"required": []interface{}{"patientName"},
			"properties": map[string]interface{}{
				"patientName": map[string]interface{}{"type": "string"},
			},
		},
		CreatedAt: time.Now(),
	}
	_ = env.store.CreateHITL(ctx, hitlReq)

	rawToken, tokenHash, _ := store.GenerateGuestToken()
	gt := &store.GuestToken{
		ID:          uuid.NewString(),
		TenantID:    store.DefaultTenantID,
		RunID:       runID,
		StateName:   "patient_form",
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Status:      "active",
		FormSchema:  hitlReq.FormSchema,
		Metadata: map[string]interface{}{
			"default_resolution": "approved",
		},
	}
	_ = env.store.CreateGuestToken(ctx, gt)

	// 1. Validation error: Missing required field
	badPayload, _ := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{"phone": "555-1234"},
	})
	badReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/%s/%s/submit", runID, rawToken), bytes.NewReader(badPayload))
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	env.router.ServeHTTP(badRec, badReq)

	if badRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing required field, got %d: %s", badRec.Code, badRec.Body.String())
	}

	// 2. Success submission
	validPayload, _ := json.Marshal(map[string]interface{}{
		"submitter": "Jane Doe",
		"data": map[string]interface{}{
			"patientName": "Jane Doe",
			"allergies":   "Penicillin",
		},
	})
	submitReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/%s/%s/submit", runID, rawToken), bytes.NewReader(validPayload))
	submitReq.Header.Set("Content-Type", "application/json")
	submitRec := httptest.NewRecorder()
	env.router.ServeHTTP(submitRec, submitReq)

	if submitRec.Code != http.StatusOK {
		t.Fatalf("submission failed with code %d: %s", submitRec.Code, submitRec.Body.String())
	}

	// Verify token is marked completed
	updatedGT, err := env.store.GetGuestTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetGuestTokenByHash failed: %v", err)
	}
	if updatedGT.Status != "completed" || updatedGT.CompletedBy != "Jane Doe" {
		t.Errorf("expected token completed by Jane Doe, got: %+v", updatedGT)
	}

	// 3. Re-submission attempt fails with Conflict
	resubmitReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/%s/%s/submit", runID, rawToken), bytes.NewReader(validPayload))
	resubmitReq.Header.Set("Content-Type", "application/json")
	resubmitRec := httptest.NewRecorder()
	env.router.ServeHTTP(resubmitRec, resubmitReq)

	if resubmitRec.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate submission, got %d", resubmitRec.Code)
	}
}

func TestPortalAPI_GenerateRunGuestToken(t *testing.T) {
	env := testRouter(t)
	ctx := context.Background()

	runID := uuid.NewString()
	run := &asmtypes.WorkflowRun{
		ID:              runID,
		TenantID:        store.DefaultTenantID,
		WorkflowName:    "approval-wf",
		WorkflowVersion: "v1",
		Status:          asmtypes.RunWaiting,
		CurrentState:    "manager_approval",
		StartedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_ = env.store.CreateRun(ctx, run)

	hitlReq := &asmtypes.HITLRequest{
		ID:           uuid.NewString(),
		RunID:        runID,
		StateName:    "manager_approval",
		WorkflowName: "approval-wf",
		FormSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"notes": map[string]interface{}{"type": "string"},
			},
		},
		CreatedAt: time.Now(),
	}
	_ = env.store.CreateHITL(ctx, hitlReq)

	genPayload, _ := json.Marshal(map[string]interface{}{
		"title":       "External Manager Review",
		"description": "Please sign off on the budget request.",
		"expires_in":  "48h",
	})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/runs/%s/guest-token", runID), bytes.NewReader(genPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", env.adminToken)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		PortalURL string    `json:"portal_url"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.PortalURL == "" || res.Token == "" {
		t.Errorf("expected portal_url and token in response: %+v", res)
	}

	// Verify the newly generated link is functional
	testReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/portal/%s/%s", runID, res.Token), nil)
	testRec := httptest.NewRecorder()
	env.router.ServeHTTP(testRec, testReq)
	if testRec.Code != http.StatusOK {
		t.Errorf("newly minted link failed with %d: %s", testRec.Code, testRec.Body.String())
	}
}

