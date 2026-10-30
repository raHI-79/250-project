package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/release"
)

const fakeAPK = "https://objects.example.test/sust-cse-portal-2.1.0.apk"

func fakeReleases(t *testing.T, tag, notes, assetName string) *release.Lookup {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assets := []map[string]any{}
		if assetName != "" {
			assets = append(assets, map[string]any{
				"name":                 assetName,
				"browser_download_url": fakeAPK,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"tag_name": tag,
			"body":     notes,
			"assets":   assets,
		}})
	}))
	t.Cleanup(server.Close)
	return release.New("", server.URL)
}

type brokenReleases struct{}

func (brokenReleases) Latest(context.Context) (release.Details, error) {
	return release.Details{}, release.ErrUnavailable
}

func (brokenReleases) ReleasesPage() string {
	return "https://github.com/raHI-79/250-project/releases/latest"
}

func appCall(t *testing.T, handlerFunc http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handlerFunc(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestDownloadAndroidRedirectsToTheNewestAPK(t *testing.T) {
	apps := NewAppRelease(fakeReleases(t, "v2.1.0", "Fixes", "sust-cse-portal-2.1.0.apk"), "https://portal.test")
	rec := appCall(t, apps.Download(), DownloadAndroidPath)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != fakeAPK {
		t.Errorf("Location = %q, want %q", got, fakeAPK)
	}
}

func TestDownloadAndroidExplainsItselfWithoutAnAsset(t *testing.T) {
	apps := NewAppRelease(fakeReleases(t, "v2.1.0", "", ""), "https://portal.test")
	rec := appCall(t, apps.Download(), DownloadAndroidPath)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "not ready to download") {
		t.Errorf("the fallback page does not explain itself: %q", body)
	}
	if !strings.Contains(body, "https://portal.test") {
		t.Errorf("the fallback page does not link the website: %q", body)
	}
}

func TestAndroidLatestReportsTheRelease(t *testing.T) {
	apps := NewAppRelease(
		fakeReleases(t, "v2.1.0", "Adds in-app updates [required]", "sust-cse-portal-2.1.0.apk"),
		"https://portal.test")
	rec := appCall(t, apps.Latest(), AndroidLatestPath)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	want := map[string]any{
		"version":      "2.1.0",
		"version_code": float64(release.VersionCode("2.1.0")),
		"apk_url":      fakeAPK,
		"notes":        "Adds in-app updates [required]",
		"required":     true,
	}
	for key, value := range want {
		if payload[key] != value {
			t.Errorf("%s = %#v, want %#v", key, payload[key], value)
		}
	}
}

func TestAndroidLatestStaysEmptyWithoutAnAsset(t *testing.T) {
	apps := NewAppRelease(fakeReleases(t, "v2.1.0", "", ""), "https://portal.test")
	rec := appCall(t, apps.Latest(), AndroidLatestPath)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["apk_url"] != "" {
		t.Errorf("apk_url = %#v, want empty", payload["apk_url"])
	}
	if payload["version"] != "" {
		t.Errorf("version = %#v, want empty without a published APK", payload["version"])
	}
}

func TestAndroidLatestAsksForPatienceWhenGitHubIsBusy(t *testing.T) {
	apps := NewAppRelease(brokenReleases{}, "https://portal.test")
	rec := appCall(t, apps.Latest(), AndroidLatestPath)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := decodeBody(t, rec)["error"]; got != MessageUpdateBusy {
		t.Errorf("error = %#v, want %q", got, MessageUpdateBusy)
	}
	if got := rec.Header().Get("Cache-Control"); got != noStore {
		t.Errorf("Cache-Control = %q, want %q", got, noStore)
	}
}

func TestDownloadAndroidFallsBackToTheReleasesPage(t *testing.T) {
	apps := NewAppRelease(brokenReleases{}, "https://portal.test")
	rec := appCall(t, apps.Download(), DownloadAndroidPath)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body = %q)", rec.Code, rec.Body.String())
	}
	want := "https://github.com/raHI-79/250-project/releases/latest"
	if got := rec.Header().Get("Location"); got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}
