package knowledge

import (
	"strings"

	"github.com/asm-platform/asm/internal/store"
	"github.com/google/uuid"
)

const (
	TargetChunkSize = 750
	ChunkOverlap    = 100
)

// ChunkPages splits paginated content into overlapping, searchable chunks with page citations.
func ChunkPages(docID, tenantID, collection string, pages []DocumentPage) []*store.KnowledgeChunk {
	var chunks []*store.KnowledgeChunk
	globalIndex := 0

	for _, page := range pages {
		pageText := strings.TrimSpace(page.Content)
		if pageText == "" {
			continue
		}

		if len(pageText) <= TargetChunkSize {
			chunks = append(chunks, &store.KnowledgeChunk{
				ID:         uuid.NewString(),
				DocumentID: docID,
				TenantID:   tenantID,
				Collection: collection,
				ChunkIndex: globalIndex,
				PageNumber: page.PageNumber,
				Content:    pageText,
			})
			globalIndex++
			continue
		}

		// Split page content into overlapping chunks
		pageChunks := splitTextWithOverlap(pageText, TargetChunkSize, ChunkOverlap)
		for _, content := range pageChunks {
			chunks = append(chunks, &store.KnowledgeChunk{
				ID:         uuid.NewString(),
				DocumentID: docID,
				TenantID:   tenantID,
				Collection: collection,
				ChunkIndex: globalIndex,
				PageNumber: page.PageNumber,
				Content:    content,
			})
			globalIndex++
		}
	}

	return chunks
}

func splitTextWithOverlap(text string, targetSize, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	total := len(runes)
	start := 0

	for start < total {
		end := start + targetSize
		if end >= total {
			chunkStr := strings.TrimSpace(string(runes[start:]))
			if chunkStr != "" {
				chunks = append(chunks, chunkStr)
			}
			break
		}

		// Look for sentence/paragraph breaks near targetSize to avoid breaking words
		adjustedEnd := findBreakPoint(runes, start, end)
		chunkStr := strings.TrimSpace(string(runes[start:adjustedEnd]))
		if chunkStr != "" {
			chunks = append(chunks, chunkStr)
		}

		// Step forward by (adjustedEnd - start - overlap)
		step := (adjustedEnd - start) - overlap
		if step <= 0 {
			step = adjustedEnd - start
		}
		start += step
	}

	return chunks
}

func findBreakPoint(runes []rune, start, preferredEnd int) int {
	// Look back up to 80 characters for a sentence boundary: '.', '?', '!', or '\n'
	lookbackLimit := preferredEnd - 80
	if lookbackLimit < start {
		lookbackLimit = start
	}

	for i := preferredEnd; i > lookbackLimit; i-- {
		ch := runes[i-1]
		if ch == '\n' || ch == '.' || ch == '?' || ch == '!' {
			return i
		}
	}

	// Fallback: look for space or tab
	for i := preferredEnd; i > lookbackLimit; i-- {
		ch := runes[i-1]
		if ch == ' ' || ch == '\t' {
			return i
		}
	}

	return preferredEnd
}
