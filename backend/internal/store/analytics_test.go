package store

import (
	"testing"
	"time"
)

func TestPageViewsRollUpPerDayAndClient(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	views := []PageView{
		{VisitorID: "v1", SessionID: "s1", Path: "/student/today", Client: ClientWeb, Device: "desktop"},
		{VisitorID: "v1", SessionID: "s1", Path: "/student/exams", Client: ClientWeb, Device: "desktop"},
		{VisitorID: "v2", SessionID: "s2", Path: "/student/today", Client: ClientAndroid, Device: "phone", AppVersion: "2.5.0"},
	}
	if err := store.RecordViews(ctx, views); err != nil {
		t.Fatalf("RecordViews() returned an unexpected error: %v", err)
	}
	var stored int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM page_views").Scan(&stored); err != nil {
		t.Fatalf("could not count the page views: %v", err)
	}
	if stored != 3 {
		t.Fatalf("stored %d page views, want 3", stored)
	}
	now := time.Now()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := store.RollUpAnalytics(ctx, day.AddDate(0, 0, -1), day.AddDate(0, 0, 1)); err != nil {
		t.Fatalf("RollUpAnalytics() returned an unexpected error: %v", err)
	}
	daily, err := store.ListDailyAnalytics(ctx, day.AddDate(0, 0, -1), day.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("ListDailyAnalytics() returned an unexpected error: %v", err)
	}
	if len(daily) != 2 {
		t.Fatalf("got %d daily rows, want one per client", len(daily))
	}
	byClient := map[string]DailyCount{}
	for _, row := range daily {
		byClient[row.Client] = row
	}
	if byClient[ClientWeb].Views != 2 || byClient[ClientWeb].Visitors != 1 {
		t.Errorf("web row = %+v, want 2 views by 1 visitor", byClient[ClientWeb])
	}
	if byClient[ClientAndroid].Views != 1 || byClient[ClientAndroid].Sessions != 1 {
		t.Errorf("app row = %+v, want 1 view in 1 session", byClient[ClientAndroid])
	}
	if err := store.RollUpAnalytics(ctx, day.AddDate(0, 0, -1), day.AddDate(0, 0, 1)); err != nil {
		t.Fatalf("RollUpAnalytics() is not repeatable: %v", err)
	}
	again, err := store.ListDailyAnalytics(ctx, day.AddDate(0, 0, -1), day.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("ListDailyAnalytics() returned an unexpected error: %v", err)
	}
	if len(again) != 2 {
		t.Errorf("got %d daily rows after a second roll-up, want 2", len(again))
	}
}

func TestOldPageViewsAreDeleted(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	view := PageView{VisitorID: "v1", Path: "/", Client: ClientWeb}
	if err := store.RecordViews(ctx, []PageView{view}); err != nil {
		t.Fatalf("RecordViews() returned an unexpected error: %v", err)
	}
	old := "UPDATE page_views SET created_at = now() - INTERVAL '120 days'"
	if _, err := pool.Exec(ctx, old); err != nil {
		t.Fatalf("could not age the page view: %v", err)
	}
	dropped, err := store.DeleteOldViews(ctx, time.Now().Add(-ViewRetained))
	if err != nil {
		t.Fatalf("DeleteOldViews() returned an unexpected error: %v", err)
	}
	if dropped != 1 {
		t.Errorf("deleted %d page views, want 1", dropped)
	}
}

func TestSystemEventsAndBackupStatusAreStored(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	if err := store.WriteSystemEvent(ctx, KindEmail, true, "Password reset code"); err != nil {
		t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
	}
	if err := store.WriteSystemEvent(ctx, KindEmail, false, "Password reset code"); err != nil {
		t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
	}
	state, err := store.BackupStatus(ctx)
	if err != nil {
		t.Fatalf("BackupStatus() returned an unexpected error: %v", err)
	}
	if state.LastSuccessAt != nil {
		t.Errorf("the fresh backup status carries %v, want no time yet", state.LastSuccessAt)
	}
	at := time.Now().Truncate(time.Second)
	if err := store.SetBackupStatus(ctx, at, "nightly"); err != nil {
		t.Fatalf("SetBackupStatus() returned an unexpected error: %v", err)
	}
	state, err = store.BackupStatus(ctx)
	if err != nil {
		t.Fatalf("BackupStatus() returned an unexpected error: %v", err)
	}
	if state.LastSuccessAt == nil || !state.LastSuccessAt.Equal(at) {
		t.Errorf("backup time = %v, want %v", state.LastSuccessAt, at)
	}
	if state.Note != "nightly" {
		t.Errorf("note = %q, want nightly", state.Note)
	}
}
