package upload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSaverServe(t *testing.T) {
	saver := NewSaver(t.TempDir(), testMaxMB)
	content := signature + "1.7 served body"
	path, err := saver.Save(multipartRequest(t, nil, "notice.pdf", content))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	name := strings.TrimPrefix(path, URLPrefix)
	tests := []struct {
		name       string
		value      string
		wantStatus int
	}{
		{name: "the saved file", value: name, wantStatus: http.StatusOK},
		{name: "an unknown but valid name", value: strings.Repeat("a", 32) + ".pdf", wantStatus: http.StatusNotFound},
		{name: "upper case hex", value: strings.ToUpper(name), wantStatus: http.StatusNotFound},
		{name: "another extension", value: strings.TrimSuffix(name, Extension) + ".txt", wantStatus: http.StatusNotFound},
		{name: "a short name", value: "abc.pdf", wantStatus: http.StatusNotFound},
		{name: "a traversal attempt", value: "../.env", wantStatus: http.StatusNotFound},
		{name: "a nested traversal attempt", value: "../" + name, wantStatus: http.StatusNotFound},
		{name: "an empty name", value: "", wantStatus: http.StatusNotFound},
	}
	serve := saver.Serve()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, URLPrefix+"placeholder.pdf", nil)
			req.SetPathValue(PathValueName, tc.value)
			rec := httptest.NewRecorder()
			serve(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != ContentType {
				t.Errorf("Content-Type = %q, want %q", got, ContentType)
			}
			if got := rec.Header().Get("Content-Disposition"); got != dispositionValue {
				t.Errorf("Content-Disposition = %q, want %q", got, dispositionValue)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			if rec.Body.String() != content {
				t.Errorf("the served body does not match the saved file")
			}
		})
	}
	t.Run("through the route pattern", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET "+URLPrefix+"{"+PathValueName+"}", serve)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if rec.Body.String() != content {
			t.Errorf("the served body does not match the saved file")
		}
	})
}
