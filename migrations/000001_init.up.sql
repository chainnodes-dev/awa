-- Phaxa Platform - Initial Schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Workflow definitions (versioned, immutable once deployed)
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    version     TEXT        NOT NULL,
    definition  JSONB       NOT NULL,
    yaml_source TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(name, version)
);

-- Workflow runs (instances of a workflow definition)
CREATE TABLE IF NOT EXISTS workflow_runs (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_name       TEXT        NOT NULL,
    workflow_version    TEXT        NOT NULL,
    status              TEXT        NOT NULL DEFAULT 'pending',
    current_state       TEXT        NOT NULL,
    blackboard          JSONB       NOT NULL DEFAULT '{}',
    temporal_workflow_id TEXT,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);

-- Full state transition history per run
CREATE TABLE IF NOT EXISTS state_transitions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id              UUID        NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    from_state          TEXT        NOT NULL,
    to_state            TEXT        NOT NULL,
    trigger             TEXT        NOT NULL,
    blackboard_snapshot JSONB       NOT NULL DEFAULT '{}',
    agent_output        JSONB,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Human-in-the-Loop requests
CREATE TABLE IF NOT EXISTS hitl_requests (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      UUID        NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    state_name  TEXT        NOT NULL,
    assignee    TEXT,
    timeout_at  TIMESTAMPTZ,
    resolved    BOOLEAN     NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolution  TEXT,       -- 'approved' | 'rejected'
    resolver    TEXT,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_runs_status        ON workflow_runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_name ON workflow_runs(workflow_name);
CREATE INDEX IF NOT EXISTS idx_transitions_run_id ON state_transitions(run_id);
CREATE INDEX IF NOT EXISTS idx_hitl_run_id        ON hitl_requests(run_id);
CREATE INDEX IF NOT EXISTS idx_hitl_unresolved    ON hitl_requests(resolved) WHERE resolved = FALSE;
