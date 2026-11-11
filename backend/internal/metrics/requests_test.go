package metrics

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/store"
)

type fakeSink struct {
	mu     sync.Mutex
	events []store.SystemEvent
}

func (f *fakeSink) WriteSystemEvent(ctx context.Context, kind string, ok bool, detail string) error {
	return f.WriteChannelEvent(ctx, store.SystemEvent{Kind: kind, OK: ok, Detail: detail})
}

func (f *fakeSink) WriteChannelEvent(ctx context.Context, event store.SystemEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
	return nil
}

func (f *fakeSink) of(kind string) []store.SystemEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	kept := make([]store.SystemEvent, 0, len(f.events))
	for _, event := range f.events {
		if event.Kind == kind {
			kept = append(kept, event)
		}
	}
	return kept
}

type fakeCounterStore struct {
	mu       sync.Mutex
	buckets  []time.Time
	requests int
	failures int
	err      error
}

func (f *fakeCounterStore) AddRequestCounts(ctx context.Context, bucket time.Time,
	requests, failures int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.buckets = append(f.buckets, bucket)
	f.requests += requests
	f.failures += failures
	return nil
}

func silence(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func answer(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
}

func maintenanceAnswer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(MaintenanceHeader, "on")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"The portal is closed for maintenance."}`))
	})
}

func TestAServerErrorFromAHandlerIsCounted(t *testing.T) {
	silence(t)
	sink := &fakeSink{}
	counts := &fakeCounterStore{}
	counter := NewCounter(counts)
	past := time.Now().Add(-2 * SettleWindow)
	handler := responsesStartedAt(past, sink, counter)(answer(http.StatusInternalServerError,
		`{"error":"Something went wrong. Please try again."}`))
	request := httptest.NewRequest(http.MethodGet, "/api/cr/upcoming", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	errors := sink.of(store.KindClient5x)
	if len(errors) != 1 {
		t.Fatalf("recorded %d server errors, want one", len(errors))
	}
	if errors[0].Detail != "GET /api/cr/upcoming" {
		t.Errorf("the detail = %q, want the method and the path", errors[0].Detail)
	}
	if errors[0].Message != "Something went wrong. Please try again." {
		t.Errorf("the message = %q, want the sentence the handler answered with",
			errors[0].Message)
	}
	counter.Flush(context.Background())
	if counts.requests != 1 || counts.failures != 1 {
		t.Errorf("counted %d requests and %d failures, want one of each",
			counts.requests, counts.failures)
	}
}

func TestTheCountersLeaveOutWhatIsNotAHandlerFault(t *testing.T) {
	silence(t)
	past := time.Now().Add(-2 * SettleWindow)
	tests := []struct {
		name        string
		handler     http.Handler
		path        string
		agent       string
		started     time.Time
		ended       error
		wantErrors  int
		wantCounted int
		wantFailed  int
	}{
		{
			name:        "a maintenance answer",
			handler:     maintenanceAnswer(),
			path:        "/api/cr/upcoming",
			started:     past,
			wantErrors:  0,
			wantCounted: 1,
			wantFailed:  0,
		},
		{
			name:        "a platform health probe",
			handler:     answer(http.StatusServiceUnavailable, `{"error":"not ready"}`),
			path:        "/api/health",
			started:     past,
			wantErrors:  0,
			wantCounted: 0,
			wantFailed:  0,
		},
		{
			name:        "a probe by its user agent",
			handler:     answer(http.StatusInternalServerError, `{"error":"boom"}`),
			path:        "/api/me",
			agent:       "Render/1.0 health",
			started:     past,
			wantErrors:  0,
			wantCounted: 0,
			wantFailed:  0,
		},
		{
			name:        "the first minute after a start",
			handler:     answer(http.StatusInternalServerError, `{"error":"the pool is not ready"}`),
			path:        "/api/me",
			started:     time.Now(),
			wantErrors:  0,
			wantCounted: 1,
			wantFailed:  0,
		},
		{
			name:        "a request the caller got wrong",
			handler:     answer(http.StatusBadRequest, `{"error":"Pick a date."}`),
			path:        "/api/cr/upcoming",
			started:     past,
			wantErrors:  0,
			wantCounted: 1,
			wantFailed:  0,
		},
		{
			name:        "a request the caller abandoned",
			handler:     answer(http.StatusInternalServerError, `{"error":"Something went wrong. Please try again."}`),
			path:        "/api/students/today",
			started:     past,
			ended:       context.Canceled,
			wantErrors:  0,
			wantCounted: 1,
			wantFailed:  0,
		},
		{
			name:        "a request that ran out of time",
			handler:     answer(http.StatusInternalServerError, `{"error":"Something went wrong. Please try again."}`),
			path:        "/api/students/today",
			started:     past,
			ended:       context.DeadlineExceeded,
			wantErrors:  1,
			wantCounted: 1,
			wantFailed:  1,
		},
		{
			name:        "a real server error",
			handler:     answer(http.StatusInternalServerError, `{"error":"boom"}`),
			path:        "/api/me",
			started:     past,
			wantErrors:  1,
			wantCounted: 1,
			wantFailed:  1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink := &fakeSink{}
			counts := &fakeCounterStore{}
			counter := NewCounter(counts)
			handler := responsesStartedAt(test.started, sink, counter)(test.handler)
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.agent != "" {
				request.Header.Set("User-Agent", test.agent)
			}
			if test.ended != nil {
				request = request.WithContext(endedContext{Context: request.Context(), err: test.ended})
			}
			handler.ServeHTTP(httptest.NewRecorder(), request)
			if got := len(sink.of(store.KindClient5x)); got != test.wantErrors {
				t.Errorf("recorded %d server errors, want %d", got, test.wantErrors)
			}
			counter.Flush(context.Background())
			if counts.requests != test.wantCounted {
				t.Errorf("counted %d requests, want %d", counts.requests, test.wantCounted)
			}
			if counts.failures != test.wantFailed {
				t.Errorf("counted %d failures, want %d", counts.failures, test.wantFailed)
			}
		})
	}
}

type endedContext struct {
	context.Context
	err error
}

func (c endedContext) Err() error { return c.err }

func TestARequestThatWentWrongIsStillCountedAsAClientError(t *testing.T) {
	silence(t)
	sink := &fakeSink{}
	handler := responsesStartedAt(time.Now().Add(-2*SettleWindow), sink, nil)(
		answer(http.StatusNotFound, `{"error":"Route not found"}`))
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/nope", nil))
	client := sink.of(store.KindClient4x)
	if len(client) != 1 || client[0].Detail != "GET /api/nope" {
		t.Errorf("recorded %+v, want the one client error", client)
	}
}

func TestTheProbeRuleNamesThePlatformChecks(t *testing.T) {
	probes := []struct {
		path  string
		agent string
	}{
		{path: "/api/health"},
		{path: "/healthz"},
		{path: "/api/healthz"},
		{path: "/health"},
		{path: "/API/Health"},
		{path: "/api/me", agent: "kube-probe/1.29"},
		{path: "/api/me", agent: "Render/2 uptime"},
		{path: "/api/me", agent: "ELB-HealthChecker/2.0"},
	}
	for _, probe := range probes {
		request := httptest.NewRequest(http.MethodGet, probe.path, nil)
		if probe.agent != "" {
			request.Header.Set("User-Agent", probe.agent)
		}
		if !IsProbe(request) {
			t.Errorf("%s with the agent %q was not read as a probe", probe.path, probe.agent)
		}
	}
	ordinary := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ordinary.Header.Set("User-Agent", "Mozilla/5.0")
	if IsProbe(ordinary) {
		t.Error("an ordinary request was read as a probe")
	}
}

func TestTheCounterWritesOneRowPerFlush(t *testing.T) {
	silence(t)
	counts := &fakeCounterStore{}
	counter := NewCounter(counts)
	for index := 0; index < 5; index++ {
		counter.Add(index == 0)
	}
	if got := counter.Flush(context.Background()); got != 5 {
		t.Errorf("the flush wrote %d requests, want 5", got)
	}
	if len(counts.buckets) != 1 {
		t.Errorf("the flush wrote %d rows, want one", len(counts.buckets))
	}
	if counts.requests != 5 || counts.failures != 1 {
		t.Errorf("wrote %d requests and %d failures, want 5 and 1", counts.requests, counts.failures)
	}
	if got := counter.Flush(context.Background()); got != 0 {
		t.Errorf("an empty flush wrote %d requests, want none", got)
	}
	if len(counts.buckets) != 1 {
		t.Errorf("an empty flush added a row: %v", counts.buckets)
	}
}

func TestANilCounterAndANilSinkAreSafe(t *testing.T) {
	silence(t)
	var counter *Counter
	counter.Add(true)
	if got := counter.Flush(context.Background()); got != 0 {
		t.Errorf("a nil counter flushed %d requests", got)
	}
	handler := Responses(nil, nil)(answer(http.StatusInternalServerError, `{"error":"boom"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want the answer to pass through", rec.Code)
	}
}
