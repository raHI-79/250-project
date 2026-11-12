package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeatureOffAnswerIsForbiddenWithACode(t *testing.T) {
	rec := httptest.NewRecorder()
	writeFeatureOff(rec, MessageFeatureOff)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a switched-off feature answered %d, want 403", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"feature_off"`) || !strings.Contains(body, MessageFeatureOff) {
		t.Fatalf("the answer is %q", body)
	}
}

func TestAnOpenGateLetsTheRequestThrough(t *testing.T) {
	gates := NewGates(nil)
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
	rec := httptest.NewRecorder()
	gates.Feature("feature_messages")(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/messages/unread-count", nil))
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("an open gate answered %d", rec.Code)
	}
}
