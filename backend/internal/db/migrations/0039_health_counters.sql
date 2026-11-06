ALTER TABLE system_events ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT '';

ALTER TABLE system_events ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

ALTER TABLE system_events ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS system_events_reason_idx
    ON system_events (kind, channel, reason, created_at DESC);

CREATE TABLE IF NOT EXISTS request_counters (
    bucket TIMESTAMPTZ PRIMARY KEY,
    requests INTEGER NOT NULL DEFAULT 0,
    failures INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT request_counters_not_negative CHECK (requests >= 0 AND failures >= 0)
);

CREATE INDEX IF NOT EXISTS request_counters_bucket_idx ON request_counters (bucket DESC);

CREATE TABLE IF NOT EXISTS alert_dismissals (
    alert_key TEXT PRIMARY KEY,
    dismissed_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    quiet_until TIMESTAMPTZ NOT NULL,
    CONSTRAINT alert_dismissals_key_not_blank CHECK (alert_key <> '')
);

CREATE INDEX IF NOT EXISTS alert_dismissals_quiet_idx ON alert_dismissals (quiet_until);
