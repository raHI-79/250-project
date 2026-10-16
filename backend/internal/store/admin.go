package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	StatusFilterAll    = "all"
	DefaultStudentPage = 50
	MaxStudentPage     = 200
)

var (
	ErrNotPending     = errors.New("store: the registration is not pending")
	ErrAlreadyRemoved = errors.New("store: the account is already removed")
	ErrNotCR          = errors.New("store: the account is not a class representative")
)

type PendingRegistration struct {
	ID             int64
	FullName       string
	RegNo          string
	Batch          string
	Section        string
	Email          string
	Phone          string
	BloodGroup     string
	HasDropCourse  bool
	DropCourseCode string
	CreatedAt      time.Time
}

type StudentRow struct {
	ID             int64
	FullName       string
	RegNo          string
	Batch          string
	Section        string
	Email          string
	Phone          string
	BloodGroup     string
	HasDropCourse  bool
	DropCourseCode string
	IsCR           bool
	Status         string
	CreatedAt      time.Time
}

type StudentFilter struct {
	Search  string
	Batch   string
	Section string
	Status  string
	Limit   int
	Offset  int
}

type CRFilter struct {
	Search string
	Batch  string
}

type TeacherRow struct {
	ID        int64
	TeacherID string
	Name      string
	Email     string
	Phone     string
	Status    string
	CreatedAt time.Time
}

type TeacherFilter struct {
	Search string
	Status string
}

type NewCR struct {
	FullName     string
	RegNo        string
	Batch        string
	Section      string
	Email        string
	Phone        string
	BloodGroup   string
	PasswordHash string
}

type NewTeacher struct {
	Name         string
	TeacherID    string
	Email        string
	Phone        string
	PasswordHash string
}

type Stats struct {
	TotalStudents    int
	TotalCRs         int
	TotalTeachers    int
	PendingApprovals int
}

const (
	listPendingRegistrationsSQL = `SELECT u.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, s.has_drop_course, COALESCE(s.drop_course_code, ''), s.created_at
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.role = 'student' AND u.status = 'pending'
ORDER BY u.id`
	setRegistrationStatusSQL = `UPDATE users
SET status = $2, updated_at = now()
WHERE id = $1 AND role = 'student' AND status = 'pending'
RETURNING id`
	studentStatusSQL = `SELECT u.status
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	listStudentsSQL = `SELECT u.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, s.has_drop_course, COALESCE(s.drop_course_code, ''), s.is_cr, u.status, s.created_at,
COUNT(*) OVER () AS total
FROM users u
JOIN students s ON s.user_id = u.id
WHERE ($1::text = '' OR s.full_name ILIKE '%' || $1::text || '%' OR s.reg_no ILIKE '%' || $1::text || '%')
  AND ($2::text = '' OR s.batch = $2::text)
  AND ($3::text = '' OR s.section = $3::text)
  AND ($4::text = 'all' OR ($4::text = '' AND u.status <> 'rejected') OR u.status = $4::text)
ORDER BY s.batch, s.section, s.full_name, s.reg_no
LIMIT $5 OFFSET $6`
	removeStudentSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'student' AND status <> 'removed'
RETURNING id`
	insertCRUserSQL = `INSERT INTO users (role, identifier, password_hash, status)
VALUES ('student', $1, $2, 'active')
RETURNING id`
	insertCRStudentSQL = `INSERT INTO students
(user_id, full_name, reg_no, batch, section, email, phone, blood_group, has_drop_course, is_cr)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, TRUE)`
	listCRsSQL = `SELECT u.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, s.is_cr, u.status, s.created_at
FROM users u
JOIN students s ON s.user_id = u.id
WHERE s.is_cr
  AND u.status <> 'rejected'
  AND ($1::text = '' OR s.full_name ILIKE '%' || $1::text || '%' OR s.reg_no ILIKE '%' || $1::text || '%')
  AND ($2::text = '' OR s.batch = $2::text)
ORDER BY s.batch, s.section, s.full_name, s.reg_no`
	demoteCRSQL = `UPDATE students
SET is_cr = FALSE, updated_at = now()
WHERE user_id = $1 AND is_cr
RETURNING id`
	deactivateCRSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND status <> 'removed'
  AND EXISTS (SELECT 1 FROM students s WHERE s.user_id = users.id AND s.is_cr)
RETURNING id`
	crStatusSQL = `SELECT u.status
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.id = $1 AND s.is_cr`
	insertTeacherUserSQL = `INSERT INTO users (role, identifier, password_hash, status)
VALUES ('teacher', $1, $2, 'active')
RETURNING id`
	insertTeacherSQL = `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, $3, $4, $5)`
	listTeachersSQL = `SELECT u.id, t.teacher_id, t.name, t.email, t.phone, u.status, t.created_at
