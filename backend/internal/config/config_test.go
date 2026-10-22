package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testDatabaseURL = "postgres://postgres:secret@localhost:5432/sust_coord?sslmode=disable"
	testJWTSecret   = "0123456789abcdef0123456789abcdef"
)

func setEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	env := map[string]string{
		"PORT":                  "",
		"APP_ENV":               "",
		"APP_TIMEZONE":          "",
		"DATABASE_URL":          testDatabaseURL,
		"TEST_DATABASE_URL":     "",
		"JWT_SECRET":            testJWTSecret,
		"JWT_TTL":               "",
		"FRONTEND_ORIGIN":       "",
		"UPLOAD_DIR":            "",
		"MAX_UPLOAD_MB":         "",
		"ADMIN_IDENTIFIER":      "",
		"ADMIN_PASSWORD":        "",
		"STUDENT_EMAIL_DOMAIN":  "",
		"DECISION_WINDOW_START": "",
		"DECISION_WINDOW_END":   "",
		"SCHEDULE_DAYS_AHEAD":   "",
		"GOOGLE_CLIENT_ID":      "",
		"FILE_STORAGE":          "",
		"STATIC_DIR":            "",
		"MAIL_PROVIDER":         "",
		"MAIL_FROM_NAME":        "",
		"BREVO_API_KEY":         "",
		"DB_MAX_CONNS":          "",
	}
	for key, value := range overrides {
		env[key] = value
	}
	for key, value := range env {
		t.Setenv(key, value)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{name: "Port", got: cfg.Port, want: "8080"},
		{name: "AppEnv", got: cfg.AppEnv, want: "development"},
		{name: "Timezone", got: cfg.Timezone, want: "Asia/Dhaka"},
		{name: "DatabaseURL", got: cfg.DatabaseURL, want: testDatabaseURL},
		{name: "TestDatabaseURL", got: cfg.TestDatabaseURL, want: ""},
		{name: "JWTSecret", got: cfg.JWTSecret, want: testJWTSecret},
		{name: "JWTTTL", got: cfg.JWTTTL, want: 24 * time.Hour},
		{name: "FrontendOrigin", got: cfg.FrontendOrigin, want: "http://localhost:5173"},
		{name: "UploadDir", got: cfg.UploadDir, want: "uploads"},
		{name: "MaxUploadMB", got: cfg.MaxUploadMB, want: 10},
		{name: "AdminIdentifier", got: cfg.AdminIdentifier, want: ""},
		{name: "AdminPassword", got: cfg.AdminPassword, want: ""},
		{name: "StudentDomain", got: cfg.StudentDomain, want: "student.sust.edu"},
		{name: "DecisionStart", got: cfg.DecisionStart, want: "16:00"},
		{name: "DecisionEnd", got: cfg.DecisionEnd, want: "23:00"},
		{name: "DecisionFrom", got: cfg.DecisionFrom, want: 16 * 60},
		{name: "DecisionTo", got: cfg.DecisionTo, want: 23 * 60},
		{name: "DaysAhead", got: cfg.DaysAhead, want: 7},
		{name: "FileStorage", got: cfg.FileStorage, want: FileStorageDisk},
		{name: "FilesInDatabase", got: cfg.FilesInDatabase(), want: false},
		{name: "StaticDir", got: cfg.StaticDir, want: ""},
		{name: "ServesStatic", got: cfg.ServesStatic(), want: false},
		{name: "MailProvider", got: cfg.MailProvider, want: MailProviderSMTP},
		{name: "MailFromName", got: cfg.MailFromName, want: defaultMailFromName},
		{name: "BrevoAPIKey", got: cfg.BrevoAPIKey, want: ""},
		{name: "DBMaxConns", got: cfg.DBMaxConns, want: defaultMaxConns},
		{name: "GoogleClientID", got: cfg.GoogleClientID, want: ""},
		{name: "GoogleEnabled", got: cfg.GoogleEnabled(), want: false},
		{name: "StudentEmailSuffix", got: cfg.StudentEmailSuffix(), want: "@student.sust.edu"},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
	if cfg.Location == nil {
		t.Fatal("Location is nil, want a loaded timezone")
	}
	if cfg.HasAdminSeed() {
		t.Error("HasAdminSeed() = true, want false when the admin variables are unset")
	}
}

