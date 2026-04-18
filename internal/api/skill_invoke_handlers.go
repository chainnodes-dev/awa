package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InvokeSkill starts a workflow run for the named skill.
// The caller supplies an optional input map that is passed as the initial blackboard.
// This endpoint accepts either a JWT Bearer token or an API key (RequireAuthOrAPIKey).
//
// POST /api/v1/invoke/:name
func (h *Handlers) InvokeSkill(c *gin.Context) {
	skillName := c.Param("name")

	var body struct {
		Input map[string]interface{} `json:"input"`
	}
	// Body is optional — an empty body is valid.
	_ = c.ShouldBindJSON(&body)

	ctx := c.Request.Context()

	// Resolve the latest version of the skill workflow definition.
	def, _, err := h.store.GetLatestDefinition(ctx, skillName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found: " + skillName})
		return
	}

	run, err := h.engine.StartRun(ctx, skillName, def.Metadata.Version, body.Input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"run_id": run.ID,
		"status": run.Status,
	})
}

// GetSkillInvocation returns the current state of a run started via InvokeSkill.
//
// GET /api/v1/invoke/:name/runs/:run_id
func (h *Handlers) GetSkillInvocation(c *gin.Context) {
	run, err := h.engine.GetRun(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
}
