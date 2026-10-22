package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakePublicStore struct {
	stats store.PublicStats
	err   error
	calls int
}

func (f *fakePublicStore) PublicStats(context.Context) (store.PublicStats, error) {
	f.calls++
	if f.err != nil {
		return store.PublicStats{}, f.err
	}
	return f.stats, nil
}

func publicCall(t *testing.T, public *Public) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	public.Stats()(rec, httptest.NewRequest(http.MethodGet, "/api/public/stats", nil))
	return rec
}

func TestPublicStatsCachesForFiveMinutes(t *testing.T) {
	numbers := &fakePublicStore{stats: store.PublicStats{Students: 240, Teachers: 18, Courses: 36, Batches: 4}}
	clock := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	public := NewPublic(numbers, func() time.Time { return clock })
	rec := publicCall(t, public)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	want := map[string]any{
		"students": float64(240),
		"teachers": float64(18),
		"courses":  float64(36),
		"batches":  float64(4),
	}
	for name, value := range want {
		if got := payload[name]; got != value {
			t.Errorf("%s = %v, want %v", name, got, value)
		}
	}
	numbers.stats = store.PublicStats{Students: 999}
	clock = clock.Add(4 * time.Minute)
	cached := decodeBody(t, publicCall(t, public))
	if cached["students"] != float64(240) {
		t.Errorf("cached students = %v, want 240", cached["students"])
	}
	if numbers.calls != 1 {
		t.Errorf("store calls = %d, want 1", numbers.calls)
	}
	clock = clock.Add(StatsCacheTTL)
	refreshed := decodeBody(t, publicCall(t, public))
	if refreshed["students"] != float64(999) {
		t.Errorf("refreshed students = %v, want 999", refreshed["students"])
	}
	if numbers.calls != 2 {
		t.Errorf("store calls after the cache expired = %d, want 2", numbers.calls)
	}
}

func TestPublicStatsReportsAFailingStore(t *testing.T) {
	silenceLog(t)
	numbers := &fakePublicStore{err: errors.New("database is down")}
	rec := publicCall(t, NewPublic(numbers, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
