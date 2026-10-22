package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/raHI-79/250-project/backend/internal/validate"
)

var (
	ErrDuplicateCourse = errors.New("store: a course with this code already exists")
	ErrCourseInUse     = errors.New("store: this course is still used by the department")
)

type Course struct {
	ID     int64
	Code   string
	Title  string
	Credit float64
}

const (
	insertCourseSQL = `INSERT INTO courses (code, title, credit)
VALUES ($1, $2, $3)
RETURNING id, code, title, credit`
	listCoursesSQL   = `SELECT id, code, title, credit FROM courses ORDER BY code`
	searchCoursesSQL = `SELECT id, code, title, credit
FROM courses
WHERE $1::text = ''
   OR replace(code, ' ', '') ILIKE '%' || replace(replace(upper($1::text), '-', ''), ' ', '') || '%'
   OR title ILIKE '%' || $1::text || '%'
ORDER BY code`
)

func (s *Store) CreateCourse(ctx context.Context, code, title string, credit float64) (Course, error) {
	var course Course
	normalised := validate.NormalizeCourseCode(code)
	err := s.pool.QueryRow(ctx, insertCourseSQL, normalised, title, credit).
		Scan(&course.ID, &course.Code, &course.Title, &course.Credit)
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
		if err := rows.Scan(&course.ID, &course.Code, &course.Title, &course.Credit); err != nil {
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
JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
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

type CourseUsage struct {
	Routines    int
	Items       int
	DropCourses int
}

func (u CourseUsage) InUse() bool {
	return u.Routines > 0 || u.Items > 0 || u.DropCourses > 0
}

const (
	courseUsageSQL = `SELECT
(SELECT COUNT(*) FROM routines WHERE course_id = $1),
(SELECT COUNT(*) FROM schedule_items WHERE course_id = $1),
(SELECT COUNT(*) FROM student_drop_courses WHERE course_id = $1)`
	deleteCourseSQL = `DELETE FROM courses WHERE id = $1 RETURNING id`
)

func (s *Store) CourseUsage(ctx context.Context, courseID int64) (CourseUsage, error) {
	var usage CourseUsage
	err := s.pool.QueryRow(ctx, courseUsageSQL, courseID).
		Scan(&usage.Routines, &usage.Items, &usage.DropCourses)
	if err != nil {
		return CourseUsage{}, fmt.Errorf("store: could not count what uses the course %d: %w", courseID, err)
	}
	return usage, nil
}

func (s *Store) DeleteCourse(ctx context.Context, courseID int64) (CourseUsage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CourseUsage{}, fmt.Errorf("store: could not start the course transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var usage CourseUsage
	err = tx.QueryRow(ctx, courseUsageSQL, courseID).
		Scan(&usage.Routines, &usage.Items, &usage.DropCourses)
	if err != nil {
		return CourseUsage{}, fmt.Errorf("store: could not count what uses the course %d: %w", courseID, err)
	}
	if usage.InUse() {
		return usage, ErrCourseInUse
	}
	var id int64
	err = tx.QueryRow(ctx, deleteCourseSQL, courseID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return CourseUsage{}, ErrNotFound
	}
	if err != nil {
		return CourseUsage{}, fmt.Errorf("store: could not delete the course %d: %w", courseID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CourseUsage{}, fmt.Errorf("store: could not commit the course deletion: %w", err)
	}
	return usage, nil
}

type RunningCourseSlot struct {
	CourseID           int64
	CourseCode         string
	CourseTitle        string
	Credit             float64
	TeacherUserID      int64
	TeacherName        string
	TeacherDesignation string
	Section            string
	Type               string
}

const listRunningCourseSlotsSQL = `SELECT r.course_id, c.code, c.title, c.credit,
COALESCE(t.user_id, 0), COALESCE(t.name, 'Deleted user'), COALESCE(t.designation, ''),
r.section, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1
ORDER BY c.code, r.section, r.day_of_week, r.start_time`

func (s *Store) ListRunningCourseSlots(ctx context.Context, batch string) ([]RunningCourseSlot, error) {
	rows, err := s.pool.Query(ctx, listRunningCourseSlotsSQL, batch)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the running courses of %s: %w", batch, err)
	}
	defer rows.Close()
	slots := make([]RunningCourseSlot, 0)
	for rows.Next() {
		var slot RunningCourseSlot
		err := rows.Scan(
			&slot.CourseID,
			&slot.CourseCode,
			&slot.CourseTitle,
			&slot.Credit,
			&slot.TeacherUserID,
			&slot.TeacherName,
			&slot.TeacherDesignation,
			&slot.Section,
			&slot.Type,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a running course slot: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the running courses: %w", err)
	}
	return slots, nil
}
