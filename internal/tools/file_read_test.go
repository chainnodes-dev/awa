package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestResolveUploadedFilePath(t *testing.T) {
	tempDir := t.TempDir()
	origBaseDir := UploadsBaseDir
	UploadsBaseDir = tempDir
	defer func() { UploadsBaseDir = origBaseDir }()

	fileID := "test-upload-uuid"
	uploadFolder := filepath.Join(tempDir, fileID)
	if err := os.MkdirAll(uploadFolder, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	// 1. Create a file on disk using NFD format (common on macOS)
	// 'ü' in NFD is 'u' + \u0308
	nfdFilename := "test_lu\u0308ftung.txt"
	nfdFilePath := filepath.Join(uploadFolder, nfdFilename)
	if err := os.WriteFile(nfdFilePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Case 1: Search using NFC query (standard LLM output)
	nfcQuery := "test_lüftung.txt"
	resolvedPath, resolvedName, err := ResolveUploadedFilePath(fileID, nfcQuery)
	if err != nil {
		t.Fatalf("expected NFC query to match NFD file on disk, got error: %v", err)
	}
	if resolvedPath != nfdFilePath {
		t.Errorf("expected path %q, got %q", nfdFilePath, resolvedPath)
	}
	if resolvedName != nfdFilename {
		t.Errorf("expected name %q, got %q", nfdFilename, resolvedName)
	}

	// Case 2: Query with empty name when single file exists
	resolvedPath, _, err = ResolveUploadedFilePath(fileID, "")
	if err != nil {
		t.Fatalf("expected empty name query to resolve single file, got error: %v", err)
	}
	if resolvedPath != nfdFilePath {
		t.Errorf("expected path %q, got %q", nfdFilePath, resolvedPath)
	}

	// Case 3: Query with case-insensitive name
	resolvedPath, _, err = ResolveUploadedFilePath(fileID, "TEST_LÜFTUNG.TXT")
	if err != nil {
		t.Fatalf("expected case-insensitive match, got error: %v", err)
	}
	if resolvedPath != nfdFilePath {
		t.Errorf("expected path %q, got %q", nfdFilePath, resolvedPath)
	}

	// Case 4: Multiple files in directory
	otherFile := filepath.Join(uploadFolder, "other.txt")
	if err := os.WriteFile(otherFile, []byte("other"), 0644); err != nil {
		t.Fatalf("failed to write other file: %v", err)
	}
	resolvedPath, resolvedName, err = ResolveUploadedFilePath(fileID, "test_lüftung.txt")
	if err != nil {
		t.Fatalf("expected NFC query to match NFD file with multiple files present: %v", err)
	}
	if resolvedPath != nfdFilePath {
		t.Errorf("expected path %q, got %q", nfdFilePath, resolvedPath)
	}
	if resolvedName != nfdFilename {
		t.Errorf("expected name %q, got %q", nfdFilename, resolvedName)
	}

	// Case 5: Non-existent fileID
	_, _, err = ResolveUploadedFilePath("non-existent-id", "file.txt")
	if err == nil {
		t.Error("expected error for non-existent fileID, got nil")
	}
}

func TestReadUploadedFileTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	origBaseDir := UploadsBaseDir
	UploadsBaseDir = tempDir
	defer func() { UploadsBaseDir = origBaseDir }()

	fileID := "execute-uuid"
	uploadFolder := filepath.Join(tempDir, fileID)
	if err := os.MkdirAll(uploadFolder, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	fileName := norm.NFD.String("Klimamann_lüftung.txt")
	testData := []byte("Invoice total: 1200 EUR\nVAT: 19%")
	if err := os.WriteFile(filepath.Join(uploadFolder, fileName), testData, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tool := &ReadUploadedFileTool{}
	input := json.RawMessage(`{"file_id": "execute-uuid", "name": "Klimamann_lüftung.txt"}`)

	resBytes, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("tool execution failed: %v", err)
	}

	var res struct {
		FileName string `json:"file_name"`
		Size     int    `json:"size"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(resBytes, &res); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if res.Content != string(testData) {
		t.Errorf("expected content %q, got %q", string(testData), res.Content)
	}
	if res.Size != len(testData) {
		t.Errorf("expected size %d, got %d", len(testData), res.Size)
	}
}
