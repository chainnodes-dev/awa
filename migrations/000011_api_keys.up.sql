-- API Keys
-- Operator-created long-lived tokens for external callers (webhooks, partner apps,
-- cross-tenant invocations). Keys are hashed (SHA-256) — the raw key is shown
-- only once at creation time.

CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    key_hash     TEXT        NOT NULL UNIQUE,  -- SHA-256 hex of the raw key
    created_by   UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
