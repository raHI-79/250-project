package release

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fakeGitHub(t *testing.T, body any, calls *int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			*calls++
		}
		if r.URL.Path != "/repos/raHI-79/250-project/releases/latest" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	return server
}

func releaseBody(tag, notes, assetName string) map[string]any {
	return map[string]any{
		"tag_name": tag,
		"body":     notes,
		"assets": []map[string]any{
			{"name": "source.zip", "browser_download_url": "https://example.test/source.zip"},
			{"name": assetName, "browser_download_url": "https://example.test/" + assetName},
		},
	}
}

func TestLatestReadsTheNewestAPKAsset(t *testing.T) {
	server := fakeGitHub(t, releaseBody("v2.1.0", "Fixes the routine", "sust-cse-portal-2.1.0.apk"), nil)
	details, err := New("", server.URL).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if details.Version != "2.1.0" {
		t.Errorf("Version = %q, want %q", details.Version, "2.1.0")
	}
	if details.APKURL != "https://example.test/sust-cse-portal-2.1.0.apk" {
		t.Errorf("APKURL = %q", details.APKURL)
	}
	if details.Notes != "Fixes the routine" {
		t.Errorf("Notes = %q", details.Notes)
	}
	if details.Required {
		t.Error("Required = true, want false")
	}
	if details.VersionCode != VersionCode("2.1.0") {
		t.Errorf("VersionCode = %d, want %d", details.VersionCode, VersionCode("2.1.0"))
	}
}

func TestLatestMarksAReleaseRequired(t *testing.T) {
	server := fakeGitHub(t, releaseBody("v3.0.0", "[REQUIRED] sign in again", "sust-cse-portal-3.0.0.apk"), nil)
	details, err := New("", server.URL).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if !details.Required {
		t.Error("Required = false, want true")
	}
}

func TestLatestReportsAReleaseWithoutAnAPK(t *testing.T) {
	server := fakeGitHub(t, map[string]any{"tag_name": "v2.1.0", "assets": []map[string]any{}}, nil)
	details, err := New("", server.URL).Latest(context.Background())
	if !errors.Is(err, ErrNoAsset) {
		t.Fatalf("Latest() error = %v, want ErrNoAsset", err)
	}
	if details.APKURL != "" {
		t.Errorf("APKURL = %q, want empty", details.APKURL)
	}
}

func TestLatestAnswersFromTheCacheWithinTenMinutes(t *testing.T) {
	calls := 0
	server := fakeGitHub(t, releaseBody("v2.1.0", "", "sust-cse-portal-2.1.0.apk"), &calls)
	clock := time.Now()
	lookup := NewWith("", server.URL, server.Client(), func() time.Time { return clock })
	for i := 0; i < 3; i++ {
		if _, err := lookup.Latest(context.Background()); err != nil {
			t.Fatalf("Latest() error = %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("GitHub was asked %d times, want 1", calls)
	}
	clock = clock.Add(CacheTTL + time.Minute)
	if _, err := lookup.Latest(context.Background()); err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if calls != 2 {
		t.Errorf("GitHub was asked %d times after the cache expired, want 2", calls)
	}
}

func TestVersionCodeOrdersReleases(t *testing.T) {
	cases := []struct {
		latest    string
		installed string
		want      bool
	}{
		{"2.1.0", "2.0.1", true},
		{"v2.1.0", "2.1.0", false},
		{"2.0.1", "2.1.0", false},
		{"3.0.0", "2.9.9", true},
		{"2.1.1", "2.1.0", true},
		{"2.1.0-beta", "2.0.9", true},
	}
	for _, test := range cases {
		if got := Newer(test.latest, test.installed); got != test.want {
			t.Errorf("Newer(%q, %q) = %t, want %t", test.latest, test.installed, got, test.want)
		}
	}
}
