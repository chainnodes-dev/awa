-- Migration 031: Embedded Vector RAG Service
-- Enables pgvector extension and creates knowledge_documents & knowledge_chunks tables

CREATE EXTENSION IF NOT EXISTS vector;

-- 1. Knowledge documents metadata & lifecycle
CREATE TABLE IF NOT EXISTS knowledge_documents (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    collection   VARCHAR(100) NOT NULL DEFAULT 'default',
    filename     TEXT         NOT NULL,
    content_type TEXT         NOT NULL DEFAULT 'application/octet-stream',
    byte_size    BIGINT       NOT NULL DEFAULT 0,
    page_count   INT          NOT NULL DEFAULT 1,
    status       VARCHAR(50)  NOT NULL DEFAULT 'indexing', -- 'indexing', 'ready', 'failed'
    chunk_count  INT          NOT NULL DEFAULT 0,
    error_msg    TEXT,
    metadata     JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by   TEXT         NOT NULL DEFAULT 'system'
);

CREATE INDEX IF NOT EXISTS idx_knowledge_docs_tenant_coll ON knowledge_documents(tenant_id, collection);

-- 2. Knowledge document chunks with 1536-dimensional embeddings
CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID         NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    collection   VARCHAR(100) NOT NULL DEFAULT 'default',
    chunk_index  INT          NOT NULL,
    page_number  INT          NOT NULL DEFAULT 1,
    content      TEXT         NOT NULL,
    embedding    vector(1536),
    metadata     JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_tenant_coll ON knowledge_chunks(tenant_id, collection);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_doc_id ON knowledge_chunks(document_id);

-- Cosine distance HNSW index for fast approximate nearest neighbor search
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_hnsw 
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);
