package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	ActionRemove      = "remove"
	ActionRestore     = "restore"
	ActionDelete      = "permanent_delete"
	DefaultAuditLimit = 100
)

var (
	ErrNotRestorable = errors.New("store: the account is not removed or rejected")
	ErrNotDeletable  = errors.New("store: only a removed or rejected account can be deleted")
	ErrProtected     = errors.New("store: this account cannot be deleted")
)

type AccountRef struct {
	ID         int64
	Role       string
	Status     string
	Identifier string
	Email      string
	Name       string
	PhotoPath  string
	IsCR       bool
}

type AuditRow struct {
	ID            int64
	AdminID       int64
	AdminName     string
	Action        string
	TargetSummary string
	CreatedAt     time.Time
}

const (
	findAccountSQL = `SELECT u.id, u.role, u.status, COALESCE(u.identifier, ''),
COALESCE(s.email, t.email, u.contact_email, ''),
COALESCE(s.full_name, t.name, ''),
COALESCE(u.photo_path, ''),
COALESCE(s.is_cr, FALSE)
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1`
	restoreAccountSQL = `UPDATE users
SET status = CASE WHEN status = 'removed' THEN 'active' ELSE 'pending' END,
    token_version = CASE WHEN status = 'removed' THEN token_version + 1 ELSE token_version END,
    updated_at = now()
WHERE id = $1 AND status IN ('removed', 'rejected')
RETURNING status`
	lockAccountSQL             = `SELECT role, status, COALESCE(photo_path, '') FROM users WHERE id = $1 FOR UPDATE`
	deleteDropCoursesOfUserSQL = `DELETE FROM student_drop_courses
WHERE student_id IN (SELECT id FROM students WHERE user_id = $1)`
	deletePollVotesOfUserSQL = `DELETE FROM poll_votes
WHERE student_id IN (SELECT id FROM students WHERE user_id = $1)`
	deleteConversationMembersSQL = `DELETE FROM conversation_members WHERE user_id = $1`
	deleteConversationReadsSQL   = `DELETE FROM conversation_reads WHERE user_id = $1`
	deleteStudentRowSQL          = `DELETE FROM students WHERE user_id = $1`
	deleteTeacherRowSQL          = `DELETE FROM teachers WHERE user_id = $1`
	deleteAccountSQL             = `DELETE FROM users WHERE id = $1`
	insertAuditSQL               = `INSERT INTO admin_audit (admin_id, action, target_summary)
VALUES ($1, $2, $3)`
	listAuditSQL = `SELECT a.id, COALESCE(a.admin_id, 0),
COALESCE(s.full_name, t.name, u.identifier, 'Administration'),
a.action, a.target_summary, a.created_at
FROM admin_audit a
LEFT JOIN users u ON u.id = a.admin_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
ORDER BY a.created_at DESC, a.id DESC
LIMIT $1`
)

func (s *Store) FindAccount(ctx context.Context, userID int64) (AccountRef, error) {
	var account AccountRef
	err := s.pool.QueryRow(ctx, findAccountSQL, userID).Scan(
		&account.ID,
		&account.Role,
		&account.Status,
		&account.Identifier,
		&account.Email,
		&account.Name,
		&account.PhotoPath,
		&account.IsCR,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountRef{}, ErrNotFound
	}
	if err != nil {
		return AccountRef{}, fmt.Errorf("store: could not load the account %d: %w", userID, err)
	}
	return account, nil
}

func (s *Store) RestoreAccount(ctx context.Context, userID int64) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, restoreAccountSQL, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, lookupErr := s.FindAccount(ctx, userID); lookupErr != nil {
			return "", lookupErr
		}
		return "", ErrNotRestorable
	}
	if err != nil {
		return "", fmt.Errorf("store: could not restore the account %d: %w", userID, err)
	}
	return status, nil
}

func (s *Store) DeleteAccount(ctx context.Context, userID int64) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("store: could not start the deletion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var role, status, photoPath string
	err = tx.QueryRow(ctx, lockAccountSQL, userID).Scan(&role, &status, &photoPath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not lock the account %d: %w", userID, err)
	}
	if role == RoleAdmin {
		return "", ErrProtected
	}
	if status != StatusRemoved && status != StatusRejected {
		return "", ErrNotDeletable
	}
	cleanups := []string{
		deleteDropCoursesOfUserSQL,
		deletePollVotesOfUserSQL,
		deleteConversationMembersSQL,
		deleteConversationReadsSQL,
		deleteStudentRowSQL,
		deleteTeacherRowSQL,
		deleteAccountSQL,
	}
	for _, statement := range cleanups {
		if _, err := tx.Exec(ctx, statement, userID); err != nil {
			return "", fmt.Errorf("store: could not delete the account %d: %w", userID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("store: could not commit the deletion of the account %d: %w", userID, err)
	}
	return photoPath, nil
}

func (s *Store) WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error {
	if _, err := s.pool.Exec(ctx, insertAuditSQL, adminID, action, targetSummary); err != nil {
		return fmt.Errorf("store: could not write the %s audit entry: %w", action, err)
	}
	return nil
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditRow, error) {
	rows, err := s.pool.Query(ctx, listAuditSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the audit entries: %w", err)
	}
	defer rows.Close()
	entries := make([]AuditRow, 0)
	for rows.Next() {
		var entry AuditRow
		err := rows.Scan(
			&entry.ID,
			&entry.AdminID,
			&entry.AdminName,
			&entry.Action,
			&entry.TargetSummary,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan an audit entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the audit entries: %w", err)
	}
	return entries, nil
}
