package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

type AttendanceSession struct {
	ID              int64
	CourseID        int64
	CourseCode      string
	CourseTitle     string
	Batch           string
	Section         string
	ScheduleItemID  int64
	TeacherID       int64
	TeacherUserID   int64
	TeacherName     string
	StartedAt       time.Time
	EndsAt          time.Time
	Status          string
	TeacherLat      float64
	TeacherLng      float64
	TeacherAccuracy float64
	LocationAt      time.Time
	HasLocation     bool
	TokenKey        string
	ClosedAt        time.Time
	Method          string
	CodeOnly        bool
	Present         int
	Roster          int
	Hours           int
	OriginalEndsAt  time.Time
	LateAfter       time.Time
	Late            int
}

type NewAttendanceSession struct {
	CourseID       int64
	Batch          string
	Section        string
	ScheduleItemID int64
	TeacherID      int64
	Minutes        int
	TokenKey       string
	Method         string
	CodeOnly       bool
	Hours          int
	LateAfter      int
}

type RosterStudent struct {
	StudentID int64
	UserID    int64
	FullName  string
	RegNo     string
	Batch     string
	Section   string
}

type AttendanceRecord struct {
	SessionID   int64
	StudentID   int64
	UserID      int64
	FullName    string
	RegNo       string
	Section     string
	PhotoPath   string
	Status      string
	Method      string
	DeviceID    string
	DistanceM   float64
	AccuracyM   float64
	HasPosition bool
	MarkedAt    time.Time
	HasMark     bool
	Note        string
	EditedBy    string
	EditedAt    time.Time
	NoLocation  bool
}

type MarkInput struct {
	SessionID  int64
	StudentID  int64
	Method     string
	DeviceID   string
	Lat        float64
	Lng        float64
	AccuracyM  float64
	DistanceM  float64
	NoLocation bool
}

type CourseAttendance struct {
	CourseID    int64
	CourseCode  string
	CourseTitle string
	TeacherName string
	Held        int
	Present     int
	Lates       int
	Credit      float64
	Sessions    int
}

type StudentSessionRow struct {
	SessionID   int64
	CourseID    int64
	CourseCode  string
	CourseTitle string
	TeacherName string
	StartedAt   time.Time
	Status      string
	Method      string
	Note        string
	MarkedAt    time.Time
	HasMark     bool
	Hours       int
}

type ClassSummary struct {
	CourseID    int64
	CourseCode  string
	CourseTitle string
	Batch       string
	Section     string
	Sessions    int
	AverageRate float64
	BelowFifty  int
}

type AttendanceAlert struct {
	StudentID   int64
	UserID      int64
	CourseID    int64
	CourseTitle string
	Held        int
	Present     int
	Credit      float64
	LastSentAt  time.Time
}

const lateWeightSQL = `COALESCE((SELECT NULLIF(v.value, '')::numeric FROM settings v WHERE v.key = 'attendance_late_weight'), 0.5)`

const sessionColumns = `a.id, a.course_id, c.code, c.title, a.batch, a.section,
COALESCE(a.schedule_item_id, 0), a.teacher_id, COALESCE(t.user_id, 0), COALESCE(t.name, ''),
a.started_at, a.ends_at, a.status, a.teacher_lat, a.teacher_lng, a.teacher_accuracy_m,
a.location_updated_at, a.token_key, a.closed_at, a.method, a.code_only,
(SELECT COUNT(*) FROM attendance_records r WHERE r.session_id = a.id AND r.status IN ('present', 'late')),
CASE WHEN a.status = 'closed'
     THEN (SELECT COUNT(*) FROM attendance_records r WHERE r.session_id = a.id)
     ELSE (SELECT COUNT(*) FROM students s
           JOIN users u ON u.id = s.user_id AND u.status = 'active'
           WHERE (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
              OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id))
END,
a.hours, a.original_ends_at, a.late_after,
(SELECT COUNT(*) FROM attendance_records r WHERE r.session_id = a.id AND r.status = 'late')`

const markedSessionSQL = `(a.status = 'open' OR EXISTS (
    SELECT 1 FROM attendance_records m WHERE m.session_id = a.id AND m.method <> 'system'))`

