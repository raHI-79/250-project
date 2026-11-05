package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const SessionTouchAfter = time.Minute

var ErrSessionGone = errors.New("store: the session is revoked or expired")

type NewSession struct {
	UserID    int64
	Device    string
	Client    string
	IPHash    string
	Country   string
	ExpiresAt time.Time
}

type Session struct {
	ID         int64
	UserID     int64
	Device     string
	Client     string
	IPHash     string
	Country    string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	SudoUntil  *time.Time
}

func (s Session) Live(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now)
}

type TwoStepChallenge struct {
	UserID    int64
	Role      string
	Email     string
	Enabled   bool
	CodeHash  string
	Attempts  int
	ExpiresAt *time.Time
	SentAt    *time.Time
}

const (
	createSessionSQL = `INSERT INTO user_sessions (user_id, device, client, ip_hash, country, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`
	loginSessionColumns = `id, user_id, device, client, ip_hash, country, created_at, last_seen_at,
expires_at, revoked_at, sudo_until`
	loadSessionSQL  = `SELECT ` + loginSessionColumns + ` FROM user_sessions WHERE id = $1`
	touchSessionSQL = `UPDATE user_sessions
SET last_seen_at = now(),
    ip_hash = CASE WHEN $2 = '' THEN ip_hash ELSE $2 END,
    country = CASE WHEN $3 = '' THEN country ELSE $3 END
WHERE id = $1 AND revoked_at IS NULL`
	listSessionsSQL = `SELECT ` + loginSessionColumns + ` FROM user_sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
ORDER BY last_seen_at DESC`
	revokeSessionSQL = `UPDATE user_sessions SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
RETURNING id`
	revokeOthersSQL = `UPDATE user_sessions SET revoked_at = now()
WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`
	revokeAllSQL = `UPDATE user_sessions SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL`
	grantSudoSQL = `UPDATE user_sessions SET sudo_until = $2
WHERE id = $1 AND revoked_at IS NULL
RETURNING id`
	dropSudoSQL          = `UPDATE user_sessions SET sudo_until = NULL WHERE user_id = $1`
	deleteOldSessionsSQL = `DELETE FROM user_sessions WHERE expires_at < $1`
	knownDeviceSQL       = `SELECT EXISTS (
    SELECT 1 FROM user_sessions
    WHERE user_id = $1 AND (device = $2 OR ($3 <> '' AND ip_hash = $3))
)`
	setTwoStepSQL = `UPDATE users SET two_step_enabled = $2, two_step_code = NULL,
two_step_expires_at = NULL, two_step_attempts = 0, updated_at = now()
WHERE id = $1
RETURNING id`
	startTwoStepSQL = `UPDATE users
SET two_step_code = $2, two_step_expires_at = $3, two_step_sent_at = $4, two_step_attempts = 0,
    updated_at = now()
WHERE id = $1
RETURNING id`
	twoStepChallengeSQL = `SELECT u.id, u.role,
COALESCE(s.email, t.email, u.contact_email, ''),
u.two_step_enabled, COALESCE(u.two_step_code, ''), u.two_step_attempts,
u.two_step_expires_at, u.two_step_sent_at
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
` + identifierMatch
	countTwoStepAttemptSQL = `UPDATE users
SET two_step_attempts = two_step_attempts + 1, updated_at = now()
WHERE id = $1
RETURNING two_step_attempts`
	clearTwoStepSQL = `UPDATE users
SET two_step_code = NULL, two_step_expires_at = NULL, two_step_attempts = 0, updated_at = now()
WHERE id = $1
RETURNING id`
	updateOfficeContactSQL = `UPDATE users
SET contact_email = COALESCE(NULLIF($2, ''), contact_email),
    contact_phone = COALESCE(NULLIF($3, ''), contact_phone),
    updated_at = now()
WHERE id = $1 AND role IN ('admin', 'super_admin')
RETURNING id`
	resetSuperAdminSQL = `UPDATE users
SET password_hash = $1, token_version = token_version + 1,
    two_step_code = NULL, two_step_expires_at = NULL, two_step_attempts = 0, updated_at = now()
WHERE role = 'super_admin'
RETURNING id, COALESCE(identifier, '')`
)

func scanLoginSession(row pgx.Row) (Session, error) {
	var session Session
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Device,
		&session.Client,
		&session.IPHash,
		&session.Country,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.SudoUntil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("store: could not read the session: %w", err)
	}
	return session, nil
}

func (s *Store) CreateSession(ctx context.Context, session NewSession) (int64, error) {
	client := session.Client
	if client != ClientAndroid {
		client = ClientWeb
	}
	var id int64
	err := s.pool.QueryRow(ctx, createSessionSQL, session.UserID, session.Device, client,
		session.IPHash, session.Country, session.ExpiresAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: could not open a session for user %d: %w", session.UserID, err)
	}
	return id, nil
}

