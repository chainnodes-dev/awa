package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asm-platform/asm/internal/store"
)

// Searcher defines the minimal search interface required by KnowledgeSearchTool.
type Searcher interface {
	Search(ctx context.Context, tenantID, collection, query string, topK int, minScore float64) ([]*store.KnowledgeCitation, error)
}

// KnowledgeSearchTool exposes RAG document search to workflow agents and action steps.
type KnowledgeSearchTool struct {
	searcher Searcher
	tenantID string
}

func NewKnowledgeSearchTool(searcher Searcher, tenantID string) *KnowledgeSearchTool {
	return &KnowledgeSearchTool{
		searcher: searcher,
		tenantID: tenantID,
	}
}

func (t *KnowledgeSearchTool) Name() string {
	return "knowledge:search"
}

func (t *KnowledgeSearchTool) Description() string {
	return "Performs semantic similarity vector search across indexed documents (PDFs, contracts, handbooks, guidelines). " +
		"Returns relevant excerpts with exact document filenames and page citations (e.g. '[Contract.pdf, Page 4]')."
}

func (t *KnowledgeSearchTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Natural language search query, question, or clause to find in the documents"
			},
			"collection": {
				"type": "string",
				"description": "Optional document collection to filter by (e.g. 'contracts', 'hr-policies', 'default'). Leave empty or 'all' to search across all collections."
			},
			"top_k": {
				"type": "integer",
				"description": "Maximum number of relevant excerpts to return (default 5, max 20)"
			},
			"min_score": {
				"type": "number",
				"description": "Minimum cosine similarity score between 0.0 and 1.0 (default 0.40)"
			}
		},
		"required": ["query"]
	}`)
}

type searchToolInput struct {
	Query      string  `json:"query"`
	Collection string  `json:"collection,omitempty"`
	TopK       int     `json:"top_k,omitempty"`
	MinScore   float64 `json:"min_score,omitempty"`
}

type searchToolOutput struct {
	Query        string                    `json:"query"`
	Collection   string                    `json:"collection"`
	MatchesCount int                       `json:"matches_count"`
	Results      []*store.KnowledgeCitation `json:"results"`
}

func (t *KnowledgeSearchTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in searchToolInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("invalid search tool input: %w", err)
	}

	if in.Query == "" {
		return nil, fmt.Errorf("'query' field is required")
	}

	topK := in.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	minScore := in.MinScore
	if minScore <= 0 {
		minScore = 0.40
	}

	tenantID := t.tenantID
	if tenantID == "" {
		tenantID = store.TenantIDFromContext(ctx)
	}
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

	results, err := t.searcher.Search(ctx, tenantID, in.Collection, in.Query, topK, minScore)
	if err != nil {
		return nil, fmt.Errorf("knowledge search error: %w", err)
	}

	out := searchToolOutput{
		Query:        in.Query,
		Collection:   in.Collection,
		MatchesCount: len(results),
		Results:      results,
	}

	return json.Marshal(out)
}