FROM users u
JOIN teachers t ON t.user_id = u.id
WHERE ($1::text = '' OR t.name ILIKE '%' || $1::text || '%' OR t.teacher_id ILIKE '%' || $1::text || '%')
  AND ($2::text = 'all' OR ($2::text = '' AND u.status <> 'rejected') OR u.status = $2::text)
ORDER BY t.name, t.teacher_id`
	deactivateTeacherSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'teacher' AND status <> 'removed'
RETURNING id`
	teacherStatusSQL = `SELECT status FROM users WHERE id = $1 AND role = 'teacher'`
	statsSQL         = `SELECT
(SELECT COUNT(*) FROM users u JOIN students s ON s.user_id = u.id WHERE u.status = 'active'),
(SELECT COUNT(*) FROM users u JOIN students s ON s.user_id = u.id WHERE u.status = 'active' AND s.is_cr),
(SELECT COUNT(*) FROM users u JOIN teachers t ON t.user_id = u.id WHERE u.status = 'active'),
(SELECT COUNT(*) FROM users WHERE role = 'student' AND status = 'pending')`
)

func (s *Store) ListPendingRegistrations(ctx context.Context) ([]PendingRegistration, error) {
	rows, err := s.pool.Query(ctx, listPendingRegistrationsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the pending registrations: %w", err)
	}
	defer rows.Close()
	registrations := make([]PendingRegistration, 0)
	for rows.Next() {
		var registration PendingRegistration
		err := rows.Scan(
			&registration.ID,
			&registration.FullName,
			&registration.RegNo,
			&registration.Batch,
			&registration.Section,
			&registration.Email,
			&registration.Phone,
			&registration.BloodGroup,
			&registration.HasDropCourse,
			&registration.DropCourseCode,
			&registration.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a pending registration: %w", err)
		}
		registrations = append(registrations, registration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the pending registrations: %w", err)
	}
	return registrations, nil
}

func (s *Store) SetRegistrationStatus(ctx context.Context, userID int64, status string) error {
	var id int64
	err := s.pool.QueryRow(ctx, setRegistrationStatusSQL, userID, status).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.pendingConflict(ctx, userID)
	}
	if err != nil {
		return fmt.Errorf("store: could not set the status of the registration %d: %w", userID, err)
	}
	return nil
}

func (s *Store) pendingConflict(ctx context.Context, userID int64) error {
	var status string
	err := s.pool.QueryRow(ctx, studentStatusSQL, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not load the registration %d: %w", userID, err)
	}
	return ErrNotPending
}

func (s *Store) ListStudents(ctx context.Context, filter StudentFilter) ([]StudentRow, int, error) {
	rows, err := s.pool.Query(ctx, listStudentsSQL,
		filter.Search,
		filter.Batch,
		filter.Section,
		filter.Status,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: could not list the students: %w", err)
	}
	defer rows.Close()
	students := make([]StudentRow, 0)
	total := 0
	for rows.Next() {
		var student StudentRow
		err := rows.Scan(
			&student.ID,
			&student.FullName,
			&student.RegNo,
			&student.Batch,
			&student.Section,
			&student.Email,
			&student.Phone,
			&student.BloodGroup,
			&student.HasDropCourse,
			&student.DropCourseCode,
			&student.IsCR,
			&student.Status,
			&student.CreatedAt,
			&total,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("store: could not scan a student: %w", err)
		}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: could not read the students: %w", err)
	}
	return students, total, nil
}

func (s *Store) RemoveStudent(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, removeStudentSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.removalConflict(ctx, studentStatusSQL, userID)
	}
	if err != nil {
		return fmt.Errorf("store: could not remove the student %d: %w", userID, err)
	}
	return nil
}

func (s *Store) removalConflict(ctx context.Context, lookupSQL string, userID int64) error {
	var status string
	err := s.pool.QueryRow(ctx, lookupSQL, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not load the account %d: %w", userID, err)
	}
	if status == StatusRemoved {
		return ErrAlreadyRemoved
	}
	return ErrNotFound
}

