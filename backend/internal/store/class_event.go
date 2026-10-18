package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	EventStatusPending       = "pending"
	EventStatusApproved      = "approved"
	EventStatusRejected      = "rejected"
	EventStatusCancelled     = "cancelled"
	EventStatusAutoCancelled = "auto_cancelled"
	BookingSourceFixed       = "fixed"
	dateLayout               = "2006-01-02"
)

var ErrRoomUnavailable = errors.New("store: the room is already booked at this time")

type StudentProfile struct {
	UserID    int64
	StudentID int64
	Batch     string
	Section   string
	IsCR      bool
}

type NewClassEvent struct {
	Batch     string
	Section   string
	EventType string
	CourseID  int64
	TeacherID int64
	EventDate string
	StartTime string
	EndTime   string
	Room      string
	CreatedBy int64
}

type ClassEventRow struct {
	ID            int64
	Batch         string
	Section       string
	EventType     string
	CourseID      int64
	CourseCode    string
	CourseTitle   string
	TeacherID     int64
	TeacherUserID int64
	TeacherName   string
	EventDate     string
	StartTime     string
	EndTime       string
	Room          string
	Status        string
	CreatedBy     int64
	CreatedAt     time.Time
	DecidedAt     *time.Time
}

type FixedSlotError struct {
	Booking RoomBooking
}

func (e *FixedSlotError) Error() string {
	return fmt.Sprintf("store: room %s is reserved for %s section %s on day %d from %s to %s",
		e.Booking.Room, e.Booking.Batch, e.Booking.Section, e.Booking.DayOfWeek,
		e.Booking.StartTime, e.Booking.EndTime)
}

type Booking struct {
	Room       string
	StartTime  string
	EndTime    string
	Batch      string
	Section    string
	CourseCode string
	Source     string
	Status     string
}

