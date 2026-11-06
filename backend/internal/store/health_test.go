package store

import (
	"testing"
	"time"
)

func healthWindow(now time.Time) Window {
	return Window{From: now.Add(-time.Hour), To: now.Add(time.Hour)}
}

func TestTheRequestCountersAddUpPerBucket(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	now := time.Now()
	bucket := now.Truncate(time.Minute)
	if err := store.AddRequestCounts(ctx, bucket, 40, 2); err != nil {
		t.Fatalf("AddRequestCounts() returned an unexpected error: %v", err)
	}
	// A second flush of the same minute adds to the row instead of replacing it.
	if err := store.AddRequestCounts(ctx, bucket, 10, 1); err != nil {
		t.Fatalf("AddRequestCounts() returned an unexpected error: %v", err)
	}
	if err := store.AddRequestCounts(ctx, bucket.Add(-time.Minute), 5, 0); err != nil {
		t.Fatalf("AddRequestCounts() returned an unexpected error: %v", err)
	}
	volume, err := store.RequestVolume(ctx, healthWindow(now))
	if err != nil {
		t.Fatalf("RequestVolume() returned an unexpected error: %v", err)
	}
	if volume.Requests != 55 || volume.Failures != 3 {
		t.Errorf("volume = %+v, want 55 requests and 3 failures", volume)
	}
	outside, err := store.RequestVolume(ctx, Window{
		From: now.Add(-48 * time.Hour),
		To:   now.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("RequestVolume() returned an unexpected error: %v", err)
	}
	if outside.Requests != 0 {
		t.Errorf("a window of its own = %+v, want nothing", outside)
	}
}

func TestThePushBreakdownGroupsByChannelAndReason(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	now := time.Now()
	events := []SystemEvent{
		{Kind: KindPush, OK: true, Channel: ChannelWebPush},
		{Kind: KindPush, OK: true, Channel: ChannelWebPush},
		{Kind: KindPush, Channel: ChannelWebPush, Reason: PushReasonExpired},
		{Kind: KindPush, Channel: ChannelWebPush, Reason: PushReasonExpired},
		{Kind: KindPush, Channel: ChannelWebPush, Reason: PushReasonNetwork},
		{Kind: KindFCM, OK: true, Channel: ChannelFCM},
		{Kind: KindFCM, Channel: ChannelFCM, Reason: PushReasonExpired},
		{Kind: KindFCM, Channel: ChannelFCM, Reason: PushReasonInvalidKey},
	}
	for _, event := range events {
		if err := store.WriteChannelEvent(ctx, event); err != nil {
			t.Fatalf("WriteChannelEvent() returned an unexpected error: %v", err)
		}
	}
	breakdown, err := store.PushBreakdown(ctx, healthWindow(now))
	if err != nil {
		t.Fatalf("PushBreakdown() returned an unexpected error: %v", err)
	}
	totals := make(map[string]int, len(breakdown))
	for _, entry := range breakdown {
		totals[entry.Channel+"/"+entry.Reason] = entry.Total
	}
	want := map[string]int{
		ChannelWebPush + "/" + PushReasonExpired: 2,
		ChannelWebPush + "/" + PushReasonNetwork: 1,
		ChannelFCM + "/" + PushReasonExpired:     1,
		ChannelFCM + "/" + PushReasonInvalidKey:  1,
	}
	if len(totals) != len(want) {
		t.Fatalf("the breakdown = %+v, want %d rows", breakdown, len(want))
	}
	for key, count := range want {
		if totals[key] != count {
			t.Errorf("%s = %d, want %d", key, totals[key], count)
		}
	}
	// A delivered notification never appears in the breakdown.
	for _, entry := range breakdown {
		if entry.Reason == "" {
			t.Errorf("a row with no reason reached the breakdown: %+v", entry)
		}
	}
	volume, err := store.PushVolume(ctx, healthWindow(now))
	if err != nil {
		t.Fatalf("PushVolume() returned an unexpected error: %v", err)
	}
	if volume.Sent != 3 || volume.Failed != 5 || volume.Cleaned != 3 {
		t.Errorf("volume = %+v, want 3 sent, 5 failed and 3 of them cleaned up", volume)
	}
	if share := volume.Share(); share < 0.39 || share > 0.41 {
		t.Errorf("Share() = %v, want about 0.4 once the cleaned ones are left out", share)
	}
}

func TestTheFailingPathsCarryTheirCountAndTheLastMessage(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	now := time.Now()
	events := []SystemEvent{
		{Kind: KindClient5x, Detail: "GET /api/cr/upcoming", Message: "an older message"},
		{Kind: KindClient5x, Detail: "GET /api/cr/upcoming", Message: "the newest message"},
		{Kind: KindClient5x, Detail: "POST /api/cr/items", Message: "the only message"},
		{Kind: KindClient4x, Detail: "GET /api/nope"},
	}
	for _, event := range events {
		if err := store.WriteChannelEvent(ctx, event); err != nil {
			t.Fatalf("WriteChannelEvent() returned an unexpected error: %v", err)
		}
	}
	paths, err := store.FailingPaths(ctx, healthWindow(now), 10)
	if err != nil {
		t.Fatalf("FailingPaths() returned an unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %+v, want the two addresses that answered with a server error", paths)
	}
	if paths[0].Path != "GET /api/cr/upcoming" || paths[0].Total != 2 {
		t.Errorf("the first row = %+v, want the address with two errors on top", paths[0])
	}
	if paths[0].Message != "the newest message" {
		t.Errorf("the message = %q, want the last one", paths[0].Message)
	}
	if paths[0].Last.IsZero() {
		t.Error("the row carries no time")
	}
	if paths[1].Path != "POST /api/cr/items" || paths[1].Total != 1 {
		t.Errorf("the second row = %+v, want the address with one error", paths[1])
	}
	short, err := store.FailingPaths(ctx, healthWindow(now), 1)
	if err != nil {
		t.Fatalf("FailingPaths() returned an unexpected error: %v", err)
	}
	if len(short) != 1 {
		t.Errorf("the limit was ignored: %+v", short)
	}
}

func TestADismissedAlertComesBackAfterItsWindow(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	if err := store.DismissAlert(ctx, "server_errors", chief, AlertQuietWindow); err != nil {
		t.Fatalf("DismissAlert() returned an unexpected error: %v", err)
	}
	quiet, err := store.QuietAlerts(ctx)
	if err != nil {
		t.Fatalf("QuietAlerts() returned an unexpected error: %v", err)
	}
	until, ok := quiet["server_errors"]
	if !ok {
		t.Fatalf("the dismissal was not stored: %+v", quiet)
	}
	if ahead := time.Until(until); ahead < 23*time.Hour || ahead > 25*time.Hour {
		t.Errorf("the alert stays quiet for %v, want about a day", ahead)
	}
	// Dismissing it again moves the window instead of failing on the key.
	if err := store.DismissAlert(ctx, "server_errors", chief, time.Minute); err != nil {
		t.Fatalf("a second DismissAlert() returned an unexpected error: %v", err)
	}
	again, err := store.QuietAlerts(ctx)
	if err != nil {
		t.Fatalf("QuietAlerts() returned an unexpected error: %v", err)
	}
	if ahead := time.Until(again["server_errors"]); ahead > 2*time.Minute {
		t.Errorf("the window was not moved, it still runs for %v", ahead)
	}
	// A window of zero or less would silence an alert for ever or not at all, so it
	// falls back to the standing day.
	if err := store.DismissAlert(ctx, "email_failures", chief, 0); err != nil {
		t.Fatalf("DismissAlert() returned an unexpected error: %v", err)
	}
	standing, err := store.QuietAlerts(ctx)
	if err != nil {
		t.Fatalf("QuietAlerts() returned an unexpected error: %v", err)
	}
	if ahead := time.Until(standing["email_failures"]); ahead < 23*time.Hour {
		t.Errorf("a window of zero runs for %v, want the standing day", ahead)
	}
	// A window that has run out leaves nothing behind for the dashboard to hide.
	expired := `INSERT INTO alert_dismissals (alert_key, dismissed_by, quiet_until)
VALUES ('push_failures', $1, now() - INTERVAL '1 hour')`
	if _, err := pool.Exec(ctx, expired, chief); err != nil {
		t.Fatalf("could not insert an expired dismissal: %v", err)
	}
	final, err := store.QuietAlerts(ctx)
	if err != nil {
		t.Fatalf("QuietAlerts() returned an unexpected error: %v", err)
	}
	if _, ok := final["push_failures"]; ok {
		t.Error("a dismissal whose window has run out was still listed as quiet")
	}
}

func TestTheNightlyCleanupDropsTheSubscriptionsNobodyUses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	owner := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	rows := []struct {
		endpoint string
		used     string
		created  string
	}{
		{endpoint: "https://push.example/fresh", used: "now()", created: "now()"},
		{endpoint: "https://push.example/stale", used: "now() - INTERVAL '70 days'", created: "now() - INTERVAL '90 days'"},
		{endpoint: "https://push.example/never-used", used: "NULL", created: "now() - INTERVAL '90 days'"},
		{endpoint: "https://push.example/new-but-unused", used: "NULL", created: "now()"},
	}
	for _, row := range rows {
		query := `INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, last_used_at, created_at)
VALUES ($1, $2, 'public-key', 'auth-secret', ` + row.used + `, ` + row.created + `)`
		if _, err := pool.Exec(ctx, query, owner, row.endpoint); err != nil {
			t.Fatalf("could not insert the subscription %s: %v", row.endpoint, err)
		}
	}
	dropped, err := store.CleanPushSubscriptions(ctx, time.Now().Add(-PushUnusedAfter))
	if err != nil {
		t.Fatalf("CleanPushSubscriptions() returned an unexpected error: %v", err)
	}
	if dropped != 2 {
		t.Errorf("dropped %d subscriptions, want the stale one and the one never used", dropped)
	}
	var left []string
	kept, err := pool.Query(ctx, `SELECT endpoint FROM push_subscriptions ORDER BY endpoint`)
	if err != nil {
		t.Fatalf("could not read the subscriptions back: %v", err)
	}
	defer kept.Close()
	for kept.Next() {
		var endpoint string
		if err := kept.Scan(&endpoint); err != nil {
			t.Fatalf("could not scan a subscription: %v", err)
		}
		left = append(left, endpoint)
	}
	if len(left) != 2 {
		t.Fatalf("the cleanup left %v, want the fresh one and the new one", left)
	}
	if left[0] != "https://push.example/fresh" || left[1] != "https://push.example/new-but-unused" {
		t.Errorf("the cleanup left %v", left)
	}
}
