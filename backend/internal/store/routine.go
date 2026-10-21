package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCourseNotFound  = errors.New("store: no course has this id")
	ErrTeacherNotFound = errors.New("store: no teacher has this id")
	ErrTeacherInactive = errors.New("store: the teacher account is not active")
)

type NewRoutine struct {
	Batch     string
	Section   string
	CourseID  int64
	TeacherID int64
	DayOfWeek int
	StartTime string
	EndTime   string
	Room      string
	Type      string
}

type RoutineRow struct {
	ID          int64
	Batch       string
	Section     string
	CourseID    int64
	CourseCode  string
	CourseTitle string
	TeacherID   int64
	TeacherName string
	DayOfWeek   int
	StartTime   string
	EndTime     string
	Room        string
	Type        string
}

type RoomBooking struct {
	Room       string
	DayOfWeek  int
	StartTime  string
	EndTime    string
	Batch      string
	Section    string
	CourseCode string
}

type RoomBookedError struct {
	Booking RoomBooking
}

func (e *RoomBookedError) Error() string {
	return fmt.Sprintf("store: room %s is already booked on day %d from %s to %s",
		e.Booking.Room, e.Booking.DayOfWeek, e.Booking.StartTime, e.Booking.EndTime)
}

const (
	lockRoomDaySQL   = `SELECT pg_advisory_xact_lock(hashtext($1::text || ':' || $2::int::text))`
	findCourseSQL    = `SELECT code, title FROM courses WHERE id = $1`
	findTeacherSQL   = `SELECT t.name, u.status FROM teachers t JOIN users u ON u.id = t.user_id WHERE t.id = $1`
	findRoomClashSQL = `SELECT r.room, r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'),
r.batch, r.section, c.code
FROM routines r
JOIN courses c ON c.id = r.course_id
WHERE r.room = $1 AND r.day_of_week = $2 AND $3::time < r.end_time AND r.start_time < $4::time
ORDER BY r.start_time
LIMIT 1`
	insertRoutineSQL = `INSERT INTO routines
(batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ($1, $2, $3, $4, $5, $6::time, $7::time, $8, $9)
RETURNING id`
	listRoutinesSQL = `SELECT r.id, r.batch, r.section, r.course_id, c.code, c.title,
COALESCE(r.teacher_id, 0), COALESCE(t.name, 'Deleted user'),
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE ($1::text = '' OR r.batch = $1::text)
  AND ($2::text = '' OR r.section = $2::text OR r.section = 'ALL')
ORDER BY r.batch, r.section, r.day_of_week, r.start_time`
	deleteRoutineSQL = `DELETE FROM routines WHERE id = $1 RETURNING id`
)

func (s *Store) CreateRoutine(ctx context.Context, routine NewRoutine) (RoutineRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not start the routine transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, lockRoomDaySQL, routine.Room, routine.DayOfWeek); err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not lock the room and day: %w", err)
	}
	row := RoutineRow{
		Batch:     routine.Batch,
		Section:   routine.Section,
		CourseID:  routine.CourseID,
		TeacherID: routine.TeacherID,
		DayOfWeek: routine.DayOfWeek,
		StartTime: routine.StartTime,
		EndTime:   routine.EndTime,
		Room:      routine.Room,
		Type:      routine.Type,
	}
	err = tx.QueryRow(ctx, findCourseSQL, routine.CourseID).Scan(&row.CourseCode, &row.CourseTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutineRow{}, ErrCourseNotFound
	}
	if err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not load the course %d: %w", routine.CourseID, err)
	}
	var teacherStatus string
	err = tx.QueryRow(ctx, findTeacherSQL, routine.TeacherID).Scan(&row.TeacherName, &teacherStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutineRow{}, ErrTeacherNotFound
	}
	if err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not load the teacher %d: %w", routine.TeacherID, err)
	}
	if teacherStatus != StatusActive {
		return RoutineRow{}, ErrTeacherInactive
	}
	var booking RoomBooking
	err = tx.QueryRow(ctx, findRoomClashSQL, routine.Room, routine.DayOfWeek, routine.StartTime, routine.EndTime).Scan(
		&booking.Room,
		&booking.DayOfWeek,
		&booking.StartTime,
		&booking.EndTime,
		&booking.Batch,
		&booking.Section,
		&booking.CourseCode,
	)
	if err == nil {
		return RoutineRow{}, &RoomBookedError{Booking: booking}
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RoutineRow{}, fmt.Errorf("store: could not check the room bookings: %w", err)
	}
	err = tx.QueryRow(ctx, insertRoutineSQL,
		routine.Batch,
		routine.Section,
		routine.CourseID,
		routine.TeacherID,
		routine.DayOfWeek,
		routine.StartTime,
		routine.EndTime,
		routine.Room,
		routine.Type,
	).Scan(&row.ID)
	if err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not create the routine entry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return RoutineRow{}, fmt.Errorf("store: could not commit the routine entry: %w", err)
	}
	return row, nil
}

func (s *Store) ListRoutines(ctx context.Context, batch, section string) ([]RoutineRow, error) {
	rows, err := s.pool.Query(ctx, listRoutinesSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the routine entries: %w", err)
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

func (s *Store) DeleteRoutine(ctx context.Context, routineID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deleteRoutineSQL, routineID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not delete the routine entry %d: %w", routineID, err)
	}
	return nil
}
