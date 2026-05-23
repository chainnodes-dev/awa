package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/store"
	"github.com/gin-gonic/gin"
)

// ListMCPServers returns all registered MCP servers.
// GET /api/v1/mcp-servers
func (h *Handlers) ListMCPServers(c *gin.Context) {
	servers, err := h.store.ListMCPServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

// CreateMCPServer adds a new MCP server to the DB registry.
// POST /api/v1/mcp-servers   (admin only)
func (h *Handlers) CreateMCPServer(c *gin.Context) {
	var body struct {
		Name        string            `json:"name"        binding:"required"`
		Transport   string            `json:"transport"   binding:"required"` // "sse" or "stdio"
		URL         string            `json:"url"`
		Command     string            `json:"command"`
		Args        []string          `json:"args"`
		EnvVars     map[string]string `json:"env_vars"`
		Description string            `json:"description"`
		DocURL      string            `json:"doc_url"`
		Tools       []store.MCPTool   `json:"tools"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		Transport:   body.Transport,
		URL:         body.URL,
		Command:     body.Command,
		Args:        body.Args,
		EnvVars:     body.EnvVars,
		Description: body.Description,
		DocURL:      body.DocURL,
		Tools:       body.Tools,
		CreatedBy:   createdBy,
	}
	slog.Info("Creating MCP server", "name", srv.Name, "tools_count", len(srv.Tools))
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
		Transport   string            `json:"transport"`
		URL         string            `json:"url"`
		Command     string            `json:"command"`
		Args        []string          `json:"args"`
		EnvVars     map[string]string `json:"env_vars"`
		Description string            `json:"description"`
		DocURL      string            `json:"doc_url"`
		Tools       []store.MCPTool   `json:"tools"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.store.GetMCPServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	srv := &store.MCPServer{
		ID:          id,
		TenantID:    existing.TenantID,
		Name:        existing.Name, // Keep existing name
		Transport:   body.Transport,
		URL:         body.URL,
		Command:     body.Command,
		Args:        body.Args,
		EnvVars:     body.EnvVars,
		Description: body.Description,
		DocURL:      body.DocURL,
		Tools:       body.Tools,
		CreatedBy:   existing.CreatedBy,
		CreatedAt:   existing.CreatedAt,
	}
	slog.Info("Updating MCP server", "id", srv.ID, "tools_count", len(srv.Tools))
	if err := h.store.UpdateMCPServer(c.Request.Context(), srv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

// PingMCPServer performs a lightweight health check against an MCP server.
// POST /api/v1/mcp-servers/:id/ping
func (h *Handlers) PingMCPServer(c *gin.Context) {
	srv, err := h.store.GetMCPServer(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()
	var latency int64
	
	if srv.Transport == "stdio" {
		_, err := h.mcpManager.GetClient(c.Request.Context(), srv.Name)
		latency = time.Since(start).Milliseconds()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "offline", "error": err.Error(), "latency_ms": latency})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "online", "latency_ms": latency})
		return
	}

	if srv.Transport == "sse" {
		if srv.URL == "" {
			c.JSON(http.StatusOK, gin.H{"status": "offline", "error": "No URL configured for SSE transport. Please set the required environment variable."})
			return
		}

		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodHead, srv.URL, nil)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": fmt.Sprintf("invalid URL '%s': %v", srv.URL, err)})
			return
		}
		resp, err := http.DefaultClient.Do(req)
		latency = time.Since(start).Milliseconds()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "offline", "error": err.Error(), "latency_ms": latency})
			return
		}
		resp.Body.Close()
		c.JSON(http.StatusOK, gin.H{"status": "online", "http_status": resp.StatusCode, "latency_ms": latency})
	}
}

