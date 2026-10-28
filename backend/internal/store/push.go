package store

import (
	"context"
	"fmt"
	"time"
)

type NotificationRecipient struct {
	UserID int64
	Role   string
}

type PushSubscription struct {
	ID        int64
	UserID    int64
	Endpoint  string
	P256dh    string
	Auth      string
	UserAgent string
}

const (
	savePushSubscriptionSQL = `INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (endpoint) DO UPDATE
SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth,
    user_agent = EXCLUDED.user_agent
RETURNING id`
	deletePushSubscriptionSQL = `DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2`
	dropPushSubscriptionSQL   = `DELETE FROM push_subscriptions WHERE endpoint = $1`
	listPushSubscriptionsSQL  = `SELECT id, user_id, endpoint, p256dh, auth, COALESCE(user_agent, '')
FROM push_subscriptions
WHERE user_id = ANY($1::bigint[])
ORDER BY id`
	touchPushSubscriptionSQL  = `UPDATE push_subscriptions SET last_used_at = $2 WHERE endpoint = $1`
	countPushSubscriptionsSQL = `SELECT COUNT(*) FROM push_subscriptions WHERE user_id = $1`
	pushRoleSQL               = `SELECT CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.id = $1`
	pushAudienceRolesSQL = `SELECT u.id,
       CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END AS effective_role
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND u.role <> 'admin'
  AND ($1::text = ''
       OR $1::text = CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END
       OR ($1::text = 'student' AND u.role = 'student'))
  AND ($2::text = '' OR s.batch = $2::text)
  AND ($3::text = '' OR s.section = $3::text)
ORDER BY u.id`
	pushAudienceSQL = `SELECT u.id
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND u.role <> 'admin'
  AND ($1::text = ''
       OR $1::text = CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END
       OR ($1::text = 'student' AND u.role = 'student'))
  AND ($2::text = '' OR s.batch = $2::text)
  AND ($3::text = '' OR s.section = $3::text)
ORDER BY u.id`
)

func (s *Store) SavePushSubscription(ctx context.Context, subscription PushSubscription) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, savePushSubscriptionSQL,
		subscription.UserID,
		subscription.Endpoint,
		subscription.P256dh,
		subscription.Auth,
		subscription.UserAgent,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: could not save the push subscription: %w", err)
	}
	return id, nil
}

func (s *Store) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error {
	if _, err := s.pool.Exec(ctx, deletePushSubscriptionSQL, endpoint, userID); err != nil {
		return fmt.Errorf("store: could not delete the push subscription: %w", err)
	}
	return nil
}

func (s *Store) DropPushSubscription(ctx context.Context, endpoint string) error {
	if _, err := s.pool.Exec(ctx, dropPushSubscriptionSQL, endpoint); err != nil {
		return fmt.Errorf("store: could not drop the push subscription: %w", err)
	}
	return nil
}

func (s *Store) ListPushSubscriptions(ctx context.Context, userIDs []int64) ([]PushSubscription, error) {
	if len(userIDs) == 0 {
		return []PushSubscription{}, nil
	}
	rows, err := s.pool.Query(ctx, listPushSubscriptionsSQL, userIDs)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the push subscriptions: %w", err)
	}
	defer rows.Close()
	subscriptions := make([]PushSubscription, 0)
	for rows.Next() {
		var subscription PushSubscription
		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.Endpoint,
			&subscription.P256dh,
			&subscription.Auth,
			&subscription.UserAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a push subscription: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the push subscriptions: %w", err)
	}
	return subscriptions, nil
}

func (s *Store) TouchPushSubscription(ctx context.Context, endpoint string, at time.Time) error {
	if _, err := s.pool.Exec(ctx, touchPushSubscriptionSQL, endpoint, at); err != nil {
		return fmt.Errorf("store: could not mark the push subscription used: %w", err)
	}
	return nil
}

func (s *Store) ListNotificationRecipients(ctx context.Context, role, batch, section string) ([]int64, error) {
	return s.userIDs(ctx, pushAudienceSQL, role, batch, section)
}

func (s *Store) ListNotificationAudience(ctx context.Context, role, batch,
	section string) ([]NotificationRecipient, error) {
	rows, err := s.pool.Query(ctx, pushAudienceRolesSQL, role, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the push audience: %w", err)
	}
	defer rows.Close()
	audience := make([]NotificationRecipient, 0)
	for rows.Next() {
		var recipient NotificationRecipient
		if err := rows.Scan(&recipient.UserID, &recipient.Role); err != nil {
			return nil, fmt.Errorf("store: could not scan a push recipient: %w", err)
		}
		audience = append(audience, recipient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the push audience: %w", err)
	}
	return audience, nil
}

func (s *Store) NotificationRole(ctx context.Context, userID int64) (string, error) {
	var role string
	if err := s.pool.QueryRow(ctx, pushRoleSQL, userID).Scan(&role); err != nil {
		return "", fmt.Errorf("store: could not read the role of an account: %w", err)
	}
	return role, nil
}

func (s *Store) CountPushSubscriptions(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, countPushSubscriptionsSQL, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: could not count the push subscriptions: %w", err)
	}
	return count, nil
}
