package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

const (
	KindEmail       = "email"
	KindPush        = "push"
	KindFCM         = "fcm"
	KindClient4x    = "http_4xx"
	KindClient5x    = "http_5xx"
	KindClientCrash = "client_crash"
	MaxViewBatch    = 20
	ViewRetained    = 90 * 24 * time.Hour
)

type PageView struct {
	VisitorID  string
	SessionID  string
	Path       string
	Client     string
	Device     string
	AppVersion string
	IPHash     string
}

type DailyCount struct {
	Day      time.Time
	Client   string
	Views    int
	Visitors int
	Sessions int
}

type BackupState struct {
	LastSuccessAt *time.Time
	Note          string
}

const (
	insertViewSQL = `INSERT INTO page_views (visitor_id, session_id, path, client, device, app_version, ip_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)`
	rollUpAnalyticsSQL = `INSERT INTO analytics_daily (day, client, views, visitors, sessions, updated_at)
SELECT date_trunc('day', created_at)::date, client, COUNT(*), COUNT(DISTINCT visitor_id),
COUNT(DISTINCT NULLIF(session_id, '')), now()
FROM page_views
WHERE created_at >= $1 AND created_at < $2
GROUP BY 1, 2
ON CONFLICT (day, client) DO UPDATE
SET views = EXCLUDED.views,
    visitors = EXCLUDED.visitors,
    sessions = EXCLUDED.sessions,
    updated_at = now()`
	deleteOldViewsSQL    = `DELETE FROM page_views WHERE created_at < $1`
	insertSystemEventSQL = `INSERT INTO system_events (kind, ok, detail) VALUES ($1, $2, $3)`
	readBackupStatusSQL  = `SELECT last_success_at, note FROM backup_status WHERE id = 1`
	writeBackupStatusSQL = `INSERT INTO backup_status (id, last_success_at, note, updated_at)
VALUES (1, $1, $2, now())
ON CONFLICT (id) DO UPDATE
SET last_success_at = EXCLUDED.last_success_at, note = EXCLUDED.note, updated_at = now()`
	listDailySQL = `SELECT day, client, views, visitors, sessions
FROM analytics_daily
WHERE day >= $1 AND day < $2
ORDER BY day ASC, client ASC`
)

func (s *Store) RecordViews(ctx context.Context, views []PageView) error {
	if len(views) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, view := range views {
		client := view.Client
		if client != ClientAndroid {
			client = ClientWeb
		}
		batch.Queue(insertViewSQL, view.VisitorID, view.SessionID, view.Path, client,
			view.Device, view.AppVersion, view.IPHash)
	}
	if err := s.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("store: could not record the page views: %w", err)
	}
	return nil
}

func (s *Store) RollUpAnalytics(ctx context.Context, from, to time.Time) error {
	if _, err := s.pool.Exec(ctx, rollUpAnalyticsSQL, from, to); err != nil {
		return fmt.Errorf("store: could not roll up the page views: %w", err)
	}
	return nil
}

func (s *Store) DeleteOldViews(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, deleteOldViewsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the old page views: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Store) WriteSystemEvent(ctx context.Context, kind string, ok bool, detail string) error {
	if _, err := s.pool.Exec(ctx, insertSystemEventSQL, kind, ok, strings.TrimSpace(detail)); err != nil {
		return fmt.Errorf("store: could not record the %s event: %w", kind, err)
	}
	return nil
}

func (s *Store) BackupStatus(ctx context.Context) (BackupState, error) {
	var state BackupState
	err := s.pool.QueryRow(ctx, readBackupStatusSQL).Scan(&state.LastSuccessAt, &state.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return BackupState{}, nil
	}
	if err != nil {
		return BackupState{}, fmt.Errorf("store: could not read the backup status: %w", err)
	}
	return state, nil
}

func (s *Store) SetBackupStatus(ctx context.Context, at time.Time, note string) error {
	if _, err := s.pool.Exec(ctx, writeBackupStatusSQL, at, strings.TrimSpace(note)); err != nil {
		return fmt.Errorf("store: could not record the backup status: %w", err)
	}
	return nil
}

func (s *Store) ListDailyAnalytics(ctx context.Context, from, to time.Time) ([]DailyCount, error) {
	rows, err := s.pool.Query(ctx, listDailySQL, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the daily analytics: %w", err)
	}
	defer rows.Close()
	counts := make([]DailyCount, 0)
	for rows.Next() {
		var count DailyCount
		if err := rows.Scan(&count.Day, &count.Client, &count.Views, &count.Visitors, &count.Sessions); err != nil {
			return nil, fmt.Errorf("store: could not scan a daily analytics row: %w", err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the daily analytics: %w", err)
	}
	return counts, nil
}
