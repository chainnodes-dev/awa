package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InvokeProcess starts an execution of a reusable workflow (process).
// POST /api/v1/invoke/:name
func (h *Handlers) InvokeProcess(c *gin.Context) {
	name := c.Param("name")
	var inputs map[string]interface{}
	if err := c.ShouldBindJSON(&inputs); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inputs: " + err.Error()})
		return
	}

	// Fetch the definition to check if it is explicitly marked as reusable.
	def, _, err := h.store.GetDefinition(c.Request.Context(), name, "latest")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "process not found"})
		return
	}

	if !def.Metadata.Reusable {
		c.JSON(http.StatusForbidden, gin.H{"error": "this workflow is not registered as a reusable process"})
		return
	}

	run, err := h.engine.StartRun(c.Request.Context(), name, "latest", inputs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"run_id": run.ID,
		"status": "started",
	})
}

// GetProcessInvocation returns the status of a specific process run.
// GET /api/v1/invoke/:name/runs/:run_id
func (h *Handlers) GetProcessInvocation(c *gin.Context) {
	runID := c.Param("run_id")
	run, err := h.engine.GetRun(c.Request.Context(), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	c.JSON(http.StatusOK, run)
}
