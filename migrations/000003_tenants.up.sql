-- Migration 003: Multi-tenant data model
-- Adds tenants table and tenant_id columns to all tenant-scoped tables.
-- Existing rows are assigned to the built-in "default" tenant so they
-- remain accessible after the migration.

-- 1. Tenants table
CREATE TABLE tenants (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT        NOT NULL,
  slug       TEXT        NOT NULL UNIQUE,   -- URL-safe identifier, e.g. "acme-corp"
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Seed the default tenant (fixed UUID for deterministic references in code)
INSERT INTO tenants (id, name, slug) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Default', 'default');

-- 3. workflow_definitions: add tenant_id, update unique constraint
ALTER TABLE workflow_definitions
  ADD COLUMN tenant_id UUID REFERENCES tenants(id);
UPDATE workflow_definitions
  SET tenant_id = '00000000-0000-0000-0000-000000000001';
ALTER TABLE workflow_definitions
  ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE workflow_definitions
  DROP CONSTRAINT IF EXISTS workflow_definitions_name_version_key;
ALTER TABLE workflow_definitions
  ADD CONSTRAINT workflow_definitions_tenant_name_version_key
  UNIQUE (tenant_id, name, version);

-- 4. workflow_runs: add tenant_id with index for fast per-tenant listing
ALTER TABLE workflow_runs
  ADD COLUMN tenant_id UUID REFERENCES tenants(id);
UPDATE workflow_runs
  SET tenant_id = '00000000-0000-0000-0000-000000000001';
ALTER TABLE workflow_runs
  ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX idx_workflow_runs_tenant ON workflow_runs (tenant_id);

-- 5. users: add tenant_id, update username uniqueness to per-tenant
ALTER TABLE users
  ADD COLUMN tenant_id UUID REFERENCES tenants(id);
UPDATE users
  SET tenant_id = '00000000-0000-0000-0000-000000000001';
ALTER TABLE users
  ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE users
  ADD CONSTRAINT users_tenant_username_key UNIQUE (tenant_id, username);

-- 6. Add super_admin to the allowed role values
ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('super_admin', 'admin', 'operator', 'viewer'));
