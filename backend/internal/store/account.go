package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const DropCourseOptionLimit = 20

var (
	ErrAlreadyCR   = errors.New("store: the student is already a class representative")
	ErrNotInvited  = errors.New("store: the account is not waiting for an invitation")
	ErrUnknownRole = errors.New("store: the account role cannot be updated this way")
)

type EmailChallenge struct {
	UserID    int64
	Status    string
	Email     string
	CodeHash  string
	Attempts  int
	ExpiresAt *time.Time
	SentAt    *time.Time
}

type PromotedCR struct {
	UserID   int64
	FullName string
	Batch    string
	Section  string
}

type TeacherInvite struct {
	UserID      int64
	TeacherID   string
	Name        string
	Designation string
	Email       string
	DateOfBirth string
	PhotoPath   string
	Attempts    int
}

type Profile struct {
	UserID       int64
	Role         string
	Status       string
	Identifier   string
	PhotoPath    string
	CreatedAt    time.Time
	FullName     string
	RegNo        string
	Batch        string
	Section      string
	Email        string
	Phone        string
	BloodGroup   string
	DateOfBirth  string
	IsCR         bool
	DropCourses  []Course
	TeacherID    string
	Designation  string
	OfficeRoom   string
	JoiningDate  string
	Bio          string
	ContactEmail string
	ContactPhone string
}

type ProfileUpdate struct {
	Phone         *string
	PhotoPath     *string
	OfficeRoom    *string
	Bio           *string
	ContactEmail  *string
	ContactPhone  *string
	DropCourseIDs *[]int64
}

type PersonCard struct {
	UserID      int64
	Role        string
	Name        string
	Batch       string
	Section     string
	Designation string
	OfficeRoom  string
	Email       string
	Phone       string
	BloodGroup  string
	PhotoPath   string
}