const (
	openSessionSQL = `INSERT INTO attendance_sessions
(course_id, batch, section, schedule_item_id, teacher_id, ends_at, original_ends_at, token_key,
 method, code_only, hours, late_after)
VALUES ($1, $2, $3, NULLIF($4, 0), $5, now() + make_interval(mins => $6),
 now() + make_interval(mins => $6), $7, $8, $9, $10,
 CASE WHEN $11::int > 0 THEN now() + make_interval(mins => $11::int) END)
RETURNING id`
	runningSessionSQL = `SELECT a.id, a.status FROM attendance_sessions a
WHERE a.teacher_id = $1 AND a.course_id = $2 AND a.batch = $3 AND a.section = $4
  AND ((a.started_at >= date_trunc('day', now()) AND a.started_at < date_trunc('day', now()) + interval '1 day')
       OR (a.status = 'open' AND a.ends_at > now()))
ORDER BY (a.status = 'open') DESC, a.started_at DESC
LIMIT 1`
	continueSessionSQL = `UPDATE attendance_sessions
SET ends_at = LEAST(GREATEST(ends_at, now() + make_interval(mins => $2)), started_at + interval '4 hours'),
    method = $3, code_only = $4, hours = $5
WHERE id = $1 AND status = 'open'
RETURNING id`
	reopenSessionSQL = `UPDATE attendance_sessions
SET status = 'open', closed_at = NULL,
    ends_at = GREATEST(LEAST(now() + make_interval(mins => $2), started_at + interval '4 hours'),
                       now() + interval '1 minute'),
    method = $3, code_only = $4, hours = $5
WHERE id = $1 AND status = 'closed'
RETURNING id`
	dropAbsenteesSQL = `DELETE FROM attendance_records WHERE session_id = $1 AND method = 'system'`
	realMarksSQL     = `SELECT COUNT(*) FROM attendance_records WHERE session_id = $1 AND method <> 'system'`
	sessionByIDSQL   = `SELECT ` + sessionColumns + `
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
WHERE a.id = $1`
	teacherSessionsSQL = `SELECT ` + sessionColumns + `
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
WHERE a.teacher_id = $1 AND a.started_at >= $2 AND a.started_at < $3
  AND ` + markedSessionSQL + `
ORDER BY a.started_at DESC`
	batchSessionsSQL = `SELECT ` + sessionColumns + `
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
WHERE ($1::text = '' OR a.batch = $1::text) AND a.started_at >= $2 AND a.started_at < $3
  AND ` + markedSessionSQL + `
ORDER BY a.started_at DESC
LIMIT 300`
	openForStudentSQL = `SELECT ` + sessionColumns + `
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
JOIN students s ON s.id = $1
WHERE a.status = 'open' AND a.ends_at > now() AND a.method = 'bluetooth'
  AND ((a.batch = s.batch AND (a.section = 'ALL' OR a.section = s.section))
       OR EXISTS (SELECT 1 FROM student_drop_courses d
                  WHERE d.student_id = s.id AND d.course_id = a.course_id))
ORDER BY a.started_at DESC
LIMIT 1`
	setLocationSQL = `UPDATE attendance_sessions
SET teacher_lat = $3, teacher_lng = $4, teacher_accuracy_m = $5, location_updated_at = now()
WHERE id = $1 AND teacher_id = $2 AND status = 'open'
RETURNING id`
	codeOnlySQL = `UPDATE attendance_sessions
SET code_only = true
WHERE id = $1 AND teacher_id = $2 AND status = 'open' AND method <> 'roll_call'
RETURNING id`
	extendSessionSQL = `UPDATE attendance_sessions
SET ends_at = ends_at + make_interval(mins => $3)
WHERE id = $1 AND teacher_id = $2 AND status = 'open'
  AND ends_at + make_interval(mins => $3) <= started_at + interval '4 hours'
RETURNING id`
	closeSessionSQL = `UPDATE attendance_sessions
SET status = 'closed', closed_at = now()
WHERE id = $1 AND teacher_id = $2 AND status = 'open'
RETURNING id`
	absentRosterSQL = `INSERT INTO attendance_records (session_id, student_id, status, method)
SELECT $1, s.id, 'absent', 'system'
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
WHERE a.id = $1
ON CONFLICT (session_id, student_id) DO NOTHING`
	rosterSQL = `SELECT s.id, s.user_id, s.full_name, s.reg_no, s.batch, s.section
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
WHERE a.id = $1
ORDER BY s.reg_no`
	onRosterSQL = `SELECT EXISTS (
    SELECT 1 FROM attendance_sessions a
    JOIN students s ON s.id = $2
    WHERE a.id = $1
      AND ((a.batch = s.batch AND (a.section = 'ALL' OR a.section = s.section))
           OR EXISTS (SELECT 1 FROM student_drop_courses d
                      WHERE d.student_id = s.id AND d.course_id = a.course_id))
)`
	markSQL = `INSERT INTO attendance_records
(session_id, student_id, status, method, device_id, lat, lng, accuracy_m, distance_m, no_location)
SELECT $1, $2,
       CASE WHEN now() > COALESCE(a.late_after, a.original_ends_at) THEN 'late' ELSE 'present' END,
       $3, $4, $5, $6, $7, $8, $9
FROM attendance_sessions a
WHERE a.id = $1 AND a.status = 'open' AND a.ends_at > now()
RETURNING status`
	sessionRecordsSQL = `SELECT a.id, s.id, s.user_id, s.full_name, s.reg_no, s.section,
COALESCE(u.photo_path, ''),
COALESCE(r.status, ''), COALESCE(r.method, ''), COALESCE(r.device_id, ''),
r.distance_m, r.accuracy_m, r.marked_at, COALESCE(r.note, ''),
COALESCE(e.name, es.full_name, ''), r.edited_at, COALESCE(r.no_location, false)
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
LEFT JOIN attendance_records r ON r.session_id = a.id AND r.student_id = s.id
LEFT JOIN teachers e ON e.user_id = r.edited_by
LEFT JOIN students es ON es.user_id = r.edited_by
WHERE a.id = $1
ORDER BY s.reg_no`
	editRecordSQL = `INSERT INTO attendance_records
(session_id, student_id, status, method, note, edited_by, edited_at)
VALUES ($1, $2, $3, 'manual', $4, $5, now())
ON CONFLICT (session_id, student_id) DO UPDATE
SET status = EXCLUDED.status, method = 'manual', note = EXCLUDED.note,
    edited_by = EXCLUDED.edited_by, edited_at = now(), no_location = false
RETURNING id`
	studentCoursesSQL = `SELECT a.course_id, c.code, c.title, COALESCE(t.name, ''),
COALESCE(SUM(a.hours), 0), COALESCE(SUM(a.hours) FILTER (WHERE r.status = 'present'), 0), COUNT(*) FILTER (WHERE r.status = 'late'), COALESCE(SUM(CASE r.status WHEN 'present' THEN a.hours WHEN 'late' THEN a.hours * ` + lateWeightSQL + ` ELSE 0 END), 0)::float8, COUNT(*)
FROM attendance_records r
JOIN attendance_sessions a ON a.id = r.session_id
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
WHERE r.student_id = $1
GROUP BY a.course_id, c.code, c.title, t.name
ORDER BY c.code`
	studentSessionsSQL = `SELECT a.id, a.course_id, c.code, c.title, COALESCE(t.name, ''),
a.started_at, r.status, r.method, r.note, r.marked_at, a.hours
FROM attendance_records r
JOIN attendance_sessions a ON a.id = r.session_id
JOIN courses c ON c.id = a.course_id
LEFT JOIN teachers t ON t.id = a.teacher_id
WHERE r.student_id = $1
ORDER BY a.started_at DESC`
	teacherClassesSQL = `SELECT a.course_id, c.code, c.title, a.batch, a.section,
COUNT(DISTINCT a.id)
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
WHERE a.teacher_id = $1 AND a.started_at >= $2 AND a.started_at < $3
GROUP BY a.course_id, c.code, c.title, a.batch, a.section
ORDER BY c.code, a.batch, a.section`
	classRatesSQL = `SELECT r.student_id, COALESCE(SUM(a.hours), 0), COALESCE(SUM(a.hours) FILTER (WHERE r.status = 'present'), 0), COUNT(*) FILTER (WHERE r.status = 'late'), COALESCE(SUM(CASE r.status WHEN 'present' THEN a.hours WHEN 'late' THEN a.hours * ` + lateWeightSQL + ` ELSE 0 END), 0)::float8
FROM attendance_records r
JOIN attendance_sessions a ON a.id = r.session_id
WHERE a.teacher_id = $1 AND a.course_id = $2 AND a.batch = $3 AND a.section = $4
  AND a.started_at >= $5 AND a.started_at < $6
GROUP BY r.student_id`
	batchClassRatesSQL = `SELECT r.student_id, COALESCE(SUM(a.hours), 0), COALESCE(SUM(a.hours) FILTER (WHERE r.status = 'present'), 0), COUNT(*) FILTER (WHERE r.status = 'late'), COALESCE(SUM(CASE r.status WHEN 'present' THEN a.hours WHEN 'late' THEN a.hours * ` + lateWeightSQL + ` ELSE 0 END), 0)::float8
FROM attendance_records r
JOIN attendance_sessions a ON a.id = r.session_id
WHERE a.course_id = $1 AND a.batch = $2 AND a.section = $3
  AND a.started_at >= $4 AND a.started_at < $5
GROUP BY r.student_id`
	exportRowsSQL = `SELECT s.id, s.full_name, s.reg_no, a.id, a.started_at,
r.status, COALESCE(r.note, ''), r.no_location, a.hours, (` + lateWeightSQL + `)::float8
FROM attendance_sessions a
JOIN attendance_records r ON r.session_id = a.id
JOIN students s ON s.id = r.student_id
WHERE a.teacher_id = $1 AND a.course_id = $2 AND a.batch = $3 AND a.section = $4
  AND a.started_at >= $5 AND a.started_at < $6
ORDER BY s.reg_no, a.started_at`
	batchExportClassesSQL = `SELECT a.course_id, c.code, c.title, a.batch, a.section, COUNT(DISTINCT a.id)
FROM attendance_sessions a
JOIN courses c ON c.id = a.course_id
WHERE a.batch = $1 AND a.started_at >= $2 AND a.started_at < $3
GROUP BY a.course_id, c.code, c.title, a.batch, a.section
ORDER BY c.code, a.section`
	batchExportRowsSQL = `SELECT s.id, s.full_name, s.reg_no, a.id, a.started_at,
r.status, COALESCE(r.note, ''), r.no_location, a.hours, (` + lateWeightSQL + `)::float8
FROM attendance_sessions a
JOIN attendance_records r ON r.session_id = a.id
JOIN students s ON s.id = r.student_id
WHERE a.course_id = $1 AND a.batch = $2 AND a.section = $3
  AND a.started_at >= $4 AND a.started_at < $5
ORDER BY s.reg_no, a.started_at`
	lowAttendanceSQL = `SELECT r.student_id, s.user_id, a.course_id, c.title,
COALESCE(SUM(a.hours), 0), COALESCE(SUM(a.hours) FILTER (WHERE r.status = 'present'), 0), COALESCE(SUM(CASE r.status WHEN 'present' THEN a.hours WHEN 'late' THEN a.hours * ` + lateWeightSQL + ` ELSE 0 END), 0)::float8, al.last_sent_at
FROM attendance_records r
JOIN attendance_sessions a ON a.id = r.session_id
JOIN courses c ON c.id = a.course_id
JOIN students s ON s.id = r.student_id
JOIN users u ON u.id = s.user_id AND u.status = 'active'
LEFT JOIN attendance_alerts al ON al.student_id = r.student_id AND al.course_id = a.course_id
WHERE a.status = 'closed'
GROUP BY r.student_id, s.user_id, a.course_id, c.title, al.last_sent_at
HAVING COUNT(*) >= $1
   AND COALESCE(SUM(CASE r.status WHEN 'present' THEN a.hours WHEN 'late' THEN a.hours * ` + lateWeightSQL + ` ELSE 0 END), 0)::float8 * 100 < COALESCE(SUM(a.hours), 0) * $2`
	joinedLaterSQL = `INSERT INTO attendance_records (session_id, student_id, status, method, note)
SELECT a.id, s.id, 'absent', 'system', 'Joined later'
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
LEFT JOIN batches b ON b.name = a.batch
WHERE a.status = 'closed'
  AND a.started_at >= COALESCE(b.semester_since, '-infinity'::timestamptz)
  AND EXISTS (SELECT 1 FROM attendance_records m WHERE m.session_id = a.id AND m.method <> 'system')
  AND ($1::bigint = 0 OR s.id = $1::bigint)
  AND ($2::bigint = 0 OR s.user_id = $2::bigint)
ON CONFLICT (session_id, student_id) DO NOTHING`
	claimAlertSQL = `INSERT INTO attendance_alerts (student_id, course_id)
VALUES ($1, $2)
ON CONFLICT (student_id, course_id) DO UPDATE SET last_sent_at = now()
WHERE attendance_alerts.last_sent_at <= $3
RETURNING student_id`
)

