package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

var (
	ErrTooLateToApprove = errors.New("store: the request date has already arrived")
	ErrNotCancellable   = errors.New("store: the class cannot be cancelled")
	ErrAlreadyCancelled = errors.New("store: this occurrence is already cancelled")
)

type Teacher struct {
	ID     int64
	UserID int64
	Name   string
}

type RequestRow struct {
	ID               int64
	EventType        string
	Batch            string
	Section          string
	CourseCode       string
	CourseTitle      string
	EventDate        string
	StartTime        string
	EndTime          string
	Room             string
	RequestedBy      string
	RequestedByRegNo string
	CreatedBy        int64
	CreatedAt        time.Time
}

type EventDecision struct {
	ID          int64
	EventType   string
	Batch       string
	Section     string
	CourseCode  string
	CourseTitle string
	EventDate   string
	StartTime   string
	EndTime     string
	Room        string
	Status      string
	CreatedBy   int64
	DecidedAt   time.Time
}

type ScheduleItem struct {
	Kind        string
	ID          int64
	Batch       string
	Section     string
	CourseCode  string
	CourseTitle string
	StartTime   string
	EndTime     string
	Room        string
	Cancelled   bool
}

const (
	findTeacherByUserSQL   = `SELECT id, user_id, name FROM teachers WHERE user_id = $1`
	listTeacherRequestsSQL = `SELECT e.id, e.event_type, e.batch, e.section, c.code, c.title,
to_char(e.event_date, 'YYYY-MM-DD'), to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
e.room, COALESCE(s.full_name, ''), COALESCE(s.reg_no, ''), e.created_by, e.created_at
FROM class_events e
JOIN courses c ON c.id = e.course_id
LEFT JOIN students s ON s.user_id = e.created_by
WHERE e.teacher_id = $1 AND e.status = 'pending'
ORDER BY e.created_at, e.id`
	findTeacherEventSQL = `SELECT e.status, e.event_type, e.batch, e.section, c.code, c.title,
to_char(e.event_date, 'YYYY-MM-DD'), to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
e.room, e.created_by, (e.event_date <= $3::date)
FROM class_events e
JOIN courses c ON c.id = e.course_id
WHERE e.id = $1 AND e.teacher_id = $2
FOR UPDATE OF e`
	decideEventSQL = `UPDATE class_events
SET status = $2, decided_at = $3
WHERE id = $1
RETURNING id`
	listTeacherScheduleSQL = `SELECT 'fixed' AS kind, r.id, r.batch, r.section, c.code, c.title,
to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room,
EXISTS (SELECT 1 FROM routine_cancellations rc WHERE rc.routine_id = r.id AND rc.cancel_date = $2::date) AS cancelled
FROM routines r
JOIN courses c ON c.id = r.course_id
WHERE r.teacher_id = $1 AND r.day_of_week = EXTRACT(DOW FROM $2::date)::int
UNION ALL
SELECT e.event_type, e.id, e.batch, e.section, c.code, c.title,
to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'), e.room,
(e.status = 'cancelled') AS cancelled
FROM class_events e
JOIN courses c ON c.id = e.course_id
WHERE e.teacher_id = $1 AND e.event_date = $2::date AND e.status IN ('approved', 'cancelled')
ORDER BY 7, 2`
	listTeacherClassesSQL = `SELECT e.id, e.event_type, e.batch, e.section, c.code, c.title,
to_char(e.event_date, 'YYYY-MM-DD'), to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
e.room, e.status, e.created_by
FROM class_events e
JOIN courses c ON c.id = e.course_id
WHERE e.teacher_id = $1 AND e.status = 'approved'
  AND (($2::text = '' AND e.event_date >= $3::date) OR ($2::text <> '' AND e.event_date = $2::date))
ORDER BY e.event_date, e.start_time, e.id`
	findTeacherRoutineSQL = `SELECT r.id, r.batch, r.section, r.course_id, c.code, c.title, r.teacher_id, t.name,
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN teachers t ON t.id = r.teacher_id
WHERE r.id = $1 AND r.teacher_id = $2`
	insertRoutineCancellationSQL = `INSERT INTO routine_cancellations (routine_id, cancel_date, cancelled_by)
VALUES ($1, $2::date, $3)
RETURNING id`
	listActiveCRsSQL = `SELECT u.id
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND s.is_cr AND s.batch = $1
  AND ($2::text = '' OR $2::text = 'ALL' OR s.section = $2::text)
ORDER BY u.id`
	autoCancelSQL = `WITH stale AS (
    UPDATE class_events
    SET status = 'auto_cancelled', decided_at = $1
    WHERE status = 'pending' AND event_date <= $2::date
    RETURNING id, created_by, event_type, course_id, batch, section, event_date, start_time, end_time, room
)
SELECT stale.id, stale.created_by, stale.event_type, c.code, c.title, stale.batch, stale.section,
to_char(stale.event_date, 'YYYY-MM-DD'), to_char(stale.start_time, 'HH24:MI'), to_char(stale.end_time, 'HH24:MI'),
stale.room
FROM stale
JOIN courses c ON c.id = stale.course_id
ORDER BY stale.id`
)