// DiscoverMCPServer connects to an MCP server and retrieves its tool list.
// POST /api/v1/mcp-servers/discover
func (h *Handlers) DiscoverMCPServer(c *gin.Context) {
	var body struct {
		Transport string            `json:"transport"`
		URL       string            `json:"url"`
		Command   string            `json:"command"`
		Args      []string          `json:"args"`
		EnvVars   map[string]string `json:"env_vars"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var client mcp.Client
	var closer func() error

	if body.Transport == "stdio" {
		clt, cl, err := h.mcpManager.TestStdioClientMap(body.Command, body.Args, body.EnvVars)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start server: %v", err)})
			return
		}
		client = clt
		closer = cl.Close
	} else {
		client = mcp.NewSSEClient(body.URL)
	}

	if closer != nil { defer closer() }

	initReq := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "phaxa-discovery", "version": "1.0"},
	}
	_, err := client.Call(ctx, "initialize", initReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": fmt.Sprintf("Initialize failed: %v", err)})
		return
	}
	_ = client.Notify(ctx, "notifications/initialized", map[string]interface{}{})

	toolsResult, err := client.Call(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	var toolsResp struct {
		Tools []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			InputSchema map[string]interface{} `json:"inputSchema"`
		} `json:"tools"`
	}
	_ = json.Unmarshal(toolsResult, &toolsResp)

	var parts []string
	for _, t := range toolsResp.Tools {
		entry := t.Name
		if t.Description != "" {
			entry += " \u2014 " + t.Description
		}
		// Identify required params from schema
		if schema, ok := t.InputSchema["properties"].(map[string]interface{}); ok {
			var reqs []string
			required, _ := t.InputSchema["required"].([]interface{})
			for k := range schema {
				isReq := false
				for _, r := range required {
					if r == k {
						isReq = true
						break
					}
				}
				if isReq {
					reqs = append(reqs, k+"*")
				} else {
					reqs = append(reqs, k)
				}
			}
			if len(reqs) > 0 {
				entry += fmt.Sprintf(" [%s]", strings.Join(reqs, ", "))
			}
		}
		parts = append(parts, entry)
	}
	description := "Tools: " + strings.Join(parts, "; ")
	if len(parts) == 0 {
		description = "No tools advertised."
	}

	c.JSON(http.StatusOK, gin.H{
		"tools":       toolsResp.Tools,
		"description": description,
		"raw_tools":   toolsResp.Tools, // Added for frontend to render structured UI
	})
}

// GenerateWorkflow calls the LLM to produce a WorkflowDef from a description.
// POST /api/v1/designer/generate
func (h *Handlers) GenerateWorkflow(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI workflow generation is not available"})
		return
	}
	var req designer.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.generator.Generate(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp, _ := designer.MarshalGenerateResult(result)
	c.JSON(http.StatusOK, resp)
}

// ScaffoldHandler calls the LLM to produce a Go HandlerFunc skeleton.
// POST /api/v1/designer/scaffold
func (h *Handlers) ScaffoldHandler(c *gin.Context) {
	var req designer.ScaffoldRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.generator.Scaffold(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CodegenHandler uses the LLM to generate code snippets.
// POST /api/v1/designer/codegen
func (h *Handlers) CodegenHandler(c *gin.Context) {
	var req designer.CodegenRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.generator.Codegen(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ProcessAnalyseHandler decomposes a process description.
func (h *Handlers) ProcessAnalyseHandler(c *gin.Context) {
	var req designer.ProcessAnalyseRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.generator.AnalyseProcess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp, _ := designer.MarshalGenerateResult(&designer.GenerateResult{
		YAML: result.YAML, Definition: result.Definition, Interactions: result.Interactions,
	})
	c.JSON(http.StatusOK, resp)
}

// ProcessSummariseHandler converts a WorkflowDef YAML into a summary.
func (h *Handlers) ProcessSummariseHandler(c *gin.Context) {
	var req designer.ProcessSummariseRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.generator.SummariseProcess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── Multi-step pipeline handlers ─────────────────────────────────────────────

func (h *Handlers) pipelineGeneratorFor(req designer.PipelineRequest) *designer.Generator {
	if req.Provider != "" {
		if p, err := h.llmRegistry.Get(req.Provider); err == nil {
			return h.generator.WithProvider(p)
		}
	}
	return h.generator
}

func writePipelineResult(c *gin.Context, result *designer.PipelineStepResult, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out, _ := designer.MarshalPipelineStepResult(result)
	c.JSON(http.StatusOK, out)
}

func (h *Handlers) PipelineDecomposeHandler(c *gin.Context) {
	var req designer.PipelineRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.pipelineGeneratorFor(req).DecomposeWorkflow(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

func (h *Handlers) PipelineCategoriseHandler(c *gin.Context) {
	var req designer.PipelineRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.pipelineGeneratorFor(req).CategoriseNodes(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

func (h *Handlers) PipelineWireHandler(c *gin.Context) {
	var req designer.PipelineRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.pipelineGeneratorFor(req).WireWorkflow(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

func (h *Handlers) PipelineImplementHandler(c *gin.Context) {
	var req designer.PipelineRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.pipelineGeneratorFor(req).ImplementNodes(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

func (h *Handlers) ScaffoldMCPHandler(c *gin.Context) {
	var req designer.MCPScaffoldRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.generator.ScaffoldMCP(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) DebugWorkflowHandler(c *gin.Context) {
	if h.generator == nil {
		slog.Warn("DebugWorkflowHandler called but h.generator is nil", "tenant_id", store.TenantIDFromContext(c.Request.Context()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI designer not configured (no LLM provider enabled)"})
		return
	}
	var req designer.DebugWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.generator.DebugWorkflow(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
