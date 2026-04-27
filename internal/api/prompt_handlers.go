package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPrompts returns the current system prompts (overrides + defaults).
// GET /api/v1/designer/prompts
func (h *Handlers) GetPrompts(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI designer not configured"})
		return
	}
	
	// For now, we return a flat list of known keys.
	// In the future we could scan the PromptManager.
	prompts := make(map[string]string)
	
	// Retrieve current values (either overrides or empty)
	prompts["workflow_generator_base"] = h.generator.GetPrompt("workflow_generator_base")
	prompts["skill_analyser_preamble"] = h.generator.GetPrompt("skill_analyser_preamble")
	prompts["workflow_refinement_addendum"] = h.generator.GetPrompt("workflow_refinement_addendum")
	
	// Pipeline steps
	prompts["workflow_decompose"] = h.generator.GetPrompt("workflow_decompose")
	prompts["workflow_categorise"] = h.generator.GetPrompt("workflow_categorise")
	prompts["workflow_wire"] = h.generator.GetPrompt("workflow_wire")
	prompts["workflow_implement_finish"] = h.generator.GetPrompt("workflow_implement_finish")

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
