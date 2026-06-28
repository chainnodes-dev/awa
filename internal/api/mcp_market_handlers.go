package api

import (
	"encoding/json"
	"net/http"

	"github.com/asm-platform/asm/internal/mcp"
	"github.com/gin-gonic/gin"
)

// GetMCPMarket returns a curated list of MCP servers from a marketplace source.
// This source can be a remote URL (curated by Chain Nodes or an enterprise) or a local file.
// GET /api/v1/mcp-market
func (h *Handlers) GetMCPMarket(c *gin.Context) {
	ctx := c.Request.Context()
	source := h.cfg.MCPMarketSource

	// Check if there is a dynamic override in the database
	if dbSource, err := h.store.GetSystemSetting(ctx, "mcp_market_source"); err == nil && dbSource != "" {
		source = dbSource
	}

	// FetchMarketplace handles both remote URLs and local file fallbacks
	entries, err := mcp.FetchMarketplace(ctx, source)
	if err != nil {
		// If the specific source fails, fall back to built-in curated list
		// rather than returning a 500 error, to ensure UI always works.
		entries = mcp.GetDefaultMarketplace()
	}

	c.JSON(http.StatusOK, entries)
}

// PublishMCPMarket allows users to publish a new custom MCP server/AgSOP to the local marketplace catalog override.
// POST /api/v1/mcp-market/publish
func (h *Handlers) PublishMCPMarket(c *gin.Context) {
	var payload mcp.MarketEntry
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid marketplace entry payload"})
		return
	}

	if payload.ID == "" || payload.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID and Name are required"})
		return
	}

	ctx := c.Request.Context()
	source := h.cfg.MCPMarketSource
	if dbSource, err := h.store.GetSystemSetting(ctx, "mcp_market_source"); err == nil && dbSource != "" {
		source = dbSource
	}

	entries, err := mcp.FetchMarketplace(ctx, source)
	if err != nil {
		entries = mcp.GetDefaultMarketplace()
	}

	// Append or update the entry
	found := false
	for i, existing := range entries {
		if existing.ID == payload.ID {
			entries[i] = payload
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, payload)
	}

	// Serialize catalog to JSON and write to settings as database catalog source override
	serialized, err := json.Marshal(entries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize catalog"})
		return
	}

	if err := h.store.SetSystemSetting(ctx, "mcp_market_source", string(serialized)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save marketplace catalog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published", "entry": payload})
}
