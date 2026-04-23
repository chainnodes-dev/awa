-- Rollback Migration 016: Unified Process Model

-- 1. Recreate skills table
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

-- 2. Recreate skill_dependencies table
CREATE TABLE IF NOT EXISTS skill_dependencies (
    skill_id       UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    depends_on_id  UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_id, depends_on_id)
);

CREATE INDEX IF NOT EXISTS idx_skill_deps_depends_on ON skill_dependencies(depends_on_id);

-- 3. Remove columns from workflow_definitions
DROP INDEX IF EXISTS idx_workflows_reusable;
ALTER TABLE workflow_definitions 
    DROP COLUMN IF EXISTS inputs,
    DROP COLUMN IF EXISTS outputs,
    DROP COLUMN IF EXISTS capabilities,
    DROP COLUMN IF EXISTS reusable;
