package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/tools"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// GetPortalForm retrieves the dynamic form schema and status for a public guest link.
// GET /api/v1/portal/:run_id/:token
func (h *Handlers) GetPortalForm(c *gin.Context) {
	runID := c.Param("run_id")
	rawToken := c.Param("token")

	if runID == "" || rawToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id and token are required"})
		return
	}

	tokenHash := store.HashGuestToken(rawToken)
	gt, err := h.store.GetGuestTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || gt == nil || gt.RunID != runID {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Guest portal link is invalid or not found.",
		})
		return
	}

	if gt.Status == "completed" {
		c.JSON(http.StatusOK, gin.H{
			"status":       "completed",
			"title":        gt.Title,
			"description":  gt.Description,
			"completed_at": gt.CompletedAt,
			"message":      "This form has already been submitted.",
		})
		return
	}

	if gt.Status != "active" {
		c.JSON(http.StatusOK, gin.H{
			"status":      gt.Status,
			"title":       gt.Title,
			"description": gt.Description,
			"message":     fmt.Sprintf("This guest link is %s.", gt.Status),
		})
		return
	}

	if time.Now().After(gt.ExpiresAt) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "expired",
			"title":       gt.Title,
			"description": gt.Description,
			"expires_at":  gt.ExpiresAt,
			"message":     "This guest link has expired.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "active",
		"run_id":      gt.RunID,
		"state_name":  gt.StateName,
		"title":       gt.Title,
		"description": gt.Description,
		"form_schema": gt.FormSchema,
		"expires_at":  gt.ExpiresAt,
	})
}

