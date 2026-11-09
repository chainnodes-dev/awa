package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *MemoryStore) CreateDocument(_ context.Context, doc *KnowledgeDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if doc.ID == "" {
		doc.ID = uuid.NewString()
	}
	if doc.Collection == "" {
		doc.Collection = "default"
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = doc.CreatedAt
	s.knowledgeDocs[doc.ID] = doc
	return nil
}

func (s *MemoryStore) GetDocument(_ context.Context, id string) (*KnowledgeDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.knowledgeDocs[id]
	if !ok {
		return nil, fmt.Errorf("knowledge document %q not found", id)
	}
	return doc, nil
}

func (s *MemoryStore) UpdateDocumentStatus(_ context.Context, id, status string, chunkCount int, pageCount int, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.knowledgeDocs[id]
	if !ok {
		return fmt.Errorf("knowledge document %q not found", id)
	}
	doc.Status = status
	if chunkCount > 0 {
		doc.ChunkCount = chunkCount
	}
	if pageCount > 0 {
		doc.PageCount = pageCount
	}
	doc.ErrorMsg = errorMsg
	doc.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) ListDocuments(_ context.Context, tenantID, collection string) ([]*KnowledgeDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*KnowledgeDocument
	for _, doc := range s.knowledgeDocs {
		if tenantID != "" && doc.TenantID != tenantID {
			continue
		}
		if collection != "" && collection != "all" && doc.Collection != collection {
			continue
		}
		result = append(result, doc)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *MemoryStore) DeleteDocument(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.knowledgeDocs, id)
	for chunkID, chunk := range s.knowledgeChunks {
		if chunk.DocumentID == id {
			delete(s.knowledgeChunks, chunkID)
		}
	}
	return nil
}

func (s *MemoryStore) ListCollections(_ context.Context, tenantID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collSet := make(map[string]struct{})
	for _, doc := range s.knowledgeDocs {
		if tenantID != "" && doc.TenantID != tenantID {
			continue
		}
		if doc.Collection != "" {
			collSet[doc.Collection] = struct{}{}
		}
	}

	result := make([]string, 0, len(collSet))
	for coll := range collSet {
		result = append(result, coll)
	}
	sort.Strings(result)
	return result, nil
}

func (s *MemoryStore) InsertChunks(_ context.Context, chunks []*KnowledgeChunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chunk := range chunks {
		if chunk.ID == "" {
			chunk.ID = uuid.NewString()
		}
		if chunk.CreatedAt.IsZero() {
			chunk.CreatedAt = time.Now()
		}
		s.knowledgeChunks[chunk.ID] = chunk
	}
	return nil
}

func (s *MemoryStore) SearchChunks(_ context.Context, tenantID, collection string, queryEmbedding []float32, topK int, minScore float64) ([]*KnowledgeCitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if topK <= 0 {
		topK = 5
	}

	type matchItem struct {
		chunk *KnowledgeChunk
		doc   *KnowledgeDocument
		score float64
	}

	var matches []matchItem
	for _, chunk := range s.knowledgeChunks {
		if tenantID != "" && chunk.TenantID != tenantID {
			continue
		}
		if collection != "" && collection != "all" && chunk.Collection != collection {
			continue
		}

		doc := s.knowledgeDocs[chunk.DocumentID]
		if doc == nil || doc.Status != "ready" {
			continue
		}

		score := cosineSimilarity(queryEmbedding, chunk.Embedding)
		if score >= minScore {
			matches = append(matches, matchItem{
				chunk: chunk,
				doc:   doc,
				score: score,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	if len(matches) > topK {
		matches = matches[:topK]
	}

	results := make([]*KnowledgeCitation, len(matches))
	for i, m := range matches {
		filename := m.doc.Filename
		citationStr := fmt.Sprintf("[%s, Page %d]", filename, m.chunk.PageNumber)
		if m.chunk.PageNumber <= 0 {
			citationStr = fmt.Sprintf("[%s, Section %d]", filename, m.chunk.ChunkIndex+1)
		}

		results[i] = &KnowledgeCitation{
			ChunkID:    m.chunk.ID,
			DocumentID: m.chunk.DocumentID,
			Filename:   filename,
			Collection: m.chunk.Collection,
			PageNumber: m.chunk.PageNumber,
			ChunkIndex: m.chunk.ChunkIndex,
			Content:    m.chunk.Content,
			Score:      math.Round(m.score*1000) / 1000,
			Citation:   citationStr,
			Metadata:   m.chunk.Metadata,
		}
	}
	return results, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		valA := float64(a[i])
		valB := float64(b[i])
		dot += valA * valB
		normA += valA * valA
		normB += valB * valB
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// VectorToString formats a float32 vector as a pgvector literal string "[0.123,0.456,...]".
func VectorToString(vec []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteByte(']')
	return sb.String()
}
