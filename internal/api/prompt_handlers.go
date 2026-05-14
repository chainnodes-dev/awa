package api

import (
	"net/http"

	"github.com/asm-platform/asm/internal/designer"
	"github.com/gin-gonic/gin"
)

// GetPrompts returns the current system prompts (overrides + defaults).
// GET /api/v1/designer/prompts
func (h *Handlers) GetPrompts(c *gin.Context) {
	prompts := make(map[string]string)
	
	keys := []string{
		"workflow_generation",
		"workflow_scaffold",
		"workflow_scaffold_mcp",
		"workflow_codegen",
		"workflow_analyse",
		"workflow_summarise",
		"workflow_categorise",
		"workflow_wire",
		"workflow_implement_finish",
		"workflow_debugger",
	}

	// Internal mapping from frontend keys to designer constants
	mapping := map[string]string{
		"workflow_generation":       designer.PromptIDBase,
		"workflow_scaffold":         designer.PromptIDSkill,
		"workflow_analyse":          designer.PromptIDDecompose,
		"workflow_categorise":       designer.PromptIDCategorise,
		"workflow_wire":             designer.PromptIDWire,
		"workflow_implement_finish": designer.PromptIDImplFinish,
		"workflow_debugger":         designer.PromptIDDebug,
	}

	for _, k := range keys {
		internalID := k
		if mapped, ok := mapping[k]; ok {
			internalID = mapped
		}

		if h.generator != nil {
			prompts[k] = h.generator.GetPrompt(internalID)
		} else {
			prompts[k] = designer.GetDefaultPrompt(internalID)
		}
	}

	c.JSON(http.StatusOK, prompts)
}

// UpdatePrompt updates a specific system prompt override.
// PUT /api/v1/designer/prompts/:id
func (h *Handlers) UpdatePrompt(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI designer not configured"})
		return
	}

	id := c.Param("id")
	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.generator.SetPrompt(id, body.Content)
	c.JSON(http.StatusOK, gin.H{"status": "updated", "id": id})
}

// Note: I need to add GetPrompt and SetPrompt methods to Generator in generator.go
