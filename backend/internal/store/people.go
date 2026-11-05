package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

var ErrNotAStudent = errors.New("store: the account is not a student")

type PersonFilter struct {
	Search string
	Role   string
	Status string
	Batch  string
	Limit  int
	Offset int
}

type PersonRow struct {
	ID         int64
	Identifier string
	Name       string
	Email      string
	Phone      string
	Role       string
	Status     string
	Batch      string
	Section    string
	IsCR       bool
	LastSignIn *time.Time
	CreatedAt  time.Time
}

type PersonDetail struct {
	PersonRow
	Sessions   int
	DeviceID   string
	DeviceSeen *time.Time
	TwoStep    bool
}

const (
	personColumns = `u.id, COALESCE(u.identifier, ''),
COALESCE(s.full_name, t.name, u.identifier, ''),
COALESCE(s.email, t.email, u.contact_email, ''),
COALESCE(s.phone, t.phone, u.contact_phone, ''),
u.role, u.status,
COALESCE(s.batch, ''), COALESCE(s.section, ''),
COALESCE(s.is_cr, FALSE), u.last_sign_in_at, u.created_at`
	personFrom = `FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	personWhere = `WHERE ($1 = '' OR u.role = $1)
  AND ($2 = '' OR u.status = $2)
  AND ($3 = '' OR COALESCE(s.batch, '') = $3)
  AND ($4 = '' OR COALESCE(s.full_name, t.name, '') ILIKE '%' || $4 || '%'
       OR COALESCE(u.identifier, '') ILIKE '%' || $4 || '%'
       OR COALESCE(s.reg_no, '') ILIKE '%' || $4 || '%'
       OR COALESCE(s.email, t.email, u.contact_email, '') ILIKE '%' || $4 || '%')`
	listPeopleSQL = `SELECT ` + personColumns + `
` + personFrom + `
` + personWhere + `
ORDER BY u.created_at DESC, u.id DESC
LIMIT $5 OFFSET $6`
	countPeopleSQL = `SELECT COUNT(*)
` + personFrom + `
` + personWhere
	findPersonRowSQL = `SELECT ` + personColumns + `,
(SELECT COUNT(*) FROM user_sessions v WHERE v.user_id = u.id AND v.revoked_at IS NULL
   AND v.expires_at > now()),
COALESCE((SELECT d.device_id FROM device_bindings d WHERE d.user_id = u.id), ''),
(SELECT d.last_seen_at FROM device_bindings d WHERE d.user_id = u.id),
u.two_step_enabled
` + personFrom + `
WHERE u.id = $1`
	suspendPersonSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND status <> 'removed' AND role NOT IN ('super_admin')
RETURNING COALESCE(identifier, '')`
	restorePersonSQL = `UPDATE users
SET status = 'active', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND status = 'removed'
RETURNING COALESCE(identifier, '')`
	dropDeviceSQL = `DELETE FROM device_bindings WHERE user_id = $1`
	setCRFlagSQL  = `UPDATE students SET is_cr = $2, updated_at = now()
WHERE user_id = $1
RETURNING id`
	bumpTokenSQL = `UPDATE users SET token_version = token_version + 1, updated_at = now()
WHERE id = $1
RETURNING token_version`
)

func scanPerson(row pgx.Row, person *PersonRow, extra ...any) error {
	targets := []any{
		&person.ID,
		&person.Identifier,
		&person.Name,
		&person.Email,
		&person.Phone,
		&person.Role,
		&person.Status,
		&person.Batch,
		&person.Section,
		&person.IsCR,
		&person.LastSignIn,
		&person.CreatedAt,
	}
	targets = append(targets, extra...)
	return row.Scan(targets...)
}

func (s *Store) ListPeople(ctx context.Context, filter PersonFilter) ([]PersonRow, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultStudentPage
	}
	if limit > MaxStudentPage {
		limit = MaxStudentPage
	}
	role := filter.Role
	if role == RoleCR {
		role = RoleStudent
	}
	search := strings.TrimSpace(filter.Search)
	rows, err := s.pool.Query(ctx, listPeopleSQL, role, filter.Status, filter.Batch, search,
		limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: could not list the accounts: %w", err)
	}
	defer rows.Close()
	people := make([]PersonRow, 0)
	for rows.Next() {
		var person PersonRow
		if err := scanPerson(rows, &person); err != nil {
			return nil, 0, fmt.Errorf("store: could not scan an account: %w", err)
		}
		if filter.Role == RoleCR && !person.IsCR {
			continue
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: could not read the accounts: %w", err)
	}
	var total int
	err = s.pool.QueryRow(ctx, countPeopleSQL, role, filter.Status, filter.Batch, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("store: could not count the accounts: %w", err)
	}
	return people, total, nil
}

func (s *Store) FindPersonDetail(ctx context.Context, userID int64) (PersonDetail, error) {
	var detail PersonDetail
	err := scanPerson(s.pool.QueryRow(ctx, findPersonRowSQL, userID), &detail.PersonRow,
		&detail.Sessions, &detail.DeviceID, &detail.DeviceSeen, &detail.TwoStep)
	if errors.Is(err, pgx.ErrNoRows) {
		return PersonDetail{}, ErrNotFound
	}
	if err != nil {
		return PersonDetail{}, fmt.Errorf("store: could not load the account %d: %w", userID, err)
	}
	return detail, nil
}

func (s *Store) SuspendPerson(ctx context.Context, userID int64) (string, error) {
	var identifier string
	err := s.pool.QueryRow(ctx, suspendPersonSQL, userID).Scan(&identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not suspend the account %d: %w", userID, err)
	}
	if _, err := s.RevokeAllSessions(ctx, userID); err != nil {
		return "", err
	}
	return identifier, nil
}

func (s *Store) RestorePerson(ctx context.Context, userID int64) (string, error) {
	var identifier string
	err := s.pool.QueryRow(ctx, restorePersonSQL, userID).Scan(&identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not restore the account %d: %w", userID, err)
	}
	return identifier, nil
}

func (s *Store) SignOutPerson(ctx context.Context, userID int64) error {
	var version int
	err := s.pool.QueryRow(ctx, bumpTokenSQL, userID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not sign the account %d out: %w", userID, err)
	}
	if _, err := s.RevokeAllSessions(ctx, userID); err != nil {
		return err
	}
	return nil
}

func (s *Store) ResetPersonDevice(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, dropDeviceSQL, userID); err != nil {
		return fmt.Errorf("store: could not reset the phone of the account %d: %w", userID, err)
	}
	return nil
}

func (s *Store) SetCR(ctx context.Context, userID int64, isCR bool) error {
	var id int64
	err := s.pool.QueryRow(ctx, setCRFlagSQL, userID, isCR).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotAStudent
	}
	if err != nil {
		return fmt.Errorf("store: could not change the CR flag of the account %d: %w", userID, err)
	}
	var version int
	if err := s.pool.QueryRow(ctx, bumpTokenSQL, userID).Scan(&version); err != nil {
		return fmt.Errorf("store: could not refresh the tokens of the account %d: %w", userID, err)
	}
	return nil
}