const (
	SessionBluetooth = "bluetooth"
	SessionRollCall  = "roll_call"
	SessionOpen      = "open"
	SessionClosed    = "closed"
)

var ErrAttendanceEmpty = errors.New("store: the attendance session had no marks and was removed")

const (
	noteOnlySQL = `UPDATE attendance_records
SET note = $3, edited_by = $4, edited_at = now()
WHERE session_id = $1 AND student_id = $2
RETURNING id`
	markAllPresentSQL = `INSERT INTO attendance_records (session_id, student_id, status, method, edited_by, edited_at)
SELECT $1, s.id, 'present', 'manual', $2, now()
FROM attendance_sessions a
JOIN students s ON (s.batch = a.batch AND (a.section = 'ALL' OR s.section = a.section))
  OR s.id IN (SELECT d.student_id FROM student_drop_courses d WHERE d.course_id = a.course_id)
JOIN users u ON u.id = s.user_id AND u.status = 'active'
WHERE a.id = $1
ON CONFLICT (session_id, student_id) DO NOTHING`
	deleteSessionSQL = `DELETE FROM attendance_sessions WHERE id = $1 RETURNING id`
	deleteSheetSQL   = `DELETE FROM attendance_sessions
WHERE course_id = $1 AND batch = $2 AND section = $3
RETURNING id`
)

