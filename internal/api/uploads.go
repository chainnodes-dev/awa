package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadFile handles both multipart/form-data and application/json (Base64) file uploads.
// POST /api/v1/uploads
func (h *Handlers) UploadFile(c *gin.Context) {
	var (
		fileID   string
		filename string
		fileSize int64
		mimeType string
	)

	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		// Handle Base64 JSON upload
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
		filename = filepath.Base(req.Name)
		fileSize = int64(len(data))
		mimeType = req.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		uploadDir := filepath.Join("data", "uploads", fileID)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create storage"})
			return
		}

		savePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(savePath, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
	} else {
		// Handle Multipart upload
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Validate extension
		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowed := map[string]bool{".pdf": true, ".docx": true, ".txt": true, ".png": true, ".jpg": true, ".jpeg": true, ".csv": true, ".json": true}
		if !allowed[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File type %s not allowed", ext)})
			return
		}

		fileID = uuid.New().String()
		filename = filepath.Base(file.Filename)
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

	// Return metadata
	c.JSON(http.StatusOK, gin.H{
		"file_id":   fileID,
		"name":      filename,
		"size":      fileSize,
		"mime_type": mimeType,
		"url":       fmt.Sprintf("/api/v1/uploads/%s/%s", fileID, filename),
	})
}

// GetUploadedFile serves an uploaded file.
// GET /api/v1/uploads/:id/:filename
func (h *Handlers) GetUploadedFile(c *gin.Context) {
	fileID := c.Param("id")
	filename := c.Param("filename")

	// Prevent path traversal
	fileID = filepath.Base(fileID)
	filename = filepath.Base(filename)

	filePath := filepath.Join("data", "uploads", fileID, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.File(filePath)
}
