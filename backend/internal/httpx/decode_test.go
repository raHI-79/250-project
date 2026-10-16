package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeTarget struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	Remember   bool   `json:"remember"`
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantOK         bool
		wantIdentifier string
		wantRemember   bool
	}{
		{
			name:           "complete body",
			body:           `{"identifier":"2023331001","password":"sust-250","remember":true}`,
			wantOK:         true,
			wantIdentifier: "2023331001",
			wantRemember:   true,
		},
		{
			name:           "missing fields fall back to zero values",
			body:           `{"identifier":"2023331001"}`,
			wantOK:         true,
			wantIdentifier: "2023331001",
		},
		{name: "empty object", body: `{}`, wantOK: true},
		{name: "empty body", body: ``},
		{name: "not json", body: `nonsense`},
		{name: "array instead of object", body: `[]`},
		{name: "unknown field", body: `{"identifier":"x","role":"admin"}`},
		{name: "wrong field type", body: `{"identifier":1}`},
		{name: "trailing data", body: `{"identifier":"x"} {"identifier":"y"}`},
		{name: "trailing garbage", body: `{"identifier":"x"}nonsense`},
		{name: "truncated object", body: `{"identifier":"x"`},
		{name: "body over one megabyte", body: `{"identifier":"` + strings.Repeat("x", MaxBodyBytes) + `"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			var target decodeTarget
			ok := DecodeJSON(rec, req, &target)
			if ok != tc.wantOK {
				t.Fatalf("DecodeJSON() = %t, want %t (body written = %q)", ok, tc.wantOK, rec.Body.String())
			}
			if !tc.wantOK {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
				}
				if got := rec.Body.String(); !strings.Contains(got, MessageInvalidBody) {
					t.Errorf("body = %q, want it to contain %q", got, MessageInvalidBody)
				}
				return
			}
			if rec.Body.Len() != 0 {
				t.Errorf("DecodeJSON() wrote %q on success, want nothing", rec.Body.String())
			}
			if target.Identifier != tc.wantIdentifier {
				t.Errorf("Identifier = %q, want %q", target.Identifier, tc.wantIdentifier)
			}
			if target.Remember != tc.wantRemember {
				t.Errorf("Remember = %t, want %t", target.Remember, tc.wantRemember)
			}
		})
	}
}
