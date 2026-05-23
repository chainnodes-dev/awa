-- Create MCP Audit Logs table
CREATE TABLE IF NOT EXISTS mcp_audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      UUID        REFERENCES workflow_runs(id) ON DELETE SET NULL,
    state_name  TEXT,
    agent_name  TEXT,
    server_url  TEXT        NOT NULL,
    method      TEXT        NOT NULL, -- 'tools/list', 'tools/call'
    tool_name   TEXT,
    input       JSONB       NOT NULL DEFAULT '{}',
    output      JSONB       NOT NULL DEFAULT '{}',
    is_error    BOOLEAN     NOT NULL DEFAULT FALSE,
    error_msg   TEXT,
    duration_ms INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster run history lookup
CREATE INDEX IF NOT EXISTS idx_mcp_logs_run_id ON mcp_audit_logs(run_id);
