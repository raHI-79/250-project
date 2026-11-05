package store

import (
	"context"
	"fmt"
	"time"
)

type DataCounts struct {
	Views         int64
	Events        int64
	Notifications int64
	Sessions      int64
}

const (
	deleteOldNotificationsSQL = `DELETE FROM notifications WHERE created_at < $1`
	deleteOldSessionRowsSQL   = `DELETE FROM user_sessions
WHERE revoked_at IS NOT NULL AND revoked_at < $1`
	dataCountsSQL = `SELECT
(SELECT COUNT(*) FROM page_views),
(SELECT COUNT(*) FROM audit_events),
(SELECT COUNT(*) FROM notifications),
(SELECT COUNT(*) FROM user_sessions WHERE revoked_at IS NULL AND expires_at > now())`
)

func (s *Store) DeleteOldNotifications(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, deleteOldNotificationsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the old notifications: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) DeleteClosedSessions(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, deleteOldSessionRowsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the closed sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) DataCounts(ctx context.Context) (DataCounts, error) {
	var counts DataCounts
	err := s.pool.QueryRow(ctx, dataCountsSQL).Scan(&counts.Views, &counts.Events,
		&counts.Notifications, &counts.Sessions)
	if err != nil {
		return DataCounts{}, fmt.Errorf("store: could not count the stored rows: %w", err)
	}
	return counts, nil
}
