CREATE TABLE IF NOT EXISTS notification_reads (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    notification_id BIGINT NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, notification_id)
);

CREATE INDEX IF NOT EXISTS notification_reads_notification_idx
    ON notification_reads (notification_id);

CREATE TABLE IF NOT EXISTS notification_read_marks (
    user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    read_before TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
