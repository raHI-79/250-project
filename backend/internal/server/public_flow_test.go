package server

import (
	"net/http"
	"testing"
)

func TestPublicStatsNeedsNoToken(t *testing.T) {
	_, _, router := newFlowRouter(t)
	rec := call(t, router, http.MethodGet, "/api/public/stats", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	for _, name := range []string{"students", "teachers", "courses", "batches"} {
		if number(t, rec, name) != 0 {
			t.Errorf("%s = %v, want 0 on an empty database", name, number(t, rec, name))
		}
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want public, max-age=300", got)
	}
}
