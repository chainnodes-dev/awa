-- Reverse of 000002_users.up.sql
DROP INDEX IF EXISTS idx_refresh_tokens_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_users_username;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
