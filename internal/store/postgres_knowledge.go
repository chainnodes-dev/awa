package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

func (s *PostgresStore) CreateDocument(ctx context.Context, doc *KnowledgeDocument) error {
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

	metaJSON, _ := json.Marshal(doc.Metadata)
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO knowledge_documents (
			id, tenant_id, collection, filename, content_type,
			byte_size, page_count, status, chunk_count, error_msg,
			metadata, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		doc.ID, doc.TenantID, doc.Collection, doc.Filename, doc.ContentType,
		doc.ByteSize, doc.PageCount, doc.Status, doc.ChunkCount, doc.ErrorMsg,
		metaJSON, doc.CreatedAt, doc.UpdatedAt, doc.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create knowledge document: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetDocument(ctx context.Context, id string) (*KnowledgeDocument, error) {
	var doc KnowledgeDocument
	var metaJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, collection, filename, content_type,
		        byte_size, page_count, status, chunk_count, COALESCE(error_msg, ''),
		        metadata, created_at, updated_at, created_by
		 FROM knowledge_documents WHERE id = $1`,
		id,
	).Scan(
		&doc.ID, &doc.TenantID, &doc.Collection, &doc.Filename, &doc.ContentType,
		&doc.ByteSize, &doc.PageCount, &doc.Status, &doc.ChunkCount, &doc.ErrorMsg,
		&metaJSON, &doc.CreatedAt, &doc.UpdatedAt, &doc.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get knowledge document %q: %w", id, err)
	}
	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &doc.Metadata)
	}
	return &doc, nil
}

func (s *PostgresStore) UpdateDocumentStatus(ctx context.Context, id, status string, chunkCount int, pageCount int, errorMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE knowledge_documents
		 SET status = $2,
		     chunk_count = CASE WHEN $3 > 0 THEN $3 ELSE chunk_count END,
		     page_count = CASE WHEN $4 > 0 THEN $4 ELSE page_count END,
		     error_msg = $5,
		     updated_at = NOW()
		 WHERE id = $1`,
		id, status, chunkCount, pageCount, errorMsg,
	)
	if err != nil {
		return fmt.Errorf("update knowledge document status: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListDocuments(ctx context.Context, tenantID, collection string) ([]*KnowledgeDocument, error) {
	query := `SELECT id, tenant_id, collection, filename, content_type,
	                 byte_size, page_count, status, chunk_count, COALESCE(error_msg, ''),
	                 metadata, created_at, updated_at, created_by
	          FROM knowledge_documents
	          WHERE ($1 = '' OR tenant_id = $1::uuid)
	            AND ($2 = '' OR $2 = 'all' OR collection = $2)
	          ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, tenantID, collection)
	if err != nil {
		return nil, fmt.Errorf("list knowledge documents: %w", err)
	}
	defer rows.Close()

	var docs []*KnowledgeDocument
	for rows.Next() {
		var doc KnowledgeDocument
		var metaJSON []byte
		if err := rows.Scan(
			&doc.ID, &doc.TenantID, &doc.Collection, &doc.Filename, &doc.ContentType,
			&doc.ByteSize, &doc.PageCount, &doc.Status, &doc.ChunkCount, &doc.ErrorMsg,
			&metaJSON, &doc.CreatedAt, &doc.UpdatedAt, &doc.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan knowledge document: %w", err)
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &doc.Metadata)
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}

func (s *PostgresStore) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM knowledge_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete knowledge document: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListCollections(ctx context.Context, tenantID string) ([]string, error) {
	query := `SELECT DISTINCT collection
	          FROM knowledge_documents
	          WHERE ($1 = '' OR tenant_id = $1::uuid)
	          ORDER BY collection ASC`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	var collections []string
	for rows.Next() {
		var coll string
		if err := rows.Scan(&coll); err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		collections = append(collections, coll)
	}
	return collections, nil
}

func (s *PostgresStore) InsertChunks(ctx context.Context, chunks []*KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin chunk insert transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	stmt := `INSERT INTO knowledge_chunks (
		id, document_id, tenant_id, collection, chunk_index,
		page_number, content, embedding, metadata, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::vector, $9, $10)`

	for _, c := range chunks {
		if c.ID == "" {
			c.ID = uuid.NewString()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now()
		}
		metaJSON, _ := json.Marshal(c.Metadata)
		if len(metaJSON) == 0 {
			metaJSON = []byte("{}")
		}

		vecStr := VectorToString(c.Embedding)
		_, err := tx.Exec(ctx, stmt,
			c.ID, c.DocumentID, c.TenantID, c.Collection, c.ChunkIndex,
			c.PageNumber, c.Content, vecStr, metaJSON, c.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", c.ChunkIndex, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit chunk inserts: %w", err)
	}
	return nil
}

func (s *PostgresStore) SearchChunks(ctx context.Context, tenantID, collection string, queryEmbedding []float32, topK int, minScore float64) ([]*KnowledgeCitation, error) {
	if topK <= 0 {
		topK = 5
	}
	vecStr := VectorToString(queryEmbedding)

	query := `SELECT
	              c.id,
	              c.document_id,
	              d.filename,
	              c.collection,
	              c.page_number,
	              c.chunk_index,
	              c.content,
	              c.metadata,
	              1 - (c.embedding <=> $1::vector) AS score
	          FROM knowledge_chunks c
	          JOIN knowledge_documents d ON c.document_id = d.id
	          WHERE ($2 = '' OR c.tenant_id = $2::uuid)
	            AND d.status = 'ready'
	            AND ($3 = '' OR $3 = 'all' OR c.collection = $3)
	            AND (1 - (c.embedding <=> $1::vector)) >= $4
	          ORDER BY c.embedding <=> $1::vector ASC
	          LIMIT $5`

	rows, err := s.pool.Query(ctx, query, vecStr, tenantID, collection, minScore, topK)
	if err != nil {
		return nil, fmt.Errorf("search knowledge chunks: %w", err)
	}
	defer rows.Close()

	var results []*KnowledgeCitation
	for rows.Next() {
		var cit KnowledgeCitation
		var metaJSON []byte
		if err := rows.Scan(
			&cit.ChunkID, &cit.DocumentID, &cit.Filename, &cit.Collection,
			&cit.PageNumber, &cit.ChunkIndex, &cit.Content, &metaJSON, &cit.Score,
		); err != nil {
			return nil, fmt.Errorf("scan chunk citation: %w", err)
		}

		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &cit.Metadata)
		}

		cit.Score = math.Round(cit.Score*1000) / 1000
		if cit.PageNumber > 0 {
			cit.Citation = fmt.Sprintf("[%s, Page %d]", cit.Filename, cit.PageNumber)
		} else {
			cit.Citation = fmt.Sprintf("[%s, Section %d]", cit.Filename, cit.ChunkIndex+1)
		}

		results = append(results, &cit)
	}
	return results, nil
}
