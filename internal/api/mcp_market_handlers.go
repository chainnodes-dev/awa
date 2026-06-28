package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/asm-platform/asm/internal/mcp"
	"github.com/gin-gonic/gin"
)

// loadAndMergeMarketplace loads the marketplace catalog from the database (if an override exists)
// and merges any new or missing entries from the local configs/mcp-market.json file.
func (h *Handlers) loadAndMergeMarketplace(ctx context.Context) ([]mcp.MarketEntry, error) {
	// 1. Load the local JSON file catalog first
	localEntries, err := mcp.FetchMarketplace(ctx, h.cfg.MCPMarketSource)
	if err != nil {
		localEntries = mcp.GetDefaultMarketplace()
	}

	// 2. Check if there is an override in the database
	dbSource, err := h.store.GetSystemSetting(ctx, "mcp_market_source")
	if err != nil || dbSource == "" {
		// No database override exists; just return the local file entries
		return localEntries, nil
	}

	// 3. Load the database catalog override
	dbEntries, err := mcp.FetchMarketplace(ctx, dbSource)
	if err != nil {
		// If database source fails to parse, return local ones
		return localEntries, nil
	}

	// 4. Merge any missing or updated local entries into the database catalog override
	modified := false
	for _, local := range localEntries {
		found := false
		for i, dbEnt := range dbEntries {
			if strings.EqualFold(dbEnt.ID, local.ID) {
				// Curated templates should always be kept up-to-date with local JSON file definitions
				dbEntries[i] = local
				found = true
				modified = true
				break
			}
		}
		if !found {
			dbEntries = append(dbEntries, local)
			modified = true
		}
	}

	// 5. If we modified (merged new servers), save the updated list back to the DB override
	if modified {
		serialized, err := json.Marshal(dbEntries)
		if err == nil {
			_ = h.store.SetSystemSetting(ctx, "mcp_market_source", string(serialized))
		}
	}

	return dbEntries, nil
}

// GetMCPMarket returns a curated list of MCP servers from a marketplace source.
// This source can be a remote URL (curated by Chain Nodes or an enterprise) or a local file.
// GET /api/v1/mcp-market
func (h *Handlers) GetMCPMarket(c *gin.Context) {
	ctx := c.Request.Context()
	entries, err := h.loadAndMergeMarketplace(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
	entries, err := h.loadAndMergeMarketplace(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
