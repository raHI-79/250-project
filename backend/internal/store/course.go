package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/validate"
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
	listCoursesSQL   = `SELECT id, code, title FROM courses ORDER BY code`
	searchCoursesSQL = `SELECT id, code, title
FROM courses
WHERE $1::text = ''
   OR replace(code, ' ', '') ILIKE '%' || replace(replace(upper($1::text), '-', ''), ' ', '') || '%'
   OR title ILIKE '%' || $1::text || '%'
ORDER BY code`
)

func (s *Store) CreateCourse(ctx context.Context, code, title string) (Course, error) {
	var course Course
	normalised := validate.NormalizeCourseCode(code)
	err := s.pool.QueryRow(ctx, insertCourseSQL, normalised, title).Scan(&course.ID, &course.Code, &course.Title)
	if isUniqueViolation(err) {
		return Course{}, ErrDuplicateCourse
	}
	if err != nil {
		return Course{}, fmt.Errorf("store: could not create the course %s: %w", code, err)
	}
	return course, nil
}

func (s *Store) ListCourses(ctx context.Context) ([]Course, error) {
	return s.courses(ctx, listCoursesSQL)
}

func (s *Store) SearchCourses(ctx context.Context, search string) ([]Course, error) {
	return s.courses(ctx, searchCoursesSQL, search)
}

func (s *Store) courses(ctx context.Context, query string, args ...any) ([]Course, error) {
	rows, err := s.pool.Query(ctx, query, args...)
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

const listBatchCourseSlotsSQL = `SELECT r.course_id, c.code, c.title,
COALESCE(r.teacher_id, 0), COALESCE(t.user_id, 0), COALESCE(t.name, 'Deleted user'),
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1 AND (r.section = $2 OR r.section = 'ALL')
ORDER BY c.code, r.day_of_week, r.start_time`

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
