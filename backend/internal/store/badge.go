package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	SectionNotices   = "notices"
	SectionMaterials = "materials"
	SectionPolls     = "polls"
	SectionExams     = "exams"
	SectionRoutine   = "routine"
	SectionCourses   = "courses"
)

var ErrUnknownSection = errors.New("store: that section has no badge")

func BadgeSections() []string {
	return []string{SectionNotices, SectionMaterials, SectionPolls, SectionExams,
		SectionRoutine, SectionCourses}
}

func IsBadgeSection(key string) bool {
	for _, section := range BadgeSections() {
		if section == key {
			return true
		}
	}
	return false
}

type BadgeViewer struct {
	UserID  int64
	Role    string
	Batch   string
	Section string
}

const (
	badgeViewerSQL = `SELECT u.id,
CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END,
COALESCE(s.batch, ''), COALESCE(s.section, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	listSectionViewsSQL = `SELECT section_key, last_seen_at FROM section_views WHERE user_id = $1`
	markSectionSeenSQL  = `INSERT INTO section_views (user_id, section_key, last_seen_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, section_key) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at`
	countNewNoticesSQL = `SELECT COUNT(*) FROM notices n
WHERE n.created_at > $4
  AND ((n.source = 'admin' AND (n.batch IS NULL OR n.batch = $1::text))
    OR (n.source = 'teacher' AND (n.batch IS NULL OR n.batch = $1::text))
    OR (n.source = 'cr' AND n.batch = $1::text
        AND (n.section IS NULL OR n.section = $2::text)))
  AND COALESCE(n.created_by, 0) <> $3`
	countNewMaterialsSQL = `SELECT COUNT(*) FROM batch_drive_links
WHERE batch = $1::text AND updated_at > $2 AND COALESCE(updated_by, 0) <> $3`
	countNewPollsSQL = `SELECT COUNT(*) FROM polls p
WHERE p.created_at > $4
  AND (p.scope = 'all_batches'
    OR (p.batch = $1::text AND (p.section = 'ALL' OR p.section = $2::text)))
  AND COALESCE(p.created_by, 0) <> $3`
	countNewExamsSQL = `SELECT COUNT(*) FROM published_items p
WHERE p.kind = 'exam' AND p.status <> 'cancelled' AND p.published_at > $3
  AND p.batch = $1::text AND (p.section = 'ALL' OR p.section = $2::text)`
	countNewRoutineSQL = `SELECT COUNT(*) FROM published_items p
WHERE p.kind <> 'exam' AND p.status <> 'cancelled' AND p.published_at > $3
  AND p.batch = $1::text AND (p.section = 'ALL' OR p.section = $2::text)`
	countNewCoursesSQL = `SELECT COUNT(*) FROM routines r
WHERE r.created_at > $3 AND r.batch = $1::text
  AND (r.section = 'ALL' OR r.section = $2::text)`
)

func (s *Store) BadgeViewer(ctx context.Context, userID int64) (BadgeViewer, error) {
	var viewer BadgeViewer
	err := s.pool.QueryRow(ctx, badgeViewerSQL, userID).
		Scan(&viewer.UserID, &viewer.Role, &viewer.Batch, &viewer.Section)
	if errors.Is(err, pgx.ErrNoRows) {
		return BadgeViewer{}, ErrNotFound
	}
	if err != nil {
		return BadgeViewer{}, fmt.Errorf("store: could not load the badge viewer %d: %w", userID, err)
	}
	return viewer, nil
}

func (s *Store) SectionViews(ctx context.Context, userID int64) (map[string]time.Time, error) {
	rows, err := s.pool.Query(ctx, listSectionViewsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the section views: %w", err)
	}
	defer rows.Close()
	seen := make(map[string]time.Time)
	for rows.Next() {
		var key string
		var at time.Time
		if err := rows.Scan(&key, &at); err != nil {
			return nil, fmt.Errorf("store: could not scan a section view: %w", err)
		}
		seen[key] = at
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the section views: %w", err)
	}
	return seen, nil
}

func (s *Store) MarkSectionSeen(ctx context.Context, userID int64, section string, at time.Time) error {
	if !IsBadgeSection(section) {
		return ErrUnknownSection
	}
	if _, err := s.pool.Exec(ctx, markSectionSeenSQL, userID, section, at); err != nil {
		return fmt.Errorf("store: could not mark the section %s as seen: %w", section, err)
	}
	return nil
}

func (s *Store) CountNewInSection(ctx context.Context, viewer BadgeViewer, section string,
	since time.Time) (int, error) {
	var count int
	var err error
	switch section {
	case SectionNotices:
		err = s.pool.QueryRow(ctx, countNewNoticesSQL, viewer.Batch, viewer.Section,
			viewer.UserID, since).Scan(&count)
	case SectionMaterials:
		err = s.pool.QueryRow(ctx, countNewMaterialsSQL, viewer.Batch, since, viewer.UserID).Scan(&count)
	case SectionPolls:
		err = s.pool.QueryRow(ctx, countNewPollsSQL, viewer.Batch, viewer.Section,
			viewer.UserID, since).Scan(&count)
	case SectionExams:
		err = s.pool.QueryRow(ctx, countNewExamsSQL, viewer.Batch, viewer.Section, since).Scan(&count)
	case SectionRoutine:
		err = s.pool.QueryRow(ctx, countNewRoutineSQL, viewer.Batch, viewer.Section, since).Scan(&count)
	case SectionCourses:
		err = s.pool.QueryRow(ctx, countNewCoursesSQL, viewer.Batch, viewer.Section, since).Scan(&count)
	default:
		return 0, ErrUnknownSection
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not count the new items of %s: %w", section, err)
	}
	return count, nil
}

type UndecidedTeacher struct {
	UserID int64
	Count  int
}

const (
	undecidedTomorrowSQL = `SELECT t.user_id, COUNT(*)
FROM schedule_items i
JOIN teachers t ON t.id = i.teacher_id
JOIN users u ON u.id = t.user_id
WHERE i.item_date = $1::date
  AND i.kind <> 'exam'
  AND i.status IN ('awaiting_teacher', 'no_response')
  AND u.status = 'active'
GROUP BY t.user_id`
	undecidedTodaySQL = `SELECT t.user_id, COUNT(*)
FROM schedule_items i
JOIN teachers t ON t.id = i.teacher_id
JOIN users u ON u.id = t.user_id
WHERE i.item_date = $1::date
  AND i.kind <> 'exam'
  AND i.status IN ('awaiting_teacher', 'no_response')
  AND i.start_time > $2::time
  AND u.status = 'active'
GROUP BY t.user_id`
	claimReminderSQL = `INSERT INTO reminder_log (user_id, slot, reminder_date)
VALUES ($1, $2, $3::date)
ON CONFLICT (user_id, slot, reminder_date) DO NOTHING
RETURNING id`
)

func (s *Store) ListUndecidedTeachers(ctx context.Context, date string) ([]UndecidedTeacher, error) {
	return s.undecided(ctx, undecidedTomorrowSQL, date)
}

func (s *Store) ListUndecidedTeachersFrom(ctx context.Context, date, after string) ([]UndecidedTeacher, error) {
	return s.undecided(ctx, undecidedTodaySQL, date, after)
}

func (s *Store) undecided(ctx context.Context, query string, args ...any) ([]UndecidedTeacher, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the undecided teachers: %w", err)
	}
	defer rows.Close()
	teachers := make([]UndecidedTeacher, 0)
	for rows.Next() {
		var teacher UndecidedTeacher
		if err := rows.Scan(&teacher.UserID, &teacher.Count); err != nil {
			return nil, fmt.Errorf("store: could not scan an undecided teacher: %w", err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the undecided teachers: %w", err)
	}
	return teachers, nil
}

func (s *Store) ClaimReminder(ctx context.Context, userID int64, slot, date string) (bool, error) {
	var id int64
	err := s.pool.QueryRow(ctx, claimReminderSQL, userID, slot, date).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: could not claim the %s reminder: %w", slot, err)
	}
	return true, nil
}
