package store

import (
	"context"
	"errors"
	"fmt"
)

var ErrDuplicateCourse = errors.New("store: a course with this code already exists")

type Course struct {
	ID    int64
	Code  string
	Title string
}

const (
	insertCourseSQL = `INSERT INTO courses (code, title)
VALUES ($1, $2)
RETURNING id, code, title`
	listCoursesSQL = `SELECT id, code, title FROM courses ORDER BY code`
)

func (s *Store) CreateCourse(ctx context.Context, code, title string) (Course, error) {
	var course Course
	err := s.pool.QueryRow(ctx, insertCourseSQL, code, title).Scan(&course.ID, &course.Code, &course.Title)
	if isUniqueViolation(err) {
		return Course{}, ErrDuplicateCourse
	}
	if err != nil {
		return Course{}, fmt.Errorf("store: could not create the course %s: %w", code, err)
	}
	return course, nil
}

func (s *Store) ListCourses(ctx context.Context) ([]Course, error) {
	rows, err := s.pool.Query(ctx, listCoursesSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the courses: %w", err)
	}
	defer rows.Close()
	courses := make([]Course, 0)
	for rows.Next() {
		var course Course
		if err := rows.Scan(&course.ID, &course.Code, &course.Title); err != nil {
			return nil, fmt.Errorf("store: could not scan a course: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the courses: %w", err)
	}
	return courses, nil
}

type CourseSlotRow struct {
	CourseID      int64
	CourseCode    string
	CourseTitle   string
	TeacherID     int64
	TeacherUserID int64
	TeacherName   string
	DayOfWeek     int
	StartTime     string
	EndTime       string
	Room          string
	Type          string
}

type DayEntryRow struct {
	Source      string
	Section     string
	CourseCode  string
	CourseTitle string
	TeacherName string
	StartTime   string
	EndTime     string
	Room        string
	Kind        string
	Status      string
}

const (
	listBatchCourseSlotsSQL = `SELECT r.course_id, c.code, c.title,
COALESCE(r.teacher_id, 0), COALESCE(t.user_id, 0), COALESCE(t.name, 'Deleted user'),
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1 AND (r.section = $2 OR r.section = 'ALL')
ORDER BY c.code, r.day_of_week, r.start_time`
	listBatchDaySQL = `SELECT 'fixed' AS source, r.section, c.code, c.title, COALESCE(t.name, 'Deleted user'),
to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type,
CASE WHEN EXISTS (
    SELECT 1 FROM routine_cancellations rc WHERE rc.routine_id = r.id AND rc.cancel_date = $2::date
) THEN 'cancelled' ELSE 'scheduled' END
FROM routines r
JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1 AND r.day_of_week = EXTRACT(DOW FROM $2::date)::int
UNION ALL
SELECT 'event' AS source, e.section, c.code, c.title, COALESCE(t.name, 'Deleted user'),
to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'), e.room, e.event_type, e.status
FROM class_events e
JOIN courses c ON c.id = e.course_id
LEFT JOIN teachers t ON t.id = e.teacher_id
WHERE e.batch = $1 AND e.event_date = $2::date AND e.status IN ('pending', 'approved')
ORDER BY 6, 2, 3`
)

func (s *Store) ListBatchCourseSlots(ctx context.Context, batch, section string) ([]CourseSlotRow, error) {
	rows, err := s.pool.Query(ctx, listBatchCourseSlotsSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the courses of %s: %w", batch, err)
	}
	defer rows.Close()
	slots := make([]CourseSlotRow, 0)
	for rows.Next() {
		var slot CourseSlotRow
		err := rows.Scan(
			&slot.CourseID,
			&slot.CourseCode,
			&slot.CourseTitle,
			&slot.TeacherID,
			&slot.TeacherUserID,
			&slot.TeacherName,
			&slot.DayOfWeek,
			&slot.StartTime,
			&slot.EndTime,
			&slot.Room,
			&slot.Type,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a course slot: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the course slots: %w", err)
	}
	return slots, nil
}

func (s *Store) ListBatchDay(ctx context.Context, batch, date string) ([]DayEntryRow, error) {
	rows, err := s.pool.Query(ctx, listBatchDaySQL, batch, date)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the day of %s: %w", batch, err)
	}
	defer rows.Close()
	entries := make([]DayEntryRow, 0)
	for rows.Next() {
		var entry DayEntryRow
		err := rows.Scan(
			&entry.Source,
			&entry.Section,
			&entry.CourseCode,
			&entry.CourseTitle,
			&entry.TeacherName,
			&entry.StartTime,
			&entry.EndTime,
			&entry.Room,
			&entry.Kind,
			&entry.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a day entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the day entries: %w", err)
	}
	return entries, nil
}
