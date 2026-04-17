-- Skill Registry
-- Skills are first-class, versioned capabilities backed by workflow implementations.

CREATE TABLE IF NOT EXISTS skills (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    workflow_ref TEXT        NOT NULL DEFAULT '',
    inputs       JSONB       NOT NULL DEFAULT '[]',
    outputs      JSONB       NOT NULL DEFAULT '[]',
    capabilities JSONB       NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_skills_tenant ON skills(tenant_id);
