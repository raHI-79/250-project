CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT settings_key_not_blank CHECK (key <> '')
);

CREATE TABLE IF NOT EXISTS ip_blocks (
    id BIGSERIAL PRIMARY KEY,
    cidr TEXT NOT NULL UNIQUE,
    reason TEXT NOT NULL DEFAULT '',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    CONSTRAINT ip_blocks_cidr_not_blank CHECK (cidr <> '')
);

CREATE INDEX IF NOT EXISTS ip_blocks_expiry_idx ON ip_blocks (expires_at);

CREATE TABLE IF NOT EXISTS approval_requests (
    id BIGSERIAL PRIMARY KEY,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '',
    requested_by BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    decided_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'declined', 'expired')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    decided_at TIMESTAMPTZ,
    CONSTRAINT approval_requests_action_not_blank CHECK (action <> ''),
    CONSTRAINT approval_requests_decided_together CHECK ((status = 'pending') = (decided_at IS NULL))
);

CREATE INDEX IF NOT EXISTS approval_requests_status_idx ON approval_requests (status, created_at DESC);

ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_sign_ins INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_failed_sign_ins_not_negative;

ALTER TABLE users ADD CONSTRAINT users_failed_sign_ins_not_negative CHECK (failed_sign_ins >= 0);