func TestLoadOverrides(t *testing.T) {
	testTimeout := "90m"
	setEnv(t, map[string]string{
		"PORT":                  "9090",
		"APP_ENV":               "production",
		"APP_TIMEZONE":          "UTC",
		"TEST_DATABASE_URL":     "postgres://postgres:secret@localhost:5432/sust_coord_test?sslmode=disable",
		"JWT_TTL":               testTimeout,
		"FRONTEND_ORIGIN":       "https://coordination.sust.edu",
		"UPLOAD_DIR":            "files",
		"MAX_UPLOAD_MB":         "25",
		"ADMIN_IDENTIFIER":      "admin",
		"ADMIN_PASSWORD":        "a-long-enough-password",
		"STUDENT_EMAIL_DOMAIN":  "Student.CSE.SUST.edu",
		"DECISION_WINDOW_START": "16:30",
		"DECISION_WINDOW_END":   "22:00",
		"SCHEDULE_DAYS_AHEAD":   "14",
		"GOOGLE_CLIENT_ID":      "client-id.apps.googleusercontent.com",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{name: "Port", got: cfg.Port, want: "9090"},
		{name: "AppEnv", got: cfg.AppEnv, want: "production"},
		{name: "Timezone", got: cfg.Timezone, want: "UTC"},
		{name: "JWTTTL", got: cfg.JWTTTL, want: 90 * time.Minute},
		{name: "FrontendOrigin", got: cfg.FrontendOrigin, want: "https://coordination.sust.edu"},
		{name: "UploadDir", got: cfg.UploadDir, want: "files"},
		{name: "MaxUploadMB", got: cfg.MaxUploadMB, want: 25},
		{name: "AdminIdentifier", got: cfg.AdminIdentifier, want: "admin"},
		{name: "AdminPassword", got: cfg.AdminPassword, want: "a-long-enough-password"},
		{name: "StudentDomain", got: cfg.StudentDomain, want: "student.cse.sust.edu"},
		{name: "DecisionFrom", got: cfg.DecisionFrom, want: 16*60 + 30},
		{name: "DecisionTo", got: cfg.DecisionTo, want: 22 * 60},
		{name: "DaysAhead", got: cfg.DaysAhead, want: 14},
		{name: "GoogleEnabled", got: cfg.GoogleEnabled(), want: true},
		{name: "StudentEmailSuffix", got: cfg.StudentEmailSuffix(), want: "@student.cse.sust.edu"},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
	if !cfg.HasAdminSeed() {
		t.Error("HasAdminSeed() = false, want true when both admin variables are set")
	}
}

func TestLoadRejectsBadConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		overrides       map[string]string
		wantErr         error
		wantErrContains string
	}{
		{
			name:            "missing DATABASE_URL",
			overrides:       map[string]string{"DATABASE_URL": ""},
			wantErr:         ErrMissingDatabaseURL,
			wantErrContains: "DATABASE_URL",
		},
		{
			name:            "MAX_UPLOAD_MB is not a number",
			overrides:       map[string]string{"MAX_UPLOAD_MB": "ten"},
			wantErrContains: "MAX_UPLOAD_MB",
		},
		{
			name:            "MAX_UPLOAD_MB is zero",
			overrides:       map[string]string{"MAX_UPLOAD_MB": "0"},
			wantErrContains: "MAX_UPLOAD_MB",
		},
		{
			name:            "missing JWT_SECRET",
			overrides:       map[string]string{"JWT_SECRET": ""},
			wantErr:         ErrMissingJWTSecret,
			wantErrContains: "JWT_SECRET",
		},
		{
			name:            "JWT_SECRET one character too short",
			overrides:       map[string]string{"JWT_SECRET": strings.Repeat("x", 31)},
			wantErr:         ErrShortJWTSecret,
			wantErrContains: "32 characters",
		},
		{
			name:            "JWT_TTL is not a duration",
			overrides:       map[string]string{"JWT_TTL": "one day"},
			wantErrContains: "JWT_TTL",
		},
		{
			name:            "JWT_TTL is zero",
			overrides:       map[string]string{"JWT_TTL": "0s"},
			wantErrContains: "positive duration",
		},
		{
			name:            "JWT_TTL is negative",
			overrides:       map[string]string{"JWT_TTL": "-1h"},
			wantErrContains: "positive duration",
		},
		{
			name:            "invalid timezone",
			overrides:       map[string]string{"APP_TIMEZONE": "Mars/Olympus"},
			wantErrContains: "Mars/Olympus",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.overrides)
			cfg, err := Load()
			if err == nil {
				t.Fatal("Load() returned no error, want one")
			}
			if cfg != nil {
				t.Error("Load() returned a config alongside the error, want nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("Load() error = %v, want %v", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("error %q does not mention %q", err, tc.wantErrContains)
			}
		})
	}
}

func TestLoadErrorsNeverContainTheSecret(t *testing.T) {
	secret := strings.Repeat("s", 31)
	setEnv(t, map[string]string{"JWT_SECRET": secret, "JWT_TTL": "not-a-duration"})
	_, err := Load()
	if err == nil {
		t.Fatal("Load() returned no error, want one")
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("the error message contains the JWT secret, want it hidden")
	}
}

