package store

import (
	"context"
	"fmt"
	"time"
)

type Window struct {
	From time.Time
	To   time.Time
}

type OverviewTotals struct {
	Visitors      int
	Views         int
	ActiveToday   int
	SignIns       int
	Registrations int
	Pending       int
}

type DayPoint struct {
	Day      time.Time
	Visitors int
	SignIns  int
}

type Slice struct {
	Label string
	Total int
}

type PagePoint struct {
	Path     string
	Views    int
	Visitors int
}

type BatchAttendance struct {
	Batch   string
	Section string
	Average float64
	Classes int
}

type CourseClasses struct {
	Code        string
	Title       string
	Held        int
	Cancelled   int
	Rescheduled int
}

type TeacherPending struct {
	Name  string
	Total int
}

type AcademicTotals struct {
	BelowHalf    int
	PhonePending int
	Notices      int
	Polls        int
	Messages     int
	ClassesHeld  int
	Cancelled    int
	Rescheduled  int
	Undecided    int
}

type SystemTotals struct {
	DatabaseBytes int64
	FileBytes     int64
	Client4xx     int
	Server5xx     int
	EmailsSent    int
	EmailsFailed  int
	PushSent      int
	PushFailed    int
	FCMSent       int
	FCMFailed     int
	PushTokens    int
	FCMTokens     int
}

const (
	overviewTotalsSQL = `SELECT
(SELECT COUNT(DISTINCT visitor_id) FROM page_views WHERE created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM page_views WHERE created_at >= $1 AND created_at < $2),
(SELECT COUNT(DISTINCT actor_id) FROM audit_events WHERE actor_id IS NOT NULL AND created_at >= $3),
(SELECT COUNT(*) FROM audit_events WHERE action = 'sign_in' AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM users WHERE role = 'student' AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM users WHERE status = 'pending')`
	visitorSeriesSQL = `WITH days AS (
    SELECT generate_series($1::date, ($2::date - INTERVAL '1 day')::date, INTERVAL '1 day')::date AS day
)
SELECT d.day,
COALESCE((SELECT COUNT(DISTINCT v.visitor_id) FROM page_views v
          WHERE v.created_at >= d.day AND v.created_at < d.day + INTERVAL '1 day'), 0),
COALESCE((SELECT COUNT(*) FROM audit_events e
          WHERE e.action = 'sign_in' AND e.created_at >= d.day
            AND e.created_at < d.day + INTERVAL '1 day'), 0)
FROM days d
ORDER BY d.day ASC`
	signInsByRoleSQL = `SELECT COALESCE(NULLIF(actor_role, ''), 'unknown'), COUNT(*)
FROM audit_events
WHERE action = 'sign_in' AND created_at >= $1 AND created_at < $2
GROUP BY 1
ORDER BY COUNT(*) DESC`
	clientSplitSQL = `SELECT client, COUNT(*)
FROM page_views
WHERE created_at >= $1 AND created_at < $2
GROUP BY client
ORDER BY COUNT(*) DESC`
	appVersionsSQL = `SELECT COALESCE(NULLIF(app_version, ''), 'unknown'), COUNT(DISTINCT visitor_id)
FROM page_views
WHERE client = 'android' AND created_at >= $1 AND created_at < $2
GROUP BY 1
ORDER BY COUNT(DISTINCT visitor_id) DESC
LIMIT 10`
	devicesSQL = `SELECT COALESCE(NULLIF(device, ''), 'unknown'), COUNT(DISTINCT visitor_id)
FROM page_views
WHERE created_at >= $1 AND created_at < $2
GROUP BY 1
ORDER BY COUNT(DISTINCT visitor_id) DESC
LIMIT 10`
	topPagesSQL = `SELECT path, COUNT(*), COUNT(DISTINCT visitor_id)
FROM page_views
WHERE created_at >= $1 AND created_at < $2
GROUP BY path
ORDER BY COUNT(*) DESC
LIMIT 10`
	batchAttendanceSQL = `SELECT a.batch, a.section,
COALESCE(AVG(CASE WHEN r.status = 'present' THEN 1.0 ELSE 0.0 END) * 100, 0),
COUNT(DISTINCT a.id)
FROM attendance_sessions a
LEFT JOIN attendance_records r ON r.session_id = a.id
WHERE a.started_at >= $1 AND a.started_at < $2
GROUP BY a.batch, a.section
ORDER BY a.batch ASC, a.section ASC`
	courseClassesSQL = `SELECT c.code, c.title,
COUNT(*) FILTER (WHERE i.status IN ('confirmed', 'scheduled')),
COUNT(*) FILTER (WHERE i.status = 'cancelled'),
COUNT(*) FILTER (WHERE i.status = 'rescheduled')
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
WHERE i.kind <> 'exam' AND i.item_date >= $1 AND i.item_date < $2
GROUP BY c.code, c.title
HAVING COUNT(*) > 0
ORDER BY COUNT(*) DESC
LIMIT 20`
	undecidedTeachersSQL = `SELECT COALESCE(t.name, 'Unassigned'), COUNT(*)
FROM schedule_items i
LEFT JOIN teachers t ON t.id = i.teacher_id
WHERE i.item_date = $1 AND i.status IN ('awaiting_teacher', 'no_response')
GROUP BY 1
ORDER BY COUNT(*) DESC
LIMIT 20`
	academicTotalsSQL = `SELECT
(SELECT COUNT(*) FROM (
    SELECT r.student_id
    FROM attendance_records r
    JOIN attendance_sessions a ON a.id = r.session_id
    WHERE a.started_at >= $1 AND a.started_at < $2
    GROUP BY r.student_id
    HAVING AVG(CASE WHEN r.status = 'present' THEN 1.0 ELSE 0.0 END) < 0.5
) low),
(SELECT COUNT(*) FROM device_change_requests WHERE status = 'pending'),
(SELECT COUNT(*) FROM notices WHERE created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM polls WHERE created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM messages WHERE created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM schedule_items WHERE kind <> 'exam' AND item_date >= $1 AND item_date < $2
   AND status IN ('confirmed', 'scheduled')),
(SELECT COUNT(*) FROM schedule_items WHERE item_date >= $1 AND item_date < $2 AND status = 'cancelled'),
(SELECT COUNT(*) FROM schedule_items WHERE item_date >= $1 AND item_date < $2 AND status = 'rescheduled'),
(SELECT COUNT(*) FROM schedule_items WHERE item_date = $3
   AND status IN ('awaiting_teacher', 'no_response'))`
	systemTotalsSQL = `SELECT
pg_database_size(current_database()),
(SELECT COALESCE(SUM(size), 0) FROM stored_files),
(SELECT COUNT(*) FROM system_events WHERE kind = 'http_4xx' AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'http_5xx' AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'email' AND ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'email' AND NOT ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'push' AND ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'push' AND NOT ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'fcm' AND ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM system_events WHERE kind = 'fcm' AND NOT ok AND created_at >= $1 AND created_at < $2),
(SELECT COUNT(*) FROM push_subscriptions),
(SELECT COUNT(*) FROM fcm_tokens)`
	latestServerErrorsSQL = `SELECT detail, created_at
FROM system_events
WHERE kind = 'http_5xx'
ORDER BY created_at DESC
LIMIT 5`
	failedSignInSpikeSQL = `SELECT COALESCE(MAX(total), 0) FROM (
    SELECT COUNT(*) AS total
    FROM audit_events
    WHERE action = 'sign_in_failed' AND ip_hash <> '' AND created_at >= $1
    GROUP BY ip_hash
) spikes`
	sheetDeletionsSQL = `SELECT COUNT(*)
FROM audit_events
WHERE action = 'delete' AND target_type = 'attendance' AND created_at >= $1`
)

