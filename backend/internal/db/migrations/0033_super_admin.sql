ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (
    role IN ('student', 'teacher', 'admin', 'super_admin')
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS office_access BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users ADD COLUMN IF NOT EXISTS last_sign_in_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_by BIGINT REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_sent_at TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_office_access_teachers_only;

ALTER TABLE users ADD CONSTRAINT users_office_access_teachers_only CHECK (
    NOT office_access OR role = 'teacher'
);

CREATE INDEX IF NOT EXISTS users_office_access_idx ON users (office_access) WHERE office_access;

CREATE INDEX IF NOT EXISTS users_role_last_sign_in_idx ON users (role, last_sign_in_at DESC);

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    actor_role TEXT NOT NULL DEFAULT '',
    actor_label TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    ip_hash TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    client TEXT NOT NULL DEFAULT 'web' CHECK (client IN ('web', 'android')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_events_created_idx ON audit_events (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS audit_events_actor_idx ON audit_events (actor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_events_action_idx ON audit_events (action, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_events_role_idx ON audit_events (actor_role, created_at DESC);
