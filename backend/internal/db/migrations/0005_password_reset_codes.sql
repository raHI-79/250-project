ALTER TABLE users ADD COLUMN IF NOT EXISTS contact_email TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS contact_phone TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_sent_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_stage TEXT;

UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL WHERE reset_token IS NOT NULL;

DROP INDEX IF EXISTS users_reset_token_key;

CREATE INDEX IF NOT EXISTS users_reset_token_idx ON users (reset_token) WHERE reset_token IS NOT NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_reset_token_has_expiry;

ALTER TABLE users ADD CONSTRAINT users_reset_token_has_expiry CHECK (
    (reset_token IS NULL AND reset_token_expires_at IS NULL AND reset_stage IS NULL)
    OR (reset_token IS NOT NULL AND reset_token_expires_at IS NOT NULL AND reset_stage IS NOT NULL)
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_reset_stage_valid;

ALTER TABLE users ADD CONSTRAINT users_reset_stage_valid CHECK (
    reset_stage IS NULL OR reset_stage IN ('code', 'verified')
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_reset_attempts_not_negative;

ALTER TABLE users ADD CONSTRAINT users_reset_attempts_not_negative CHECK (reset_attempts >= 0);
