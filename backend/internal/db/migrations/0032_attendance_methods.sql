ALTER TABLE attendance_sessions
ADD COLUMN IF NOT EXISTS method TEXT NOT NULL DEFAULT 'bluetooth';

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_method_check;

ALTER TABLE attendance_sessions ADD CONSTRAINT attendance_sessions_method_check
CHECK (method IN ('bluetooth', 'roll_call'));

ALTER TABLE attendance_records DROP CONSTRAINT IF EXISTS attendance_records_method_check;

ALTER TABLE attendance_records ADD CONSTRAINT attendance_records_method_check
CHECK (method IN ('app', 'code', 'web', 'manual', 'system'));

CREATE INDEX IF NOT EXISTS attendance_sessions_sheet_idx
ON attendance_sessions (course_id, batch, section);
