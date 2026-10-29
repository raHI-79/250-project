CREATE TABLE IF NOT EXISTS fcm_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL DEFAULT 'android',
    device_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fcm_tokens_token_not_blank CHECK (token <> ''),
    CONSTRAINT fcm_tokens_platform_not_blank CHECK (platform <> '')
);

CREATE INDEX IF NOT EXISTS fcm_tokens_user_idx ON fcm_tokens (user_id);
