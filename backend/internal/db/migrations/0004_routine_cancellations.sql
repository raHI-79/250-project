CREATE TABLE IF NOT EXISTS routine_cancellations (
    id BIGSERIAL PRIMARY KEY,
    routine_id BIGINT NOT NULL REFERENCES routines (id) ON DELETE CASCADE,
    cancel_date DATE NOT NULL,
    cancelled_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT routine_cancellations_one_per_date UNIQUE (routine_id, cancel_date)
);

CREATE INDEX IF NOT EXISTS routine_cancellations_date_idx ON routine_cancellations (cancel_date);

CREATE INDEX IF NOT EXISTS routine_cancellations_cancelled_by_idx ON routine_cancellations (cancelled_by);
