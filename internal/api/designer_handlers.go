package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/store"
)

// registryPath resolves the location of mcp_registry.yaml relative to the
// project root regardless of where the binary is started from.
func registryPath() string {
	// Walk up from this source file to the project root at compile time.
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "mcp_registry.yaml")
}

// ListMCPServers returns the list of registered MCP servers (name + description only).
// GET /api/v1/mcp-servers
//
// Strategy: use the DB-backed registry when it has entries; fall back to the
// static mcp_registry.yaml for teams that haven't migrated to the DB yet.
func (h *Handlers) ListMCPServers(c *gin.Context) {
	ctx := c.Request.Context()

	// Try DB first.
	hasAny, err := h.store.HasAnyMCPServer(ctx)
	if err == nil && hasAny {
		servers, err := h.store.ListMCPServers(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return the full store model so the admin UI has IDs.
		c.JSON(http.StatusOK, servers)
		return
	}

	// Fall back to static YAML registry.
	entries, err := designer.LoadMCPRegistry(registryPath())
	if err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, designer.ToListItems(entries))
}

// CreateMCPServer adds a new MCP server to the DB registry.
// POST /api/v1/mcp-servers   (admin only)
func (h *Handlers) CreateMCPServer(c *gin.Context) {
	var body struct {
		Name        string `json:"name"        binding:"required"`
		URL         string `json:"url"`
		EnvVar      string `json:"env_var"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.URL == "" && body.EnvVar == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of url or env_var is required"})
		return
	}

	claims := auth.ClaimsFrom(c)
	tenantID := ""
	createdBy := ""
	if claims != nil {
		tenantID = claims.TenantID
		createdBy = claims.Username
	}

	srv := &store.MCPServer{
		TenantID:    tenantID,
		Name:        body.Name,
		URL:         body.URL,
		EnvVar:      body.EnvVar,
		Description: body.Description,
		CreatedBy:   createdBy,
	}
	if err := h.store.CreateMCPServer(c.Request.Context(), srv); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, srv)
}

// UpdateMCPServer replaces the mutable fields of an existing MCP server.
// PUT /api/v1/mcp-servers/:id   (admin only)
func (h *Handlers) UpdateMCPServer(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		URL         string `json:"url"`
		EnvVar      string `json:"env_var"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	srv := &store.MCPServer{
		ID:          id,
		URL:         body.URL,
		EnvVar:      body.EnvVar,
		Description: body.Description,
	}
	if err := h.store.UpdateMCPServer(c.Request.Context(), srv); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, srv)
}

// DeleteMCPServer removes an MCP server from the registry.
// DELETE /api/v1/mcp-servers/:id   (admin only)
func (h *Handlers) DeleteMCPServer(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteMCPServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// PingMCPServer performs a lightweight HTTP health check against an MCP server.
// POST /api/v1/mcp-servers/:id/ping
func (h *Handlers) PingMCPServer(c *gin.Context) {
	srv, err := h.store.GetMCPServer(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Resolve the URL: prefer direct URL, fall back to env var.
	url := srv.URL
	if url == "" && srv.EnvVar != "" {
		url = os.Getenv(srv.EnvVar)
	}
	if url == "" {
		c.JSON(http.StatusOK, gin.H{"status": "unknown", "error": "no URL configured"})
		return
	}

	// Attempt a HEAD request with a short timeout.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "error": fmt.Sprintf("invalid URL: %v", err)})
		return
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "offline", "error": err.Error(), "latency_ms": latency})
		return
	}
	resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{"status": "online", "http_status": resp.StatusCode, "latency_ms": latency})
}

