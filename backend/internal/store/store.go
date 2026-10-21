package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RoleStudent      = "student"
	RoleTeacher      = "teacher"
	RoleAdmin        = "admin"
	StatusActive     = "active"
	StatusPending    = "pending"
	StatusRejected   = "rejected"
	StatusRemoved    = "removed"
	StatusUnverified = "unverified"
	StatusInvited    = "invited"
)

var (
	ErrNotFound          = errors.New("store: no matching row")
	ErrEmptyIdentifier   = errors.New("store: the admin identifier must not be empty")
	ErrEmptyPasswordHash = errors.New("store: the admin password hash must not be empty")
)

type AuthUser struct {
	ID           int64
	Identifier   string
	Role         string
	Status       string
	TokenVersion int
	IsCR         bool
	PhotoPath    string
}

func (u AuthUser) IsActive() bool {
	return u.Status == StatusActive
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const getAuthUserSQL = `SELECT u.id, COALESCE(u.identifier, ''), u.role, u.status, u.token_version, COALESCE(s.is_cr, FALSE),
COALESCE(u.photo_path, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.id = $1`

func (s *Store) GetAuthUser(ctx context.Context, userID int64) (AuthUser, error) {
	var user AuthUser
	err := s.pool.QueryRow(ctx, getAuthUserSQL, userID).Scan(
		&user.ID,
		&user.Identifier,
		&user.Role,
		&user.Status,
		&user.TokenVersion,
		&user.IsCR,
		&user.PhotoPath,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, ErrNotFound
	}
	if err != nil {
		return AuthUser{}, fmt.Errorf("store: could not load the user with id %d: %w", userID, err)
	}
	return user, nil
}

const ensureAdminSQL = `INSERT INTO users (role, identifier, password_hash, status, contact_email, contact_phone)
VALUES ('admin', $1, $2, 'active', NULLIF($3, ''), NULLIF($4, ''))
ON CONFLICT (identifier) DO NOTHING
RETURNING id`

const fillAdminContactSQL = `UPDATE users
SET contact_email = COALESCE(NULLIF(contact_email, ''), NULLIF($2, '')),
    contact_phone = COALESCE(NULLIF(contact_phone, ''), NULLIF($3, '')),
    updated_at = now()
WHERE identifier = $1
  AND role = 'admin'
  AND (COALESCE(contact_email, '') = '' OR COALESCE(contact_phone, '') = '')`

func (s *Store) EnsureAdmin(ctx context.Context, identifier, passwordHash, contactEmail, contactPhone string) (bool, error) {
	if identifier == "" {
		return false, ErrEmptyIdentifier
	}
	if passwordHash == "" {
		return false, ErrEmptyPasswordHash
	}
	created := true
	var id int64
	err := s.pool.QueryRow(ctx, ensureAdminSQL, identifier, passwordHash, contactEmail, contactPhone).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
	} else if err != nil {
		return false, fmt.Errorf("store: could not create the admin account: %w", err)
	}
	if contactEmail == "" && contactPhone == "" {
		return created, nil
	}
	if _, err := s.pool.Exec(ctx, fillAdminContactSQL, identifier, contactEmail, contactPhone); err != nil {
		return created, fmt.Errorf("store: could not fill the admin contact details: %w", err)
	}
	return created, nil
}
