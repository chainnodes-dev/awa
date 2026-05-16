-- Add tools column to mcp_servers table to persist discovered tool definitions
ALTER TABLE mcp_servers ADD COLUMN IF NOT EXISTS tools JSONB DEFAULT '[]'::jsonb;
