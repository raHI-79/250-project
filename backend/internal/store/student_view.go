package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const NoDayFilter = -1

type StudentProfile struct {
	UserID    int64
	StudentID int64
	Batch     string
	Section   string
	IsCR      bool
}

type PlanItem struct {
	Batch       string
	Section     string
	StartTime   string
	EndTime     string
	CourseCode  string
	CourseTitle string
	Kind        string
	Mode        string
	Room        string
	MeetingLink string
	TeacherName string
	Status      string
	Note        string
}

type StudentExam struct {
	ID           int64
	ItemDate     string
	StartTime    string
	EndTime      string
	CourseCode   string
	CourseTitle  string
	Batch        string
	Section      string
	Mode         string
	Room         string
	MeetingLink  string
	Syllabus     string
	Instructions string
	Origin       string
	Status       string
	Published    bool
}

type RoutineFilter struct {
	Batch      string
	Semester   string
	Section    string
	DayOfWeek  int
	CourseCode string
}

const (
	findStudentProfileSQL = `SELECT u.id, s.id, s.batch, s.section, s.is_cr
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	listPublishedDaySQL = `SELECT p.batch, p.section,
COALESCE(to_char(p.start_time, 'HH24:MI'), ''), COALESCE(to_char(p.end_time, 'HH24:MI'), ''),
p.course_code, p.course_title, p.kind, p.mode, p.room, p.meeting_link, p.teacher_name, p.status, p.note
FROM published_items p
WHERE p.item_date = $2::date
  AND (p.batch = $1 OR (p.batch <> $1 AND replace(p.course_code, ' ', '') = ANY($3::text[])))
ORDER BY p.start_time NULLS LAST, p.batch, p.section`
	listStudentExamsSQL = `SELECT i.id, to_char(i.item_date, 'YYYY-MM-DD'),
COALESCE(to_char(pi.start_time, 'HH24:MI'), ''), COALESCE(to_char(pi.end_time, 'HH24:MI'), ''),
c.code, c.title, i.batch, i.section,
COALESCE(pi.mode, ''), COALESCE(pi.room, ''), COALESCE(pi.meeting_link, ''),
i.syllabus, i.instructions, i.origin, i.status, (pi.item_id IS NOT NULL)
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
LEFT JOIN published_items pi ON pi.item_id = i.id
WHERE i.origin = 'teacher_exam' AND i.status <> 'cancelled'
  AND ((i.batch = $1 AND (i.section = $2::text OR i.section = 'ALL'))
    OR (i.batch <> $1 AND replace(c.code, ' ', '') = ANY($4::text[])))
  AND i.item_date >= $3::date
UNION ALL
SELECT p.item_id, to_char(p.item_date, 'YYYY-MM-DD'),
COALESCE(to_char(p.start_time, 'HH24:MI'), ''), COALESCE(to_char(p.end_time, 'HH24:MI'), ''),
p.course_code, p.course_title, p.batch, p.section,
p.mode, p.room, p.meeting_link, p.syllabus, p.instructions, 'cr', p.status, TRUE
FROM published_items p
JOIN schedule_items i ON i.id = p.item_id
WHERE p.kind = 'exam' AND i.origin = 'cr' AND p.status <> 'cancelled'
  AND ((p.batch = $1 AND (p.section = $2::text OR p.section = 'ALL'))
    OR (p.batch <> $1 AND replace(p.course_code, ' ', '') = ANY($4::text[])))
  AND p.item_date >= $3::date
ORDER BY 2, 3`
	listFilteredRoutinesSQL = `SELECT r.id, r.batch, r.semester, r.section, r.course_id, c.code, c.title,
COALESCE(r.teacher_id, 0), COALESCE(t.name, 'Deleted user'),
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.batch = $1
  AND ($2::text = '' OR r.section = $2::text OR r.section = 'ALL')
  AND ($3::int < 0 OR r.day_of_week = $3::int)
  AND ($4::text = '' OR replace(c.code, ' ', '') = $4::text)
  AND r.semester = $5::text
ORDER BY r.day_of_week, r.start_time, r.section, r.id`
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

func (s *Store) ListPublishedDay(ctx context.Context, batch, date string, dropCourseCodes []string) ([]PlanItem, error) {
	rows, err := s.pool.Query(ctx, listPublishedDaySQL, batch, date, dropCourseCodes)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the published plan of %s on %s: %w", batch, date, err)
	}
	defer rows.Close()
	items := make([]PlanItem, 0)
	for rows.Next() {
		var item PlanItem
		err := rows.Scan(
			&item.Batch,
			&item.Section,
			&item.StartTime,
			&item.EndTime,
			&item.CourseCode,
			&item.CourseTitle,
			&item.Kind,
			&item.Mode,
			&item.Room,
			&item.MeetingLink,
			&item.TeacherName,
			&item.Status,
			&item.Note,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a published plan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the published plan: %w", err)
	}
	return items, nil
}

func (s *Store) ListStudentExams(ctx context.Context, batch, section string, now time.Time,
	dropCourseCodes []string) ([]StudentExam, error) {
	rows, err := s.pool.Query(ctx, listStudentExamsSQL, batch, section, now.Format(dateLayout),
		dropCourseCodes)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the exams of %s: %w", batch, err)
	}
	defer rows.Close()
	exams := make([]StudentExam, 0)
	for rows.Next() {
		var exam StudentExam
		err := rows.Scan(
			&exam.ID,
			&exam.ItemDate,
			&exam.StartTime,
			&exam.EndTime,
			&exam.CourseCode,
			&exam.CourseTitle,
			&exam.Batch,
			&exam.Section,
			&exam.Mode,
			&exam.Room,
			&exam.MeetingLink,
			&exam.Syllabus,
			&exam.Instructions,
			&exam.Origin,
			&exam.Status,
			&exam.Published,
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
	return s.routines(ctx, listFilteredRoutinesSQL,
		filter.Batch,
		filter.Section,
		filter.DayOfWeek,
		filter.CourseCode,
		filter.Semester,
	)
}
