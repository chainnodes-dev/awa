-- Rollback migration 007: Remove version_number column and restore original constraint.

DROP INDEX IF EXISTS idx_workflow_def_latest;

ALTER TABLE workflow_definitions
  DROP CONSTRAINT IF EXISTS workflow_definitions_tenant_name_vnum_key;

ALTER TABLE workflow_definitions
  ADD CONSTRAINT workflow_definitions_tenant_name_version_key
  UNIQUE (tenant_id, name, version);

ALTER TABLE workflow_definitions
  DROP COLUMN IF EXISTS version_number;
