package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// UploadsBaseDir is the relative or absolute directory where uploads are stored.
var UploadsBaseDir = filepath.Join("data", "uploads")

// ReadUploadedFileTool allows agents to read the content of files uploaded via the platform UI.
type ReadUploadedFileTool struct{}

func (t *ReadUploadedFileTool) Name() string { return "read_file" }

func (t *ReadUploadedFileTool) Description() string {
	return "Reads the text content of an uploaded document (PDF, TXT, CSV, JSON). " +
		"Use the 'file_id' and optionally 'name' found in the blackboard to identify the file."
}

func (t *ReadUploadedFileTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_id": { "type": "string", "description": "The unique ID of the uploaded file" },
			"name": { "type": "string", "description": "The original filename (optional if file_id is provided)" }
		},
		"required": ["file_id"]
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

	if args.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	targetPath, resolvedName, err := ResolveUploadedFilePath(args.FileID, args.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to locate uploaded file: %w", err)
	}

	raw, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(resolvedName))
	content := extractFileContent(ctx, targetPath, ext, raw)

	// Limit content length to avoid exceeding LLM context limits
	if len(content) > 50000 {
		content = content[:50000] + "\n\n[TRUNCATED...]"
	}

	return json.Marshal(map[string]interface{}{
		"file_name": resolvedName,
		"size":      len(raw),
		"content":   content,
	})
}

// ResolveUploadedFilePath resolves the path to an uploaded file, accounting for:
// 1. Direct path match
// 2. Single-file uploads in the fileID directory
// 3. Unicode normalization differences (NFC vs NFD, common when uploading from macOS)
// 4. Case-insensitive matches
func ResolveUploadedFilePath(fileID, name string) (string, string, error) {
	fileID = filepath.Base(fileID)
	dir := filepath.Join(UploadsBaseDir, fileID)

	// Check if directory exists
	if _, err := os.Stat(dir); err != nil {
		return "", "", fmt.Errorf("upload directory not found: %w", err)
	}

	// Scan directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", fmt.Errorf("cannot read upload directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}

	if len(files) == 0 {
		return "", "", fmt.Errorf("no files found in upload folder for id %s", fileID)
	}

	// If name was omitted, or if there is exactly 1 file in this upload's folder, return it
	if name == "" || len(files) == 1 {
		return filepath.Join(dir, files[0]), files[0], nil
	}

	cleanName := filepath.Base(name)
	targetNFC := norm.NFC.String(cleanName)
	targetNFD := norm.NFD.String(cleanName)

	// Check for Unicode normalization matches (NFC / NFD)
	for _, f := range files {
		fNFC := norm.NFC.String(f)
		fNFD := norm.NFD.String(f)
		if fNFC == targetNFC || fNFD == targetNFD || fNFC == targetNFD || fNFD == targetNFC {
			return filepath.Join(dir, f), f, nil
		}
	}

	// Check for case-insensitive match
	for _, f := range files {
		if strings.EqualFold(norm.NFC.String(f), targetNFC) {
			return filepath.Join(dir, f), f, nil
		}
	}

	return "", "", fmt.Errorf("file %q not found in %s (available: %s)", name, dir, strings.Join(files, ", "))
}

func extractFileContent(ctx context.Context, path, ext string, raw []byte) string {
	if ext == ".pdf" {
		// 1. Try pdftotext (poppler-utils)
		if _, err := exec.LookPath("pdftotext"); err == nil {
			cmd := exec.CommandContext(ctx, "pdftotext", "-layout", path, "-")
			out, err := cmd.Output()
			if err == nil && len(strings.TrimSpace(string(out))) > 0 {
				return string(out)
			}
		}

		// 2. Try python3 with pypdf if available
		if _, err := exec.LookPath("python3"); err == nil {
			pyScript := `
import sys
try:
    import pypdf
    reader = pypdf.PdfReader(sys.argv[1])
    text = "\n".join([p.extract_text() or "" for p in reader.pages])
    print(text)
except Exception:
    sys.exit(1)
`
			cmd := exec.CommandContext(ctx, "python3", "-c", pyScript, path)
			out, err := cmd.Output()
			if err == nil && len(strings.TrimSpace(string(out))) > 0 {
				return string(out)
			}
		}

		// 3. Fallback: extract printable strings
		printable := extractPrintableStrings(raw)
		if len(strings.TrimSpace(printable)) > 0 {
			return "[Extracted PDF text]:\n" + printable
		}

		return fmt.Sprintf("[PDF binary file: %d bytes, install poppler-utils for text extraction]", len(raw))
	}

	return string(raw)
}

func extractPrintableStrings(data []byte) string {
	var sb strings.Builder
	var current strings.Builder
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			current.WriteByte(b)
		} else {
			if current.Len() >= 4 {
				sb.WriteString(current.String())
				sb.WriteByte('\n')
			}
			current.Reset()
		}
	}
	if current.Len() >= 4 {
		sb.WriteString(current.String())
	}
	return sb.String()
}
