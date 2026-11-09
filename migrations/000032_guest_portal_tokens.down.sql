-- Rollback Migration: 000032_guest_portal_tokens

DROP INDEX IF EXISTS idx_guest_tokens_run_state;
DROP INDEX IF EXISTS idx_guest_tokens_hash;
DROP TABLE IF EXISTS guest_tokens;

