package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type crashViews struct {
	events []store.SystemEvent
}

func (c *crashViews) RecordViews(ctx context.Context, views []store.PageView) error {
	return nil
}

func (c *crashViews) SetBackupStatus(ctx context.Context, at time.Time, note string) error {
	return nil
}

func (c *crashViews) WriteChannelEvent(ctx context.Context, event store.SystemEvent) error {
	c.events = append(c.events, event)
	return nil
}

func TestCrashReportKeepsTheRouteWithoutItsQuery(t *testing.T) {
	views := &crashViews{}
	body := `{"name":"TypeError","message":"Cannot read properties of null","route":"/student/upcoming-routine?date=2026-11-14#top","client":"app","version":"2.7.3"}`
	request := httptest.NewRequest(http.MethodPost, "/api/client-log", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	NewAnalytics(nil, views).Crash()(rec, request)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("the report answered %d: %s", rec.Code, rec.Body.String())
	}
	if len(views.events) != 1 {
		t.Fatalf("recorded %d events, want 1", len(views.events))
	}
	event := views.events[0]
	if event.Kind != store.KindClientCrash || event.OK {
		t.Fatalf("recorded %+v", event)
	}
	if event.Detail != "/student/upcoming-routine" {
		t.Fatalf("the route kept its query: %q", event.Detail)
	}
	if event.Channel != "app" || event.Message != "TypeError: Cannot read properties of null" {
		t.Fatalf("recorded %+v", event)
	}
}

func TestCrashReportNeedsSomethingToSay(t *testing.T) {
	views := &crashViews{}
	request := httptest.NewRequest(http.MethodPost, "/api/client-log", strings.NewReader(`{"route":"/"}`))
	request.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	NewAnalytics(nil, views).Crash()(rec, request)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty report answered %d", rec.Code)
	}
	if len(views.events) != 0 {
		t.Fatalf("an empty report was recorded")
	}
}
