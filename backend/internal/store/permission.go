package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

const (
	listPermissionsSQL  = `SELECT permission FROM admin_permissions WHERE user_id = $1 ORDER BY permission`
	clearPermissionsSQL = `DELETE FROM admin_permissions WHERE user_id = $1`
	insertPermissionSQL = `INSERT INTO admin_permissions (user_id, permission, granted_by)
VALUES ($1, $2, NULLIF($3, 0))
ON CONFLICT DO NOTHING`
	officePermissionsSQL = `SELECT user_id, permission FROM admin_permissions`
)

func (s *Store) Permissions(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, listPermissionsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the permissions of user %d: %w", userID, err)
	}
	defer rows.Close()
	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: could not scan a permission: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the permissions of user %d: %w", userID, err)
	}
	return names, nil
}

func (s *Store) SetPermissions(ctx context.Context, userID int64, names []string, grantedBy int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the permission transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, clearPermissionsSQL, userID); err != nil {
		return fmt.Errorf("store: could not clear the permissions of user %d: %w", userID, err)
	}
	batch := &pgx.Batch{}
	for _, name := range names {
		batch.Queue(insertPermissionSQL, userID, name, grantedBy)
	}
	if batch.Len() > 0 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("store: could not store the permissions of user %d: %w", userID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the permissions of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) AllPermissions(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.pool.Query(ctx, officePermissionsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the office permissions: %w", err)
	}
	defer rows.Close()
	granted := make(map[int64][]string)
	for rows.Next() {
		var userID int64
		var name string
		if err := rows.Scan(&userID, &name); err != nil {
			return nil, fmt.Errorf("store: could not scan an office permission: %w", err)
		}
		granted[userID] = append(granted[userID], name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the office permissions: %w", err)
	}
	return granted, nil
}
