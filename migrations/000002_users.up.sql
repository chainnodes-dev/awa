-- Phaxa Platform - Users & Auth Schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,                -- bcrypt hash, cost factor 12
    role          TEXT        NOT NULL DEFAULT 'viewer', -- 'admin' | 'operator' | 'viewer'
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh tokens (one-to-many per user; invalidated on logout or rotation)
-- token_hash stores hex(sha256(rawToken)) — raw token is never persisted.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username          ON users(username);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id  ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash     ON refresh_tokens(token_hash);