func (s *Store) UpdateAttendanceNote(ctx context.Context, sessionID, studentID,
	editorUserID int64, note string) error {
	var id int64
	err := s.pool.QueryRow(ctx, noteOnlySQL, sessionID, studentID, note, editorUserID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not save the note of the student %d: %w", studentID, err)
	}
	return nil
}

func (s *Store) MarkAllPresent(ctx context.Context, sessionID, editorUserID int64) (int, error) {
	tag, err := s.pool.Exec(ctx, markAllPresentSQL, sessionID, editorUserID)
	if err != nil {
		return 0, fmt.Errorf("store: could not mark the roster of %d present: %w", sessionID, err)
	}
	return int(tag.RowsAffected()), nil
}

func (s *Store) DeleteAttendanceSession(ctx context.Context, sessionID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deleteSessionSQL, sessionID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not delete the attendance session %d: %w", sessionID, err)
	}
	return nil
}

func (s *Store) DeleteAttendanceSheet(ctx context.Context, courseID int64,
	batch, section string) (int, error) {
	rows, err := s.pool.Query(ctx, deleteSheetSQL, courseID, batch, section)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the attendance sheet of %d: %w", courseID, err)
	}
	defer rows.Close()
	deleted := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("store: could not count the deleted sessions: %w", err)
		}
		deleted++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("store: could not read the deleted sessions: %w", err)
	}
	if deleted == 0 {
		return 0, ErrNotFound
	}
	return deleted, nil
}

