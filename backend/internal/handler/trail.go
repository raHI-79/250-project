package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"strings"
)

type trailWriter struct {
	http.ResponseWriter
	status int
}

func (w *trailWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *trailWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *trailWriter) ok() bool {
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	return status >= 200 && status < 300
}

var skippedTrails = []string{"/api/auth/", "/api/analytics/", "/api/me", "/api/push/client-log"}

var ownTrails = []string{"/remove", "/restore", "/permanent", "/deactivate"}

func TrailAction(method, path string) string {
	switch {
	case strings.HasSuffix(path, "/publish"):
		return store.ActionPublished
	case method == http.MethodPost:
		return store.ActionCreated
	case method == http.MethodPatch, method == http.MethodPut:
		return store.ActionUpdated
	case method == http.MethodDelete:
		return store.ActionDeleted
	default:
		return ""
	}
}

func TrailTarget(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if part != "api" && part != "" {
			if index+1 < len(parts) {
				return parts[index+1]
			}
			return part
		}
	}
	return "request"
}

func Trail(rec *Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			action := TrailAction(r.Method, r.URL.Path)
			if action == "" || skipTrail(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			watched := &trailWriter{ResponseWriter: w}
			next.ServeHTTP(watched, r)
			if !watched.ok() {
				return
			}
			detached := r.WithContext(context.WithoutCancel(r.Context()))
			rec.Record(detached, action, TrailTarget(r.URL.Path), "",
				r.Method+" "+r.URL.Path)
		})
	}
}

func skipTrail(path string) bool {
	for _, prefix := range skippedTrails {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if !strings.HasPrefix(path, "/api/admin/") {
		return false
	}
	for _, suffix := range ownTrails {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}
