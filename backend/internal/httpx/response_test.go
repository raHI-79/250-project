package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantBody   string
		wantHeader string
	}{
		{
			name:       "map payload",
			status:     http.StatusOK,
			data:       map[string]string{"status": "ok"},
			wantBody:   `{"status":"ok"}`,
			wantHeader: "application/json",
		},
		{
			name:   "created with struct payload",
			status: http.StatusCreated,
			data: struct {
				Name string `json:"name"`
			}{Name: "Ratul"},
			wantBody:   `{"name":"Ratul"}`,
			wantHeader: "application/json",
		},
		{
			name:       "slice payload",
			status:     http.StatusOK,
			data:       []int{1, 2, 3},
			wantBody:   `[1,2,3]`,
			wantHeader: "application/json",
		},
		{
			name:       "nil payload writes no body",
			status:     http.StatusNoContent,
			data:       nil,
			wantBody:   "",
			wantHeader: "application/json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteJSON(rec, tc.status, tc.data)
			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := rec.Header().Get("Content-Type"); got != tc.wantHeader {
				t.Errorf("Content-Type = %q, want %q", got, tc.wantHeader)
			}
			if got := rec.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{name: "not found", status: http.StatusNotFound, message: "Route not found"},
		{name: "unauthorized", status: http.StatusUnauthorized, message: "Unauthorized"},
		{name: "conflict", status: http.StatusConflict, message: "This room is not available at the selected time."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, tc.status, tc.message)
			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
			}
			if len(body) != 1 {
				t.Errorf("body has %d fields, want exactly 1: %v", len(body), body)
			}
			if body["error"] != tc.message {
				t.Errorf(`body["error"] = %q, want %q`, body["error"], tc.message)
			}
		})
	}
}
