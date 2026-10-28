CREATE TABLE IF NOT EXISTS section_views (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    section_key TEXT NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, section_key),
    CONSTRAINT section_views_key_not_blank CHECK (section_key <> '')
);

CREATE TABLE IF NOT EXISTS reminder_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slot TEXT NOT NULL,
    reminder_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reminder_log_slot_not_blank CHECK (slot <> ''),
    CONSTRAINT reminder_log_once UNIQUE (user_id, slot, reminder_date)
);

CREATE INDEX IF NOT EXISTS reminder_log_date_idx ON reminder_log (reminder_date);

DELETE FROM notifications WHERE type = 'message';
