-- Reverse of 000003_tenants.up.sql

-- 6. Restore original role check (without super_admin)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('admin', 'operator', 'viewer'));

-- 5. Revert users: restore global username uniqueness, drop tenant_id
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tenant_username_key;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;

-- 4. Revert workflow_runs: drop tenant_id
DROP INDEX IF EXISTS idx_workflow_runs_tenant;
ALTER TABLE workflow_runs DROP COLUMN IF EXISTS tenant_id;

-- 3. Revert workflow_definitions: restore global unique constraint, drop tenant_id
ALTER TABLE workflow_definitions DROP CONSTRAINT IF EXISTS workflow_definitions_tenant_name_version_key;
ALTER TABLE workflow_definitions ADD CONSTRAINT workflow_definitions_name_version_key UNIQUE (name, version);
ALTER TABLE workflow_definitions DROP COLUMN IF EXISTS tenant_id;

-- 2 & 1. Drop tenants (cascade removes FK refs, but columns already dropped above)
DROP TABLE IF EXISTS tenants;
