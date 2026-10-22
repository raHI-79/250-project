ALTER TABLE routines ADD COLUMN IF NOT EXISTS semester TEXT NOT NULL DEFAULT '1-1';

ALTER TABLE routines ALTER COLUMN semester DROP DEFAULT;

ALTER TABLE routines DROP CONSTRAINT IF EXISTS routines_semester_check;

ALTER TABLE routines ADD CONSTRAINT routines_semester_check
CHECK (semester IN ('1-1', '1-2', '2-1', '2-2', '3-1', '3-2', '4-1', '4-2'));

CREATE INDEX IF NOT EXISTS routines_batch_semester_idx ON routines (batch, semester, section);

CREATE TABLE IF NOT EXISTS batch_semesters (
    batch TEXT PRIMARY KEY,
    running_semester TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT batch_semesters_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT batch_semesters_running_check
    CHECK (running_semester IN ('1-1', '1-2', '2-1', '2-2', '3-1', '3-2', '4-1', '4-2'))
);

INSERT INTO batch_semesters (batch, running_semester)
SELECT DISTINCT r.batch, '1-1' FROM routines r
ON CONFLICT (batch) DO NOTHING;
