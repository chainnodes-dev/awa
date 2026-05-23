-- Add env_vars JSONB column and migrate data from env_var/env_value
ALTER TABLE mcp_servers ADD COLUMN env_vars JSONB DEFAULT '{}';

-- Migrate existing data
UPDATE mcp_servers 
SET env_vars = jsonb_build_object(env_var, env_value)
WHERE env_var IS NOT NULL AND env_var != '';

-- Drop old columns
ALTER TABLE mcp_servers DROP COLUMN env_var;
ALTER TABLE mcp_servers DROP COLUMN env_value;
