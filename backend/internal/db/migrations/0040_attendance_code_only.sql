ALTER TABLE attendance_sessions ADD COLUMN IF NOT EXISTS code_only BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE attendance_records ADD COLUMN IF NOT EXISTS no_location BOOLEAN NOT NULL DEFAULT false;

DROP INDEX IF EXISTS attendance_records_one_per_device;

CREATE UNIQUE INDEX IF NOT EXISTS attendance_records_one_per_device
ON attendance_records (session_id, device_id) WHERE method IN ('app', 'code') AND device_id <> '';
