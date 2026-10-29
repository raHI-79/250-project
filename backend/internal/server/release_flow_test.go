package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/release"
)

const flowAPK = "https://objects.example.test/sust-cse-portal-2.1.0.apk"

type fakeReleaseLookup struct {
	details release.Details
	err     error
}

func (f fakeReleaseLookup) Latest(context.Context) (release.Details, error) {
	return f.details, f.err
}

func newReleaseRouter(t *testing.T, lookup handler.ReleaseLookup) http.Handler {
	t.Helper()
	deps, _ := newAccessRouter(t)
	deps.Releases = lookup
	return NewRouter(deps)
}

func TestTheDownloadLinkIsPublicAndLandsOnTheAPK(t *testing.T) {
	router := newReleaseRouter(t, fakeReleaseLookup{details: release.Details{
		Version: "2.1.0",
		APKURL:  flowAPK,
	}})
	rec := serve(t, router, http.MethodGet, handler.DownloadAndroidPath, "", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != flowAPK {
		t.Errorf("Location = %q, want %q", got, flowAPK)
	}
}

func TestTheAndroidLatestEndpointIsPublic(t *testing.T) {
	router := newReleaseRouter(t, fakeReleaseLookup{details: release.Details{
		Version:     "2.1.0",
		VersionCode: release.VersionCode("2.1.0"),
		APKURL:      flowAPK,
		Notes:       "Adds in-app updates",
	}})
	rec := serve(t, router, http.MethodGet, handler.AndroidLatestPath, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	if payload["version"] != "2.1.0" || payload["apk_url"] != flowAPK {
		t.Errorf("payload = %#v", payload)
	}
	if payload["required"] != false {
		t.Errorf("required = %#v, want false", payload["required"])
	}
}

func TestTheDownloadLinkFallsBackWithoutAnAsset(t *testing.T) {
	router := newReleaseRouter(t, fakeReleaseLookup{
		details: release.Details{Version: "2.1.0"},
		err:     release.ErrNoAsset,
	})
	rec := serve(t, router, http.MethodGet, handler.DownloadAndroidPath, "", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want an HTML page", got)
	}
}
