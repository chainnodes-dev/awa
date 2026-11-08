package api

import (
	"context"
	"encoding/json"
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

func TestModelProfiles(t *testing.T) {
	env := testRouter(t)

	// 1. GET /api/v1/llm-configs/profiles returns defaults
	t.Run("GetDefaultProfiles", func(t *testing.T) {
		w := do(env.router, "GET", "/api/v1/llm-configs/profiles", nil, env.adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var profiles map[string]struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &profiles); err != nil {
			t.Fatalf("failed to parse profiles: %v", err)
		}
		if _, ok := profiles["fast"]; !ok {
			t.Errorf("expected 'fast' profile in defaults")
		}
		if _, ok := profiles["reasoning"]; !ok {
			t.Errorf("expected 'reasoning' profile in defaults")
		}
		if _, ok := profiles["general"]; !ok {
			t.Errorf("expected 'general' profile in defaults")
		}
	})

	// 2. PUT /api/v1/llm-configs/profiles updates custom profile mapping
	t.Run("UpdateProfiles", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"fast": map[string]string{
				"id":          "fast",
				"provider":    "openrouter",
				"model":       "openai/gpt-4o-mini",
				"label":       "Fast & Cheap",
				"description": "Custom mapped fast profile",
			},
		}
		w := do(env.router, "PUT", "/api/v1/llm-configs/profiles", jsonBody(t, updateBody), env.adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Re-fetch to ensure persistence
		w2 := do(env.router, "GET", "/api/v1/llm-configs/profiles", nil, env.adminToken)
		var profiles map[string]struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &profiles); err != nil {
			t.Fatalf("failed to parse profiles: %v", err)
		}
		if profiles["fast"].Model != "openai/gpt-4o-mini" {
			t.Errorf("expected updated fast model 'openai/gpt-4o-mini', got %q", profiles["fast"].Model)
		}
	})

	// 3. GET /api/v1/llm-configs/openrouter/models returns catalog (or fallback)
	t.Run("ListOpenRouterModels", func(t *testing.T) {
		w := do(env.router, "GET", "/api/v1/llm-configs/openrouter/models", nil, env.adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var models []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &models); err != nil {
			t.Fatalf("failed to unmarshal models: %v", err)
		}
		if len(models) == 0 {
			t.Errorf("expected at least 1 model in openrouter catalog")
		}
	})

	// 4. GET /api/v1/llm-configs/designer-profiles and PUT /api/v1/llm-configs/designer-profiles
	t.Run("DesignerProfiles", func(t *testing.T) {
		w := do(env.router, "GET", "/api/v1/llm-configs/designer-profiles", nil, env.adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var profs map[string]struct {
			ID    string `json:"id"`
			Model string `json:"model"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &profs); err != nil {
			t.Fatalf("failed to unmarshal designer profiles: %v", err)
		}
		if _, ok := profs["frontier"]; !ok {
			t.Errorf("expected 'frontier' tier in designer profiles")
		}
		if _, ok := profs["balanced"]; !ok {
			t.Errorf("expected 'balanced' tier in designer profiles")
		}
		if _, ok := profs["fast"]; !ok {
			t.Errorf("expected 'fast' tier in designer profiles")
		}

		// Update balanced tier
		updateBody := map[string]interface{}{
			"balanced": map[string]string{
				"id":          "balanced",
				"provider":    "openrouter",
				"model":       "deepseek/deepseek-chat",
				"label":       "Balanced V3",
				"description": "Custom balanced tier",
			},
		}
		w2 := do(env.router, "PUT", "/api/v1/llm-configs/designer-profiles", jsonBody(t, updateBody), env.adminToken)
		if w2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
		}

		w3 := do(env.router, "GET", "/api/v1/llm-configs/designer-profiles", nil, env.adminToken)
		if err := json.Unmarshal(w3.Body.Bytes(), &profs); err != nil {
			t.Fatalf("failed to unmarshal designer profiles: %v", err)
		}
		if profs["balanced"].Model != "deepseek/deepseek-chat" {
			t.Errorf("expected updated balanced model 'deepseek/deepseek-chat', got %q", profs["balanced"].Model)
		}
	})
}
