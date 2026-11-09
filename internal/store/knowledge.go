package store

import (
	"context"
	"time"
)

// KnowledgeDocument tracks an ingested source document (PDF, TXT, etc.).
type KnowledgeDocument struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Collection  string         `json:"collection"`
	Filename    string         `json:"filename"`
	ContentType string         `json:"content_type"`
	ByteSize    int64          `json:"byte_size"`
	PageCount   int            `json:"page_count"`
	Status      string         `json:"status"` // "indexing", "ready", "failed"
	ChunkCount  int            `json:"chunk_count"`
	ErrorMsg    string         `json:"error_msg,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   string         `json:"created_by"`
}

// KnowledgeChunk represents an embedded, searchable section of a document.
type KnowledgeChunk struct {
	ID         string         `json:"id"`
	DocumentID string         `json:"document_id"`
	TenantID   string         `json:"tenant_id"`
	Collection string         `json:"collection"`
	ChunkIndex int            `json:"chunk_index"`
	PageNumber int            `json:"page_number"`
	Content    string         `json:"content"`
	Embedding  []float32      `json:"-"` // excluded from regular JSON serialization
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// KnowledgeCitation is a search result with exact document and page reference.
type KnowledgeCitation struct {
	ChunkID    string         `json:"chunk_id"`
	DocumentID string         `json:"document_id"`
	Filename   string         `json:"filename"`
	Collection string         `json:"collection"`
	PageNumber int            `json:"page_number"`
	ChunkIndex int            `json:"chunk_index"`
	Content    string         `json:"content"`
	Score      float64        `json:"score"`
	Citation   string         `json:"citation"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// KnowledgeStore defines persistence and vector similarity search operations.
type KnowledgeStore interface {
	CreateDocument(ctx context.Context, doc *KnowledgeDocument) error
	GetDocument(ctx context.Context, id string) (*KnowledgeDocument, error)
	UpdateDocumentStatus(ctx context.Context, id, status string, chunkCount int, pageCount int, errorMsg string) error
	ListDocuments(ctx context.Context, tenantID, collection string) ([]*KnowledgeDocument, error)
	DeleteDocument(ctx context.Context, id string) error
	ListCollections(ctx context.Context, tenantID string) ([]string, error)

	InsertChunks(ctx context.Context, chunks []*KnowledgeChunk) error
	SearchChunks(ctx context.Context, tenantID, collection string, queryEmbedding []float32, topK int, minScore float64) ([]*KnowledgeCitation, error)
}
