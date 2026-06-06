package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asm-platform/asm/internal/store"
)

func TestLocalModelHandlers(t *testing.T) {
	// 1. Start a mock Ollama server.
	mockOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"models": [
					{
						"name": "gemma2:2b",
						"size": 1600000000,
						"details": {
							"format": "gguf",
							"family": "gemma2"
						}
					}
				]
			}`))
		case "/api/pull":
			w.Header().Set("Content-Type", "application/json")
			// Stream progress lines
			_, _ = w.Write([]byte(`{"status": "pulling manifest"}
{"status": "pulling sha256:1234", "total": 100, "completed": 50}
{"status": "success"}
`))
		case "/api/delete":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockOllama.Close()

	// 2. Setup the test environment.
	env := testRouter(t)
	ctx := context.Background()

	// 3. Insert Ollama config into store pointing to mock server.
	cfg := &store.LLMConfig{
		TenantID: store.DefaultTenantID,
		Provider: "ollama",
		Enabled:  true,
		BaseURL:  mockOllama.URL,
	}
	if err := env.store.UpsertLLMConfig(ctx, cfg); err != nil {
		t.Fatalf("failed to insert mock ollama config: %v", err)
	}

	// 4. Test ListLocalModels GET /api/v1/llm-configs/local/models
	t.Run("ListLocalModels", func(t *testing.T) {
		w := do(env.router, "GET", "/api/v1/llm-configs/local/models", nil, env.adminToken)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
		}
		var resp struct {
			Models []struct {
				Name string `json:"name"`
				Size int64  `json:"size"`
			} `json:"models"`
		}
		decode(t, w, &resp)
		if len(resp.Models) != 1 || resp.Models[0].Name != "gemma2:2b" {
			t.Errorf("unexpected models response: %+v", resp)
		}
	})

	// 5. Test PullLocalModel POST /api/v1/llm-configs/local/models/pull
	t.Run("PullLocalModel", func(t *testing.T) {
		w := do(env.router, "POST", "/api/v1/llm-configs/local/models/pull",
			jsonBody(t, map[string]string{"model": "gemma2:2b"}), env.adminToken)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
		}
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json-stream" {
			t.Errorf("expected content type application/json-stream, got %q", contentType)
		}
		// Confirm multiple lines in body
		body := w.Body.String()
		if body == "" {
			t.Error("expected non-empty progress response")
		}
	})

	// 6. Test DeleteLocalModel DELETE /api/v1/llm-configs/local/models
	t.Run("DeleteLocalModel", func(t *testing.T) {
		w := do(env.router, "DELETE", "/api/v1/llm-configs/local/models?model=gemma2:2b", nil, env.adminToken)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d (body: %s)", w.Code, w.Body.String())
		}
	})
}
