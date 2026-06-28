package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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

// HandleSlackResolve handles the URL-encoded payload sent when a user clicks a Slack interactive button.
func (h *Handlers) HandleSlackResolve(c *gin.Context) {
	rawPayload := c.PostForm("payload")
	if rawPayload == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing payload"})
		return
	}

	var slackPayload struct {
		User struct {
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"user"`
		Actions []struct {
			Value string `json:"value"` // expected format: approved:hitlID or rejected:hitlID
		} `json:"actions"`
	}

	if err := json.Unmarshal([]byte(rawPayload), &slackPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Slack payload JSON"})
		return
	}

	if len(slackPayload.Actions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No actions found in payload"})
		return
	}

	actionValue := slackPayload.Actions[0].Value
	parts := strings.SplitN(actionValue, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action value format"})
		return
	}

	resolution := parts[0] // "approved" or "rejected"
	hitlID := parts[1]

	hitlReq, err := h.store.GetHITL(c.Request.Context(), hitlID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "HITL request not found"})
		return
	}

	if hitlReq.Resolved {
		c.JSON(http.StatusOK, gin.H{"text": "This HITL request has already been resolved."})
		return
	}

	resolver := slackPayload.User.Name
	if resolver == "" {
		resolver = slackPayload.User.Username
	}
	if resolver == "" {
		resolver = "Slack ChatOps"
	}

	sig := asmtypes.HITLSignal{
		RunID:      hitlReq.RunID,
		Resolution: resolution,
		Resolver:   resolver,
		Comment:    "Resolved via Slack button",
	}

	if err := h.engine.SignalHITL(c.Request.Context(), sig); err != nil {
		slog.Error("Failed to resolve HITL request via Slack ChatOps", "hitl_id", hitlID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve HITL"})
		return
	}

	statusText := "Approved"
	if resolution == "rejected" {
		statusText = "Rejected"
	}

	c.JSON(http.StatusOK, gin.H{
		"text": fmt.Sprintf("✓ HITL Request *%s* has been %s by %s.", hitlReq.StateName, statusText, resolver),
	})
}
