package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/mcp"
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
		Name        string   `json:"name"        binding:"required"`
		Transport   string   `json:"transport"   binding:"required"` // "sse" or "stdio"
		URL         string   `json:"url"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		EnvVar      string   `json:"env_var"`
		EnvValue    string   `json:"env_value"`
		Description string   `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Transport == "stdio" && !h.allowStdioMCP {
		c.JSON(http.StatusForbidden, gin.H{"error": "Local stdio MCP servers are disabled by the system administrator"})
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
		EnvVar:      body.EnvVar,
		EnvValue:    body.EnvValue,
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
		Transport   string   `json:"transport"`
		URL         string   `json:"url"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		EnvVar      string   `json:"env_var"`
		EnvValue    string   `json:"env_value"`
		Description string   `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Transport == "stdio" && !h.allowStdioMCP {
		c.JSON(http.StatusForbidden, gin.H{"error": "Local stdio MCP servers are disabled by the system administrator"})
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
		EnvVar:      body.EnvVar,
		EnvValue:    body.EnvValue,
		Description: body.Description,
		CreatedBy:   existing.CreatedBy,
		CreatedAt:   existing.CreatedAt,
	}
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

// PingMCPServer performs a lightweight HTTP health check against an MCP server.
// POST /api/v1/mcp-servers/:id/ping
func (h *Handlers) PingMCPServer(c *gin.Context) {
	srv, err := h.store.GetMCPServer(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if srv.Transport == "stdio" {
		start := time.Now()
		_, err := h.mcpManager.GetClient(c.Request.Context(), srv.Name)
		duration := time.Since(start).Milliseconds()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "offline", "error": err.Error(), "latency_ms": duration})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "online", "latency_ms": duration})
		return
	}

	// Handle stdio transport
	if srv.Transport == "stdio" {
		if srv.Command == "" {
			c.JSON(http.StatusOK, gin.H{"status": "unknown", "error": "no command configured"})
			return
		}
		// Try to start a test client
		_, closer, err := h.mcpManager.TestStdioClient(srv.Command, srv.Args, srv.EnvVar, srv.EnvValue)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "offline", "error": fmt.Sprintf("failed to start process: %v", err)})
			return
		}
		closer.Close()
		c.JSON(http.StatusOK, gin.H{"status": "online", "latency_ms": 0})
		return
	}

	// Resolve the URL: prefer direct URL, fall back to env var.
	url := srv.URL

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
		Transport string   `json:"transport"`
		URL       string   `json:"url"`
		Command   string   `json:"command"`
		Args      []string `json:"args"`
		EnvVar    string   `json:"env_var"`
		EnvValue  string   `json:"env_value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if body.Transport == "stdio" && !h.allowStdioMCP {
		c.JSON(http.StatusForbidden, gin.H{"error": "Local stdio MCP servers are disabled"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var client mcp.Client

	if body.Transport == "stdio" {
		if body.Command == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "command is required for stdio transport"})
			return
		}
		stdioClient, closer, err := h.mcpManager.TestStdioClient(body.Command, body.Args, body.EnvVar, body.EnvValue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start server: %v", err)})
			return
		}
		defer closer.Close()
		client = stdioClient
	} else {
		if body.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url is required for sse transport"})
			return
		}
		client = mcp.NewSSEClient(body.URL)
	}

	// 1. Initialize
	initReq := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "asm-platform-discovery",
			"version": "1.0",
		},
	}
	_, err := client.Call(ctx, "initialize", initReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": fmt.Sprintf("Failed to initialize MCP: %v", err)})
		return
	}

	// 2. Initialized notification
	if err := client.Notify(ctx, "notifications/initialized", map[string]interface{}{}); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": fmt.Sprintf("Failed to send initialized notification: %v", err)})
		return
	}

	// 3. Fetch tools
	toolsResult, err := client.Call(ctx, "tools/list", map[string]interface{}{})
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

// ProcessAnalyseHandler decomposes a process description into an ASM workflow YAML.
// POST /api/v1/designer/process/analyse
func (h *Handlers) ProcessAnalyseHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "process analysis is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.ProcessAnalyseRequest
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

	result, err := gen.AnalyseProcess(c.Request.Context(), req)
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

// ProcessSummariseHandler converts a WorkflowDef YAML into a human-readable process summary.
// POST /api/v1/designer/process/summarise
func (h *Handlers) ProcessSummariseHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "process summarisation is not available: no LLM provider configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)",
		})
		return
	}

	var req designer.ProcessSummariseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.WorkflowYAML) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow_yaml is required"})
		return
	}

	result, err := h.generator.SummariseProcess(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── Multi-step pipeline handlers ─────────────────────────────────────────────

// pipelineGeneratorFor resolves the generator for a pipeline request (provider override).
func (h *Handlers) pipelineGeneratorFor(req designer.PipelineRequest) *designer.Generator {
	if req.Provider != "" {
		if p, err := h.llmRegistry.Get(req.Provider); err == nil {
			return h.generator.WithProvider(p)
		}
	}
	return h.generator
}

// requirePipelineYAML returns false and writes a 400 if current_yaml is missing.
func requirePipelineYAML(c *gin.Context, yaml string) bool {
	if strings.TrimSpace(yaml) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_yaml is required"})
		return false
	}
	return true
}

func writePipelineResult(c *gin.Context, result *designer.PipelineStepResult, err error) {
	if err != nil {
		resp := gin.H{"error": err.Error()}
		if result != nil {
			resp["interactions"] = result.Interactions
			if result.YAML != "" {
				resp["yaml"] = result.YAML
			}
		}
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	out, mErr := designer.MarshalPipelineStepResult(result)
	if mErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialise result"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// PipelineDecomposeHandler is Step 1: identify states/nodes from a process description.
// POST /api/v1/designer/pipeline/decompose
func (h *Handlers) PipelineDecomposeHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no LLM provider configured"})
		return
	}
	var req designer.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ProcessDescription) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "process_description is required"})
		return
	}
	result, err := h.pipelineGeneratorFor(req).DecomposeWorkflow(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

// PipelineCategoriseHandler is Step 2: assign correct ASM types to all nodes.
// POST /api/v1/designer/pipeline/categorise
func (h *Handlers) PipelineCategoriseHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no LLM provider configured"})
		return
	}
	var req designer.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ProcessDescription) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "process_description is required"})
		return
	}
	if !requirePipelineYAML(c, req.CurrentYAML) {
		return
	}
	result, err := h.pipelineGeneratorFor(req).CategoriseNodes(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

// PipelineWireHandler is Step 3: define transitions and blackboard schema.
// POST /api/v1/designer/pipeline/wire
func (h *Handlers) PipelineWireHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no LLM provider configured"})
		return
	}
	var req designer.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ProcessDescription) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "process_description is required"})
		return
	}
	if !requirePipelineYAML(c, req.CurrentYAML) {
		return
	}
	result, err := h.pipelineGeneratorFor(req).WireWorkflow(c.Request.Context(), req)
	writePipelineResult(c, result, err)
}

// PipelineImplementHandler is Step 4: generate final code for all nodes in parallel.
// POST /api/v1/designer/pipeline/implement
func (h *Handlers) PipelineImplementHandler(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no LLM provider configured"})
		return
	}
	var req designer.PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ProcessDescription) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "process_description is required"})
		return
	}
	if !requirePipelineYAML(c, req.CurrentYAML) {
		return
	}
	result, err := h.pipelineGeneratorFor(req).ImplementNodes(c.Request.Context(), req)
	writePipelineResult(c, result, err)
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
