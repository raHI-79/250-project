CREATE TABLE IF NOT EXISTS holidays (
    holiday_date DATE PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    working_day BOOLEAN NOT NULL DEFAULT FALSE,
    follows_weekday SMALLINT,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT holidays_follows_weekday_range CHECK (
        follows_weekday IS NULL OR follows_weekday BETWEEN 0 AND 6
    ),
    CONSTRAINT holidays_working_day_follows_a_weekday CHECK (
        NOT working_day OR follows_weekday IS NOT NULL
    ),
    CONSTRAINT holidays_name_not_blank CHECK (working_day OR name <> '')
);

CREATE INDEX IF NOT EXISTS holidays_working_day_idx ON holidays (working_day, holiday_date);

INSERT INTO settings (key, value)
VALUES ('weekly_off_days', '5,6')
ON CONFLICT (key) DO NOTHING;