func (s *Store) LoadSession(ctx context.Context, sessionID int64) (Session, error) {
	return scanLoginSession(s.pool.QueryRow(ctx, loadSessionSQL, sessionID))
}

func (s *Store) TouchSession(ctx context.Context, sessionID int64, ipHash, country string) error {
	if _, err := s.pool.Exec(ctx, touchSessionSQL, sessionID, ipHash, country); err != nil {
		return fmt.Errorf("store: could not touch the session %d: %w", sessionID, err)
	}
	return nil
}

func (s *Store) ListSessions(ctx context.Context, userID int64) ([]Session, error) {
	rows, err := s.pool.Query(ctx, listSessionsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the sessions of user %d: %w", userID, err)
	}
	defer rows.Close()
	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanLoginSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the sessions of user %d: %w", userID, err)
	}
	return sessions, nil
}

func (s *Store) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, revokeSessionSQL, sessionID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not revoke the session %d: %w", sessionID, err)
	}
	return nil
}

func (s *Store) RevokeOtherSessions(ctx context.Context, userID, keepID int64) (int64, error) {
	tag, err := s.pool.Exec(ctx, revokeOthersSQL, userID, keepID)
	if err != nil {
		return 0, fmt.Errorf("store: could not revoke the other sessions of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) RevokeAllSessions(ctx context.Context, userID int64) (int64, error) {
	tag, err := s.pool.Exec(ctx, revokeAllSQL, userID)
	if err != nil {
		return 0, fmt.Errorf("store: could not revoke the sessions of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) GrantSudo(ctx context.Context, sessionID int64, until time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, grantSudoSQL, sessionID, until).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionGone
	}
	if err != nil {
		return fmt.Errorf("store: could not open the confirmation window: %w", err)
	}
	return nil
}

func (s *Store) DropSudo(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, dropSudoSQL, userID); err != nil {
		return fmt.Errorf("store: could not close the confirmation windows of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) DeleteOldSessions(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, deleteOldSessionsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) SetTwoStep(ctx context.Context, userID int64, enabled bool) error {
	var id int64
	err := s.pool.QueryRow(ctx, setTwoStepSQL, userID, enabled).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not change the two-step setting of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) StartTwoStep(ctx context.Context, userID int64, codeHash string,
	expiresAt, sentAt time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, startTwoStepSQL, userID, codeHash, expiresAt, sentAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not store a two-step code for user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) TwoStepChallenge(ctx context.Context, identifier string) (TwoStepChallenge, error) {
	var challenge TwoStepChallenge
	err := s.pool.QueryRow(ctx, twoStepChallengeSQL, identifier).Scan(
		&challenge.UserID,
		&challenge.Role,
		&challenge.Email,
		&challenge.Enabled,
		&challenge.CodeHash,
		&challenge.Attempts,
		&challenge.ExpiresAt,
		&challenge.SentAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TwoStepChallenge{}, ErrNotFound
	}
	if err != nil {
		return TwoStepChallenge{}, fmt.Errorf("store: could not load the two-step challenge: %w", err)
	}
	return challenge, nil
}

func (s *Store) CountTwoStepAttempt(ctx context.Context, userID int64) (int, error) {
	var attempts int
	err := s.pool.QueryRow(ctx, countTwoStepAttemptSQL, userID).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not count a two-step attempt: %w", err)
	}
	return attempts, nil
}

func (s *Store) ClearTwoStep(ctx context.Context, userID int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, clearTwoStepSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not clear the two-step code of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) UpdateOfficeContact(ctx context.Context, userID int64, email, phone string) error {
	var id int64
	err := s.pool.QueryRow(ctx, updateOfficeContactSQL, userID, email, phone).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateAccount
		}
		return fmt.Errorf("store: could not change the contact details of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) ResetSuperAdminPassword(ctx context.Context, passwordHash string) (string, error) {
	var userID int64
	var identifier string
	err := s.pool.QueryRow(ctx, resetSuperAdminSQL, passwordHash).Scan(&userID, &identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not reset the super admin password: %w", err)
	}
	if _, err := s.RevokeAllSessions(ctx, userID); err != nil {
		return "", err
	}
	return identifier, nil
}

func (s *Store) KnownDevice(ctx context.Context, userID int64, device, ipHash string) (bool, error) {
	var seen bool
	if err := s.pool.QueryRow(ctx, knownDeviceSQL, userID, device, ipHash).Scan(&seen); err != nil {
		return false, fmt.Errorf("store: could not check the devices of user %d: %w", userID, err)
	}
	return seen, nil
}
