package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/asm-platform/asm/internal/catalog"
	"github.com/gin-gonic/gin"
)

// loadAndMergeNodeCatalog loads built-in and persistent custom node entries.
func (h *Handlers) loadAndMergeNodeCatalog(ctx context.Context) ([]catalog.NodeCatalogEntry, error) {
	// 1. Load from configured source (configs/node-catalog.json)
	localEntries, err := catalog.FetchNodeCatalog(ctx, h.cfg.NodeCatalogSource)
	if err != nil {
		localEntries = catalog.GetDefaultCatalog()
	}

	// 2. Check if there are overrides/additions in the database system settings
	dbSource, err := h.store.GetSystemSetting(ctx, "node_catalog_source")
	if err != nil || dbSource == "" {
		return localEntries, nil
	}

	// 3. Parse DB entries
	var dbEntries []catalog.NodeCatalogEntry
	if err := json.Unmarshal([]byte(dbSource), &dbEntries); err != nil {
		return localEntries, nil
	}

	// 4. Merge local curated entries into dbEntries
	mergedMap := make(map[string]catalog.NodeCatalogEntry)
	for _, entry := range localEntries {
		mergedMap[entry.ID] = entry
	}
	for _, entry := range dbEntries {
		// user entries override or add
		mergedMap[entry.ID] = entry
	}

	res := make([]catalog.NodeCatalogEntry, 0, len(mergedMap))
	// preserve curated order first
	seen := make(map[string]bool)
	for _, e := range localEntries {
		res = append(res, mergedMap[e.ID])
		seen[e.ID] = true
	}
	for _, e := range dbEntries {
		if !seen[e.ID] {
			res = append(res, e)
			seen[e.ID] = true
		}
	}

	return res, nil
}

// GetNodeCatalog returns the catalog of reusable custom nodes.
// GET /api/v1/node-catalog
func (h *Handlers) GetNodeCatalog(c *gin.Context) {
	ctx := c.Request.Context()
	entries, err := h.loadAndMergeNodeCatalog(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

// PublishNodeCatalog creates or updates a custom node in the catalog.
// POST /api/v1/node-catalog
func (h *Handlers) PublishNodeCatalog(c *gin.Context) {
	var payload catalog.NodeCatalogEntry
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node catalog payload"})
		return
	}

	if payload.ID == "" || payload.Name == "" || payload.CodeTemplate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID, Name, and CodeTemplate are required"})
		return
	}

	if payload.Version == "" {
		payload.Version = "1.0.0"
	}
	if payload.CreatedAt == "" {
		payload.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	ctx := c.Request.Context()
	entries, err := h.loadAndMergeNodeCatalog(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	serialized, err := json.Marshal(entries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize catalog"})
		return
	}

	if err := h.store.SetSystemSetting(ctx, "node_catalog_source", string(serialized)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save node catalog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published", "entry": payload})
}

// DeleteNodeCatalog deletes a custom node from the catalog.
// DELETE /api/v1/node-catalog/:id
func (h *Handlers) DeleteNodeCatalog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter required"})
		return
	}

	ctx := c.Request.Context()
	entries, err := h.loadAndMergeNodeCatalog(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filtered := make([]catalog.NodeCatalogEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catalog item not found"})
		return
	}

	serialized, err := json.Marshal(filtered)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize catalog"})
		return
	}

	if err := h.store.SetSystemSetting(ctx, "node_catalog_source", string(serialized)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save node catalog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
}

