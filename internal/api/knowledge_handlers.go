package api

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadKnowledgeDocument handles PDF / TXT / Markdown file uploads for Vector RAG.
// POST /api/v1/knowledge/upload
func (h *Handlers) UploadKnowledgeDocument(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

	claims := auth.ClaimsFrom(c)
	createdBy := "system"
	if claims != nil && claims.Username != "" {
		createdBy = claims.Username
	}

	var (
		filename    string
		contentType string
		fileBytes   []byte
		collection  string
	)

	contentTypeHeader := c.GetHeader("Content-Type")
	if strings.Contains(contentTypeHeader, "application/json") {
		var req struct {
			Name          string `json:"name"`
			Collection    string `json:"collection"`
			ContentBase64 string `json:"content_base64"`
			ContentType   string `json:"content_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload: " + err.Error()})
			return
		}
		if req.Name == "" || req.ContentBase64 == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and content_base64 are required"})
			return
		}
		data, err := base64.StdEncoding.DecodeString(req.ContentBase64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		filename = req.Name
		contentType = req.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		collection = req.Collection
		fileBytes = data
	} else {
		// Multipart upload
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file form field is required"})
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file: " + err.Error()})
			return
		}

		filename = header.Filename
		contentType = header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		collection = c.Request.FormValue("collection")
		fileBytes = data
	}

	if collection == "" {
		collection = "default"
	}

	doc := &store.KnowledgeDocument{
		ID:          uuid.NewString(),
		TenantID:    tenantID,
		Collection:  collection,
		Filename:    filename,
		ContentType: contentType,
		ByteSize:    int64(len(fileBytes)),
		Status:      "indexing",
		CreatedBy:   createdBy,
	}

	if err := h.store.CreateDocument(c.Request.Context(), doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document record: " + err.Error()})
		return
	}

	// Trigger asynchronous indexing
	h.GetKnowledgeService().QueueIngest(doc.ID, tenantID, collection, filename, contentType, fileBytes)

	c.JSON(http.StatusCreated, doc)
}

// ListKnowledgeDocuments returns all uploaded documents for a tenant, optionally filtered by collection.
// GET /api/v1/knowledge/documents
func (h *Handlers) ListKnowledgeDocuments(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}
	collection := c.Query("collection")

	docs, err := h.GetKnowledgeService().ListDocuments(c.Request.Context(), tenantID, collection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list documents: " + err.Error()})
		return
	}
	if docs == nil {
		docs = []*store.KnowledgeDocument{}
	}
	c.JSON(http.StatusOK, docs)
}

// DeleteKnowledgeDocument removes a document and its indexed vector chunks.
// DELETE /api/v1/knowledge/documents/:id
func (h *Handlers) DeleteKnowledgeDocument(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document ID is required"})
		return
	}

	if err := h.GetKnowledgeService().DeleteDocument(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
}

// ListKnowledgeCollections returns distinct collection names for a tenant.
// GET /api/v1/knowledge/collections
func (h *Handlers) ListKnowledgeCollections(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

	colls, err := h.GetKnowledgeService().ListCollections(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list collections: " + err.Error()})
		return
	}
	if colls == nil {
		colls = []string{}
	}
	c.JSON(http.StatusOK, colls)
}

// SearchKnowledge allows live query testing from the platform console.
// POST /api/v1/knowledge/search
func (h *Handlers) SearchKnowledge(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

	var req struct {
		Query      string  `json:"query"`
		Collection string  `json:"collection"`
		TopK       int     `json:"top_k"`
		MinScore   float64 `json:"min_score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search request: " + err.Error()})
		return
	}
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query cannot be empty"})
		return
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = 0.40
	}

	results, err := h.GetKnowledgeService().Search(c.Request.Context(), tenantID, req.Collection, req.Query, topK, minScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed: " + err.Error()})
		return
	}
	if results == nil {
		results = []*store.KnowledgeCitation{}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":         req.Query,
		"collection":    req.Collection,
		"matches_count": len(results),
		"results":       results,
	})
}