func (s *Store) OpenAttendanceSession(ctx context.Context,
	fresh NewAttendanceSession) (AttendanceSession, error) {
	method := fresh.Method
	if method == "" {
		method = SessionBluetooth
	}
	codeOnly := fresh.CodeOnly && method != SessionRollCall
	hours := fresh.Hours
	if hours < 1 {
		hours = 1
	}
	var running int64
	var status string
	err := s.pool.QueryRow(ctx, runningSessionSQL, fresh.TeacherID, fresh.CourseID,
		fresh.Batch, fresh.Section).Scan(&running, &status)
	if err == nil {
		return s.continueSession(ctx, running, status, fresh.Minutes, method, codeOnly, hours)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, fmt.Errorf("store: could not look for a running attendance: %w", err)
	}
	var id int64
	err = s.pool.QueryRow(ctx, openSessionSQL, fresh.CourseID, fresh.Batch, fresh.Section,
		fresh.ScheduleItemID, fresh.TeacherID, fresh.Minutes, fresh.TokenKey, method,
		codeOnly, hours, fresh.LateAfter).Scan(&id)
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not open an attendance session: %w", err)
	}
	return s.AttendanceSession(ctx, id)
}

func (s *Store) continueSession(ctx context.Context, sessionID int64, status string, minutes int,
	method string, codeOnly bool, hours int) (AttendanceSession, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not start continuing the session %d: %w", sessionID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query := continueSessionSQL
	if status != SessionOpen {
		query = reopenSessionSQL
	}
	var id int64
	if err := tx.QueryRow(ctx, query, sessionID, minutes, method, codeOnly, hours).Scan(&id); err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not continue the session %d: %w", sessionID, err)
	}
	if status != SessionOpen {
		if _, err := tx.Exec(ctx, dropAbsenteesSQL, sessionID); err != nil {
			return AttendanceSession{}, fmt.Errorf("store: could not reopen the marks of %d: %w", sessionID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not finish continuing the session %d: %w", sessionID, err)
	}
	return s.AttendanceSession(ctx, sessionID)
}

func (s *Store) AttendanceSession(ctx context.Context, sessionID int64) (AttendanceSession, error) {
	session, err := scanSession(s.pool.QueryRow(ctx, sessionByIDSQL, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, ErrNotFound
	}
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not read the attendance session %d: %w",
			sessionID, err)
	}
	return session, nil
}

func (s *Store) OpenSessionForStudent(ctx context.Context, studentID int64) (AttendanceSession, error) {
	session, err := scanSession(s.pool.QueryRow(ctx, openForStudentSQL, studentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, ErrNotFound
	}
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not find an open attendance for %d: %w",
			studentID, err)
	}
	return session, nil
}

func (s *Store) ListBatchSessions(ctx context.Context, batch string,
	from, to time.Time) ([]AttendanceSession, error) {
	return s.sessionList(ctx, batchSessionsSQL, batch, from, to)
}

func (s *Store) ListTeacherAttendance(ctx context.Context, teacherID int64,
	from, to time.Time) ([]AttendanceSession, error) {
	return s.sessionList(ctx, teacherSessionsSQL, teacherID, from, to)
}

func (s *Store) sessionList(ctx context.Context, query string,
	args ...any) ([]AttendanceSession, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the attendance sessions: %w", err)
	}
	defer rows.Close()
	sessions := make([]AttendanceSession, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an attendance session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the attendance sessions: %w", err)
	}
	return sessions, nil
}