const (
	deleteDropCoursesSQL = `DELETE FROM student_drop_courses WHERE student_id = $1`
	insertDropCourseSQL  = `INSERT INTO student_drop_courses (student_id, course_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`
	searchDropCourseOptionsSQL = `SELECT c.id, c.code, c.title
FROM courses c
WHERE EXISTS (SELECT 1 FROM routines r WHERE r.course_id = c.id)
  AND ($1::text = '' OR c.code ILIKE '%' || $1::text || '%' OR c.title ILIKE '%' || $1::text || '%')
ORDER BY c.code
LIMIT $2`
	filterDropCourseIDsSQL = `SELECT c.id
FROM courses c
WHERE c.id = ANY($1::bigint[])
  AND EXISTS (SELECT 1 FROM routines r WHERE r.course_id = c.id)`
	listStudentDropCoursesSQL = `SELECT c.id, c.code, c.title
FROM student_drop_courses d
JOIN courses c ON c.id = d.course_id
WHERE d.student_id = $1
ORDER BY c.code`
	findEmailChallengeSQL = `SELECT u.id, u.status, s.email, COALESCE(u.email_code, ''), u.email_code_attempts,
u.email_code_expires_at, u.email_code_sent_at
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.identifier = $1`
	startEmailVerificationSQL = `UPDATE users
SET email_code = $2,
    email_code_expires_at = $3,
    email_code_attempts = 0,
    email_code_sent_at = $4,
    updated_at = now()
WHERE id = $1 AND status = 'unverified'
RETURNING id`
	countEmailAttemptSQL = `UPDATE users
SET email_code_attempts = email_code_attempts + 1, updated_at = now()
WHERE id = $1
RETURNING email_code_attempts`
	clearEmailCodeSQL = `UPDATE users
SET email_code = NULL,
    email_code_expires_at = NULL,
    email_code_attempts = 0,
    updated_at = now()
WHERE id = $1
RETURNING id`
	confirmEmailSQL = `UPDATE users
SET status = 'pending',
    email_verified_at = $2,
    email_code = NULL,
    email_code_expires_at = NULL,
    email_code_attempts = 0,
    email_code_sent_at = NULL,
    updated_at = now()
WHERE id = $1 AND status = 'unverified'
RETURNING id`
	promoteCRSQL = `UPDATE students
SET is_cr = TRUE, updated_at = now()
WHERE user_id = $1 AND NOT is_cr
  AND EXISTS (SELECT 1 FROM users u WHERE u.id = students.user_id AND u.role = 'student' AND u.status = 'active')
RETURNING full_name, batch, section`
	findActiveStudentCRSQL = `SELECT s.is_cr
FROM students s
JOIN users u ON u.id = s.user_id
WHERE s.user_id = $1 AND u.role = 'student' AND u.status = 'active'`
	getPasswordHashSQL = `SELECT password_hash FROM users WHERE id = $1 AND status = 'active'`
	setPasswordSQL     = `UPDATE users
SET password_hash = $2, token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING token_version`
	getProfileSQL = `SELECT u.id, u.role, u.status, COALESCE(u.identifier, ''), COALESCE(u.photo_path, ''), u.created_at,
COALESCE(s.id, 0),
COALESCE(s.full_name, ''), COALESCE(s.reg_no, ''), COALESCE(s.batch, ''), COALESCE(s.section, ''),
COALESCE(s.email, ''), COALESCE(s.phone, ''), COALESCE(s.blood_group, ''),
COALESCE(to_char(s.date_of_birth, 'YYYY-MM-DD'), ''), COALESCE(s.is_cr, FALSE),
COALESCE(t.teacher_id, ''), COALESCE(t.name, ''), COALESCE(t.designation, ''), COALESCE(t.email, ''),
COALESCE(t.phone, ''), COALESCE(t.blood_group, ''), COALESCE(to_char(t.date_of_birth, 'YYYY-MM-DD'), ''),
COALESCE(t.office_room, ''), COALESCE(to_char(t.joining_date, 'YYYY-MM-DD'), ''), COALESCE(t.bio, ''),
COALESCE(u.contact_email, ''), COALESCE(u.contact_phone, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1`
	updateStudentContactSQL = `UPDATE students
SET phone = COALESCE($2, phone), updated_at = now()
WHERE user_id = $1
RETURNING id`
	updateTeacherContactSQL = `UPDATE teachers
SET phone = COALESCE($2, phone),
    office_room = COALESCE($3, office_room),
    bio = COALESCE($4, bio),
    updated_at = now()
WHERE user_id = $1
RETURNING id`
	updateAdminContactSQL = `UPDATE users
SET contact_email = COALESCE($2, contact_email),
    contact_phone = COALESCE($3, contact_phone),
    updated_at = now()
WHERE id = $1 AND role = 'admin'
RETURNING id`
	updatePhotoSQL = `UPDATE users
SET photo_path = NULLIF($2, ''), updated_at = now()
WHERE id = $1`
	currentPhotoSQL = `SELECT COALESCE(photo_path, '') FROM users WHERE id = $1`
	findPersonSQL   = `SELECT u.id, u.role, COALESCE(s.is_cr, FALSE),
COALESCE(s.full_name, t.name, ''),
COALESCE(s.batch, ''), COALESCE(s.section, ''),
COALESCE(t.designation, ''), COALESCE(t.office_room, ''),
COALESCE(s.email, t.email, ''), COALESCE(s.phone, t.phone, ''),
COALESCE(s.blood_group, t.blood_group, ''), COALESCE(u.photo_path, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'active' AND u.role IN ('student', 'teacher')`
)

func replaceDropCourses(ctx context.Context, tx pgx.Tx, studentID int64, courseIDs []int64) error {
	if _, err := tx.Exec(ctx, deleteDropCoursesSQL, studentID); err != nil {
		return fmt.Errorf("store: could not clear the drop courses of the student %d: %w", studentID, err)
	}
	for _, courseID := range courseIDs {
		if _, err := tx.Exec(ctx, insertDropCourseSQL, studentID, courseID); err != nil {
			return fmt.Errorf("store: could not add the drop course %d: %w", courseID, err)
		}
	}
	return nil
}

