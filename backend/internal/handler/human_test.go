package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
)

type stubPasses struct {
	accept string
}

func (s stubPasses) Issue(time.Duration) (string, time.Time, error) {
	return s.accept, time.Now().Add(time.Hour), nil
}

func (s stubPasses) Check(token string) error {
	if token == s.accept {
		return nil
	}
	return auth.ErrInvalidHumanPass
}

func checkedHandler(cfg *config.Config, passes HumanPassIssuer) http.Handler {
	return RequireHuman(cfg, passes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestRequireHumanPassesWhenTheCheckIsOff(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recorder := httptest.NewRecorder()
	checkedHandler(&config.Config{}, stubPasses{accept: "good"}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d when no keys are set", recorder.Code, http.StatusNoContent)
	}
}

func TestRequireHumanNeedsAPass(t *testing.T) {
	cfg := &config.Config{TurnstileSiteKey: "site", TurnstileSecretKey: "secret"}
	tests := []struct {
		name   string
		pass   string
		status int
	}{
		{name: "no pass", pass: "", status: http.StatusForbidden},
		{name: "a wrong pass", pass: "stale", status: http.StatusForbidden},
		{name: "the issued pass", pass: "good", status: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
			if test.pass != "" {
				request.Header.Set(HumanPassHeader, test.pass)
			}
			recorder := httptest.NewRecorder()
			checkedHandler(cfg, stubPasses{accept: "good"}).ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Errorf("status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

func TestRequireHumanSkipsTheAndroidApp(t *testing.T) {
	cfg := &config.Config{TurnstileSiteKey: "site", TurnstileSecretKey: "secret"}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	recorder := httptest.NewRecorder()
	checkedHandler(cfg, stubPasses{accept: "good"}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d for the app without a pass", recorder.Code, http.StatusNoContent)
	}
}
