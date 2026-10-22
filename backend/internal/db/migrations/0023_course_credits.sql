ALTER TABLE courses ADD COLUMN IF NOT EXISTS credit NUMERIC(3,2) NOT NULL DEFAULT 3.0;

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_credit_range;

ALTER TABLE courses ADD CONSTRAINT courses_credit_range CHECK (
    credit >= 0.5 AND credit <= 6.0 AND credit * 4 = floor(credit * 4)
);
