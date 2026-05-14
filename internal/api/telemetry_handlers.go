package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/asm-platform/asm/internal/telemetry"
)

func (h *Handlers) GetTelemetryStatus(c *gin.Context) {
	enabled, _ := h.store.GetSystemSetting(c.Request.Context(), "telemetry_enabled")
	if enabled == "" {
		enabled = "true" // default
	}

	// Generate a preview of what would be sent
	svc := telemetry.NewService(h.store, 0)
	report, err := svc.Collect(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled == "true",
		"report":  report,
	})
}

func (h *Handlers) ToggleTelemetry(c *gin.Context) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	val := "false"
	if body.Enabled {
		val = "true"
	}

	if err := h.store.SetSystemSetting(c.Request.Context(), "telemetry_enabled", val); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": body.Enabled})
}
