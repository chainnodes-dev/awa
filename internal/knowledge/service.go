package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/store"
)

type ingestJob struct {
	docID       string
	tenantID    string
	collection  string
	filename    string
	contentType string
	data        []byte
}

// Service coordinates document ingestion, background indexing, and semantic search.
type Service struct {
	store    store.Store
	embedder Embedder
	queue    chan *ingestJob
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewService instantiates the knowledge service with background worker pool.
func NewService(st store.Store, cfg *config.Config) *Service {
	embedder := NewCascadeEmbedder(cfg)
	s := &Service{
		store:    st,
		embedder: embedder,
		queue:    make(chan *ingestJob, 100),
		stopCh:   make(chan struct{}),
	}

	// Launch background indexing worker
	s.wg.Add(1)
	go s.workerLoop()

	return s
}

// Close gracefully shuts down the background indexing workers.
func (s *Service) Close() {
	close(s.stopCh)
	s.wg.Wait()
}

// QueueIngest schedules asynchronous document parsing, chunking, and vector embedding.
func (s *Service) QueueIngest(docID, tenantID, collection, filename, contentType string, data []byte) {
	select {
	case s.queue <- &ingestJob{
		docID:       docID,
		tenantID:    tenantID,
		collection:  collection,
		filename:    filename,
		contentType: contentType,
		data:        data,
	}:
	default:
		slog.Error("Knowledge ingestion queue full; dropping job", "docID", docID)
		_ = s.store.UpdateDocumentStatus(context.Background(), docID, "failed", 0, 0, "Ingestion queue capacity reached")
	}
}

func (s *Service) workerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case job := <-s.queue:
			s.processJob(job)
		}
	}
}

func (s *Service) processJob(job *ingestJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	slog.Info("Starting document RAG indexing", "docID", job.docID, "filename", job.filename)

	// 1. Parse pages
	pages, err := ParseDocument(job.filename, job.contentType, job.data)
	if err != nil {
		slog.Error("Failed to parse document", "docID", job.docID, "error", err)
		_ = s.store.UpdateDocumentStatus(ctx, job.docID, "failed", 0, 0, fmt.Sprintf("Parsing error: %v", err))
		return
	}

	// 2. Split into chunks
	chunks := ChunkPages(job.docID, job.tenantID, job.collection, pages)
	if len(chunks) == 0 {
		slog.Warn("Document yielded 0 text chunks", "docID", job.docID)
		_ = s.store.UpdateDocumentStatus(ctx, job.docID, "failed", 0, len(pages), "Document contains no readable text")
		return
	}

	// 3. Batch generate embeddings
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}

	embeddings, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		slog.Error("Embedding generation failed", "docID", job.docID, "error", err)
		_ = s.store.UpdateDocumentStatus(ctx, job.docID, "failed", 0, len(pages), fmt.Sprintf("Embedding error: %v", err))
		return
	}

	for i, emb := range embeddings {
		chunks[i].Embedding = emb
	}

	// 4. Store chunks in database
	if err := s.store.InsertChunks(ctx, chunks); err != nil {
		slog.Error("Failed to insert document chunks", "docID", job.docID, "error", err)
		_ = s.store.UpdateDocumentStatus(ctx, job.docID, "failed", 0, len(pages), fmt.Sprintf("Database insert error: %v", err))
		return
	}

	// 5. Update status to 'ready'
	if err := s.store.UpdateDocumentStatus(ctx, job.docID, "ready", len(chunks), len(pages), ""); err != nil {
		slog.Error("Failed to update document status to ready", "docID", job.docID, "error", err)
		return
	}

	slog.Info("Document RAG indexing complete", "docID", job.docID, "chunks", len(chunks), "pages", len(pages))
}

// Search performs semantic similarity search across indexed document chunks.
func (s *Service) Search(ctx context.Context, tenantID, collection, query string, topK int, minScore float64) ([]*store.KnowledgeCitation, error) {
	if topK <= 0 {
		topK = 5
	}
	if minScore < 0 {
		minScore = 0.0
	} else if minScore == 0 {
		minScore = 0.20 // default baseline for meaningful cosine match
	}

	// Generate embedding for search query
	queryEmbedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed search query: %w", err)
	}

	return s.store.SearchChunks(ctx, tenantID, collection, queryEmbedding, topK, minScore)
}

// ListDocuments lists documents in a given collection.
func (s *Service) ListDocuments(ctx context.Context, tenantID, collection string) ([]*store.KnowledgeDocument, error) {
	return s.store.ListDocuments(ctx, tenantID, collection)
}

// DeleteDocument removes a document and its indexed vector chunks.
func (s *Service) DeleteDocument(ctx context.Context, docID string) error {
	return s.store.DeleteDocument(ctx, docID)
}

// ListCollections lists all distinct collections for a tenant.
func (s *Service) ListCollections(ctx context.Context, tenantID string) ([]string, error) {
	return s.store.ListCollections(ctx, tenantID)
}
