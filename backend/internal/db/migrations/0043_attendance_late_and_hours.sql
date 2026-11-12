ALTER TABLE attendance_records DROP CONSTRAINT IF EXISTS attendance_records_status_check;

ALTER TABLE attendance_records ADD CONSTRAINT attendance_records_status_check
CHECK (status IN ('present', 'late', 'absent'));

ALTER TABLE attendance_sessions ADD COLUMN IF NOT EXISTS hours INTEGER NOT NULL DEFAULT 1;

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_hours_check;

ALTER TABLE attendance_sessions ADD CONSTRAINT attendance_sessions_hours_check
CHECK (hours BETWEEN 1 AND 4);

ALTER TABLE attendance_sessions ADD COLUMN IF NOT EXISTS original_ends_at TIMESTAMPTZ;

UPDATE attendance_sessions SET original_ends_at = ends_at WHERE original_ends_at IS NULL;

ALTER TABLE attendance_sessions ALTER COLUMN original_ends_at SET DEFAULT now();

ALTER TABLE attendance_sessions ALTER COLUMN original_ends_at SET NOT NULL;

ALTER TABLE attendance_sessions ADD COLUMN IF NOT EXISTS late_after TIMESTAMPTZ;

INSERT INTO settings (key, value) VALUES ('attendance_late_weight', '0.5')
ON CONFLICT (key) DO NOTHING;

INSERT INTO attendance_records (session_id, student_id, status, method, note)
SELECT a.id, s.id, 'absent', 'system', 'Joined later'
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
LEFT JOIN batch_semesters b ON b.batch = a.batch
WHERE a.status = 'closed'
  AND a.started_at >= COALESCE(b.updated_at, '-infinity'::timestamptz)
  AND EXISTS (SELECT 1 FROM attendance_records m WHERE m.session_id = a.id AND m.method <> 'system')
ON CONFLICT (session_id, student_id) DO NOTHING;
