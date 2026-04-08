-- Reverse of 000001_init.up.sql
DROP INDEX IF EXISTS idx_hitl_unresolved;
DROP INDEX IF EXISTS idx_hitl_run_id;
DROP INDEX IF EXISTS idx_transitions_run_id;
DROP INDEX IF EXISTS idx_runs_workflow_name;
DROP INDEX IF EXISTS idx_runs_status;

DROP TABLE IF EXISTS hitl_requests;
DROP TABLE IF EXISTS state_transitions;
DROP TABLE IF EXISTS workflow_runs;
DROP TABLE IF EXISTS workflow_definitions;
