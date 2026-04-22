-- Add process_description to workflow_definitions
ALTER TABLE workflow_definitions ADD COLUMN IF NOT EXISTS process_description TEXT;
