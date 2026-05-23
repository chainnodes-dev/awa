package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/health"
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- test fixtures ---

const simpleWorkflowYAML = `apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: test-wf
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

// stubTemporalClient satisfies orchestrator.TemporalEngineClient for API tests.
type stubTemporalClient struct{}

func (s *stubTemporalClient) StartWorkflow(_ context.Context, _ *asmtypes.WorkflowRun, _ *asmtypes.WorkflowDef) (string, error) {
	return "stub-temporal-id", nil
}
func (s *stubTemporalClient) SendTriggerSignal(_ context.Context, _, _ string, _ map[string]interface{}) error {
	return nil
}
func (s *stubTemporalClient) SendHITLSignal(_ context.Context, _ string, _ asmtypes.HITLSignal) error {
	return nil
}
func (s *stubTemporalClient) SendChatSignal(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *stubTemporalClient) TerminateWorkflow(_ context.Context, _ string) error {
	return nil
}
func (s *stubTemporalClient) AwaitWorkflowCompletion(_ context.Context, _ string) error {
	return nil
}
func (s *stubTemporalClient) Close() {}

// testEnv bundles all test dependencies together.
type testEnv struct {
	router     *gin.Engine
	store      store.Store
	engine     *orchestrator.Engine
	adminToken string // pre-generated Bearer token for the test admin
}

// testRouter builds a full router backed by in-memory dependencies and returns
// a pre-generated admin token ready to use in request headers.
func testRouter(t *testing.T) testEnv {
	t.Helper()
	s := store.NewMemoryStore()
	bus := events.NewLocalBus()
	temporal := &stubTemporalClient{}
	eng := orchestrator.NewEngine(s, bus, temporal)

	jwtSvc := auth.NewJWTService("test-secret-do-not-use-in-prod")
	adminToken, err := jwtSvc.GenerateAccessToken("test-admin-id", store.DefaultTenantID, "testadmin", string(auth.RoleAdmin))
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}

	mcpMgr := mcp.NewManager(s)
	handlers := NewHandlers(&config.Config{}, eng, s, jwtSvc, nil, nil, nil, nil, nil, nil, mcpMgr, nil, nil) // nil generator, etc.
	router := gin.New()
	registerRoutes(router, handlers, NewHub(bus), jwtSvc, health.New())

	return testEnv{
		router:     router,
		store:      s,
		engine:     eng,
		adminToken: "Bearer " + adminToken,
	}
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewBuffer(b)
}

// do executes a request on the router. Pass authHeader as "Bearer <token>" to
// include authentication; pass "" for unauthenticated requests (public routes).
func do(router *gin.Engine, method, path string, body *bytes.Buffer, authHeader string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
}

// --- Health (public — no auth required) ---

func TestHealth(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/health", nil, "")
	if w.Code != http.StatusOK {
		t.Errorf("health: got %d, want 200", w.Code)
	}
}

// --- Auth middleware ---

func TestAuth_MissingToken(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/workflows", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/workflows", nil, "Bearer not-a-real-token")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with bad token, got %d", w.Code)
	}
}

// --- Workflow definitions ---

func TestCreateWorkflow_Valid(t *testing.T) {
	env := testRouter(t)

	w := do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	var def asmtypes.WorkflowDef
	decode(t, w, &def)
	if def.Metadata.Name != "test-wf" {
		t.Errorf("name: got %q, want %q", def.Metadata.Name, "test-wf")
	}
}

func TestCreateWorkflow_InvalidYAML(t *testing.T) {
	env := testRouter(t)

	w := do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": "not: valid: yaml: [[["}), env.adminToken)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateWorkflow_MissingBody(t *testing.T) {
	env := testRouter(t)

	w := do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{}), env.adminToken) // missing 'yaml' field

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListWorkflows_Empty(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/workflows", nil, env.adminToken)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var defs []asmtypes.WorkflowDef
	decode(t, w, &defs)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
}

func TestListWorkflows_AfterCreate(t *testing.T) {
	env := testRouter(t)

	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	w := do(env.router, "GET", "/api/v1/workflows", nil, env.adminToken)
	var defs []asmtypes.WorkflowDef
	decode(t, w, &defs)
	if len(defs) != 1 {
		t.Errorf("expected 1 def, got %d", len(defs))
	}
}

func TestGetWorkflow(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	w := do(env.router, "GET", "/api/v1/workflows/test-wf/v1", nil, env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decode(t, w, &resp)
	if _, ok := resp["definition"]; !ok {
		t.Error("expected 'definition' key in response")
	}
	if _, ok := resp["yaml"]; !ok {
		t.Error("expected 'yaml' key in response")
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/workflows/missing/v1", nil, env.adminToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteWorkflow(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	w := do(env.router, "DELETE", "/api/v1/workflows/test-wf/v1", nil, env.adminToken)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Should no longer be retrievable.
	w2 := do(env.router, "GET", "/api/v1/workflows/test-wf/v1", nil, env.adminToken)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w2.Code)
	}
}

// --- Runs ---

func TestStartRun_Valid(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	w := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "test-wf",
		"workflow_version": "v1",
		"input":            map[string]interface{}{"note": "hello"},
	}), env.adminToken)
	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	var run asmtypes.WorkflowRun
	decode(t, w, &run)
	if run.ID == "" {
		t.Error("expected non-empty run ID")
	}
	if run.WorkflowName != "test-wf" {
		t.Errorf("workflow_name: got %q, want %q", run.WorkflowName, "test-wf")
	}
	if run.CurrentState != "start" {
		t.Errorf("current_state: got %q, want %q", run.CurrentState, "start")
	}
}

func TestStartRun_UnknownWorkflow(t *testing.T) {
	env := testRouter(t)

	w := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "no-such",
		"workflow_version": "v1",
	}), env.adminToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestStartRun_MissingFields(t *testing.T) {
	env := testRouter(t)

	w := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name": "test-wf",
		// missing workflow_version
	}), env.adminToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetRun(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	wStart := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "test-wf",
		"workflow_version": "v1",
	}), env.adminToken)
	var run asmtypes.WorkflowRun
	decode(t, wStart, &run)

	// Poll until the run appears in the store (engine runs async).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := env.store.GetRun(context.Background(), run.ID); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	w := do(env.router, "GET", "/api/v1/runs/"+run.ID, nil, env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var got asmtypes.WorkflowRun
	decode(t, w, &got)
	if got.ID != run.ID {
		t.Errorf("run ID mismatch: got %q, want %q", got.ID, run.ID)
	}
}

func TestGetRun_NotFound(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/runs/ghost-id", nil, env.adminToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListRuns_Empty(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/runs", nil, env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	var runs []asmtypes.WorkflowRun
	decode(t, w, &runs)
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestTriggerRun(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	wStart := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "test-wf",
		"workflow_version": "v1",
	}), env.adminToken)
	var run asmtypes.WorkflowRun
	decode(t, wStart, &run)

	w := do(env.router, "POST", "/api/v1/runs/"+run.ID+"/trigger",
		jsonBody(t, map[string]interface{}{"trigger": "finish"}), env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("trigger status: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	// Wait for completion.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := env.store.GetRun(context.Background(), run.ID)
		if r != nil && r.Status == asmtypes.RunComplete {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	r, _ := env.store.GetRun(context.Background(), run.ID)
	if r.Status != asmtypes.RunComplete {
		t.Errorf("expected complete status, got %q", r.Status)
	}
}

func TestTriggerRun_InvalidTrigger(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	wStart := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "test-wf",
		"workflow_version": "v1",
	}), env.adminToken)
	var run asmtypes.WorkflowRun
	decode(t, wStart, &run)

	w := do(env.router, "POST", "/api/v1/runs/"+run.ID+"/trigger",
		jsonBody(t, map[string]interface{}{"trigger": "nonexistent"}), env.adminToken)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetRunHistory(t *testing.T) {
	env := testRouter(t)
	do(env.router, "POST", "/api/v1/workflows",
		jsonBody(t, map[string]string{"yaml": simpleWorkflowYAML}), env.adminToken)

	wStart := do(env.router, "POST", "/api/v1/runs", jsonBody(t, map[string]interface{}{
		"workflow_name":    "test-wf",
		"workflow_version": "v1",
	}), env.adminToken)
	var run asmtypes.WorkflowRun
	decode(t, wStart, &run)

	// Trigger and wait for completion so transitions are recorded.
	do(env.router, "POST", "/api/v1/runs/"+run.ID+"/trigger",
		jsonBody(t, map[string]interface{}{"trigger": "finish"}), env.adminToken)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := env.store.GetRun(context.Background(), run.ID)
		if r != nil && r.Status == asmtypes.RunComplete {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	w := do(env.router, "GET", "/api/v1/runs/"+run.ID+"/history", nil, env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var history []asmtypes.TransitionRecord
	decode(t, w, &history)
	if len(history) == 0 {
		t.Error("expected at least one transition in history")
	}
}

// --- HITL ---

func TestGetPendingHITL_Empty(t *testing.T) {
	env := testRouter(t)
	w := do(env.router, "GET", "/api/v1/hitl/pending", nil, env.adminToken)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	var reqs []asmtypes.HITLRequest
	decode(t, w, &reqs)
	if len(reqs) != 0 {
		t.Errorf("expected 0 pending HITL, got %d", len(reqs))
	}
}
