package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

const (
	ResetStageCode     = "code"
	ResetStageVerified = "verified"
)

var (
	ErrDuplicateAccount    = errors.New("store: an account with this registration number or email already exists")
	ErrDuplicateIdentifier = errors.New("store: this user id is already taken")
	ErrInvalidResetToken   = errors.New("store: the reset token is missing, wrong or expired")
)

type Registration struct {
	FullName        string
	RegNo           string
	Batch           string
	Section         string
	Email           string
	Phone           string
	BloodGroup      string
	DateOfBirth     string
	PasswordHash    string
	PhotoPath       string
	Status          string
	DropCourseIDs   []int64
	EmailVerifiedAt *time.Time
	CodeHash        string
	CodeExpiresAt   *time.Time
	CodeSentAt      *time.Time
}

type RegistrationResult struct {
	UserID         int64
	ReplacedPhotos []string
}

type LoginUser struct {
	AuthUser
	PasswordHash    string
	Name            string
	Batch           string
	Section         string
	RejectionReason string
}

type ResetTarget struct {
	AuthUser
	Email       string
	Phone       string
	DateOfBirth string
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
	findConflictingAccountsSQL = `SELECT u.id, u.status, COALESCE(u.photo_path, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.identifier = $1 OR lower(s.email) = lower($2)`
	deleteUserSQL = `DELETE FROM users WHERE id = $1`
	insertUserSQL = `INSERT INTO users
(role, identifier, password_hash, status, photo_path, email_verified_at, email_code, email_code_expires_at, email_code_sent_at)
VALUES ('student', $1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, $8)
RETURNING id`
	insertStudentSQL = `INSERT INTO students
(user_id, full_name, reg_no, batch, section, email, phone, blood_group, date_of_birth, is_cr)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::date, FALSE)
RETURNING id`
	identifierMatch = `WHERE (lower(u.identifier) = lower($1)
   OR (u.role = 'teacher' AND lower(t.email) = lower($1)))
ORDER BY (lower(u.identifier) = lower($1)) DESC NULLS LAST, u.id
LIMIT 1`
	findLoginUserSQL = `SELECT u.id, COALESCE(u.identifier, ''), u.role, u.status, u.token_version, COALESCE(u.photo_path, ''),
u.password_hash,
COALESCE(s.is_cr, FALSE),
COALESCE(s.full_name, t.name, ''),
COALESCE(s.batch, ''),
COALESCE(s.section, ''),
COALESCE(u.rejection_reason, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
` + identifierMatch
	incrementTokenVersionSQL = `UPDATE users
SET token_version = token_version + 1, updated_at = now()
WHERE id = $1
RETURNING token_version`
	findResetTargetSQL = `SELECT u.id, COALESCE(u.identifier, ''), u.role, u.status, u.token_version,
COALESCE(s.is_cr, FALSE),
COALESCE(s.email, t.email, u.contact_email, ''),
COALESCE(s.phone, t.phone, u.contact_phone, ''),
COALESCE(to_char(s.date_of_birth, 'YYYY-MM-DD'), to_char(t.date_of_birth, 'YYYY-MM-DD'), ''),
u.reset_sent_at
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
` + identifierMatch
	findResetChallengeSQL = `SELECT u.id, u.status, COALESCE(u.reset_token, ''), COALESCE(u.reset_stage, ''),
u.reset_attempts, u.reset_token_expires_at
FROM users u
LEFT JOIN teachers t ON t.user_id = u.id
` + identifierMatch
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
WHERE id = (SELECT u.id FROM users u LEFT JOIN teachers t ON t.user_id = u.id ` + identifierMatch + `)
  AND status = 'active'
  AND reset_token = $2
  AND reset_stage = 'verified'
  AND reset_token_expires_at > now()
RETURNING id`
)

func (s *Store) CreateRegistration(ctx context.Context, registration Registration) (RegistrationResult, error) {
	var result RegistrationResult
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("store: could not start the registration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	replaceable, photos, err := conflictingAccounts(ctx, tx, registration.RegNo, registration.Email)
	if err != nil {
		return result, err
	}
	if !replaceable {
		return result, ErrDuplicateAccount
	}
	result.ReplacedPhotos = photos
	var userID int64
	err = tx.QueryRow(ctx, insertUserSQL,
		registration.RegNo,
		registration.PasswordHash,
		registration.Status,
		registration.PhotoPath,
		registration.EmailVerifiedAt,
		registration.CodeHash,
		registration.CodeExpiresAt,
		registration.CodeSentAt,
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return RegistrationResult{}, ErrDuplicateAccount
		}
		return RegistrationResult{}, fmt.Errorf("store: could not create the user row: %w", err)
	}
	var studentID int64
	err = tx.QueryRow(ctx, insertStudentSQL,
		userID,
		registration.FullName,
		registration.RegNo,
		registration.Batch,
		registration.Section,
		registration.Email,
		registration.Phone,
		registration.BloodGroup,
		registration.DateOfBirth,
	).Scan(&studentID)
	if err != nil {
		if isUniqueViolation(err) {
			return RegistrationResult{}, ErrDuplicateAccount
		}
		return RegistrationResult{}, fmt.Errorf("store: could not create the student row: %w", err)
	}
	if err := replaceDropCourses(ctx, tx, studentID, registration.DropCourseIDs); err != nil {
		return RegistrationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RegistrationResult{}, fmt.Errorf("store: could not commit the registration: %w", err)
	}
	result.UserID = userID
	return result, nil
}

func conflictingAccounts(ctx context.Context, tx pgx.Tx, regNo, email string) (bool, []string, error) {
	rows, err := tx.Query(ctx, findConflictingAccountsSQL, regNo, email)
	if err != nil {
		return false, nil, fmt.Errorf("store: could not look for an existing account: %w", err)
	}
	type candidate struct {
		id     int64
		status string
		photo  string
	}
	candidates := make([]candidate, 0, 2)
	for rows.Next() {
		var found candidate
		if err := rows.Scan(&found.id, &found.status, &found.photo); err != nil {
			rows.Close()
			return false, nil, fmt.Errorf("store: could not scan an existing account: %w", err)
		}
		candidates = append(candidates, found)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("store: could not read the existing accounts: %w", err)
	}
	photos := make([]string, 0, len(candidates))
	for _, found := range candidates {
		if found.status != StatusUnverified {
			return false, nil, nil
		}
		if _, err := tx.Exec(ctx, deleteUserSQL, found.id); err != nil {
			return false, nil, fmt.Errorf("store: could not replace the unverified account %d: %w", found.id, err)
		}
		if found.photo != "" {
			photos = append(photos, found.photo)
		}
	}
	return true, photos, nil
}

func (s *Store) FindLoginUser(ctx context.Context, identifier string) (LoginUser, error) {
	var user LoginUser
	err := s.pool.QueryRow(ctx, findLoginUserSQL, identifier).Scan(
		&user.ID,
		&user.Identifier,
		&user.Role,
		&user.Status,
		&user.TokenVersion,
		&user.PhotoPath,
		&user.PasswordHash,
		&user.IsCR,
		&user.Name,
		&user.Batch,
		&user.Section,
		&user.RejectionReason,
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
		&target.DateOfBirth,
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
	return hasPgCode(err, uniqueViolationCode)
}

func isForeignKeyViolation(err error) bool {
	return hasPgCode(err, foreignKeyViolationCode)
}

func hasPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