func scanSession(row rowScanner) (AttendanceSession, error) {
	var session AttendanceSession
	var lat, lng, accuracy *float64
	var locationAt, closedAt, lateAfter *time.Time
	err := row.Scan(
		&session.ID,
		&session.CourseID,
		&session.CourseCode,
		&session.CourseTitle,
		&session.Batch,
		&session.Section,
		&session.ScheduleItemID,
		&session.TeacherID,
		&session.TeacherUserID,
		&session.TeacherName,
		&session.StartedAt,
		&session.EndsAt,
		&session.Status,
		&lat,
		&lng,
		&accuracy,
		&locationAt,
		&session.TokenKey,
		&closedAt,
		&session.Method,
		&session.CodeOnly,
		&session.Present,
		&session.Roster,
		&session.Hours,
		&session.OriginalEndsAt,
		&lateAfter,
		&session.Late,
	)
	if err != nil {
		return AttendanceSession{}, err
	}
	if lat != nil && lng != nil {
		session.TeacherLat, session.TeacherLng, session.HasLocation = *lat, *lng, true
	}
	if accuracy != nil {
		session.TeacherAccuracy = *accuracy
	}
	if locationAt != nil {
		session.LocationAt = *locationAt
	}
	if closedAt != nil {
		session.ClosedAt = *closedAt
	}
	if lateAfter != nil {
		session.LateAfter = *lateAfter
	}
	return session, nil
}

func (s *Store) SetAttendanceLocation(ctx context.Context, sessionID, teacherID int64,
	lat, lng, accuracy float64) error {
	var id int64
	err := s.pool.QueryRow(ctx, setLocationSQL, sessionID, teacherID, lat, lng, accuracy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAttendanceClosed
	}
	if err != nil {
		return fmt.Errorf("store: could not save the position of the session %d: %w", sessionID, err)
	}
	return nil
}

func (s *Store) ExtendAttendanceSession(ctx context.Context, sessionID, teacherID int64,
	minutes int) (AttendanceSession, error) {
	var id int64
	err := s.pool.QueryRow(ctx, extendSessionSQL, sessionID, teacherID, minutes).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, ErrAttendanceClosed
	}
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not extend the session %d: %w", sessionID, err)
	}
	return s.AttendanceSession(ctx, sessionID)
}

func (s *Store) SwitchAttendanceToCodeOnly(ctx context.Context, sessionID,
	teacherID int64) (AttendanceSession, error) {
	var id int64
	err := s.pool.QueryRow(ctx, codeOnlySQL, sessionID, teacherID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, ErrAttendanceClosed
	}
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not switch the session %d to code only: %w",
			sessionID, err)
	}
	return s.AttendanceSession(ctx, sessionID)
}

func (s *Store) CloseAttendanceSession(ctx context.Context, sessionID,
	teacherID int64) (AttendanceSession, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not start the attendance close: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, closeSessionSQL, sessionID, teacherID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, ErrAttendanceClosed
	}
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not close the session %d: %w", sessionID, err)
	}
	var marks int
	if err := tx.QueryRow(ctx, realMarksSQL, sessionID).Scan(&marks); err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not count the marks of %d: %w", sessionID, err)
	}
	if marks == 0 {
		if _, err := tx.Exec(ctx, deleteSessionSQL, sessionID); err != nil {
			return AttendanceSession{}, fmt.Errorf("store: could not drop the empty session %d: %w", sessionID, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return AttendanceSession{}, fmt.Errorf("store: could not finish dropping the session %d: %w", sessionID, err)
		}
		return AttendanceSession{ID: sessionID, Status: SessionClosed}, ErrAttendanceEmpty
	}
	if _, err := tx.Exec(ctx, absentRosterSQL, sessionID); err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not mark the absentees of %d: %w", sessionID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return AttendanceSession{}, fmt.Errorf("store: could not finish the attendance close: %w", err)
	}
	return s.AttendanceSession(ctx, sessionID)
}

func (s *Store) ListRoster(ctx context.Context, sessionID int64) ([]RosterStudent, error) {
	rows, err := s.pool.Query(ctx, rosterSQL, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the roster of %d: %w", sessionID, err)
	}
	defer rows.Close()
	roster := make([]RosterStudent, 0)
	for rows.Next() {
		var entry RosterStudent
		err := rows.Scan(&entry.StudentID, &entry.UserID, &entry.FullName,
			&entry.RegNo, &entry.Batch, &entry.Section)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a roster student: %w", err)
		}
		roster = append(roster, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the roster: %w", err)
	}
	return roster, nil
}