const (
	findStudentProfileSQL = `SELECT u.id, s.id, s.batch, s.section, s.is_cr
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	listSectionRoutinesSQL = `SELECT r.id, r.batch, r.section, r.course_id, c.code, c.title, r.teacher_id, t.name,
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1 AND (r.section = $2 OR r.section = 'ALL')
ORDER BY r.day_of_week, r.start_time, r.room`
	lockRoomDateSQL        = `SELECT pg_advisory_xact_lock(hashtext($1::text || ':' || $2::text))`
	findTeacherForEventSQL = `SELECT t.name, t.user_id, u.status
FROM teachers t
JOIN users u ON u.id = t.user_id
WHERE t.id = $1`
	findDatedClashSQL = `SELECT e.id
FROM class_events e
WHERE e.room = $1
  AND e.event_date = $2::date
  AND e.status IN ('pending', 'approved')
  AND $3::time < e.end_time AND e.start_time < $4::time
LIMIT 1`
	findFixedClashSQL = `SELECT r.room, r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'),
r.batch, r.section, c.code
FROM routines r
JOIN courses c ON c.id = r.course_id
WHERE r.room = $1
  AND r.day_of_week = EXTRACT(DOW FROM $2::date)::int
  AND r.batch <> $3
  AND $4::time < r.end_time AND r.start_time < $5::time
ORDER BY r.start_time
LIMIT 1`
	insertClassEventSQL = `INSERT INTO class_events
(batch, section, course_id, teacher_id, event_date, start_time, end_time, room, event_type, status, created_by)
VALUES ($1, $2, $3, $4, $5::date, $6::time, $7::time, $8, $9, 'pending', $10)
RETURNING id, created_at`
	listRoomBookingsSQL = `SELECT r.room, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'),
r.batch, r.section, c.code, 'fixed' AS source, 'fixed' AS status
FROM routines r
JOIN courses c ON c.id = r.course_id
WHERE r.day_of_week = EXTRACT(DOW FROM $1::date)::int
  AND ($2::text = '' OR r.room = $2::text)
UNION ALL
SELECT e.room, to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
e.batch, e.section, c.code, e.event_type AS source, e.status AS status
FROM class_events e
JOIN courses c ON c.id = e.course_id
WHERE e.event_date = $1::date
  AND e.status IN ('pending', 'approved')
  AND ($2::text = '' OR e.room = $2::text)
ORDER BY 1, 2, 3`
	listClassRequestsSQL = `SELECT e.id, e.batch, e.section, e.event_type, e.course_id, c.code, c.title,
e.teacher_id, t.user_id, t.name, to_char(e.event_date, 'YYYY-MM-DD'),
to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'), e.room, e.status, e.created_at, e.decided_at
FROM class_events e
JOIN courses c ON c.id = e.course_id
JOIN teachers t ON t.id = e.teacher_id
WHERE e.created_by = $1 AND ($2::text = '' OR e.status = $2::text)
ORDER BY e.created_at DESC, e.id DESC`
)

func (s *Store) GetStudentProfile(ctx context.Context, userID int64) (StudentProfile, error) {
	var profile StudentProfile
	err := s.pool.QueryRow(ctx, findStudentProfileSQL, userID).Scan(
		&profile.UserID,
		&profile.StudentID,
		&profile.Batch,
		&profile.Section,
		&profile.IsCR,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StudentProfile{}, ErrNotFound
	}
	if err != nil {
		return StudentProfile{}, fmt.Errorf("store: could not load the student profile of the user %d: %w", userID, err)
	}
	return profile, nil
}

func (s *Store) ListSectionRoutines(ctx context.Context, batch, section string) ([]RoutineRow, error) {
	rows, err := s.pool.Query(ctx, listSectionRoutinesSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the routine of %s section %s: %w", batch, section, err)
	}
	defer rows.Close()
	entries := make([]RoutineRow, 0)
	for rows.Next() {
		var entry RoutineRow
		err := rows.Scan(
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
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a routine entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the routine entries: %w", err)
	}
	return entries, nil
}

func (s *Store) CreateClassRequest(ctx context.Context, event NewClassEvent) (ClassEventRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not start the class request transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, lockRoomDateSQL, event.Room, event.EventDate); err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not lock the room and date: %w", err)
	}
	row := ClassEventRow{
		Batch:     event.Batch,
		Section:   event.Section,
		EventType: event.EventType,
		CourseID:  event.CourseID,
		TeacherID: event.TeacherID,
		EventDate: event.EventDate,
		StartTime: event.StartTime,
		EndTime:   event.EndTime,
		Room:      event.Room,
		Status:    EventStatusPending,
	}
	err = tx.QueryRow(ctx, findCourseSQL, event.CourseID).Scan(&row.CourseCode, &row.CourseTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClassEventRow{}, ErrCourseNotFound
	}
	if err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not load the course %d: %w", event.CourseID, err)
	}
	var teacherStatus string
	err = tx.QueryRow(ctx, findTeacherForEventSQL, event.TeacherID).Scan(&row.TeacherName, &row.TeacherUserID, &teacherStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClassEventRow{}, ErrTeacherNotFound
	}
	if err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not load the teacher %d: %w", event.TeacherID, err)
	}
	if teacherStatus != StatusActive {
		return ClassEventRow{}, ErrTeacherInactive
	}
	var clashID int64
	err = tx.QueryRow(ctx, findDatedClashSQL, event.Room, event.EventDate, event.StartTime, event.EndTime).Scan(&clashID)
	if err == nil {
		return ClassEventRow{}, ErrRoomUnavailable
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ClassEventRow{}, fmt.Errorf("store: could not check the dated bookings: %w", err)
	}
	var booking RoomBooking
	err = tx.QueryRow(ctx, findFixedClashSQL, event.Room, event.EventDate, event.Batch, event.StartTime, event.EndTime).Scan(
		&booking.Room,
		&booking.DayOfWeek,
		&booking.StartTime,
		&booking.EndTime,
		&booking.Batch,
		&booking.Section,
		&booking.CourseCode,
	)
	if err == nil {
		return ClassEventRow{}, &FixedSlotError{Booking: booking}
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ClassEventRow{}, fmt.Errorf("store: could not check the fixed routine: %w", err)
	}
	err = tx.QueryRow(ctx, insertClassEventSQL,
		event.Batch,
		event.Section,
		event.CourseID,
		event.TeacherID,
		event.EventDate,
		event.StartTime,
		event.EndTime,
		event.Room,
		event.EventType,
		event.CreatedBy,
	).Scan(&row.ID, &row.CreatedAt)
	if err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not create the class request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ClassEventRow{}, fmt.Errorf("store: could not commit the class request: %w", err)
	}
	return row, nil
}

func (s *Store) ListRoomBookings(ctx context.Context, date, room string) ([]Booking, error) {
	rows, err := s.pool.Query(ctx, listRoomBookingsSQL, date, room)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the bookings of %s: %w", date, err)
	}
	defer rows.Close()
	bookings := make([]Booking, 0)
	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.Room,
			&booking.StartTime,
			&booking.EndTime,
			&booking.Batch,
			&booking.Section,
			&booking.CourseCode,
			&booking.Source,
			&booking.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a booking: %w", err)
		}
		bookings = append(bookings, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the bookings: %w", err)
	}
	return bookings, nil
}

func (s *Store) ListClassRequests(ctx context.Context, createdBy int64, status string) ([]ClassEventRow, error) {
	rows, err := s.pool.Query(ctx, listClassRequestsSQL, createdBy, status)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class requests of the user %d: %w", createdBy, err)
	}
	defer rows.Close()
	requests := make([]ClassEventRow, 0)
	for rows.Next() {
		var request ClassEventRow
		err := rows.Scan(
			&request.ID,
			&request.Batch,
			&request.Section,
			&request.EventType,
			&request.CourseID,
			&request.CourseCode,
			&request.CourseTitle,
			&request.TeacherID,
			&request.TeacherUserID,
			&request.TeacherName,
			&request.EventDate,
			&request.StartTime,
			&request.EndTime,
			&request.Room,
			&request.Status,
			&request.CreatedAt,
			&request.DecidedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a class request: %w", err)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class requests: %w", err)
	}
	return requests, nil
}