// UploadPortalFile allows guest users to upload required files (e.g. PDF documents, photos)
// without platform authentication, secured by the active guest token.
// POST /api/v1/portal/:run_id/:token/upload
func (h *Handlers) UploadPortalFile(c *gin.Context) {
	runID := c.Param("run_id")
	rawToken := c.Param("token")

	tokenHash := store.HashGuestToken(rawToken)
	gt, err := h.store.GetGuestTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || gt == nil || gt.RunID != runID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest portal link is invalid."})
		return
	}

	if !gt.IsValid() {
		c.JSON(http.StatusGone, gin.H{"error": "Guest portal link is expired or inactive."})
		return
	}

	var (
		fileID   string
		filename string
		fileSize int64
		mimeType string
	)

	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		var req struct {
			Name          string `json:"name"`
			ContentBase64 string `json:"content_base64"`
			MimeType      string `json:"mime_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
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

		fileID = uuid.New().String()
		filename = norm.NFC.String(filepath.Base(req.Name))
		fileSize = int64(len(data))
		mimeType = req.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		uploadDir := filepath.Join("data", "uploads", fileID)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create storage directory"})
			return
		}

		savePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(savePath, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
			return
		}
	} else {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowed := map[string]bool{
			".pdf": true, ".docx": true, ".txt": true, ".png": true,
			".jpg": true, ".jpeg": true, ".csv": true, ".json": true,
		}
		if !allowed[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File extension %s is not permitted", ext)})
			return
		}

		fileID = uuid.New().String()
		filename = norm.NFC.String(filepath.Base(file.Filename))
		fileSize = file.Size
		mimeType = file.Header.Get("Content-Type")

		uploadDir := filepath.Join("data", "uploads", fileID)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create storage: %v", err)})
			return
		}

		savePath := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":   fileID,
		"name":      filename,
		"size":      fileSize,
		"mime_type": mimeType,
		"url":       fmt.Sprintf("/api/v1/portal/%s/%s/files/%s/%s", runID, rawToken, fileID, filename),
	})
}

// GetPortalFile serves an uploaded file in the context of an active guest token.
// GET /api/v1/portal/:run_id/:token/files/:id/:filename
func (h *Handlers) GetPortalFile(c *gin.Context) {
	runID := c.Param("run_id")
	rawToken := c.Param("token")
	fileID := c.Param("id")
	filename := c.Param("filename")

	tokenHash := store.HashGuestToken(rawToken)
	gt, err := h.store.GetGuestTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || gt == nil || gt.RunID != runID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest portal link is invalid."})
		return
	}

	targetPath, _, err := tools.ResolveUploadedFilePath(fileID, filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.File(targetPath)
}

// PortalSubmitRequest represents the payload from guest form submission.
type PortalSubmitRequest struct {
	Resolution string                 `json:"resolution,omitempty"` // "approved", "rejected", "submitted"
	Submitter  string                 `json:"submitter,omitempty"`  // optional name/email
	Data       map[string]interface{} `json:"data"`                // form fields
}

// SubmitPortalForm processes form submission from an unauthenticated guest,
// unblocking the waiting workflow run and invalidating the guest token.
// POST /api/v1/portal/:run_id/:token/submit
func (h *Handlers) SubmitPortalForm(c *gin.Context) {
	runID := c.Param("run_id")
	rawToken := c.Param("token")

	tokenHash := store.HashGuestToken(rawToken)
	gt, err := h.store.GetGuestTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || gt == nil || gt.RunID != runID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest portal link is invalid."})
		return
	}

	if gt.Status == "completed" {
		c.JSON(http.StatusConflict, gin.H{"error": "This form has already been submitted."})
		return
	}

	if !gt.IsValid() {
		c.JSON(http.StatusGone, gin.H{"error": "This guest link is expired or inactive."})
		return
	}

	var req PortalSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission payload: " + err.Error()})
		return
	}

	if req.Data == nil {
		req.Data = make(map[string]interface{})
	}

	// Validate required fields if schema defines them
	if requiredList, ok := gt.FormSchema["required"].([]interface{}); ok {
		for _, reqField := range requiredList {
			if strField, ok := reqField.(string); ok {
				val, exists := req.Data[strField]
				if !exists || val == nil || val == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("Field '%s' is required", strField),
					})
					return
				}
			}
		}
	}

	// Determine resolution trigger
	resolution := req.Resolution
	if resolution == "" {
		if defaultRes, ok := gt.Metadata["default_resolution"].(string); ok && defaultRes != "" {
			resolution = defaultRes
		} else {
			resolution = "approved"
		}
	}

	submitter := req.Submitter
	if submitter == "" {
		submitter = "guest"
	}

	sig := asmtypes.HITLSignal{
		RunID:      gt.RunID,
		Resolution: resolution,
		Resolver:   fmt.Sprintf("guest:%s", submitter),
		Comment:    "Submitted via Guest Portal",
		Payload:    req.Data,
	}

	if err := h.engine.SignalHITL(c.Request.Context(), sig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve workflow task: " + err.Error()})
		return
	}

	// Mark token as completed immediately to prevent replay attacks
	_ = h.store.MarkGuestTokenCompleted(c.Request.Context(), gt.ID, submitter)

	c.JSON(http.StatusOK, gin.H{
		"status":  "completed",
		"message": "Thank you! Your submission has been received and processed.",
	})
}

// GenerateRunGuestToken generates a new guest portal token for an active HITL state on a run.
// POST /api/v1/runs/:id/guest-token
func (h *Handlers) GenerateRunGuestToken(c *gin.Context) {
	runID := c.Param("id")
	hitl, err := h.store.GetHITL(c.Request.Context(), runID)
	if err != nil || hitl == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No pending HITL task found for run " + runID})
		return
	}

	run, err := h.store.GetRun(c.Request.Context(), runID)
	if err != nil || run == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	var req struct {
		ExpiresIn   string `json:"expires_in"` // e.g. "72h", "7d"
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	_ = c.ShouldBindJSON(&req)

	ttl := 7 * 24 * time.Hour
	if req.ExpiresIn != "" {
		if d, err := time.ParseDuration(req.ExpiresIn); err == nil && d > 0 {
			ttl = d
		}
	}

	tokenPlain, tokenHash, err := store.GenerateGuestToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	title := req.Title
	if title == "" {
		if t, ok := hitl.FormSchema["title"].(string); ok && t != "" {
			title = t
		} else {
			title = fmt.Sprintf("Action Required: %s", hitl.StateName)
		}
	}

	desc := req.Description
	if desc == "" {
		if d, ok := hitl.FormSchema["description"].(string); ok && d != "" {
			desc = d
		}
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	portalURL := fmt.Sprintf("%s/portal/%s/%s", strings.TrimRight(baseURL, "/"), runID, tokenPlain)

	gt := &store.GuestToken{
		ID:          uuid.NewString(),
		TenantID:    run.TenantID,
		RunID:       runID,
		StateName:   hitl.StateName,
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(ttl),
		Status:      "active",
		Title:       title,
		Description: desc,
		FormSchema:  hitl.FormSchema,
		Metadata: map[string]interface{}{
			"workflow_name":    hitl.WorkflowName,
			"workflow_version": hitl.WorkflowVersion,
		},
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateGuestToken(c.Request.Context(), gt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist guest token: " + err.Error()})
		return
	}

	// Update HITL request metadata with link
	hitl.GuestAccess = true
	hitl.GuestPortalURL = portalURL
	hitl.GuestToken = tokenPlain
	_ = h.store.CreateHITL(c.Request.Context(), hitl)

	c.JSON(http.StatusOK, gin.H{
		"portal_url": portalURL,
		"token":      tokenPlain,
		"expires_at": gt.ExpiresAt,
	})
}