func (s *Store) CountRoster(ctx context.Context, sessionID int64) (int, error) {
	roster, err := s.ListRoster(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return len(roster), nil
}

func (s *Store) OnRoster(ctx context.Context, sessionID, studentID int64) (bool, error) {
	var listed bool
	if err := s.pool.QueryRow(ctx, onRosterSQL, sessionID, studentID).Scan(&listed); err != nil {
		return false, fmt.Errorf("store: could not check the roster of %d: %w", sessionID, err)
	}
	return listed, nil
}

func (s *Store) MarkAttendance(ctx context.Context, mark MarkInput) (string, error) {
	var status string
	var lat, lng, accuracy, distance *float64
	if !mark.NoLocation {
		lat, lng, accuracy, distance = &mark.Lat, &mark.Lng, &mark.AccuracyM, &mark.DistanceM
	}
	err := s.pool.QueryRow(ctx, markSQL, mark.SessionID, mark.StudentID, mark.Method,
		mark.DeviceID, lat, lng, accuracy, distance, mark.NoLocation).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrAttendanceClosed
	}
	if isUniqueViolation(err) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "attendance_records_one_per_device" {
			return "", ErrDeviceUsed
		}
		return "", ErrAlreadyMarked
	}
	if err != nil {
		return "", fmt.Errorf("store: could not mark the student %d: %w", mark.StudentID, err)
	}
	return status, nil
}

func (s *Store) ListSessionRecords(ctx context.Context, sessionID int64) ([]AttendanceRecord, error) {
	rows, err := s.pool.Query(ctx, sessionRecordsSQL, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the marks of %d: %w", sessionID, err)
	}
	defer rows.Close()
	records := make([]AttendanceRecord, 0)
	for rows.Next() {
		var record AttendanceRecord
		var distance, accuracy *float64
		var markedAt, editedAt *time.Time
		err := rows.Scan(
			&record.SessionID,
			&record.StudentID,
			&record.UserID,
			&record.FullName,
			&record.RegNo,
			&record.Section,
			&record.PhotoPath,
			&record.Status,
			&record.Method,
			&record.DeviceID,
			&distance,
			&accuracy,
			&markedAt,
			&record.Note,
			&record.EditedBy,
			&editedAt,
			&record.NoLocation,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an attendance mark: %w", err)
		}
		if distance != nil {
			record.DistanceM, record.HasPosition = *distance, true
		}
		if accuracy != nil {
			record.AccuracyM = *accuracy
		}
		if markedAt != nil {
			record.MarkedAt, record.HasMark = *markedAt, true
		}
		if editedAt != nil {
			record.EditedAt = *editedAt
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the attendance marks: %w", err)
	}
	return records, nil
}

func (s *Store) EditAttendanceRecord(ctx context.Context, sessionID, studentID, editorUserID int64,
	status, note string) error {
	var id int64
	err := s.pool.QueryRow(ctx, editRecordSQL, sessionID, studentID, status, note, editorUserID).Scan(&id)
	if err != nil {
		return fmt.Errorf("store: could not edit the mark of the student %d: %w", studentID, err)
	}
	return nil
}

func (s *Store) StudentCourseAttendance(ctx context.Context, studentID int64) ([]CourseAttendance, error) {
	rows, err := s.pool.Query(ctx, studentCoursesSQL, studentID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the attendance of the student %d: %w", studentID, err)
	}
	defer rows.Close()
	courses := make([]CourseAttendance, 0)
	for rows.Next() {
		var course CourseAttendance
		err := rows.Scan(&course.CourseID, &course.CourseCode, &course.CourseTitle,
			&course.TeacherName, &course.Held, &course.Present, &course.Lates, &course.Credit,
			&course.Sessions)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a course attendance: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the course attendance: %w", err)
	}
	return courses, nil
}

func (s *Store) StudentAttendanceSessions(ctx context.Context, studentID int64) ([]StudentSessionRow, error) {
	rows, err := s.pool.Query(ctx, studentSessionsSQL, studentID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the classes of the student %d: %w", studentID, err)
	}
	defer rows.Close()
	entries := make([]StudentSessionRow, 0)
	for rows.Next() {
		var entry StudentSessionRow
		var markedAt *time.Time
		err := rows.Scan(&entry.SessionID, &entry.CourseID, &entry.CourseCode, &entry.CourseTitle,
			&entry.TeacherName, &entry.StartedAt, &entry.Status, &entry.Method, &entry.Note, &markedAt,
			&entry.Hours)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a student attendance row: %w", err)
		}
		if markedAt != nil {
			entry.MarkedAt, entry.HasMark = *markedAt, true
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the student attendance rows: %w", err)
	}
	return entries, nil
}

func (s *Store) ListTeacherClasses(ctx context.Context, teacherID int64,
	from, to time.Time) ([]ClassSummary, error) {
	return s.classSummaries(ctx, teacherClassesSQL, teacherID, from, to)
}

func (s *Store) ListBatchClasses(ctx context.Context, batch string,
	from, to time.Time) ([]ClassSummary, error) {
	return s.classSummaries(ctx, batchExportClassesSQL, batch, from, to)
}

func (s *Store) classSummaries(ctx context.Context, query string, args ...any) ([]ClassSummary, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the attendance classes: %w", err)
	}
	defer rows.Close()
	classes := make([]ClassSummary, 0)
	for rows.Next() {
		var class ClassSummary
		err := rows.Scan(&class.CourseID, &class.CourseCode, &class.CourseTitle,
			&class.Batch, &class.Section, &class.Sessions)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an attendance class: %w", err)
		}
		classes = append(classes, class)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the attendance classes: %w", err)
	}
	return classes, nil
}

