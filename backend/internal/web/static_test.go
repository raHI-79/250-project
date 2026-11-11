package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func staticDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		IndexFile:                       "<!doctype html><title>Portal</title>",
		filepath.Join("assets", "a.js"): "console.log(1)",
		"manifest.webmanifest":          "{}",
		"sw.js":                         "self.addEventListener('push', () => {})",
	}
	for name, body := range files {
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("could not create %s: %v", target, err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatalf("could not write %s: %v", target, err)
		}
	}
	return dir
}

func fallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Route not found"}`))
	})
}

func TestStaticServesFilesAndFallsBackToIndex(t *testing.T) {
	handler := Static(staticDir(t), fallbackHandler())
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
		wantCache  string
	}{
		{name: "the hashed asset", method: http.MethodGet, target: "/assets/a.js",
			wantStatus: http.StatusOK, wantBody: "console.log(1)", wantCache: assetCache},
		{name: "a file beside the page", method: http.MethodGet, target: "/manifest.webmanifest",
			wantStatus: http.StatusOK, wantBody: "{}", wantCache: pageCache},
		{name: "the root", method: http.MethodGet, target: "/",
			wantStatus: http.StatusOK, wantBody: "<!doctype html><title>Portal</title>", wantCache: pageCache},
		{name: "a client route", method: http.MethodGet, target: "/student/today",
			wantStatus: http.StatusOK, wantBody: "<!doctype html><title>Portal</title>", wantCache: pageCache},
		{name: "a deep client route", method: http.MethodGet, target: "/office-portal/class-periods",
			wantStatus: http.StatusOK, wantBody: "<!doctype html><title>Portal</title>", wantCache: pageCache},
		{name: "an unknown api route", method: http.MethodGet, target: "/api/nothing",
			wantStatus: http.StatusNotFound, wantBody: `{"error":"Route not found"}`},
		{name: "an unknown upload", method: http.MethodGet, target: "/uploads/nothing.pdf",
			wantStatus: http.StatusNotFound, wantBody: `{"error":"Route not found"}`},
		{name: "a post to a client route", method: http.MethodPost, target: "/student/today",
			wantStatus: http.StatusNotFound, wantBody: `{"error":"Route not found"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(test.method, test.target, nil))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, test.wantStatus)
			}
			if body := rec.Body.String(); body != test.wantBody {
				t.Errorf("body = %q, want %q", body, test.wantBody)
			}
			if test.wantCache == "" {
				return
			}
			if got := rec.Header().Get("Cache-Control"); got != test.wantCache {
				t.Errorf("cache = %q, want %q", got, test.wantCache)
			}
		})
	}
}

func TestStaticSetsSecurityHeaders(t *testing.T) {
	handler := Static(staticDir(t), fallbackHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}
	for name, want := range headers {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestStaticKeepsEscapesInsideTheFolder(t *testing.T) {
	dir := staticDir(t)
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("not for the web"), 0o644); err != nil {
		t.Fatalf("could not write the secret: %v", err)
	}
	handler := Static(dir, fallbackHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/../secret.txt", nil))
	if rec.Body.String() == "not for the web" {
		t.Fatal("the handler served a file from outside the static folder")
	}
}

func TestStaticServesTheServiceWorkerAndManifestForInstalling(t *testing.T) {
	handler := Static(staticDir(t), fallbackHandler())
	tests := []struct {
		name      string
		target    string
		wantType  string
		wantScope string
		wantCache string
	}{
		{
			name:      "the service worker",
			target:    WorkerPath,
			wantType:  WorkerType,
			wantScope: WorkerScope,
			wantCache: pageCache,
		},
		{
			name:      "the manifest",
			target:    ManifestPath,
			wantType:  ManifestType,
			wantCache: pageCache,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, test.target, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != test.wantType {
				t.Errorf("content type = %q, want %q", got, test.wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != test.wantCache {
				t.Errorf("cache = %q, want %q", got, test.wantCache)
			}
			if got := rec.Header().Get("Service-Worker-Allowed"); got != test.wantScope {
				t.Errorf("service worker scope = %q, want %q", got, test.wantScope)
			}
		})
	}
}
