ALTER TABLE attendance_sessions ADD COLUMN IF NOT EXISTS teacher_device_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS attendance_sessions_teacher_device_idx
ON attendance_sessions (teacher_device_id) WHERE teacher_device_id <> '';

WITH wrong AS (
    SELECT r.id, a.status AS session_status
    FROM attendance_records r
    JOIN attendance_sessions a ON a.id = r.session_id
    WHERE r.status IN ('present', 'late')
      AND r.marked_at >= TIMESTAMPTZ '2026-11-12 00:00:00+06'
      AND (
          r.method = 'system'
          OR (r.method IN ('app', 'code', 'web') AND r.marked_at < a.started_at)
          OR (r.method IN ('app', 'code', 'web') AND r.lat IS NULL AND NOT COALESCE(r.no_location, FALSE))
      )
),
unmarked AS (
    DELETE FROM attendance_records r
    USING wrong w
    WHERE r.id = w.id AND w.session_status = 'open'
    RETURNING r.id
),
absent AS (
    UPDATE attendance_records r
    SET status = 'absent', method = 'system', device_id = '', lat = NULL, lng = NULL,
        accuracy_m = NULL, distance_m = NULL, no_location = FALSE,
        note = 'Automatic mark corrected'
    FROM wrong w
    WHERE r.id = w.id AND w.session_status <> 'open'
    RETURNING r.id
)
INSERT INTO audit_events (actor_role, actor_label, action, target_type, summary)
SELECT 'system', 'migration', 'attendance_cleanup', 'attendance',
       format('%s automatic present marks set back: %s unmarked in open sessions, %s absent in ended sessions',
              c.unmarked + c.absent, c.unmarked, c.absent)
FROM (SELECT (SELECT COUNT(*) FROM unmarked) AS unmarked, (SELECT COUNT(*) FROM absent) AS absent) c
WHERE c.unmarked + c.absent > 0;