type StudentRate struct {
	StudentID int64
	Held      int
	Present   int
	Lates     int
	Credit    float64
}

func (s *Store) ClassRates(ctx context.Context, teacherID, courseID int64, batch, section string,
	from, to time.Time) ([]StudentRate, error) {
	rows, err := s.pool.Query(ctx, classRatesSQL, teacherID, courseID, batch, section, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: could not count the attendance of a class: %w", err)
	}
	defer rows.Close()
	rates := make([]StudentRate, 0)
	for rows.Next() {
		var rate StudentRate
		if err := rows.Scan(&rate.StudentID, &rate.Held, &rate.Present, &rate.Lates, &rate.Credit); err != nil {
			return nil, fmt.Errorf("store: could not scan a class rate: %w", err)
		}
		rates = append(rates, rate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class rates: %w", err)
	}
	return rates, nil
}

func (s *Store) BatchClassRates(ctx context.Context, courseID int64, batch, section string,
	from, to time.Time) ([]StudentRate, error) {
	rows, err := s.pool.Query(ctx, batchClassRatesSQL, courseID, batch, section, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: could not count the attendance of a batch class: %w", err)
	}
	defer rows.Close()
	rates := make([]StudentRate, 0)
	for rows.Next() {
		var rate StudentRate
		if err := rows.Scan(&rate.StudentID, &rate.Held, &rate.Present, &rate.Lates, &rate.Credit); err != nil {
			return nil, fmt.Errorf("store: could not scan a batch class rate: %w", err)
		}
		rates = append(rates, rate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the batch class rates: %w", err)
	}
	return rates, nil
}

type ExportRow struct {
	StudentID  int64
	FullName   string
	RegNo      string
	SessionID  int64
	StartedAt  time.Time
	Status     string
	Note       string
	NoLocation bool
	Hours      int
	LateWeight float64
}

func (s *Store) TeacherExportRows(ctx context.Context, teacherID, courseID int64, batch, section string,
	from, to time.Time) ([]ExportRow, error) {
	return s.exportRows(ctx, exportRowsSQL, teacherID, courseID, batch, section, from, to)
}

func (s *Store) BatchExportRows(ctx context.Context, courseID int64, batch, section string,
	from, to time.Time) ([]ExportRow, error) {
	return s.exportRows(ctx, batchExportRowsSQL, courseID, batch, section, from, to)
}

func (s *Store) exportRows(ctx context.Context, query string, args ...any) ([]ExportRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the attendance export rows: %w", err)
	}
	defer rows.Close()
	entries := make([]ExportRow, 0)
	for rows.Next() {
		var entry ExportRow
		err := rows.Scan(&entry.StudentID, &entry.FullName, &entry.RegNo,
			&entry.SessionID, &entry.StartedAt, &entry.Status, &entry.Note, &entry.NoLocation,
			&entry.Hours, &entry.LateWeight)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an attendance export row: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the attendance export: %w", err)
	}
	return entries, nil
}

func (s *Store) ListLowAttendance(ctx context.Context, minSessions int,
	threshold float64) ([]AttendanceAlert, error) {
	rows, err := s.pool.Query(ctx, lowAttendanceSQL, minSessions, threshold)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the low attendance rows: %w", err)
	}
	defer rows.Close()
	alerts := make([]AttendanceAlert, 0)
	for rows.Next() {
		var alert AttendanceAlert
		var lastSent *time.Time
		err := rows.Scan(&alert.StudentID, &alert.UserID, &alert.CourseID, &alert.CourseTitle,
			&alert.Held, &alert.Present, &alert.Credit, &lastSent)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a low attendance row: %w", err)
		}
		if lastSent != nil {
			alert.LastSentAt = *lastSent
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the low attendance rows: %w", err)
	}
	return alerts, nil
}

func (s *Store) ClaimAttendanceAlert(ctx context.Context, studentID, courseID int64,
	before time.Time) (bool, error) {
	var claimed int64
	err := s.pool.QueryRow(ctx, claimAlertSQL, studentID, courseID, before).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: could not claim the attendance alert of %d: %w", studentID, err)
	}
	return true, nil
}

func (s *Store) BackfillJoinedLater(ctx context.Context, studentID, userID int64) (int, error) {
	tag, err := s.pool.Exec(ctx, joinedLaterSQL, studentID, userID)
	if err != nil {
		return 0, fmt.Errorf("store: could not count the earlier classes of a student who joined later: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
