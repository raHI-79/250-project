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

const (
	ResetStageCode     = "code"
	ResetStageVerified = "verified"
)

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

type ResetTarget struct {
	AuthUser
	Email       string
	Phone       string
	ResetSentAt *time.Time
}

type ResetChallenge struct {
	UserID    int64
	Status    string
	TokenHash string
	Stage     string
	Attempts  int
	ExpiresAt *time.Time
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
	findResetTargetSQL = `SELECT u.id, u.identifier, u.role, u.status, u.token_version,
COALESCE(s.is_cr, FALSE),
COALESCE(s.email, t.email, u.contact_email, ''),
COALESCE(s.phone, t.phone, u.contact_phone, ''),
u.reset_sent_at
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.identifier = $1`
	findResetChallengeSQL = `SELECT id, status, COALESCE(reset_token, ''), COALESCE(reset_stage, ''), reset_attempts, reset_token_expires_at
FROM users
WHERE identifier = $1`
	startPasswordResetSQL = `UPDATE users
SET reset_token = $2,
    reset_token_expires_at = $3,
    reset_stage = 'code',
    reset_attempts = 0,
    reset_sent_at = $4,
    updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING id`
	countResetAttemptSQL = `UPDATE users
SET reset_attempts = reset_attempts + 1, updated_at = now()
WHERE id = $1
RETURNING reset_attempts`
	clearPasswordResetSQL = `UPDATE users
SET reset_token = NULL,
    reset_token_expires_at = NULL,
    reset_stage = NULL,
    reset_attempts = 0,
    reset_sent_at = NULL,
    updated_at = now()
WHERE id = $1
RETURNING id`
	verifyResetCodeSQL = `UPDATE users
SET reset_token = $2,
    reset_token_expires_at = $3,
    reset_stage = 'verified',
    reset_attempts = 0,
    updated_at = now()
WHERE id = $1 AND status = 'active' AND reset_stage = 'code'
RETURNING id`
	finishPasswordResetSQL = `UPDATE users
SET password_hash = $3,
    reset_token = NULL,
    reset_token_expires_at = NULL,
    reset_stage = NULL,
    reset_attempts = 0,
    reset_sent_at = NULL,
    token_version = token_version + 1,
    updated_at = now()
WHERE identifier = $1
  AND status = 'active'
  AND reset_token = $2
  AND reset_stage = 'verified'
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

func (s *Store) FindResetTarget(ctx context.Context, identifier string) (ResetTarget, error) {
	var target ResetTarget
	err := s.pool.QueryRow(ctx, findResetTargetSQL, identifier).Scan(
		&target.ID,
		&target.Identifier,
		&target.Role,
		&target.Status,
		&target.TokenVersion,
		&target.IsCR,
		&target.Email,
		&target.Phone,
		&target.ResetSentAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResetTarget{}, ErrNotFound
	}
	if err != nil {
		return ResetTarget{}, fmt.Errorf("store: could not load the password reset target: %w", err)
	}
	return target, nil
}

func (s *Store) FindResetChallenge(ctx context.Context, identifier string) (ResetChallenge, error) {
	var challenge ResetChallenge
	err := s.pool.QueryRow(ctx, findResetChallengeSQL, identifier).Scan(
		&challenge.UserID,
		&challenge.Status,
		&challenge.TokenHash,
		&challenge.Stage,
		&challenge.Attempts,
		&challenge.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResetChallenge{}, ErrNotFound
	}
	if err != nil {
		return ResetChallenge{}, fmt.Errorf("store: could not load the password reset challenge: %w", err)
	}
	return challenge, nil
}

func (s *Store) StartPasswordReset(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, startPasswordResetSQL, userID, codeHash, expiresAt, sentAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not store the password reset code: %w", err)
	}
	return nil
}

func (s *Store) CountResetAttempt(ctx context.Context, userID int64) (int, error) {
	var attempts int
	err := s.pool.QueryRow(ctx, countResetAttemptSQL, userID).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not count the password reset attempt: %w", err)
	}
	return attempts, nil
}

func (s *Store) ClearPasswordReset(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, clearPasswordResetSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not clear the password reset fields: %w", err)
	}
	return nil
}

func (s *Store) VerifyResetCode(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, verifyResetCodeSQL, userID, tokenHash, expiresAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return fmt.Errorf("store: could not store the verified password reset token: %w", err)
	}
	return nil
}

func (s *Store) FinishPasswordReset(ctx context.Context, identifier, tokenHash, passwordHash string) error {
	var id int64
	err := s.pool.QueryRow(ctx, finishPasswordResetSQL, identifier, tokenHash, passwordHash).Scan(&id)
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
