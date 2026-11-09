-- Migration: 000032_guest_portal_tokens
-- Table for public guest portal access links allowing unauthenticated users
-- (e.g. candidates, patients, vendors) to resolve workflow HITL pauses.

CREATE TABLE IF NOT EXISTS guest_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    run_id UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    state_name TEXT NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'completed', 'expired', 'revoked'
    title TEXT,
    description TEXT,
    form_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    completed_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_guest_tokens_hash ON guest_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_guest_tokens_run_state ON guest_tokens(run_id, state_name);
