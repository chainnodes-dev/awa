package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadUploadedFileTool allows agents to read the content of files uploaded via the platform UI.
type ReadUploadedFileTool struct{}

func (t *ReadUploadedFileTool) Name() string { return "read_file" }

func (t *ReadUploadedFileTool) Description() string {
	return "Reads the text content of an uploaded document (PDF, TXT, CSV, JSON). " +
		"Use the 'file_id' and 'name' found in the blackboard to identify the file."
}

func (t *ReadUploadedFileTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_id": { "type": "string", "description": "The unique ID of the uploaded file" },
			"name": { "type": "string", "description": "The original filename" }
		},
		"required": ["file_id", "name"]
	}`)
}

func (t *ReadUploadedFileTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args struct {
		FileID string `json:"file_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	// Security: Prevent path traversal
	fileID := filepath.Base(args.FileID)
	name := filepath.Base(args.Name)

	path := filepath.Join("data", "uploads", fileID, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(name))
	
	var content string
	if ext == ".pdf" {
		// In a production environment, we would use a library like rsc.io/pdf or 
		// call an external OCR service. For this demo, we'll return a placeholder
		// or attempt a raw text read if it's a simple PDF.
		content = "[PDF CONTENT EXTRACTION LOGIC WOULD GO HERE]\n\n" + string(data)
	} else {
		content = string(data)
	}

	// Limit content length to avoid exceeding LLM context limits
	if len(content) > 50000 {
		content = content[:50000] + "\n\n[TRUNCATED...]"
	}

	return json.Marshal(map[string]interface{}{
		"file_name": name,
		"size":      len(data),
		"content":   content,
	})
}
