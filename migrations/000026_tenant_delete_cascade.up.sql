-- Migration 026: Add ON DELETE CASCADE to tenant_id foreign keys
-- This ensures that deleting a tenant also removes its workflows, runs, and users.

-- 1. workflow_definitions
ALTER TABLE workflow_definitions
  DROP CONSTRAINT IF EXISTS workflow_definitions_tenant_id_fkey,
  ADD CONSTRAINT workflow_definitions_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 2. workflow_runs
ALTER TABLE workflow_runs
  DROP CONSTRAINT IF EXISTS workflow_runs_tenant_id_fkey,
  ADD CONSTRAINT workflow_runs_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 3. users
ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_tenant_id_fkey,
  ADD CONSTRAINT users_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
