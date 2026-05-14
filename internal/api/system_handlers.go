package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSystemTelemetry returns the current telemetry state.
func (h *Handlers) GetSystemTelemetry(c *gin.Context) {
	enabled, _ := h.store.GetSystemSetting(c.Request.Context(), "enable_telemetry")
	c.JSON(http.StatusOK, gin.H{"enabled": enabled == "true"})
}

// UpdateSystemTelemetry updates the telemetry state.
func (h *Handlers) UpdateSystemTelemetry(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	val := "false"
	if req.Enabled {
		val = "true"
	}
	if err := h.store.SetSystemSetting(c.Request.Context(), "enable_telemetry", val); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
