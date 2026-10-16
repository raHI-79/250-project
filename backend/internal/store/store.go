package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusActive   = "active"
	StatusPending  = "pending"
	StatusRejected = "rejected"
	StatusRemoved  = "removed"
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

const getAuthUserSQL = `SELECT u.id, u.identifier, u.role, u.status, u.token_version, COALESCE(s.is_cr, FALSE)
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
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, ErrNotFound
	}
	if err != nil {
		return AuthUser{}, fmt.Errorf("store: could not load the user with id %d: %w", userID, err)
	}
	return user, nil
}

const ensureAdminSQL = `INSERT INTO users (role, identifier, password_hash, status)
VALUES ('admin', $1, $2, 'active')
ON CONFLICT (identifier) DO NOTHING
RETURNING id`

func (s *Store) EnsureAdmin(ctx context.Context, identifier, passwordHash string) (bool, error) {
	if identifier == "" {
		return false, ErrEmptyIdentifier
	}
	if passwordHash == "" {
		return false, ErrEmptyPasswordHash
	}
	var id int64
	err := s.pool.QueryRow(ctx, ensureAdminSQL, identifier, passwordHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: could not create the admin account: %w", err)
	}
	return true, nil
}
