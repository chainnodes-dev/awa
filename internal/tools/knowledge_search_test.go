package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/asm-platform/asm/internal/store"
)

type mockSearcher struct {
	results []*store.KnowledgeCitation
}

func (m *mockSearcher) Search(ctx context.Context, tenantID, collection, query string, topK int, minScore float64) ([]*store.KnowledgeCitation, error) {
	return m.results, nil
}

func TestKnowledgeSearchTool(t *testing.T) {
	mock := &mockSearcher{
		results: []*store.KnowledgeCitation{
			{
				ChunkID:    "chunk-1",
				DocumentID: "doc-1",
				Filename:   "Vendor_Contract_2026.pdf",
				Collection: "contracts",
				PageNumber: 7,
				ChunkIndex: 3,
				Content:    "Payment terms are net 30 days upon invoice receipt.",
				Score:      0.88,
				Citation:   "[Vendor_Contract_2026.pdf, Page 7]",
			},
		},
	}

	tool := NewKnowledgeSearchTool(mock, "test-tenant")
	if tool.Name() != "knowledge:search" {
		t.Fatalf("expected knowledge:search, got %s", tool.Name())
	}

	inputJSON := []byte(`{"query": "payment terms net days", "collection": "contracts"}`)
	outputJSON, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("tool execution failed: %v", err)
	}

	var out searchToolOutput
	if err := json.Unmarshal(outputJSON, &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}

	if out.MatchesCount != 1 {
		t.Errorf("expected 1 match, got %d", out.MatchesCount)
	}
	if len(out.Results) != 1 || out.Results[0].Citation != "[Vendor_Contract_2026.pdf, Page 7]" {
		t.Errorf("unexpected results: %+v", out.Results)
	}
}
