package api

import (
	"log/slog"
	"net/http"

	"github.com/asm-platform/asm/pkg/asmtypes"
	"github.com/gin-gonic/gin"
)

// HandleChatOpsResolve provides a headless HITL endpoint for external ChatOps (Slack, Teams).
// In a full implementation, this should verify Slack signatures or require an API token.
func (h *Handlers) HandleChatOpsResolve(c *gin.Context) {
	id := c.Param("id")

	var payload struct {
		Resolution string                 `json:"resolution"` // "approved" | "rejected"
		Resolver   string                 `json:"resolver"`
		Comment    string                 `json:"comment"`
		Payload    map[string]interface{} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
		return
	}

	// Fetch HITL Request to verify existence and get RunID
	hitlReq, err := h.store.GetHITL(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "HITL request not found"})
		return
	}

	if hitlReq.Resolved {
		c.JSON(http.StatusConflict, gin.H{"error": "HITL request already resolved"})
		return
	}

	// Create Signal
	sig := asmtypes.HITLSignal{
		RunID:      hitlReq.RunID,
		Resolution: payload.Resolution,
		Resolver:   payload.Resolver,
		Comment:    payload.Comment,
		Payload:    payload.Payload,
	}

	// Resolve the HITL request via Orchestrator Engine
	if err := h.engine.SignalHITL(c.Request.Context(), sig); err != nil {
		slog.Error("Failed to resolve HITL request via ChatOps", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve HITL"})
		return
	}

	slog.Info("Successfully resolved HITL request via Headless ChatOps", "hitl_id", id, "run_id", hitlReq.RunID, "resolver", payload.Resolver)
	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