func (s *Store) GetTeacherByUser(ctx context.Context, userID int64) (Teacher, error) {
	var teacher Teacher
	err := s.pool.QueryRow(ctx, findTeacherByUserSQL, userID).Scan(&teacher.ID, &teacher.UserID, &teacher.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	if err != nil {
		return Teacher{}, fmt.Errorf("store: could not load the teacher of the user %d: %w", userID, err)
	}
	return teacher, nil
}

func (s *Store) ListTeacherRequests(ctx context.Context, teacherID int64) ([]RequestRow, error) {
	rows, err := s.pool.Query(ctx, listTeacherRequestsSQL, teacherID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the requests of the teacher %d: %w", teacherID, err)
	}
	defer rows.Close()
	requests := make([]RequestRow, 0)
	for rows.Next() {
		var request RequestRow
		err := rows.Scan(
			&request.ID,
			&request.EventType,
			&request.Batch,
			&request.Section,
			&request.CourseCode,
			&request.CourseTitle,
			&request.EventDate,
			&request.StartTime,
			&request.EndTime,
			&request.Room,
			&request.RequestedBy,
			&request.RequestedByRegNo,
			&request.CreatedBy,
			&request.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a request: %w", err)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the requests: %w", err)
	}
	return requests, nil
}

func (s *Store) DecideRequest(ctx context.Context, eventID, teacherID int64, status string, now time.Time) (EventDecision, error) {
	today := now.Format(dateLayout)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EventDecision{}, fmt.Errorf("store: could not start the decision transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	decision := EventDecision{ID: eventID, Status: status, DecidedAt: now}
	var currentStatus string
	var tooLate bool
	err = tx.QueryRow(ctx, findTeacherEventSQL, eventID, teacherID, today).Scan(
		&currentStatus,
		&decision.EventType,
		&decision.Batch,
		&decision.Section,
		&decision.CourseCode,
		&decision.CourseTitle,
		&decision.EventDate,
		&decision.StartTime,
		&decision.EndTime,
		&decision.Room,
		&decision.CreatedBy,
		&tooLate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventDecision{}, ErrNotFound
	}
	if err != nil {
		return EventDecision{}, fmt.Errorf("store: could not load the request %d: %w", eventID, err)
	}
	if currentStatus != EventStatusPending {
		return EventDecision{}, ErrNotPending
	}
	if status == EventStatusApproved && tooLate {
		return EventDecision{}, ErrTooLateToApprove
	}
	var id int64
	if err := tx.QueryRow(ctx, decideEventSQL, eventID, status, now).Scan(&id); err != nil {
		return EventDecision{}, fmt.Errorf("store: could not decide the request %d: %w", eventID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return EventDecision{}, fmt.Errorf("store: could not commit the decision: %w", err)
	}
	return decision, nil
}

func (s *Store) CancelClassEvent(ctx context.Context, eventID, teacherID int64, now time.Time) (EventDecision, error) {
	today := now.Format(dateLayout)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EventDecision{}, fmt.Errorf("store: could not start the cancellation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	decision := EventDecision{ID: eventID, Status: EventStatusCancelled, DecidedAt: now}
	var currentStatus string
	var past bool
	err = tx.QueryRow(ctx, findTeacherEventSQL, eventID, teacherID, today).Scan(
		&currentStatus,
		&decision.EventType,
		&decision.Batch,
		&decision.Section,
		&decision.CourseCode,
		&decision.CourseTitle,
		&decision.EventDate,
		&decision.StartTime,
		&decision.EndTime,
		&decision.Room,
		&decision.CreatedBy,
		&past,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventDecision{}, ErrNotFound
	}
	if err != nil {
		return EventDecision{}, fmt.Errorf("store: could not load the class %d: %w", eventID, err)
	}
	if currentStatus != EventStatusApproved {
		return EventDecision{}, ErrNotCancellable
	}
	if past && decision.EventDate != today {
		return EventDecision{}, ErrNotCancellable
	}
	var id int64
	if err := tx.QueryRow(ctx, decideEventSQL, eventID, EventStatusCancelled, now).Scan(&id); err != nil {
		return EventDecision{}, fmt.Errorf("store: could not cancel the class %d: %w", eventID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return EventDecision{}, fmt.Errorf("store: could not commit the cancellation: %w", err)
	}
	return decision, nil
}

func (s *Store) ListTeacherSchedule(ctx context.Context, teacherID int64, date string) ([]ScheduleItem, error) {
	rows, err := s.pool.Query(ctx, listTeacherScheduleSQL, teacherID, date)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the schedule of the teacher %d: %w", teacherID, err)
	}
	defer rows.Close()
	items := make([]ScheduleItem, 0)
	for rows.Next() {
		var item ScheduleItem
		err := rows.Scan(
			&item.Kind,
			&item.ID,
			&item.Batch,
			&item.Section,
			&item.CourseCode,
			&item.CourseTitle,
			&item.StartTime,
			&item.EndTime,
			&item.Room,
			&item.Cancelled,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a schedule item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the schedule: %w", err)
	}
	return items, nil
}

func (s *Store) ListTeacherClasses(ctx context.Context, teacherID int64, date string, now time.Time) ([]ClassEventRow, error) {
	rows, err := s.pool.Query(ctx, listTeacherClassesSQL, teacherID, date, now.Format(dateLayout))
	if err != nil {
		return nil, fmt.Errorf("store: could not list the classes of the teacher %d: %w", teacherID, err)
	}
	defer rows.Close()
	classes := make([]ClassEventRow, 0)
	for rows.Next() {
		var class ClassEventRow
		err := rows.Scan(
			&class.ID,
			&class.EventType,
			&class.Batch,
			&class.Section,
			&class.CourseCode,
			&class.CourseTitle,
			&class.EventDate,
			&class.StartTime,
			&class.EndTime,
			&class.Room,
			&class.Status,
			&class.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a class: %w", err)
		}
		classes = append(classes, class)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the classes: %w", err)
	}
	return classes, nil
}

func (s *Store) GetTeacherRoutine(ctx context.Context, routineID, teacherID int64) (RoutineRow, error) {
	var entry RoutineRow
	err := s.pool.QueryRow(ctx, findTeacherRoutineSQL, routineID, teacherID).Scan(
		&entry.ID,
		&entry.Batch,
		&entry.Section,
		&entry.CourseID,
		&entry.CourseCode,
		&entry.CourseTitle,
		&entry.TeacherID,
		&entry.TeacherName,
		&entry.DayOfWeek,
		&entry.StartTime,
		&entry.EndTime,
		&entry.Room,
		&entry.Type,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutineRow{}, ErrNotFound
	}
	if err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not load the routine entry %d: %w", routineID, err)
	}
	return entry, nil
}

func (s *Store) CancelRoutineOccurrence(ctx context.Context, routineID, cancelledBy int64, date string) error {
	var id int64
	err := s.pool.QueryRow(ctx, insertRoutineCancellationSQL, routineID, date, cancelledBy).Scan(&id)
	if isUniqueViolation(err) {
		return ErrAlreadyCancelled
	}
	if err != nil {
		return fmt.Errorf("store: could not cancel the routine entry %d on %s: %w", routineID, date, err)
	}
	return nil
}

func (s *Store) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	rows, err := s.pool.Query(ctx, listActiveCRsSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class representatives of %s: %w", batch, err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan a class representative: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class representatives: %w", err)
	}
	return ids, nil
}

func (s *Store) AutoCancelStaleRequests(ctx context.Context, now time.Time) ([]EventDecision, error) {
	rows, err := s.pool.Query(ctx, autoCancelSQL, now, now.Format(dateLayout))
	if err != nil {
		return nil, fmt.Errorf("store: could not auto cancel the stale requests: %w", err)
	}
	defer rows.Close()
	cancelled := make([]EventDecision, 0)
	for rows.Next() {
		decision := EventDecision{Status: EventStatusAutoCancelled, DecidedAt: now}
		err := rows.Scan(
			&decision.ID,
			&decision.CreatedBy,
			&decision.EventType,
			&decision.CourseCode,
			&decision.CourseTitle,
			&decision.Batch,
			&decision.Section,
			&decision.EventDate,
			&decision.StartTime,
			&decision.EndTime,
			&decision.Room,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an auto cancelled request: %w", err)
		}
		cancelled = append(cancelled, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the auto cancelled requests: %w", err)
	}
	return cancelled, nil
}
