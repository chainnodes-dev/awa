package knowledge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// DocumentPage holds the extracted text of a single document page.
type DocumentPage struct {
	PageNumber int
	Content    string
}

// ParseDocument extracts paginated text content from file bytes based on content type.
func ParseDocument(filename, contentType string, data []byte) ([]DocumentPage, error) {
	lowerName := strings.ToLower(filename)
	lowerType := strings.ToLower(contentType)

	if strings.HasSuffix(lowerName, ".pdf") || strings.Contains(lowerType, "pdf") {
		return parsePDFPages(data)
	}

	// Plain text, markdown, CSV, JSON
	text := norm.NFC.String(string(data))
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	return []DocumentPage{
		{
			PageNumber: 1,
			Content:    strings.TrimSpace(text),
		},
	}, nil
}

func parsePDFPages(data []byte) ([]DocumentPage, error) {
	// Write to temporary file for pdftotext execution
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "rag-parse-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp pdf file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp pdf: %w", err)
	}
	tmpFile.Close()

	// Invoke pdftotext with layout preserving form-feed (\x0c) page separation
	cmd := exec.Command("pdftotext", "-layout", tmpFile.Name(), "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil && stdout.Len() > 0 {
		raw := norm.NFC.String(stdout.String())
		raw = strings.ReplaceAll(raw, "\r\n", "\n")
		// pdftotext delimits pages with form feed \x0c (\f)
		rawPages := strings.Split(raw, "\x0c")

		var pages []DocumentPage
		pageNum := 1
		for _, p := range rawPages {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				pages = append(pages, DocumentPage{
					PageNumber: pageNum,
					Content:    trimmed,
				})
			}
			pageNum++
		}
		if len(pages) > 0 {
			return pages, nil
		}
	}

	// Fallback if pdftotext is not available or extracted no text: extract printable strings
	extracted := extractPrintableASCII(data)
	if extracted == "" {
		extracted = fmt.Sprintf("[Document: %s (Binary PDF content could not be read as text)]", filepath.Base(tmpFile.Name()))
	}
	return []DocumentPage{
		{
			PageNumber: 1,
			Content:    extracted,
		},
	}, nil
}

func extractPrintableASCII(data []byte) string {
	var sb strings.Builder
	var word strings.Builder
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\t' {
			word.WriteByte(b)
		} else {
			if word.Len() >= 4 { // keep meaningful words
				sb.WriteString(word.String())
				sb.WriteByte(' ')
			}
			word.Reset()
		}
	}
	if word.Len() >= 4 {
		sb.WriteString(word.String())
	}
	return strings.TrimSpace(sb.String())
}
