package knowledge

import (
	"context"
	"testing"
	"time"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/store"
)

func TestParseDocument_PlainText(t *testing.T) {
	content := "Line 1\r\nLine 2\nLine 3"
	pages, err := ParseDocument("test.txt", "text/plain", []byte(content))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].PageNumber != 1 {
		t.Errorf("expected page 1, got %d", pages[0].PageNumber)
	}
	if pages[0].Content != "Line 1\nLine 2\nLine 3" {
		t.Errorf("unexpected content: %q", pages[0].Content)
	}
}

func TestChunkPages_ShortAndLong(t *testing.T) {
	pages := []DocumentPage{
		{
			PageNumber: 1,
			Content:    "This is a short page.",
		},
		{
			PageNumber: 2,
			Content:    "Sentence one. " + string(make([]byte, 1000)) + " Sentence two.",
		},
	}

	chunks := ChunkPages("doc-1", "tenant-1", "policies", pages)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	if chunks[0].PageNumber != 1 {
		t.Errorf("chunk 0 should be page 1, got %d", chunks[0].PageNumber)
	}
	if chunks[0].Collection != "policies" {
		t.Errorf("chunk 0 collection should be policies, got %s", chunks[0].Collection)
	}
	if chunks[1].PageNumber != 2 {
		t.Errorf("chunk 1 should be page 2, got %d", chunks[1].PageNumber)
	}
}

func TestKnowledgeService_IngestAndSearch(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		EmbeddingModel: "text-embedding-3-small",
		EmbeddingDim:   64, // fast test dimension
	}

	svc := NewService(memStore, cfg)
	defer svc.Close()

	ctx := context.Background()
	tenantID := "test-tenant"
	docID := "doc-handbook-1"

	doc := &store.KnowledgeDocument{
		ID:          docID,
		TenantID:    tenantID,
		Collection:  "hr-policies",
		Filename:    "Employee_Handbook_2026.txt",
		ContentType: "text/plain",
		Status:      "indexing",
	}
	if err := memStore.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}

	text := "Vacation policy: All employees receive 25 days of paid annual leave. " +
		"Notice for leave must be submitted two weeks in advance. " +
		"Health benefits: Comprehensive medical and dental insurance is provided from day one."

	svc.QueueIngest(docID, tenantID, "hr-policies", "Employee_Handbook_2026.txt", "text/plain", []byte(text))

	// Wait for background worker
	var indexedDoc *store.KnowledgeDocument
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		indexedDoc, _ = memStore.GetDocument(ctx, docID)
		if indexedDoc != nil && indexedDoc.Status == "ready" {
			break
		}
	}

	if indexedDoc == nil || indexedDoc.Status != "ready" {
		t.Fatalf("document did not reach 'ready' status in time, got: %+v", indexedDoc)
	}
	if indexedDoc.ChunkCount == 0 {
		t.Fatalf("expected chunk count > 0, got %d", indexedDoc.ChunkCount)
	}

	// Perform search
	citations, err := svc.Search(ctx, tenantID, "hr-policies", "vacation days annual leave", 3, 0.0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(citations) == 0 {
		t.Fatalf("expected search citations, got 0")
	}

	first := citations[0]
	if first.Filename != "Employee_Handbook_2026.txt" {
		t.Errorf("expected filename Employee_Handbook_2026.txt, got %s", first.Filename)
	}
	if first.Citation == "" {
		t.Errorf("expected citation string, got empty")
	}
	if first.Score <= 0 {
		t.Errorf("expected positive similarity score, got %f", first.Score)
	}
}
