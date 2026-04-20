-- Add form_schema to hitl_requests
-- Allows workflows to define custom input forms for human resolution.

ALTER TABLE hitl_requests ADD COLUMN IF NOT EXISTS form_schema JSONB;
