package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	ClientWeb             = "web"
	ClientAndroid         = "android"
	DefaultEventPage      = 50
	MaxEventPage          = 200
	ActionSignIn          = "sign_in"
	ActionSignOut         = "sign_out"
	ActionSignInFailed    = "sign_in_failed"
	ActionCreated         = "create"
	ActionUpdated         = "update"
	ActionDeleted         = "delete"
	ActionPublished       = "publish"
	ActionInviteAdmin     = "invite_admin"
	ActionResendInvite    = "resend_invite"
	ActionCancelInvite    = "cancel_invite"
	ActionSuspendAdmin    = "suspend_admin"
	ActionRestoreAdmin    = "restore_admin"
	ActionRemoveAdmin     = "remove_admin"
	ActionGrantOffice     = "grant_office_access"
	ActionRevokeOffice    = "revoke_office_access"
	ActionActivateAdmin   = "activate_admin"
	ActionConfirmPassword = "confirm_password"
	ActionPasswordChange  = "password_change"
	ActionPasswordReset   = "password_reset"
	ActionContactChange   = "contact_change"
	ActionTwoStepOn       = "two_step_on"
	ActionTwoStepOff      = "two_step_off"
	ActionSessionOut      = "session_out"
	ActionSessionOutAll   = "session_out_all"
	ActionTwoStepFailed   = "two_step_failed"
	ActionTwoStepSent     = "two_step_sent"
	ActionNewDevice       = "new_device"
	ActionSuspendUser     = "suspend_user"
	ActionRestoreUser     = "restore_user"
	ActionResetPhone      = "reset_phone"
	ActionRoleChange      = "role_change"
	ActionPermissions     = "permissions"
)

type AuditEvent struct {
	ActorID    int64
	ActorRole  string
	ActorLabel string
	Action     string
	TargetType string
	TargetID   string
	Summary    string
	IPHash     string
	UserAgent  string
	Client     string
}

type AuditEntry struct {
	ID         int64
	ActorID    int64
	ActorRole  string
	ActorLabel string
	Action     string
	TargetType string
	TargetID   string
	Summary    string
	Client     string
	UserAgent  string
	CreatedAt  time.Time
}

type AuditFilter struct {
	ActorID int64
	Role    string
	Action  string
	Client  string
	Search  string
	From    *time.Time
	To      *time.Time
	Before  int64
	Limit   int
}

type ActionCount struct {
	Action string
	Total  int
}

type AuditPerson struct {
	ID    int64
	Label string
	Role  string
	Total int
}

const (
	insertEventSQL = `INSERT INTO audit_events (actor_id, actor_role, actor_label, action, target_type,
target_id, summary, ip_hash, user_agent, client)
VALUES (NULLIF($1, 0), $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	listEventsSQL = `SELECT e.id, COALESCE(e.actor_id, 0), e.actor_role,
COALESCE(NULLIF(e.actor_label, ''), s.full_name, t.name, u.identifier, 'System'),
e.action, e.target_type, e.target_id, e.summary, e.client, e.user_agent, e.created_at
FROM audit_events e
LEFT JOIN users u ON u.id = e.actor_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE ($1 = 0 OR e.actor_id = $1)
  AND ($2 = '' OR e.actor_role = $2)
  AND ($3 = '' OR e.action = $3)
  AND ($4 = '' OR e.client = $4)
  AND ($5::timestamptz IS NULL OR e.created_at >= $5)
  AND ($6::timestamptz IS NULL OR e.created_at < $6)
  AND ($7 = '' OR e.summary ILIKE '%' || $7 || '%' OR e.action ILIKE '%' || $7 || '%'
       OR COALESCE(NULLIF(e.actor_label, ''), s.full_name, t.name, u.identifier, '') ILIKE '%' || $7 || '%')
  AND ($8 = 0 OR e.id < $8)
ORDER BY e.id DESC
LIMIT $9`
	countActionsSQL = `SELECT action, COUNT(*)
FROM audit_events
WHERE actor_id = $1
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at < $3)
GROUP BY action
ORDER BY COUNT(*) DESC, action ASC`
	listAuditPeopleSQL = `SELECT COALESCE(e.actor_id, 0), e.actor_role,
COALESCE(NULLIF(e.actor_label, ''), s.full_name, t.name, u.identifier, 'System'),
COUNT(*)
FROM audit_events e
LEFT JOIN users u ON u.id = e.actor_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
GROUP BY e.actor_id, e.actor_role, COALESCE(NULLIF(e.actor_label, ''), s.full_name, t.name, u.identifier, 'System')
ORDER BY COUNT(*) DESC
LIMIT 200`
	deleteOldEventsSQL = `DELETE FROM audit_events WHERE created_at < $1`
)

func (s *Store) WriteEvent(ctx context.Context, event AuditEvent) error {
	client := strings.TrimSpace(event.Client)
	if client != ClientAndroid {
		client = ClientWeb
	}
	_, err := s.pool.Exec(ctx, insertEventSQL,
		event.ActorID,
		event.ActorRole,
		event.ActorLabel,
		event.Action,
		event.TargetType,
		event.TargetID,
		event.Summary,
		event.IPHash,
		event.UserAgent,
		client,
	)
	if err != nil {
		return fmt.Errorf("store: could not write the %s event: %w", event.Action, err)
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultEventPage
	}
	if limit > MaxEventPage {
		limit = MaxEventPage
	}
	rows, err := s.pool.Query(ctx, listEventsSQL,
		filter.ActorID,
		filter.Role,
		filter.Action,
		filter.Client,
		filter.From,
		filter.To,
		filter.Search,
		filter.Before,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the audit events: %w", err)
	}
	defer rows.Close()
	entries := make([]AuditEntry, 0)
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.ActorID,
			&entry.ActorRole,
			&entry.ActorLabel,
			&entry.Action,
			&entry.TargetType,
			&entry.TargetID,
			&entry.Summary,
			&entry.Client,
			&entry.UserAgent,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: could not scan an audit event: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the audit events: %w", err)
	}
	return entries, nil
}

func (s *Store) CountActions(ctx context.Context, actorID int64, from, to *time.Time) ([]ActionCount, error) {
	rows, err := s.pool.Query(ctx, countActionsSQL, actorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: could not count the actions of user %d: %w", actorID, err)
	}
	defer rows.Close()
	counts := make([]ActionCount, 0)
	for rows.Next() {
		var count ActionCount
		if err := rows.Scan(&count.Action, &count.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan an action count: %w", err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the action counts: %w", err)
	}
	return counts, nil
}

func (s *Store) ListAuditPeople(ctx context.Context) ([]AuditPerson, error) {
	rows, err := s.pool.Query(ctx, listAuditPeopleSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the audit actors: %w", err)
	}
	defer rows.Close()
	people := make([]AuditPerson, 0)
	for rows.Next() {
		var person AuditPerson
		if err := rows.Scan(&person.ID, &person.Role, &person.Label, &person.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan an audit actor: %w", err)
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the audit actors: %w", err)
	}
	return people, nil
}

func (s *Store) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, deleteOldEventsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the old audit events: %w", err)
	}
	return tag.RowsAffected(), nil
}
