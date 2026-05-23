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
		"workflow_generator_base",
		"skill_analyser_preamble",
		"workflow_refinement_addendum",
		"workflow_decompose",
		"workflow_categorise",
		"workflow_wire",
		"workflow_implement_finish",
		"workflow_debugger",
	}

	// Internal mapping from frontend keys to designer constants
	mapping := map[string]string{
		"workflow_generator_base":      designer.PromptIDBase,
		"skill_analyser_preamble":      designer.PromptIDSkill,
		"workflow_refinement_addendum": designer.PromptIDRefine,
		"workflow_decompose":           designer.PromptIDDecompose,
		"workflow_categorise":          designer.PromptIDCategorise,
		"workflow_wire":                designer.PromptIDWire,
		"workflow_implement_finish":    designer.PromptIDImplFinish,
		"workflow_debugger":            designer.PromptIDDebug,
	}

	for _, k := range keys {
		internalID := mapping[k]
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
	
	// Map frontend key to internal designer ID
	mapping := map[string]string{
		"workflow_generator_base":      designer.PromptIDBase,
		"skill_analyser_preamble":      designer.PromptIDSkill,
		"workflow_refinement_addendum": designer.PromptIDRefine,
		"workflow_decompose":           designer.PromptIDDecompose,
		"workflow_categorise":          designer.PromptIDCategorise,
		"workflow_wire":                designer.PromptIDWire,
		"workflow_implement_finish":    designer.PromptIDImplFinish,
		"workflow_debugger":            designer.PromptIDDebug,
	}

	internalID := id
	if mapped, ok := mapping[id]; ok {
		internalID = mapped
	}

	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.generator.SetPrompt(internalID, body.Content)
	c.JSON(http.StatusOK, gin.H{"status": "updated", "id": internalID})
}

// Note: I need to add GetPrompt and SetPrompt methods to Generator in generator.go
