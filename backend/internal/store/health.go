package store

import (
	"context"
	"fmt"
	"time"
)

const (
	ChannelWebPush = "web"
	ChannelFCM     = "fcm"

	// The reasons a notification does not arrive. They are the groups System health
	// shows, chosen so that each one points at a different thing to do about it.
	PushReasonExpired     = "expired"
	PushReasonInvalidKey  = "invalid_key"
	PushReasonRateLimited = "rate_limited"
	PushReasonNetwork     = "network"
	PushReasonPayload     = "payload"
	PushReasonOther       = "other"

	// PushUnusedAfter is how long a web push subscription may sit unused before the
	// nightly cleanup drops it. A browser that has not asked for a notification in two
	// months has almost certainly thrown the subscription away already.
	PushUnusedAfter = 60 * 24 * time.Hour
	// AlertQuietWindow is how long a dismissed alert stays quiet.
	AlertQuietWindow = 24 * time.Hour
)

// SystemEvent is one line of the health log: an email, a notification or a failing
// answer. Channel and Reason are filled in for the notifications, so that the failures
// can be grouped by where they were going and by what went wrong.
type SystemEvent struct {
	Kind    string
	OK      bool
	Detail  string
	Channel string
	Reason  string
	Message string
}

// PushBreakdown is the failure count of one channel for one reason.
type PushBreakdown struct {
	Channel string
	Reason  string
	Total   int
}

// FailingPath is one address that answered with a server error, with the message of the
// last one and when it happened.
type FailingPath struct {
	Path    string
	Total   int
	Message string
	Last    time.Time
}

// RequestVolume is how many requests the handlers answered in a period, and how many of
// them were server errors. The alerts need the volume to tell a handful of errors in a
// quiet hour from the same handful in a busy one.
type RequestVolume struct {
	Requests int
	Failures int
}

// PushVolume counts the notifications of one period. Cleaned holds the ones that failed
// because the subscription was gone and was dropped on the spot, so the alert can leave
// them out: they are not a fault anybody can fix and they never happen twice.
type PushVolume struct {
	Sent    int
	Failed  int
	Cleaned int
}

func (v PushVolume) Attempts() int {
	return v.Sent + v.Failed
}

// Share is the failure share the alert compares against its threshold. The endpoints the
// server has already cleaned up are left out of both the numerator and the denominator.
func (v PushVolume) Share() float64 {
	attempts := v.Attempts() - v.Cleaned
	failed := v.Failed - v.Cleaned
	if attempts <= 0 || failed <= 0 {
		return 0
	}
	return float64(failed) / float64(attempts)
}

const (
	insertChannelEventSQL = `INSERT INTO system_events (kind, ok, detail, channel, reason, message)
VALUES ($1, $2, $3, $4, $5, $6)`
	addRequestCountsSQL = `INSERT INTO request_counters (bucket, requests, failures, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (bucket) DO UPDATE
SET requests = request_counters.requests + EXCLUDED.requests,
    failures = request_counters.failures + EXCLUDED.failures,
    updated_at = now()`
	requestVolumeSQL = `SELECT COALESCE(SUM(requests), 0), COALESCE(SUM(failures), 0)
FROM request_counters
WHERE bucket >= $1 AND bucket < $2`
	pushBreakdownSQL = `SELECT channel, reason, COUNT(*)
FROM system_events
WHERE kind IN ('push', 'fcm') AND NOT ok AND reason <> ''
  AND created_at >= $1 AND created_at < $2
GROUP BY channel, reason
ORDER BY COUNT(*) DESC, channel, reason`
	pushVolumeSQL = `SELECT
COUNT(*) FILTER (WHERE ok),
COUNT(*) FILTER (WHERE NOT ok),
COUNT(*) FILTER (WHERE NOT ok AND reason = 'expired')
FROM system_events
WHERE kind IN ('push', 'fcm') AND created_at >= $1 AND created_at < $2`
	// The last message of each address comes from the newest row of that address, which
	// DISTINCT ON picks without a second pass over the table.
	failingPathsSQL = `WITH latest AS (
    SELECT DISTINCT ON (detail) detail, message, created_at
    FROM system_events
    WHERE kind = 'http_5xx' AND created_at >= $1 AND created_at < $2
    ORDER BY detail, created_at DESC
), totals AS (
    SELECT detail, COUNT(*) AS total
    FROM system_events
    WHERE kind = 'http_5xx' AND created_at >= $1 AND created_at < $2
    GROUP BY detail
)
SELECT totals.detail, totals.total, latest.message, latest.created_at
FROM totals
JOIN latest ON latest.detail = totals.detail
ORDER BY totals.total DESC, latest.created_at DESC
LIMIT $3`
	dismissAlertSQL = `INSERT INTO alert_dismissals (alert_key, dismissed_by, dismissed_at, quiet_until)
VALUES ($1, NULLIF($2::bigint, 0), now(), now() + make_interval(secs => $3::double precision))
ON CONFLICT (alert_key) DO UPDATE
SET dismissed_by = EXCLUDED.dismissed_by,
    dismissed_at = now(),
    quiet_until = EXCLUDED.quiet_until`
	quietAlertsSQL = `SELECT alert_key, quiet_until
FROM alert_dismissals
WHERE quiet_until > now()`
	cleanPushSubscriptionsSQL = `DELETE FROM push_subscriptions
WHERE COALESCE(last_used_at, created_at) < $1`
)

