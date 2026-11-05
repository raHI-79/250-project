package handler

import (
	"encoding/json"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/release"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	apkVersion = "2.6.0"
	apkBody    = "fake-apk-bytes"
)

type fakeGitHub struct {
	server    *httptest.Server
	storage   *httptest.Server
	calls     map[string]int
	token     string
	assetAuth []string
	status    int
}

func newFakeGitHub(t *testing.T, version string) *fakeGitHub {
	t.Helper()
	github := &fakeGitHub{calls: map[string]int{}, token: "secret-token"}
	github.storage = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		github.calls["storage"]++
		github.assetAuth = append(github.assetAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprint(len(apkBody)))
		_, _ = w.Write([]byte(apkBody))
	}))
	t.Cleanup(github.storage.Close)
	github.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/assets/") {
			github.calls["asset"]++
			if github.status >= http.StatusBadRequest {
				w.WriteHeader(github.status)
				return
			}
			if r.Header.Get("Accept") != release.AssetAccept {
				t.Errorf("asset Accept = %q, want %q", r.Header.Get("Accept"), release.AssetAccept)
			}
			if r.Header.Get("Authorization") != "Bearer "+github.token {
				t.Errorf("asset Authorization = %q, want the token", r.Header.Get("Authorization"))
			}
			http.Redirect(w, r, github.storage.URL+"/download", http.StatusFound)
			return
		}
		github.calls["releases"]++
		w.Header().Set("Content-Type", "application/json")
		payload := []map[string]any{{
			"tag_name": "v" + version,
			"name":     "SUST CSE Portal " + version,
			"body":     "Release notes",
			"assets": []map[string]any{{
				"name":                 release.AssetName(version),
				"browser_download_url": github.storage.URL + "/public",
				"url":                  github.server.URL + "/repos/test/repo/releases/assets/42",
				"size":                 len(apkBody),
				"download_count":       7,
			}},
		}}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(github.server.Close)
	return github
}

func newAPKs(t *testing.T, github *fakeGitHub, token string) (*APKs, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{UploadDir: dir}
	lookup := release.New("test/repo", github.server.URL).WithToken(token)
	return NewAPKs(cfg, lookup), dir
}

func getAPK(t *testing.T, apks *APKs, version string) *httptest.ResponseRecorder {
	t.Helper()
	path := DownloadAndroidPath
	pattern := "GET " + DownloadAndroidPath
	if version != "" {
		path = DownloadAndroidPath + "/" + version
		pattern = "GET " + DownloadAndroidPath + "/{version}"
	}
	mux := http.NewServeMux()
	mux.Handle(pattern, apks.Serve())
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func TestTheServerStreamsTheNewestAPK(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	apks, dir := newAPKs(t, github, github.token)
	rec := getAPK(t, apks, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != apkBody {
		t.Errorf("body = %q, want the APK bytes", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != release.APKMime {
		t.Errorf("Content-Type = %q, want %q", got, release.APKMime)
	}
	wanted := `attachment; filename="` + release.AssetName(apkVersion) + `"`
	if got := rec.Header().Get("Content-Disposition"); got != wanted {
		t.Errorf("Content-Disposition = %q, want %q", got, wanted)
	}
	if got := rec.Header().Get("Content-Length"); got != fmt.Sprint(len(apkBody)) {
		t.Errorf("Content-Length = %q, want %d", got, len(apkBody))
	}
	cached := filepath.Join(dir, apkFolder, release.AssetName(apkVersion))
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("the APK was not cached on disk: %v", err)
	}
}

func TestTheStorageHostNeverSeesTheToken(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	apks, _ := newAPKs(t, github, github.token)
	if rec := getAPK(t, apks, ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if github.calls["storage"] != 1 {
		t.Fatalf("the storage host was called %d times, want 1", github.calls["storage"])
	}
	for _, header := range github.assetAuth {
		if header != "" {
			t.Errorf("the storage host received an Authorization header: %q", header)
		}
	}
}

func TestGitHubIsCalledOncePerRelease(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	apks, _ := newAPKs(t, github, github.token)
	for attempt := 1; attempt <= 3; attempt++ {
		if rec := getAPK(t, apks, ""); rec.Code != http.StatusOK {
			t.Fatalf("attempt %d answered %d", attempt, rec.Code)
		}
	}
	if github.calls["asset"] != 1 {
		t.Errorf("the asset API was called %d times, want 1", github.calls["asset"])
	}
	if github.calls["storage"] != 1 {
		t.Errorf("the storage host was called %d times, want 1", github.calls["storage"])
	}
}

func TestOneVersionCanBeAskedForByName(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	apks, _ := newAPKs(t, github, github.token)
	rec := getAPK(t, apks, apkVersion)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	missing := getAPK(t, apks, "1.0.0")
	if missing.Code != http.StatusNotFound {
		t.Errorf("an unknown version answered %d, want 404", missing.Code)
	}
}

func TestWithoutATokenTheDownloadIsRefused(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	apks, _ := newAPKs(t, github, "")
	rec := getAPK(t, apks, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), MessageAPKBusy) {
		t.Errorf("body = %q, want the busy message", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), github.token) {
		t.Error("the answer leaked the token")
	}
}

func TestWhenGitHubRefusesTheAssetWeAnswer503(t *testing.T) {
	github := newFakeGitHub(t, apkVersion)
	github.status = http.StatusUnauthorized
	apks, _ := newAPKs(t, github, github.token)
	rec := getAPK(t, apks, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
}
