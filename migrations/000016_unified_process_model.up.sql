-- Migration 016: Unified Process Model
-- Merges the Skill Registry into the Workflow Definitions.

-- 1. Add skill metadata columns to workflow_definitions
ALTER TABLE workflow_definitions 
    ADD COLUMN inputs JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN outputs JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN capabilities JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN reusable BOOLEAN NOT NULL DEFAULT FALSE;

-- 2. Clean up legacy skill tables
DROP TABLE IF EXISTS skill_dependencies;
DROP TABLE IF EXISTS skills;

-- 3. Update indexes
CREATE INDEX idx_workflows_reusable ON workflow_definitions (reusable) WHERE reusable = TRUE;
