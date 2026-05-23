-- Migration 007: Auto-increment integer versioning for workflow definitions.
-- Adds version_number column, backfills existing rows with 1, and creates
-- an index for efficient "latest version" lookups.

-- 1. Add version_number column (nullable initially for backfill).
ALTER TABLE workflow_definitions
  ADD COLUMN version_number INTEGER;

-- 2. Backfill: all existing rows get version_number = 1.
UPDATE workflow_definitions SET version_number = 1;

-- 3. Make it NOT NULL after backfill.
ALTER TABLE workflow_definitions
  ALTER COLUMN version_number SET NOT NULL;

-- 4. Drop the old unique constraint (tenant_id, name, version) and add a new
--    one that uses version_number instead.  The TEXT "version" column is kept
--    for backward compatibility but is no longer part of the uniqueness key.
ALTER TABLE workflow_definitions
  DROP CONSTRAINT IF EXISTS workflow_definitions_tenant_name_version_key;

ALTER TABLE workflow_definitions
  ADD CONSTRAINT workflow_definitions_tenant_name_vnum_key
  UNIQUE (tenant_id, name, version_number);

-- 5. Index for fast "get latest version" queries.
CREATE INDEX idx_workflow_def_latest
  ON workflow_definitions (tenant_id, name, version_number DESC);
