package api

import (
	"net/http"

	"github.com/asm-platform/asm/internal/mcp"
	"github.com/gin-gonic/gin"
)

// GetMCPMarket returns a curated list of MCP servers from a marketplace source.
// This source can be a remote URL (curated by Phaxa or an enterprise) or a local file.
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