func TestLoadAdminSeedNeedsBothVariables(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      bool
	}{
		{name: "neither set", want: false},
		{name: "only identifier", overrides: map[string]string{"ADMIN_IDENTIFIER": "admin"}, want: false},
		{name: "only password", overrides: map[string]string{"ADMIN_PASSWORD": "a-long-enough-password"}, want: false},
		{
			name:      "both set",
			overrides: map[string]string{"ADMIN_IDENTIFIER": "admin", "ADMIN_PASSWORD": "a-long-enough-password"},
			want:      true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.overrides)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned an unexpected error: %v", err)
			}
			if got := cfg.HasAdminSeed(); got != tc.want {
				t.Errorf("HasAdminSeed() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestNow(t *testing.T) {
	setEnv(t, map[string]string{"APP_TIMEZONE": "Asia/Dhaka"})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	_, offset := cfg.Now().Zone()
	wantOffset := int((6 * time.Hour).Seconds())
	if offset != wantOffset {
		t.Errorf("Now() offset = %d seconds, want %d seconds (UTC+6)", offset, wantOffset)
	}
}

func TestGoogleEnabled(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		want     bool
	}{
		{name: "unset", clientID: "", want: false},
		{name: "spaces", clientID: "   ", want: false},
		{name: "a client id", clientID: "client-id.apps.googleusercontent.com", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, map[string]string{"GOOGLE_CLIENT_ID": tc.clientID})
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned an unexpected error: %v", err)
			}
			if cfg.GoogleEnabled() != tc.want {
				t.Errorf("GoogleEnabled() = %v, want %v", cfg.GoogleEnabled(), tc.want)
			}
		})
	}
}

func TestDecisionWindow(t *testing.T) {
	setEnv(t, map[string]string{"DECISION_WINDOW_START": "17:00", "DECISION_WINDOW_END": "23:00"})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name       string
		clock      string
		wantOpen   bool
		wantClosed bool
	}{
		{name: "the morning", clock: "09:00", wantOpen: false, wantClosed: false},
		{name: "one minute early", clock: "16:59", wantOpen: false, wantClosed: false},
		{name: "the window opens", clock: "17:00", wantOpen: true, wantClosed: false},
		{name: "inside the window", clock: "20:15", wantOpen: true, wantClosed: false},
		{name: "the last minute", clock: "22:59", wantOpen: true, wantClosed: false},
		{name: "the deadline", clock: "23:00", wantOpen: false, wantClosed: true},
		{name: "after the deadline", clock: "23:45", wantOpen: false, wantClosed: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			at, err := time.ParseInLocation("15:04", tc.clock, cfg.Location)
			if err != nil {
				t.Fatalf("could not parse %q: %v", tc.clock, err)
			}
			if got := cfg.DecisionOpen(at); got != tc.wantOpen {
				t.Errorf("DecisionOpen(%s) = %v, want %v", tc.clock, got, tc.wantOpen)
			}
			if got := cfg.DecisionClosed(at); got != tc.wantClosed {
				t.Errorf("DecisionClosed(%s) = %v, want %v", tc.clock, got, tc.wantClosed)
			}
		})
	}
}

func TestLoadRejectsABrokenWindow(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
	}{
		{name: "a broken start", start: "five", end: "23:00"},
		{name: "a broken end", start: "17:00", end: "late"},
		{name: "the end before the start", start: "23:00", end: "17:00"},
		{name: "the same minute", start: "17:00", end: "17:00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, map[string]string{"DECISION_WINDOW_START": tc.start, "DECISION_WINDOW_END": tc.end})
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted the window %q to %q", tc.start, tc.end)
			}
		})
	}
}

func TestLoadRejectsABrokenDaysAhead(t *testing.T) {
	for _, value := range []string{"zero", "0", "-1", "40"} {
		t.Run(value, func(t *testing.T) {
			setEnv(t, map[string]string{"SCHEDULE_DAYS_AHEAD": value})
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted SCHEDULE_DAYS_AHEAD=%q", value)
			}
		})
	}
}

func TestLoadReadsTheDeploymentSettings(t *testing.T) {
	setEnv(t, map[string]string{
		"FILE_STORAGE":   "database",
		"STATIC_DIR":     "  /app/static  ",
		"MAIL_PROVIDER":  "BREVO",
		"MAIL_FROM_NAME": "CSE Office",
		"BREVO_API_KEY":  "  secret-key  ",
		"DB_MAX_CONNS":   "5",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{name: "FileStorage", got: cfg.FileStorage, want: FileStorageDatabase},
		{name: "FilesInDatabase", got: cfg.FilesInDatabase(), want: true},
		{name: "StaticDir", got: cfg.StaticDir, want: "/app/static"},
		{name: "ServesStatic", got: cfg.ServesStatic(), want: true},
		{name: "MailProvider", got: cfg.MailProvider, want: MailProviderBrevo},
		{name: "MailFromName", got: cfg.MailFromName, want: "CSE Office"},
		{name: "BrevoAPIKey", got: cfg.BrevoAPIKey, want: "secret-key"},
		{name: "DBMaxConns", got: cfg.DBMaxConns, want: 5},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
}

func TestLoadRefusesBadDeploymentSettings(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
	}{
		{name: "an unknown file storage", overrides: map[string]string{"FILE_STORAGE": "s3"}},
		{name: "an unknown mail provider", overrides: map[string]string{"MAIL_PROVIDER": "postman"}},
		{name: "no connections", overrides: map[string]string{"DB_MAX_CONNS": "0"}},
		{name: "too many connections", overrides: map[string]string{"DB_MAX_CONNS": "500"}},
		{name: "connections that are not a number", overrides: map[string]string{"DB_MAX_CONNS": "five"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setEnv(t, test.overrides)
			if _, err := Load(); err == nil {
				t.Fatal("Load() accepted a setting it should refuse")
			}
		})
	}
}
