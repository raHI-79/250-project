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
	ID          int64
	StudentID   int64
	FullName    string
	RegNo       string
	Batch       string
	Section     string
	Email       string
	Phone       string
	BloodGroup  string
	DateOfBirth string
	PhotoPath   string
	DropCourses []Course
	CreatedAt   time.Time
}

type StudentRow struct {
	ID          int64
	StudentID   int64
	FullName    string
	RegNo       string
	Batch       string
	Section     string
	Email       string
	Phone       string
	BloodGroup  string
	DateOfBirth string
	PhotoPath   string
	IsCR        bool
	Status      string
	CreatedAt   time.Time
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
	ID          int64
	TeacherID   string
	Name        string
	Designation string
	Email       string
	Phone       string
	BloodGroup  string
	DateOfBirth string
	OfficeRoom  string
	JoiningDate string
	Bio         string
	PhotoPath   string
	Status      string
	CreatedAt   time.Time
}

type TeacherFilter struct {
	Search string
	Status string
}

type NewTeacher struct {
	Name            string
	TeacherID       string
	Designation     string
	Email           string
	Phone           string
	BloodGroup      string
	DateOfBirth     string
	OfficeRoom      string
	JoiningDate     string
	PhotoPath       string
	InviteHash      string
	InviteExpiresAt time.Time
}

type TeacherUpdate struct {
	Name        *string
	TeacherID   *string
	Designation *string
	Email       *string
	DateOfBirth *string
	BloodGroup  *string
	JoiningDate *string
	OfficeRoom  *string
}

type Stats struct {
	TotalStudents    int
	TotalCRs         int
	TotalTeachers    int
	PendingApprovals int
}

