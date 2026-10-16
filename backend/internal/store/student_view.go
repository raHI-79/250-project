package store

import (
	"context"
	"fmt"
	"time"
)

const NoDayFilter = -1

type PlanItem struct {
	Section     string
	StartTime   string
	EndTime     string
	CourseCode  string
	CourseTitle string
	Kind        string
	Room        string
	TeacherName string
	Cancelled   bool
}

type DropCourseItem struct {
	Source      string
	Batch       string
	Section     string
	CourseCode  string
	CourseTitle string
	Kind        string
	DayOfWeek   int
	EventDate   string
	StartTime   string
	EndTime     string
	Room        string
	TeacherName string
}

type ExamRow struct {
	ID          int64
	EventDate   string
	StartTime   string
	EndTime     string
	CourseCode  string
	CourseTitle string
	Batch       string
	Section     string
	Room        string
}

type RoutineFilter struct {
	Batch      string
	Section    string
	DayOfWeek  int
	CourseCode string
}

const (
	listDayPlanSQL = `SELECT r.section, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'),
c.code, c.title, r.type, r.room, t.name,
EXISTS (SELECT 1 FROM routine_cancellations rc WHERE rc.routine_id = r.id AND rc.cancel_date = $2::date) AS cancelled
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1 AND r.day_of_week = EXTRACT(DOW FROM $2::date)::int
UNION ALL
SELECT e.section, to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
c.code, c.title, e.event_type, e.room, t.name, (e.status = 'cancelled') AS cancelled
FROM class_events e
JOIN courses c ON c.id = e.course_id
JOIN teachers t ON t.id = e.teacher_id
WHERE e.batch = $1 AND e.event_date = $2::date AND e.status IN ('approved', 'cancelled')
ORDER BY 1, 2`
	listDropCourseSQL = `SELECT 'fixed' AS source, r.batch, r.section, c.code, c.title, r.type,
r.day_of_week, '' AS event_date, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, t.name
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN teachers t ON t.id = r.teacher_id
WHERE c.code = $1 AND r.batch <> $2
UNION ALL
SELECT 'dated' AS source, e.batch, e.section, c.code, c.title, e.event_type,
EXTRACT(DOW FROM e.event_date)::int, to_char(e.event_date, 'YYYY-MM-DD'),
to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'), e.room, t.name
FROM class_events e
JOIN courses c ON c.id = e.course_id
JOIN teachers t ON t.id = e.teacher_id
WHERE c.code = $1 AND e.batch <> $2 AND e.status = 'approved'
  AND e.event_type IN ('class', 'exam') AND e.event_date >= $3::date
ORDER BY 2, 7, 9`
	listExamsSQL = `SELECT e.id, to_char(e.event_date, 'YYYY-MM-DD'), to_char(e.start_time, 'HH24:MI'),
to_char(e.end_time, 'HH24:MI'), c.code, c.title, e.batch, e.section, e.room
FROM class_events e
JOIN courses c ON c.id = e.course_id
WHERE e.batch = $1 AND (e.section = $2 OR e.section = 'ALL') AND e.status = 'approved'
  AND e.event_type = 'exam' AND e.event_date >= $3::date
ORDER BY e.event_date, e.start_time, e.id`
	listFilteredRoutinesSQL = `SELECT r.id, r.batch, r.section, r.course_id, c.code, c.title, r.teacher_id, t.name,
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1
  AND ($2::text = '' OR r.section = $2::text OR r.section = 'ALL')
  AND ($3::int < 0 OR r.day_of_week = $3::int)
  AND ($4::text = '' OR c.code = $4::text)
ORDER BY r.day_of_week, r.start_time, r.section, r.id`
)

func (s *Store) ListDayPlan(ctx context.Context, batch, date string) ([]PlanItem, error) {
	rows, err := s.pool.Query(ctx, listDayPlanSQL, batch, date)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the plan of %s on %s: %w", batch, date, err)
	}
	defer rows.Close()
	items := make([]PlanItem, 0)
	for rows.Next() {
		var item PlanItem
		err := rows.Scan(
			&item.Section,
			&item.StartTime,
			&item.EndTime,
			&item.CourseCode,
			&item.CourseTitle,
			&item.Kind,
			&item.Room,
			&item.TeacherName,
			&item.Cancelled,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a plan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the plan: %w", err)
	}
	return items, nil
}

func (s *Store) ListDropCourseSchedule(ctx context.Context, courseCode, ownBatch string, now time.Time) ([]DropCourseItem, error) {
	rows, err := s.pool.Query(ctx, listDropCourseSQL, courseCode, ownBatch, now.Format(dateLayout))
	if err != nil {
		return nil, fmt.Errorf("store: could not list the schedule of the course %s: %w", courseCode, err)
	}
	defer rows.Close()
	items := make([]DropCourseItem, 0)
	for rows.Next() {
		var item DropCourseItem
		err := rows.Scan(
			&item.Source,
			&item.Batch,
			&item.Section,
			&item.CourseCode,
			&item.CourseTitle,
			&item.Kind,
			&item.DayOfWeek,
			&item.EventDate,
			&item.StartTime,
			&item.EndTime,
			&item.Room,
			&item.TeacherName,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a drop course item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drop course schedule: %w", err)
	}
	return items, nil
}

func (s *Store) ListExams(ctx context.Context, batch, section string, now time.Time) ([]ExamRow, error) {
	rows, err := s.pool.Query(ctx, listExamsSQL, batch, section, now.Format(dateLayout))
	if err != nil {
		return nil, fmt.Errorf("store: could not list the exams of %s: %w", batch, err)
	}
	defer rows.Close()
	exams := make([]ExamRow, 0)
	for rows.Next() {
		var exam ExamRow
		err := rows.Scan(
			&exam.ID,
			&exam.EventDate,
			&exam.StartTime,
			&exam.EndTime,
			&exam.CourseCode,
			&exam.CourseTitle,
			&exam.Batch,
			&exam.Section,
			&exam.Room,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an exam: %w", err)
		}
		exams = append(exams, exam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the exams: %w", err)
	}
	return exams, nil
}

func (s *Store) ListFilteredRoutines(ctx context.Context, filter RoutineFilter) ([]RoutineRow, error) {
	rows, err := s.pool.Query(ctx, listFilteredRoutinesSQL,
		filter.Batch,
		filter.Section,
		filter.DayOfWeek,
		filter.CourseCode,
	)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the routine of %s: %w", filter.Batch, err)
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
