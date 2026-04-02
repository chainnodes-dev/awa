package api

import (
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/designer"
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
func (h *Handlers) ListMCPServers(c *gin.Context) {
	entries, err := designer.LoadMCPRegistry(registryPath())
	if err != nil {
		// Non-fatal: if the registry file is missing, return an empty list.
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, designer.ToListItems(entries))
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

	result, err := h.generator.Generate(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp, err := designer.MarshalGenerateResult(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialise result"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