type ErrorPath struct {
	Path string
	At   time.Time
}

func (s *Store) OverviewTotals(ctx context.Context, window Window, dayStart time.Time) (OverviewTotals, error) {
	var totals OverviewTotals
	err := s.pool.QueryRow(ctx, overviewTotalsSQL, window.From, window.To, dayStart).Scan(
		&totals.Visitors,
		&totals.Views,
		&totals.ActiveToday,
		&totals.SignIns,
		&totals.Registrations,
		&totals.Pending,
	)
	if err != nil {
		return OverviewTotals{}, fmt.Errorf("store: could not read the overview totals: %w", err)
	}
	return totals, nil
}

func (s *Store) VisitorSeries(ctx context.Context, window Window) ([]DayPoint, error) {
	rows, err := s.pool.Query(ctx, visitorSeriesSQL, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the visitor series: %w", err)
	}
	defer rows.Close()
	points := make([]DayPoint, 0)
	for rows.Next() {
		var point DayPoint
		if err := rows.Scan(&point.Day, &point.Visitors, &point.SignIns); err != nil {
			return nil, fmt.Errorf("store: could not scan a visitor point: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the visitor series: %w", err)
	}
	return points, nil
}

func (s *Store) SignInsByRole(ctx context.Context, window Window) ([]Slice, error) {
	return s.slices(ctx, signInsByRoleSQL, window)
}

func (s *Store) ClientSplit(ctx context.Context, window Window) ([]Slice, error) {
	return s.slices(ctx, clientSplitSQL, window)
}

func (s *Store) AppVersions(ctx context.Context, window Window) ([]Slice, error) {
	return s.slices(ctx, appVersionsSQL, window)
}

func (s *Store) Devices(ctx context.Context, window Window) ([]Slice, error) {
	return s.slices(ctx, devicesSQL, window)
}

func (s *Store) slices(ctx context.Context, query string, window Window) ([]Slice, error) {
	rows, err := s.pool.Query(ctx, query, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read a dashboard breakdown: %w", err)
	}
	defer rows.Close()
	slices := make([]Slice, 0)
	for rows.Next() {
		var slice Slice
		if err := rows.Scan(&slice.Label, &slice.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan a dashboard breakdown: %w", err)
		}
		slices = append(slices, slice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read a dashboard breakdown: %w", err)
	}
	return slices, nil
}

func (s *Store) TopPages(ctx context.Context, window Window) ([]PagePoint, error) {
	rows, err := s.pool.Query(ctx, topPagesSQL, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the top pages: %w", err)
	}
	defer rows.Close()
	pages := make([]PagePoint, 0)
	for rows.Next() {
		var page PagePoint
		if err := rows.Scan(&page.Path, &page.Views, &page.Visitors); err != nil {
			return nil, fmt.Errorf("store: could not scan a page row: %w", err)
		}
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the top pages: %w", err)
	}
	return pages, nil
}

func (s *Store) BatchAttendance(ctx context.Context, window Window) ([]BatchAttendance, error) {
	rows, err := s.pool.Query(ctx, batchAttendanceSQL, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the attendance averages: %w", err)
	}
	defer rows.Close()
	averages := make([]BatchAttendance, 0)
	for rows.Next() {
		var average BatchAttendance
		if err := rows.Scan(&average.Batch, &average.Section, &average.Average, &average.Classes); err != nil {
			return nil, fmt.Errorf("store: could not scan an attendance average: %w", err)
		}
		averages = append(averages, average)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the attendance averages: %w", err)
	}
	return averages, nil
}

func (s *Store) CourseClasses(ctx context.Context, window Window) ([]CourseClasses, error) {
	rows, err := s.pool.Query(ctx, courseClassesSQL, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the class counts: %w", err)
	}
	defer rows.Close()
	courses := make([]CourseClasses, 0)
	for rows.Next() {
		var course CourseClasses
		if err := rows.Scan(&course.Code, &course.Title, &course.Held,
			&course.Cancelled, &course.Rescheduled); err != nil {
			return nil, fmt.Errorf("store: could not scan a class count: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class counts: %w", err)
	}
	return courses, nil
}

func (s *Store) UndecidedTeachers(ctx context.Context, day time.Time) ([]TeacherPending, error) {
	rows, err := s.pool.Query(ctx, undecidedTeachersSQL, day)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the undecided classes: %w", err)
	}
	defer rows.Close()
	teachers := make([]TeacherPending, 0)
	for rows.Next() {
		var teacher TeacherPending
		if err := rows.Scan(&teacher.Name, &teacher.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan an undecided class count: %w", err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the undecided classes: %w", err)
	}
	return teachers, nil
}

func (s *Store) AcademicTotals(ctx context.Context, window Window, tomorrow time.Time) (AcademicTotals, error) {
	var totals AcademicTotals
	err := s.pool.QueryRow(ctx, academicTotalsSQL, window.From, window.To, tomorrow).Scan(
		&totals.BelowHalf,
		&totals.PhonePending,
		&totals.Notices,
		&totals.Polls,
		&totals.Messages,
		&totals.ClassesHeld,
		&totals.Cancelled,
		&totals.Rescheduled,
		&totals.Undecided,
	)
	if err != nil {
		return AcademicTotals{}, fmt.Errorf("store: could not read the academic totals: %w", err)
	}
	return totals, nil
}

func (s *Store) SystemTotals(ctx context.Context, window Window) (SystemTotals, error) {
	var totals SystemTotals
	err := s.pool.QueryRow(ctx, systemTotalsSQL, window.From, window.To).Scan(
		&totals.DatabaseBytes,
		&totals.FileBytes,
		&totals.Client4xx,
		&totals.Server5xx,
		&totals.EmailsSent,
		&totals.EmailsFailed,
		&totals.PushSent,
		&totals.PushFailed,
		&totals.FCMSent,
		&totals.FCMFailed,
		&totals.PushTokens,
		&totals.FCMTokens,
	)
	if err != nil {
		return SystemTotals{}, fmt.Errorf("store: could not read the system totals: %w", err)
	}
	return totals, nil
}

func (s *Store) LatestServerErrors(ctx context.Context) ([]ErrorPath, error) {
	rows, err := s.pool.Query(ctx, latestServerErrorsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the latest server errors: %w", err)
	}
	defer rows.Close()
	paths := make([]ErrorPath, 0)
	for rows.Next() {
		var path ErrorPath
		if err := rows.Scan(&path.Path, &path.At); err != nil {
			return nil, fmt.Errorf("store: could not scan a server error: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the latest server errors: %w", err)
	}
	return paths, nil
}

func (s *Store) FailedSignInSpike(ctx context.Context, since time.Time) (int, error) {
	var highest int
	if err := s.pool.QueryRow(ctx, failedSignInSpikeSQL, since).Scan(&highest); err != nil {
		return 0, fmt.Errorf("store: could not read the failed sign-in spike: %w", err)
	}
	return highest, nil
}

func (s *Store) SheetDeletions(ctx context.Context, since time.Time) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, sheetDeletionsSQL, since).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: could not count the attendance sheet deletions: %w", err)
	}
	return total, nil
}