const (
	listPendingRegistrationsSQL = `SELECT u.id, s.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, COALESCE(to_char(s.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(u.photo_path, ''), s.created_at
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.role = 'student' AND u.status = 'pending'
ORDER BY u.id`
	listDropCoursesForStudentsSQL = `SELECT d.student_id, c.id, c.code, c.title
FROM student_drop_courses d
JOIN courses c ON c.id = d.course_id
WHERE d.student_id = ANY($1::bigint[])
ORDER BY d.student_id, c.code`
	setRegistrationStatusSQL = `UPDATE users
SET status = $2, updated_at = now()
WHERE id = $1 AND role = 'student' AND status = 'pending'
RETURNING id`
	studentStatusSQL = `SELECT u.status
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	listStudentsSQL = `SELECT u.id, s.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, COALESCE(to_char(s.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(u.photo_path, ''),
s.is_cr, u.status, s.created_at,
COUNT(*) OVER () AS total
FROM users u
JOIN students s ON s.user_id = u.id
WHERE ($1::text = '' OR s.full_name ILIKE '%' || $1::text || '%' OR s.reg_no ILIKE '%' || $1::text || '%')
  AND ($2::text = '' OR s.batch = $2::text)
  AND ($3::text = '' OR s.section = $3::text)
  AND ($4::text = 'all' OR ($4::text = '' AND u.status NOT IN ('rejected', 'unverified')) OR u.status = $4::text)
ORDER BY s.batch, s.section, s.full_name, s.reg_no
LIMIT $5 OFFSET $6`
	removeStudentSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'student' AND status <> 'removed'
RETURNING id`
	listCRsSQL = `SELECT u.id, s.id, s.full_name, s.reg_no, s.batch, s.section, s.email, s.phone,
s.blood_group, COALESCE(to_char(s.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(u.photo_path, ''),
s.is_cr, u.status, s.created_at
FROM users u
JOIN students s ON s.user_id = u.id
WHERE s.is_cr
  AND u.status NOT IN ('rejected', 'unverified')
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
	insertTeacherUserSQL = `INSERT INTO users
(role, identifier, password_hash, status, photo_path, invite_token, invite_expires_at)
VALUES ('teacher', $1, '', 'invited', NULLIF($2, ''), $3, $4)
RETURNING id`
	insertTeacherSQL = `INSERT INTO teachers
(user_id, teacher_id, name, designation, email, phone, blood_group, date_of_birth, office_room, joining_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::date, $9, NULLIF($10, '')::date)`
	listTeachersSQL = `SELECT u.id, t.teacher_id, t.name, t.designation, t.email, t.phone,
COALESCE(t.blood_group, ''), COALESCE(to_char(t.date_of_birth, 'YYYY-MM-DD'), ''),
t.office_room, COALESCE(to_char(t.joining_date, 'YYYY-MM-DD'), ''), t.bio,
COALESCE(u.photo_path, ''), u.status, t.created_at
FROM users u
JOIN teachers t ON t.user_id = u.id
WHERE ($1::text = '' OR t.name ILIKE '%' || $1::text || '%' OR t.teacher_id ILIKE '%' || $1::text || '%')
  AND ($2::text = 'all' OR ($2::text = '' AND u.status <> 'rejected') OR u.status = $2::text)
ORDER BY t.name, t.teacher_id`
	deactivateTeacherSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'teacher' AND status <> 'removed'
RETURNING id`
	teacherStatusSQL     = `SELECT status FROM users WHERE id = $1 AND role = 'teacher'`
	updateTeacherUserSQL = `UPDATE users
SET identifier = COALESCE($2, identifier), updated_at = now()
WHERE id = $1 AND role = 'teacher'
RETURNING id`
	updateTeacherRecordSQL = `UPDATE teachers
SET name = COALESCE($2, name),
    teacher_id = COALESCE($3, teacher_id),
    designation = COALESCE($4, designation),
    email = COALESCE($5, email),
    date_of_birth = COALESCE($6::date, date_of_birth),
    blood_group = COALESCE($7, blood_group),
    joining_date = COALESCE($8::date, joining_date),
    office_room = COALESCE($9, office_room),
    updated_at = now()
WHERE user_id = $1
RETURNING id`
	resetTeacherInviteSQL = `UPDATE users
SET invite_token = $2, invite_expires_at = $3, invite_attempts = 0, updated_at = now()
WHERE id = $1 AND role = 'teacher' AND status = 'invited'
RETURNING id`
	findInviteSQL = `SELECT u.id, t.teacher_id, t.name, t.designation, t.email,
COALESCE(to_char(t.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(u.photo_path, ''), u.invite_attempts
FROM users u
JOIN teachers t ON t.user_id = u.id
WHERE u.invite_token = $1 AND u.status = 'invited' AND u.invite_expires_at > now()`
	findTeacherInviteByIDSQL = `SELECT u.id, t.teacher_id, t.name, t.designation, t.email,
COALESCE(to_char(t.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(u.photo_path, ''), u.invite_attempts
FROM users u
JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'invited'`
	countInviteAttemptSQL = `UPDATE users
SET invite_attempts = invite_attempts + 1, updated_at = now()
WHERE id = $1
RETURNING invite_attempts`
	clearInviteSQL = `UPDATE users
SET invite_token = NULL, invite_expires_at = NULL, updated_at = now()
WHERE id = $1
RETURNING id`
	activateTeacherSQL = `UPDATE users
SET status = 'active',
    password_hash = $2,
    photo_path = COALESCE(NULLIF($3, ''), photo_path),
    invite_token = NULL,
    invite_expires_at = NULL,
    invite_attempts = 0,
    email_verified_at = now(),
    token_version = token_version + 1,
    updated_at = now()
WHERE id = $1 AND role = 'teacher' AND status = 'invited'
RETURNING id`
	statsSQL = `SELECT
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
			&registration.StudentID,
			&registration.FullName,
			&registration.RegNo,
			&registration.Batch,
			&registration.Section,
			&registration.Email,
			&registration.Phone,
			&registration.BloodGroup,
			&registration.DateOfBirth,
			&registration.PhotoPath,
			&registration.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a pending registration: %w", err)
		}
		registration.DropCourses = make([]Course, 0)
		registrations = append(registrations, registration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the pending registrations: %w", err)
	}
	rows.Close()
	studentIDs := make([]int64, 0, len(registrations))
	for _, registration := range registrations {
		studentIDs = append(studentIDs, registration.StudentID)
	}
	dropCourses, err := s.dropCoursesByStudent(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	for index := range registrations {
		if courses, ok := dropCourses[registrations[index].StudentID]; ok {
			registrations[index].DropCourses = courses
		}
	}
	return registrations, nil
}

func (s *Store) dropCoursesByStudent(ctx context.Context, studentIDs []int64) (map[int64][]Course, error) {
	grouped := make(map[int64][]Course)
	if len(studentIDs) == 0 {
		return grouped, nil
	}
	rows, err := s.pool.Query(ctx, listDropCoursesForStudentsSQL, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the drop courses of the students: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var studentID int64
		var course Course
		if err := rows.Scan(&studentID, &course.ID, &course.Code, &course.Title); err != nil {
			return nil, fmt.Errorf("store: could not scan a drop course: %w", err)
		}
		grouped[studentID] = append(grouped[studentID], course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drop courses of the students: %w", err)
	}
	return grouped, nil
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
			&student.StudentID,
			&student.FullName,
			&student.RegNo,
			&student.Batch,
			&student.Section,
			&student.Email,
			&student.Phone,
			&student.BloodGroup,
			&student.DateOfBirth,
			&student.PhotoPath,
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
			&cr.StudentID,
			&cr.FullName,
			&cr.RegNo,
			&cr.Batch,
			&cr.Section,
			&cr.Email,
			&cr.Phone,
			&cr.BloodGroup,
			&cr.DateOfBirth,
			&cr.PhotoPath,
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
	err = tx.QueryRow(ctx, insertTeacherUserSQL,
		teacher.TeacherID,
		teacher.PhotoPath,
		teacher.InviteHash,
		teacher.InviteExpiresAt,
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the teacher user row: %w", err)
	}
	_, err = tx.Exec(ctx, insertTeacherSQL,
		userID,
		teacher.TeacherID,
		teacher.Name,
		teacher.Designation,
		teacher.Email,
		teacher.Phone,
		teacher.BloodGroup,
		teacher.DateOfBirth,
		teacher.OfficeRoom,
		teacher.JoiningDate,
	)
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
			&teacher.Designation,
			&teacher.Email,
			&teacher.Phone,
			&teacher.BloodGroup,
			&teacher.DateOfBirth,
			&teacher.OfficeRoom,
			&teacher.JoiningDate,
			&teacher.Bio,
			&teacher.PhotoPath,
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

func (s *Store) UpdateTeacher(ctx context.Context, userID int64, update TeacherUpdate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the teacher update transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, updateTeacherUserSQL, userID, update.TeacherID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateAccount
		}
		return fmt.Errorf("store: could not update the account of the teacher %d: %w", userID, err)
	}
	var teacherID int64
	err = tx.QueryRow(ctx, updateTeacherRecordSQL,
		userID,
		update.Name,
		update.TeacherID,
		update.Designation,
		update.Email,
		update.DateOfBirth,
		update.BloodGroup,
		update.JoiningDate,
		update.OfficeRoom,
	).Scan(&teacherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateAccount
		}
		return fmt.Errorf("store: could not update the teacher %d: %w", userID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the teacher update: %w", err)
	}
	return nil
}

func (s *Store) ResetTeacherInvite(ctx context.Context, userID int64, inviteHash string, expiresAt time.Time) (TeacherInvite, error) {
	var id int64
	err := s.pool.QueryRow(ctx, resetTeacherInviteSQL, userID, inviteHash, expiresAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return TeacherInvite{}, s.inviteConflict(ctx, userID)
	}
	if err != nil {
		return TeacherInvite{}, fmt.Errorf("store: could not refresh the invitation of the teacher %d: %w", userID, err)
	}
	return s.teacherInvite(ctx, findTeacherInviteByIDSQL, userID)
}

func (s *Store) inviteConflict(ctx context.Context, userID int64) error {
	var status string
	err := s.pool.QueryRow(ctx, teacherStatusSQL, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not load the teacher %d: %w", userID, err)
	}
	return ErrNotInvited
}

func (s *Store) FindInvite(ctx context.Context, inviteHash string) (TeacherInvite, error) {
	return s.teacherInvite(ctx, findInviteSQL, inviteHash)
}

func (s *Store) teacherInvite(ctx context.Context, query string, arg any) (TeacherInvite, error) {
	var invite TeacherInvite
	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&invite.UserID,
		&invite.TeacherID,
		&invite.Name,
		&invite.Designation,
		&invite.Email,
		&invite.DateOfBirth,
		&invite.PhotoPath,
		&invite.Attempts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TeacherInvite{}, ErrNotFound
	}
	if err != nil {
		return TeacherInvite{}, fmt.Errorf("store: could not load the teacher invitation: %w", err)
	}
	return invite, nil
}

func (s *Store) CountInviteAttempt(ctx context.Context, userID int64) (int, error) {
	var attempts int
	err := s.pool.QueryRow(ctx, countInviteAttemptSQL, userID).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not count the invitation attempt: %w", err)
	}
	return attempts, nil
}

func (s *Store) ClearInvite(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, clearInviteSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not clear the invitation of the teacher %d: %w", userID, err)
	}
	return nil
}

func (s *Store) ActivateTeacher(ctx context.Context, userID int64, passwordHash, photoPath string) error {
	var id int64
	err := s.pool.QueryRow(ctx, activateTeacherSQL, userID, passwordHash, photoPath).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not activate the teacher %d: %w", userID, err)
	}
	return nil
}