func (s *Store) WriteChannelEvent(ctx context.Context, event SystemEvent) error {
	_, err := s.pool.Exec(ctx, insertChannelEventSQL, event.Kind, event.OK, event.Detail,
		event.Channel, event.Reason, event.Message)
	if err != nil {
		return fmt.Errorf("store: could not record the %s event: %w", event.Kind, err)
	}
	return nil
}

func (s *Store) AddRequestCounts(ctx context.Context, bucket time.Time, requests, failures int) error {
	_, err := s.pool.Exec(ctx, addRequestCountsSQL, bucket, requests, failures)
	if err != nil {
		return fmt.Errorf("store: could not add the request counters: %w", err)
	}
	return nil
}

func (s *Store) RequestVolume(ctx context.Context, window Window) (RequestVolume, error) {
	var volume RequestVolume
	err := s.pool.QueryRow(ctx, requestVolumeSQL, window.From, window.To).
		Scan(&volume.Requests, &volume.Failures)
	if err != nil {
		return RequestVolume{}, fmt.Errorf("store: could not read the request volume: %w", err)
	}
	return volume, nil
}

func (s *Store) PushVolume(ctx context.Context, window Window) (PushVolume, error) {
	var volume PushVolume
	err := s.pool.QueryRow(ctx, pushVolumeSQL, window.From, window.To).
		Scan(&volume.Sent, &volume.Failed, &volume.Cleaned)
	if err != nil {
		return PushVolume{}, fmt.Errorf("store: could not read the push volume: %w", err)
	}
	return volume, nil
}

func (s *Store) PushBreakdown(ctx context.Context, window Window) ([]PushBreakdown, error) {
	rows, err := s.pool.Query(ctx, pushBreakdownSQL, window.From, window.To)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the push breakdown: %w", err)
	}
	defer rows.Close()
	kept := make([]PushBreakdown, 0)
	for rows.Next() {
		var entry PushBreakdown
		if err := rows.Scan(&entry.Channel, &entry.Reason, &entry.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan a push breakdown row: %w", err)
		}
		kept = append(kept, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the push breakdown: %w", err)
	}
	return kept, nil
}

func (s *Store) FailingPaths(ctx context.Context, window Window, limit int) ([]FailingPath, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, failingPathsSQL, window.From, window.To, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the failing paths: %w", err)
	}
	defer rows.Close()
	kept := make([]FailingPath, 0)
	for rows.Next() {
		var entry FailingPath
		if err := rows.Scan(&entry.Path, &entry.Total, &entry.Message, &entry.Last); err != nil {
			return nil, fmt.Errorf("store: could not scan a failing path: %w", err)
		}
		kept = append(kept, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the failing paths: %w", err)
	}
	return kept, nil
}

func (s *Store) DismissAlert(ctx context.Context, key string, by int64, quiet time.Duration) error {
	if quiet <= 0 {
		quiet = AlertQuietWindow
	}
	_, err := s.pool.Exec(ctx, dismissAlertSQL, key, by, quiet.Seconds())
	if err != nil {
		return fmt.Errorf("store: could not dismiss the alert %q: %w", key, err)
	}
	return nil
}

// QuietAlerts lists the alerts the super admin has set aside and the moment each one
// comes back.
func (s *Store) QuietAlerts(ctx context.Context) (map[string]time.Time, error) {
	rows, err := s.pool.Query(ctx, quietAlertsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the dismissed alerts: %w", err)
	}
	defer rows.Close()
	quiet := make(map[string]time.Time)
	for rows.Next() {
		var key string
		var until time.Time
		if err := rows.Scan(&key, &until); err != nil {
			return nil, fmt.Errorf("store: could not scan a dismissed alert: %w", err)
		}
		quiet[key] = until
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the dismissed alerts: %w", err)
	}
	return quiet, nil
}

// CleanPushSubscriptions drops the web push subscriptions nothing has used for a long
// time. A subscription the browser threw away answers every send with the same refusal,
// and leaving it in place keeps the failure share of the whole portal high.
func (s *Store) CleanPushSubscriptions(ctx context.Context, before time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, cleanPushSubscriptionsSQL, before)
	if err != nil {
		return 0, fmt.Errorf("store: could not clean the push subscriptions: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