func (s *Store) SearchDropCourseOptions(ctx context.Context, search string) ([]Course, error) {
	rows, err := s.pool.Query(ctx, searchDropCourseOptionsSQL, search, DropCourseOptionLimit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the drop course options: %w", err)
	}
	defer rows.Close()
	courses := make([]Course, 0)
	for rows.Next() {
		var course Course
		if err := rows.Scan(&course.ID, &course.Code, &course.Title); err != nil {
			return nil, fmt.Errorf("store: could not scan a drop course option: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drop course options: %w", err)
	}
	return courses, nil
}

func (s *Store) FilterDropCourseIDs(ctx context.Context, courseIDs []int64) ([]int64, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, filterDropCourseIDsSQL, courseIDs)
	if err != nil {
		return nil, fmt.Errorf("store: could not check the drop course choices: %w", err)
	}
	defer rows.Close()
	allowed := make([]int64, 0, len(courseIDs))
	for rows.Next() {
		var courseID int64
		if err := rows.Scan(&courseID); err != nil {
			return nil, fmt.Errorf("store: could not scan a drop course choice: %w", err)
		}
		allowed = append(allowed, courseID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drop course choices: %w", err)
	}
	return allowed, nil
}

func (s *Store) ListStudentDropCourses(ctx context.Context, studentID int64) ([]Course, error) {
	rows, err := s.pool.Query(ctx, listStudentDropCoursesSQL, studentID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the drop courses of the student %d: %w", studentID, err)
	}
	defer rows.Close()
	courses := make([]Course, 0)
	for rows.Next() {
		var course Course
		if err := rows.Scan(&course.ID, &course.Code, &course.Title); err != nil {
			return nil, fmt.Errorf("store: could not scan a drop course: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drop courses: %w", err)
	}
	return courses, nil
}

func (s *Store) FindEmailChallenge(ctx context.Context, regNo string) (EmailChallenge, error) {
	var challenge EmailChallenge
	err := s.pool.QueryRow(ctx, findEmailChallengeSQL, regNo).Scan(
		&challenge.UserID,
		&challenge.Status,
		&challenge.Email,
		&challenge.CodeHash,
		&challenge.Attempts,
		&challenge.ExpiresAt,
		&challenge.SentAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailChallenge{}, ErrNotFound
	}
	if err != nil {
		return EmailChallenge{}, fmt.Errorf("store: could not load the email verification challenge: %w", err)
	}
	return challenge, nil
}

func (s *Store) StartEmailVerification(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, startEmailVerificationSQL, userID, codeHash, expiresAt, sentAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not store the email verification code: %w", err)
	}
	return nil
}

func (s *Store) CountEmailAttempt(ctx context.Context, userID int64) (int, error) {
	var attempts int
	err := s.pool.QueryRow(ctx, countEmailAttemptSQL, userID).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not count the email verification attempt: %w", err)
	}
	return attempts, nil
}

func (s *Store) ClearEmailCode(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, clearEmailCodeSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not clear the email verification code: %w", err)
	}
	return nil
}

func (s *Store) ConfirmEmail(ctx context.Context, userID int64, verifiedAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, confirmEmailSQL, userID, verifiedAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not confirm the email address: %w", err)
	}
	return nil
}

func (s *Store) PromoteCR(ctx context.Context, userID int64) (PromotedCR, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PromotedCR{}, fmt.Errorf("store: could not start the promotion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	promoted := PromotedCR{UserID: userID}
	err = tx.QueryRow(ctx, promoteCRSQL, userID).Scan(&promoted.FullName, &promoted.Batch, &promoted.Section)
	if errors.Is(err, pgx.ErrNoRows) {
		var isCR bool
		lookupErr := tx.QueryRow(ctx, findActiveStudentCRSQL, userID).Scan(&isCR)
		if errors.Is(lookupErr, pgx.ErrNoRows) {
			return PromotedCR{}, ErrNotFound
		}
		if lookupErr != nil {
			return PromotedCR{}, fmt.Errorf("store: could not load the student %d: %w", userID, lookupErr)
		}
		return PromotedCR{}, ErrAlreadyCR
	}
	if err != nil {
		return PromotedCR{}, fmt.Errorf("store: could not promote the student %d: %w", userID, err)
	}
	var tokenVersion int
	if err := tx.QueryRow(ctx, incrementTokenVersionSQL, userID).Scan(&tokenVersion); err != nil {
		return PromotedCR{}, fmt.Errorf("store: could not invalidate the tokens of the account %d: %w", userID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PromotedCR{}, fmt.Errorf("store: could not commit the promotion: %w", err)
	}
	return promoted, nil
}

func (s *Store) GetPasswordHash(ctx context.Context, userID int64) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, getPasswordHashSQL, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not load the password of the user %d: %w", userID, err)
	}
	return hash, nil
}