func (s *Store) CreateCR(ctx context.Context, cr NewCR) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: could not start the class representative transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID int64
	if err := tx.QueryRow(ctx, insertCRUserSQL, cr.RegNo, cr.PasswordHash).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the class representative user row: %w", err)
	}
	_, err = tx.Exec(ctx, insertCRStudentSQL,
		userID,
		cr.FullName,
		cr.RegNo,
		cr.Batch,
		cr.Section,
		cr.Email,
		cr.Phone,
		cr.BloodGroup,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the class representative student row: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: could not commit the class representative: %w", err)
	}
	return userID, nil
}

func (s *Store) ListCRs(ctx context.Context, filter CRFilter) ([]StudentRow, error) {
	rows, err := s.pool.Query(ctx, listCRsSQL, filter.Search, filter.Batch)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class representatives: %w", err)
	}
	defer rows.Close()
	crs := make([]StudentRow, 0)
	for rows.Next() {
		var cr StudentRow
		err := rows.Scan(
			&cr.ID,
			&cr.FullName,
			&cr.RegNo,
			&cr.Batch,
			&cr.Section,
			&cr.Email,
			&cr.Phone,
			&cr.BloodGroup,
			&cr.IsCR,
			&cr.Status,
			&cr.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a class representative: %w", err)
		}
		crs = append(crs, cr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class representatives: %w", err)
	}
	return crs, nil
}

func (s *Store) DemoteCR(ctx context.Context, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the demotion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var studentID int64
	err = tx.QueryRow(ctx, demoteCRSQL, userID).Scan(&studentID)
	if errors.Is(err, pgx.ErrNoRows) {
		var status string
		lookupErr := tx.QueryRow(ctx, studentStatusSQL, userID).Scan(&status)
		if errors.Is(lookupErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if lookupErr != nil {
			return fmt.Errorf("store: could not load the account %d: %w", userID, lookupErr)
		}
		return ErrNotCR
	}
	if err != nil {
		return fmt.Errorf("store: could not demote the class representative %d: %w", userID, err)
	}
	var tokenVersion int
	if err := tx.QueryRow(ctx, incrementTokenVersionSQL, userID).Scan(&tokenVersion); err != nil {
		return fmt.Errorf("store: could not invalidate the tokens of the account %d: %w", userID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the demotion: %w", err)
	}
	return nil
}

func (s *Store) DeactivateCR(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deactivateCRSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.removalConflict(ctx, crStatusSQL, userID)
	}
	if err != nil {
		return fmt.Errorf("store: could not deactivate the class representative %d: %w", userID, err)
	}
	return nil
}

func (s *Store) CreateTeacher(ctx context.Context, teacher NewTeacher) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: could not start the teacher transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID int64
	if err := tx.QueryRow(ctx, insertTeacherUserSQL, teacher.TeacherID, teacher.PasswordHash).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the teacher user row: %w", err)
	}
	_, err = tx.Exec(ctx, insertTeacherSQL, userID, teacher.TeacherID, teacher.Name, teacher.Email, teacher.Phone)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the teacher row: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: could not commit the teacher: %w", err)
	}
	return userID, nil
}

func (s *Store) ListTeachers(ctx context.Context, filter TeacherFilter) ([]TeacherRow, error) {
	rows, err := s.pool.Query(ctx, listTeachersSQL, filter.Search, filter.Status)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the teachers: %w", err)
	}
	defer rows.Close()
	teachers := make([]TeacherRow, 0)
	for rows.Next() {
		var teacher TeacherRow
		err := rows.Scan(
			&teacher.ID,
			&teacher.TeacherID,
			&teacher.Name,
			&teacher.Email,
			&teacher.Phone,
			&teacher.Status,
			&teacher.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a teacher: %w", err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the teachers: %w", err)
	}
	return teachers, nil
}

func (s *Store) DeactivateTeacher(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deactivateTeacherSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.removalConflict(ctx, teacherStatusSQL, userID)
	}
	if err != nil {
		return fmt.Errorf("store: could not deactivate the teacher %d: %w", userID, err)
	}
	return nil
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	err := s.pool.QueryRow(ctx, statsSQL).Scan(
		&stats.TotalStudents,
		&stats.TotalCRs,
		&stats.TotalTeachers,
		&stats.PendingApprovals,
	)
	if err != nil {
		return Stats{}, fmt.Errorf("store: could not load the overview stats: %w", err)
	}
	return stats, nil
}
