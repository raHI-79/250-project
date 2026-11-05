CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    device TEXT NOT NULL DEFAULT '',
    client TEXT NOT NULL DEFAULT 'web' CHECK (client IN ('web', 'android')),
    ip_hash TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    sudo_until TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS user_sessions_user_idx ON user_sessions (user_id, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS user_sessions_expiry_idx ON user_sessions (expires_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_step_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_step_code TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_step_expires_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_step_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_step_sent_at TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_two_step_attempts_not_negative;

ALTER TABLE users ADD CONSTRAINT users_two_step_attempts_not_negative CHECK (two_step_attempts >= 0);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_two_step_has_expiry;

ALTER TABLE users ADD CONSTRAINT users_two_step_has_expiry CHECK (
    (two_step_code IS NULL AND two_step_expires_at IS NULL)
    OR (two_step_code IS NOT NULL AND two_step_expires_at IS NOT NULL)
);
