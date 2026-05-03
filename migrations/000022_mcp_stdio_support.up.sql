-- Add support for stdio-based MCP servers (local commands)
ALTER TABLE mcp_servers ADD COLUMN transport TEXT NOT NULL DEFAULT 'sse';
ALTER TABLE mcp_servers ADD COLUMN command TEXT;
ALTER TABLE mcp_servers ADD COLUMN args TEXT[]; -- array of strings

-- Update comment for clarification
COMMENT ON COLUMN mcp_servers.transport IS 'Connection type: sse (default) or stdio (local command)';
