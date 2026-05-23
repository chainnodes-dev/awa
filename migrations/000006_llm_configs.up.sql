CREATE TABLE IF NOT EXISTS llm_configs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider      TEXT        NOT NULL,
    api_key       TEXT        NOT NULL DEFAULT '',
    base_url      TEXT        NOT NULL DEFAULT '',
    default_model TEXT        NOT NULL DEFAULT '',
    enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    is_default    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, provider)
);
CREATE INDEX IF NOT EXISTS idx_llm_configs_tenant ON llm_configs(tenant_id);
