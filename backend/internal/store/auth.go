package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

const uniqueViolationCode = "23505"

var (
	ErrDuplicateAccount  = errors.New("store: an account with this registration number or email already exists")
	ErrInvalidResetToken = errors.New("store: the reset token is missing, wrong or expired")
)

type Registration struct {
	FullName       string
	RegNo          string
	Batch          string
	Section        string
	Email          string
	Phone          string
	BloodGroup     string
	PasswordHash   string
	HasDropCourse  bool
	DropCourseCode string
}

type LoginUser struct {
	AuthUser
	PasswordHash string
	Name         string
	Batch        string
	Section      string
}

const (
	insertUserSQL = `INSERT INTO users (role, identifier, password_hash, status)
VALUES ('student', $1, $2, 'pending')
RETURNING id`
	insertStudentSQL = `INSERT INTO students
(user_id, full_name, reg_no, batch, section, email, phone, blood_group, has_drop_course, drop_course_code, is_cr)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, FALSE)`
	findLoginUserSQL = `SELECT u.id, u.identifier, u.role, u.status, u.token_version, u.password_hash,
COALESCE(s.is_cr, FALSE),
COALESCE(s.full_name, t.name, ''),
COALESCE(s.batch, ''),
COALESCE(s.section, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.identifier = $1`
	incrementTokenVersionSQL = `UPDATE users
SET token_version = token_version + 1, updated_at = now()
WHERE id = $1
RETURNING token_version`
	setResetTokenSQL = `UPDATE users
SET reset_token = $2, reset_token_expires_at = $3, updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING id`
	resetPasswordSQL = `UPDATE users
SET password_hash = $3,
    reset_token = NULL,
    reset_token_expires_at = NULL,
    token_version = token_version + 1,
    updated_at = now()
WHERE identifier = $1
  AND status = 'active'
  AND reset_token = $2
  AND reset_token_expires_at > now()
RETURNING id`
)

func (s *Store) CreateRegistration(ctx context.Context, registration Registration) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: could not start the registration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID int64
	if err := tx.QueryRow(ctx, insertUserSQL, registration.RegNo, registration.PasswordHash).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the user row: %w", err)
	}
	var dropCourseCode *string
	if registration.HasDropCourse && registration.DropCourseCode != "" {
		code := registration.DropCourseCode
		dropCourseCode = &code
	}
	_, err = tx.Exec(ctx, insertStudentSQL,
		userID,
		registration.FullName,
		registration.RegNo,
		registration.Batch,
		registration.Section,
		registration.Email,
		registration.Phone,
		registration.BloodGroup,
		registration.HasDropCourse,
		dropCourseCode,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not create the student row: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: could not commit the registration: %w", err)
	}
	return userID, nil
}

func (s *Store) FindLoginUser(ctx context.Context, identifier string) (LoginUser, error) {
	var user LoginUser
	err := s.pool.QueryRow(ctx, findLoginUserSQL, identifier).Scan(
		&user.ID,
		&user.Identifier,
		&user.Role,
		&user.Status,
		&user.TokenVersion,
		&user.PasswordHash,
		&user.IsCR,
		&user.Name,
		&user.Batch,
		&user.Section,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginUser{}, ErrNotFound
	}
	if err != nil {
		return LoginUser{}, fmt.Errorf("store: could not load the login user: %w", err)
	}
	return user, nil
}

func (s *Store) IncrementTokenVersion(ctx context.Context, userID int64) (int, error) {
	var tokenVersion int
	err := s.pool.QueryRow(ctx, incrementTokenVersionSQL, userID).Scan(&tokenVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not increment the token version: %w", err)
	}
	return tokenVersion, nil
}

func (s *Store) SetResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, setResetTokenSQL, userID, tokenHash, expiresAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not store the reset token: %w", err)
	}
	return nil
}

func (s *Store) ResetPassword(ctx context.Context, identifier, tokenHash, passwordHash string) error {
	var id int64
	err := s.pool.QueryRow(ctx, resetPasswordSQL, identifier, tokenHash, passwordHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return fmt.Errorf("store: could not reset the password: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