func (s *Store) SetPassword(ctx context.Context, userID int64, passwordHash string) (int, error) {
	var tokenVersion int
	err := s.pool.QueryRow(ctx, setPasswordSQL, userID, passwordHash).Scan(&tokenVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not change the password of the user %d: %w", userID, err)
	}
	return tokenVersion, nil
}

func (s *Store) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	var profile Profile
	var studentID int64
	var teacherName, teacherEmail, teacherPhone, teacherBlood, teacherDateOfBirth string
	err := s.pool.QueryRow(ctx, getProfileSQL, userID).Scan(
		&profile.UserID,
		&profile.Role,
		&profile.Status,
		&profile.Identifier,
		&profile.PhotoPath,
		&profile.CreatedAt,
		&studentID,
		&profile.FullName,
		&profile.RegNo,
		&profile.Batch,
		&profile.Section,
		&profile.Email,
		&profile.Phone,
		&profile.BloodGroup,
		&profile.DateOfBirth,
		&profile.IsCR,
		&profile.TeacherID,
		&teacherName,
		&profile.Designation,
		&teacherEmail,
		&teacherPhone,
		&teacherBlood,
		&teacherDateOfBirth,
		&profile.OfficeRoom,
		&profile.JoiningDate,
		&profile.Bio,
		&profile.ContactEmail,
		&profile.ContactPhone,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("store: could not load the profile of the user %d: %w", userID, err)
	}
	if profile.Role == "teacher" {
		profile.FullName = teacherName
		profile.Email = teacherEmail
		profile.Phone = teacherPhone
		profile.BloodGroup = teacherBlood
		profile.DateOfBirth = teacherDateOfBirth
	}
	profile.DropCourses = make([]Course, 0)
	if studentID > 0 {
		courses, err := s.ListStudentDropCourses(ctx, studentID)
		if err != nil {
			return Profile{}, err
		}
		profile.DropCourses = courses
	}
	return profile, nil
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, role string, update ProfileUpdate) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("store: could not start the profile transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	oldPhoto := ""
	if update.PhotoPath != nil {
		err = tx.QueryRow(ctx, currentPhotoSQL, userID).Scan(&oldPhoto)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("store: could not read the photo of the user %d: %w", userID, err)
		}
		if _, err := tx.Exec(ctx, updatePhotoSQL, userID, *update.PhotoPath); err != nil {
			return "", fmt.Errorf("store: could not change the photo of the user %d: %w", userID, err)
		}
	}
	switch role {
	case "student", "cr":
		var studentID int64
		err = tx.QueryRow(ctx, updateStudentContactSQL, userID, update.Phone).Scan(&studentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("store: could not update the student %d: %w", userID, err)
		}
		if update.DropCourseIDs != nil {
			if err := replaceDropCourses(ctx, tx, studentID, *update.DropCourseIDs); err != nil {
				return "", err
			}
		}
	case "teacher":
		var teacherID int64
		err = tx.QueryRow(ctx, updateTeacherContactSQL, userID, update.Phone, update.OfficeRoom, update.Bio).Scan(&teacherID)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("store: could not update the teacher %d: %w", userID, err)
		}
	case "admin":
		var id int64
		err = tx.QueryRow(ctx, updateAdminContactSQL, userID, update.ContactEmail, update.ContactPhone).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("store: could not update the admin %d: %w", userID, err)
		}
	default:
		return "", ErrUnknownRole
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("store: could not commit the profile change: %w", err)
	}
	return oldPhoto, nil
}

func (s *Store) FindPerson(ctx context.Context, userID int64) (PersonCard, error) {
	var card PersonCard
	var role string
	var isCR bool
	err := s.pool.QueryRow(ctx, findPersonSQL, userID).Scan(
		&card.UserID,
		&role,
		&isCR,
		&card.Name,
		&card.Batch,
		&card.Section,
		&card.Designation,
		&card.OfficeRoom,
		&card.Email,
		&card.Phone,
		&card.BloodGroup,
		&card.PhotoPath,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PersonCard{}, ErrNotFound
	}
	if err != nil {
		return PersonCard{}, fmt.Errorf("store: could not load the person %d: %w", userID, err)
	}
	card.Role = role
	if role == "student" && isCR {
		card.Role = "cr"
	}
	return card, nil
}