// DiscoverMCPServer connects to an MCP server and retrieves its tool list via
// the MCP JSON-RPC protocol (initialize → tools/list).
// POST /api/v1/mcp-servers/discover
func (h *Handlers) DiscoverMCPServer(c *gin.Context) {
	var body struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}

	// Helper: send a JSON-RPC request to the MCP server.
	sendRPC := func(id int, method string, params interface{}) (json.RawMessage, error) {
		reqBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  params,
		}
		payload, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, body.URL, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("connect to MCP server: %w", err)
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB cap
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		var rpcResp struct {
			Result json.RawMessage        `json:"result"`
			Error  map[string]interface{} `json:"error"`
		}
		if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
			return nil, fmt.Errorf("parse JSON-RPC response: %w (body: %s)", err, string(respBytes[:min(200, len(respBytes))]))
		}
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("MCP error: %v", rpcResp.Error["message"])
		}
		return rpcResp.Result, nil
	}

	// Step 1: initialize
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "asm-platform",
			"version": "1.0.0",
		},
	}
	_, err := sendRPC(1, "initialize", initParams)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	// Step 2: tools/list
	toolsResult, err := sendRPC(2, "tools/list", map[string]interface{}{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	var toolsResp struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(toolsResult, &toolsResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": fmt.Sprintf("parse tools response: %v", err)})
		return
	}

	// Build a human-readable description from the tool list.
	var parts []string
	for _, t := range toolsResp.Tools {
		entry := t.Name
		if t.Description != "" {
			entry += " — " + t.Description
		}
		parts = append(parts, entry)
	}
	description := "Tools: " + strings.Join(parts, "; ")
	if len(parts) == 0 {
		description = "No tools advertised by this server."
	}

	c.JSON(http.StatusOK, gin.H{
		"tools":       toolsResp.Tools,
		"description": description,
	})
}

// GenerateWorkflow calls the LLM to produce a WorkflowDef from a description.
// POST /api/v1/designer/generate
func (h *Handlers) GenerateWorkflow(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI workflow generation is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is required"})
		return
	}

	gen := h.generator
	if req.Provider != "" {
		p, err := h.llmRegistry.Get(req.Provider)
		if err == nil {
			gen = gen.WithProvider(p)
		}
	}

	result, err := gen.Generate(c.Request.Context(), req)
	if err != nil {
		resp := gin.H{"error": err.Error()}
		if result != nil {
			resp["interactions"] = result.Interactions
		}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp, err := designer.MarshalGenerateResult(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialise result"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ScaffoldHandler calls the LLM to produce a Go HandlerFunc skeleton for a specialist worker.
// POST /api/v1/designer/scaffold
func (h *Handlers) ScaffoldHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "code scaffolding is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.ScaffoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AgentName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_name is required"})
		return
	}

	result, err := h.generator.Scaffold(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CodegenHandler calls the LLM to produce JavaScript code for a Code node.
// POST /api/v1/designer/codegen
func (h *Handlers) CodegenHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "code generation is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.CodegenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instructions are required"})
		return
	}

	result, err := h.generator.Codegen(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// AnalyseSkillHandler decomposes a skill description into an ASM workflow YAML.
// POST /api/v1/designer/skill/analyse
func (h *Handlers) AnalyseSkillHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "skill analysis is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.AnalyseSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is required"})
		return
	}

	gen := h.generator
	if req.Provider != "" {
		p, err := h.llmRegistry.Get(req.Provider)
		if err == nil {
			gen = gen.WithProvider(p)
		}
	}

	result, err := gen.AnalyseSkill(c.Request.Context(), req)
	if err != nil {
		resp := gin.H{"error": err.Error()}
		if result != nil {
			resp["interactions"] = result.Interactions
		}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	// Reuse MarshalGenerateResult for identical response shape {yaml, definition, interactions}.
	resp, err := designer.MarshalGenerateResult(&designer.GenerateResult{
		YAML:         result.YAML,
		Definition:   result.Definition,
		Interactions: result.Interactions,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialise result"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RenderSkillHandler converts a WorkflowDef YAML into a human-readable skill description.
// POST /api/v1/designer/skill/render
func (h *Handlers) RenderSkillHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "skill rendering is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.RenderSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.WorkflowYAML) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow_yaml is required"})
		return
	}

	result, err := h.generator.RenderSkill(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ScaffoldMCPHandler generates a Go MCP server implementation for the described tools.
// POST /api/v1/designer/scaffold-mcp
func (h *Handlers) ScaffoldMCPHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "MCP server scaffolding is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.MCPScaffoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.generator.ScaffoldMCP(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
