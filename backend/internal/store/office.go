package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

var (
	ErrNotAnOfficeAdmin = errors.New("store: the account is not an office admin")
	ErrNotATeacher      = errors.New("store: the account is not an active teacher")
	ErrSuperAdminLocked = errors.New("store: the super admin account cannot be changed here")
)

const (
	OfficeSourceAccount = "account"
	OfficeSourceTeacher = "teacher"
)

type OfficeAdmin struct {
	ID          int64
	Identifier  string
	Name        string
	Email       string
	Role        string
	Status      string
	Source      string
	LastSignIn  *time.Time
	InviteEnds  *time.Time
	InvitedBy   string
	ActionCount int
	CreatedAt   time.Time
}

type OfficeTeacher struct {
	ID           int64
	TeacherID    string
	Name         string
	Designation  string
	Email        string
	OfficeAccess bool
	LastSignIn   *time.Time
}

type AdminInvite struct {
	UserID   int64
	Email    string
	Attempts int
}

const (
	listOfficeAdminsSQL = `SELECT u.id, COALESCE(u.identifier, ''),
COALESCE(t.name, ''),
COALESCE(t.email, u.contact_email, ''),
u.role, u.status,
CASE WHEN u.role = 'teacher' THEN 'teacher' ELSE 'account' END,
u.last_sign_in_at, u.invite_expires_at,
COALESCE(inviter.identifier, ''),
(SELECT COUNT(*) FROM audit_events e WHERE e.actor_id = u.id),
u.created_at
FROM users u
LEFT JOIN teachers t ON t.user_id = u.id
LEFT JOIN users inviter ON inviter.id = u.invited_by
WHERE u.role IN ('admin', 'super_admin') OR u.office_access
ORDER BY CASE u.role WHEN 'super_admin' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
         lower(COALESCE(u.identifier, t.name, '')) ASC`
	createAdminInviteSQL = `INSERT INTO users (role, status, contact_email, invite_token, invite_expires_at,
invite_sent_at, invited_by, password_hash)
VALUES ('admin', 'invited', $1, $2, $3, now(), $4, '')
RETURNING id`
	emailTakenSQL = `SELECT EXISTS (
    SELECT 1 FROM users u
    LEFT JOIN students s ON s.user_id = u.id
    LEFT JOIN teachers t ON t.user_id = u.id
    WHERE lower(COALESCE(s.email, t.email, u.contact_email, '')) = lower($1)
      AND u.status <> 'removed'
)`
	resendAdminInviteSQL = `UPDATE users
SET invite_token = $2, invite_expires_at = $3, invite_attempts = 0, invite_sent_at = now(), updated_at = now()
WHERE id = $1 AND role = 'admin' AND status = 'invited'
RETURNING COALESCE(contact_email, '')`
	cancelAdminInviteSQL = `DELETE FROM users
WHERE id = $1 AND role = 'admin' AND status = 'invited'
RETURNING COALESCE(contact_email, '')`
	findAdminInviteSQL = `SELECT u.id, COALESCE(u.contact_email, ''), u.invite_attempts
FROM users u
WHERE u.invite_token = $1 AND u.role = 'admin' AND u.status = 'invited'
  AND u.invite_expires_at > now()`
	activateAdminSQL = `UPDATE users
SET status = 'active',
    identifier = $2,
    password_hash = $3,
    invite_token = NULL,
    invite_expires_at = NULL,
    invite_attempts = 0,
    email_verified_at = now(),
    token_version = token_version + 1,
    updated_at = now()
WHERE id = $1 AND role = 'admin' AND status = 'invited'
RETURNING id`
	suspendAdminSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'admin' AND status = 'active'
RETURNING COALESCE(identifier, '')`
	restoreAdminSQL = `UPDATE users
SET status = 'active', token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'admin' AND status = 'removed'
RETURNING COALESCE(identifier, '')`
	removeAdminSQL = `DELETE FROM users
WHERE id = $1 AND role = 'admin'
RETURNING COALESCE(identifier, '')`
	officeAccountSQL   = `SELECT COALESCE(identifier, ''), role, status FROM users WHERE id = $1`
	setOfficeAccessSQL = `UPDATE users
SET office_access = $2, token_version = token_version + 1, updated_at = now()
WHERE id = $1 AND role = 'teacher' AND status = 'active'
RETURNING id`
	officeTeacherNameSQL  = `SELECT t.name FROM teachers t WHERE t.user_id = $1`
	listOfficeTeachersSQL = `SELECT u.id, COALESCE(t.teacher_id, ''), t.name, t.designation, t.email,
u.office_access, u.last_sign_in_at
FROM users u
JOIN teachers t ON t.user_id = u.id
WHERE u.role = 'teacher' AND u.status = 'active'
  AND ($1 = '' OR t.name ILIKE '%' || $1 || '%' OR t.email ILIKE '%' || $1 || '%'
       OR COALESCE(t.teacher_id, '') ILIKE '%' || $1 || '%')
ORDER BY u.office_access DESC, lower(t.name) ASC
LIMIT $2`
	touchSignInSQL = `UPDATE users SET last_sign_in_at = now() WHERE id = $1`
)

func (s *Store) ListOfficeAdmins(ctx context.Context) ([]OfficeAdmin, error) {
	rows, err := s.pool.Query(ctx, listOfficeAdminsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the office admins: %w", err)
	}
	defer rows.Close()
	admins := make([]OfficeAdmin, 0)
	for rows.Next() {
		var admin OfficeAdmin
		if err := rows.Scan(
			&admin.ID,
			&admin.Identifier,
			&admin.Name,
			&admin.Email,
			&admin.Role,
			&admin.Status,
			&admin.Source,
			&admin.LastSignIn,
			&admin.InviteEnds,
			&admin.InvitedBy,
			&admin.ActionCount,
			&admin.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: could not scan an office admin: %w", err)
		}
		admins = append(admins, admin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the office admins: %w", err)
	}
	return admins, nil
}

func (s *Store) EmailTaken(ctx context.Context, email string) (bool, error) {
	var taken bool
	if err := s.pool.QueryRow(ctx, emailTakenSQL, email).Scan(&taken); err != nil {
		return false, fmt.Errorf("store: could not check the email %q: %w", email, err)
	}
	return taken, nil
}

func (s *Store) CreateAdminInvite(ctx context.Context, email, inviteHash string,
	expiresAt time.Time, invitedBy int64) (int64, error) {
	taken, err := s.EmailTaken(ctx, email)
	if err != nil {
		return 0, err
	}
	if taken {
		return 0, ErrDuplicateAccount
	}
	var id int64
	err = s.pool.QueryRow(ctx, createAdminInviteSQL, email, inviteHash, expiresAt, invitedBy).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccount
		}
		return 0, fmt.Errorf("store: could not invite an office admin: %w", err)
	}
	return id, nil
}

func (s *Store) ResendAdminInvite(ctx context.Context, userID int64, inviteHash string,
	expiresAt time.Time) (string, error) {
	var email string
	err := s.pool.QueryRow(ctx, resendAdminInviteSQL, userID, inviteHash, expiresAt).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not refresh the invitation of the admin %d: %w", userID, err)
	}
	return email, nil
}

func (s *Store) CancelAdminInvite(ctx context.Context, userID int64) (string, error) {
	var email string
	err := s.pool.QueryRow(ctx, cancelAdminInviteSQL, userID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not cancel the invitation of the admin %d: %w", userID, err)
	}
	return email, nil
}

func (s *Store) FindAdminInvite(ctx context.Context, inviteHash string) (AdminInvite, error) {
	var invite AdminInvite
	err := s.pool.QueryRow(ctx, findAdminInviteSQL, inviteHash).
		Scan(&invite.UserID, &invite.Email, &invite.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminInvite{}, ErrNotFound
	}
	if err != nil {
		return AdminInvite{}, fmt.Errorf("store: could not load the admin invitation: %w", err)
	}
	return invite, nil
}

func (s *Store) ActivateAdmin(ctx context.Context, userID int64, chosenID, passwordHash string) error {
	var id int64
	err := s.pool.QueryRow(ctx, activateAdminSQL, userID, chosenID, passwordHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateIdentifier
		}
		return fmt.Errorf("store: could not activate the admin %d: %w", userID, err)
	}
	return nil
}

func (s *Store) SuspendAdmin(ctx context.Context, userID int64) (string, error) {
	return s.changeAdmin(ctx, suspendAdminSQL, userID)
}

func (s *Store) RestoreAdmin(ctx context.Context, userID int64) (string, error) {
	return s.changeAdmin(ctx, restoreAdminSQL, userID)
}

func (s *Store) RemoveAdmin(ctx context.Context, userID int64) (string, error) {
	return s.changeAdmin(ctx, removeAdminSQL, userID)
}

func (s *Store) changeAdmin(ctx context.Context, query string, userID int64) (string, error) {
	var identifier string
	err := s.pool.QueryRow(ctx, query, userID).Scan(&identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", s.officeAccountConflict(ctx, userID)
	}
	if err != nil {
		return "", fmt.Errorf("store: could not change the office admin %d: %w", userID, err)
	}
	return identifier, nil
}

func (s *Store) officeAccountConflict(ctx context.Context, userID int64) error {
	_, role, _, err := s.OfficeAccount(ctx, userID)
	if err != nil {
		return err
	}
	if role == RoleSuperAdmin {
		return ErrSuperAdminLocked
	}
	return ErrNotAnOfficeAdmin
}

func (s *Store) OfficeAccount(ctx context.Context, userID int64) (string, string, string, error) {
	var identifier, role, status string
	err := s.pool.QueryRow(ctx, officeAccountSQL, userID).Scan(&identifier, &role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrNotFound
	}
	if err != nil {
		return "", "", "", fmt.Errorf("store: could not load the office account %d: %w", userID, err)
	}
	return identifier, role, status, nil
}

func (s *Store) SetTeacherOfficeAccess(ctx context.Context, userID int64, granted bool) (string, error) {
	var id int64
	err := s.pool.QueryRow(ctx, setOfficeAccessSQL, userID, granted).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotATeacher
	}
	if err != nil {
		return "", fmt.Errorf("store: could not change the office access of the teacher %d: %w", userID, err)
	}
	var name string
	if err := s.pool.QueryRow(ctx, officeTeacherNameSQL, userID).Scan(&name); err != nil {
		return "", fmt.Errorf("store: could not load the teacher %d: %w", userID, err)
	}
	return name, nil
}

func (s *Store) ListOfficeTeachers(ctx context.Context, search string, limit int) ([]OfficeTeacher, error) {
	if limit <= 0 {
		limit = DefaultStudentPage
	}
	rows, err := s.pool.Query(ctx, listOfficeTeachersSQL, search, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the teachers for office access: %w", err)
	}
	defer rows.Close()
	teachers := make([]OfficeTeacher, 0)
	for rows.Next() {
		var teacher OfficeTeacher
		if err := rows.Scan(
			&teacher.ID,
			&teacher.TeacherID,
			&teacher.Name,
			&teacher.Designation,
			&teacher.Email,
			&teacher.OfficeAccess,
			&teacher.LastSignIn,
		); err != nil {
			return nil, fmt.Errorf("store: could not scan a teacher for office access: %w", err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the teachers for office access: %w", err)
	}
	return teachers, nil
}

func (s *Store) TouchSignIn(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, touchSignInSQL, userID); err != nil {
		return fmt.Errorf("store: could not record the sign-in of user %d: %w", userID, err)
	}
	return nil
}
