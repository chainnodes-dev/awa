package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/store"
	"github.com/gin-gonic/gin"
)

func setupTestMCPHandler() (*Handlers, *store.MemoryStore, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret: "test-secret-key-at-least-32-bytes-long!",
	}
	mcpMgr := mcp.NewManager(memStore)
	h := &Handlers{
		store:      memStore,
		cfg:        cfg,
		mcpManager: mcpMgr,
	}

	r := gin.New()
	r.POST("/api/v1/mcp-servers", h.CreateMCPServer)
	r.GET("/api/v1/mcp-servers", h.ListMCPServers)
	r.PUT("/api/v1/mcp-servers/:id", h.UpdateMCPServer)
	r.DELETE("/api/v1/mcp-servers/:id", h.DeleteMCPServer)

	return h, memStore, r
}

func TestMCPServer_CRUD(t *testing.T) {
	_, memStore, router := setupTestMCPHandler()

	// 1. Create MCP server
	createPayload := map[string]interface{}{
		"name":        "test-server",
		"transport":   "stdio",
		"command":     "npx",
		"args":        []string{"-y", "@modelcontextprotocol/server-postgres"},
		"env_vars":    map[string]string{"DATABASE_URL": "postgres://localhost/test"},
		"description": "Postgres connector",
		"scopes":      []string{"read-only"},
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var created store.MCPServer
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created server: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected ID to be set")
	}
	if len(created.Scopes) != 1 || created.Scopes[0] != "read-only" {
		t.Errorf("expected scopes ['read-only'], got %v", created.Scopes)
	}

	// 2. Update MCP server config (change env_vars and scopes)
	updatePayload := map[string]interface{}{
		"name":        "test-server-renamed",
		"transport":   "stdio",
		"command":     "npx",
		"args":        []string{"-y", "@modelcontextprotocol/server-postgres", "--schema=public"},
		"env_vars":    map[string]string{"DATABASE_URL": "postgres://prod-db:5432/main", "NEW_KEY": "val"},
		"description": "Updated description",
		"scopes":      []string{"read-only", "write"},
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/mcp-servers/"+created.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	router.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var updated store.MCPServer
	if err := json.Unmarshal(updateW.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode updated server: %v", err)
	}
	if updated.Name != "test-server-renamed" {
		t.Errorf("expected updated name 'test-server-renamed', got %q", updated.Name)
	}
	if updated.EnvVars["DATABASE_URL"] != "postgres://prod-db:5432/main" || updated.EnvVars["NEW_KEY"] != "val" {
		t.Errorf("expected updated env_vars, got %v", updated.EnvVars)
	}
	if len(updated.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %v", updated.Scopes)
	}

	// Verify in store
	inStore, err := memStore.GetMCPServer(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetMCPServer failed: %v", err)
	}
	if inStore.Description != "Updated description" {
		t.Errorf("expected store to have updated description, got %q", inStore.Description)
	}

	// 3. Delete MCP server
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/mcp-servers/"+created.ID, nil)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify deleted from store
	_, err = memStore.GetMCPServer(context.Background(), created.ID)
	if err == nil {
		t.Fatalf("expected server to be deleted from store")
	}
}
