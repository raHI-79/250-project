package store

import (
	"context"
	"fmt"
	"time"
)

type Notification struct {
	ID        int64
	Type      string
	Title     string
	Body      string
	CreatedAt time.Time
}

const (
	insertDirectNotificationSQL = `INSERT INTO notifications (recipient_user_id, type, title, body)
VALUES ($1, $2, $3, $4)`
	insertBroadcastNotificationSQL = `INSERT INTO notifications (recipient_role, recipient_batch, recipient_section, type, title, body)
VALUES (NULLIF($1::text, ''), NULLIF($2::text, ''), NULLIF($3::text, ''), $4, $5, $6)`
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
SELECT n.id, n.type, n.title, n.body, n.created_at
FROM notifications n, viewer v
WHERE (n.recipient_user_id = v.id
    OR (n.recipient_user_id IS NULL
        AND (n.recipient_role IS NULL
             OR n.recipient_role = v.effective_role
             OR (n.recipient_role = 'student' AND v.base_role = 'student'))
        AND (n.recipient_batch IS NULL OR n.recipient_batch = v.batch)
        AND (n.recipient_section IS NULL OR n.recipient_section = v.section)))`
	listNotificationsSQL = notificationAudienceSQL + `
ORDER BY n.created_at DESC, n.id DESC
LIMIT $2`
	countNotificationsSQL = `SELECT COUNT(*) FROM (` + notificationAudienceSQL + `
AND n.created_at > $2) matches`
)

func (s *Store) CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string) error {
	if _, err := s.pool.Exec(ctx, insertDirectNotificationSQL, userID, kind, title, body); err != nil {
		return fmt.Errorf("store: could not create the notification for the user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string) error {
	_, err := s.pool.Exec(ctx, insertBroadcastNotificationSQL, role, batch, section, kind, title, body)
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
			&notification.CreatedAt,
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
