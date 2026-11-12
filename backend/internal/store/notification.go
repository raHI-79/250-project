package store

import (
	"context"
	"fmt"
	"time"
)

type NotificationLink struct {
	Kind string
	Ref  int64
	Date string
}

type Notification struct {
	ID        int64
	Type      string
	Title     string
	Body      string
	Link      NotificationLink
	CreatedAt time.Time
	Read      bool
}

const (
	insertDirectNotificationSQL = `INSERT INTO notifications
(recipient_user_id, type, title, body, link_kind, link_ref, link_date)
VALUES ($1, $2, $3, $4, $5, NULLIF($6::bigint, 0), NULLIF($7::text, '')::date)`
	insertBroadcastNotificationSQL = `INSERT INTO notifications
(recipient_role, recipient_batch, recipient_section, type, title, body, link_kind, link_ref, link_date)
VALUES (NULLIF($1::text, ''), NULLIF($2::text, ''), NULLIF($3::text, ''), $4, $5, $6,
$7, NULLIF($8::bigint, 0), NULLIF($9::text, '')::date)`
	notificationAudienceSQL = `WITH viewer AS (
    SELECT u.id,
           CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END AS effective_role,
           u.role AS base_role,
           s.batch AS batch,
           s.section AS section
    FROM users u
    LEFT JOIN students s ON s.user_id = u.id
    WHERE u.id = $1
)
SELECT n.id, n.type, n.title, n.body, COALESCE(n.link_kind, ''), COALESCE(n.link_ref, 0),
COALESCE(to_char(n.link_date, 'YYYY-MM-DD'), ''), n.created_at,
(EXISTS (SELECT 1 FROM notification_reads r WHERE r.user_id = v.id AND r.notification_id = n.id)
 OR n.created_at <= COALESCE(
    (SELECT m.read_before FROM notification_read_marks m WHERE m.user_id = v.id),
    '-infinity'::timestamptz)) AS is_read
FROM notifications n, viewer v
WHERE (n.recipient_user_id = v.id
    OR (n.recipient_user_id IS NULL
        AND (n.recipient_role IS NULL
             OR n.recipient_role = v.effective_role
             OR (n.recipient_role = 'student' AND v.base_role = 'student'))
        AND (n.recipient_batch IS NULL OR n.recipient_batch = v.batch)
        AND (n.recipient_section IS NULL OR n.recipient_section = v.section)))
  AND n.type <> 'message'`
	listNotificationsSQL = notificationAudienceSQL + `
ORDER BY n.created_at DESC, n.id DESC
LIMIT $2`
	countNotificationsSQL = `SELECT COUNT(*) FROM (` + notificationAudienceSQL + `
AND n.created_at > $2) matches`
	countUnreadNotificationsSQL = `SELECT COUNT(*) FROM (` + notificationAudienceSQL + `) matches
WHERE NOT matches.is_read`
	markNotificationsReadSQL = `INSERT INTO notification_reads (user_id, notification_id)
SELECT $1, matches.id FROM (` + notificationAudienceSQL + `) matches
WHERE matches.id = ANY($2::bigint[])
ON CONFLICT (user_id, notification_id) DO NOTHING`
	markAllNotificationsReadSQL = `INSERT INTO notification_read_marks (user_id, read_before)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET read_before = GREATEST(notification_read_marks.read_before, EXCLUDED.read_before),
    updated_at = now()`
)

func (s *Store) CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string,
	link NotificationLink) error {
	_, err := s.pool.Exec(ctx, insertDirectNotificationSQL,
		userID, kind, title, body, link.Kind, link.Ref, link.Date)
	if err != nil {
		return fmt.Errorf("store: could not create the notification for the user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string,
	link NotificationLink) error {
	_, err := s.pool.Exec(ctx, insertBroadcastNotificationSQL,
		role, batch, section, kind, title, body, link.Kind, link.Ref, link.Date)
	if err != nil {
		return fmt.Errorf("store: could not create the broadcast notification: %w", err)
	}
	return nil
}

func (s *Store) ListNotifications(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	rows, err := s.pool.Query(ctx, listNotificationsSQL, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the notifications of the user %d: %w", userID, err)
	}
	defer rows.Close()
	notifications := make([]Notification, 0)
	for rows.Next() {
		var notification Notification
		err := rows.Scan(
			&notification.ID,
			&notification.Type,
			&notification.Title,
			&notification.Body,
			&notification.Link.Kind,
			&notification.Link.Ref,
			&notification.Link.Date,
			&notification.CreatedAt,
			&notification.Read,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a notification: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the notifications: %w", err)
	}
	return notifications, nil
}

func (s *Store) CountNotificationsSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, countNotificationsSQL, userID, since).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: could not count the notifications of the user %d: %w", userID, err)
	}
	return total, nil
}

func (s *Store) CountUnreadNotifications(ctx context.Context, userID int64) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, countUnreadNotificationsSQL, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: could not count the unread notifications of the user %d: %w", userID, err)
	}
	return total, nil
}

func (s *Store) MarkNotificationsRead(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, markNotificationsReadSQL, userID, ids); err != nil {
		return fmt.Errorf("store: could not mark the notifications of the user %d read: %w", userID, err)
	}
	return nil
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID int64, before time.Time) error {
	if _, err := s.pool.Exec(ctx, markAllNotificationsReadSQL, userID, before); err != nil {
		return fmt.Errorf("store: could not mark every notification of the user %d read: %w", userID, err)
	}
	return nil
}
