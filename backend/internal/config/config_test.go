package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://postgres:secret@localhost:5432/sust_coord?sslmode=disable"

func TestLoad(t *testing.T) {
	tests := []struct {
		name                string
		env                 map[string]string
		wantPort            string
		wantAppEnv          string
		wantTimezone        string
		wantDatabaseURL     string
		wantTestDatabaseURL string
		wantErr             bool
		wantErrContains     string
	}{
		{
			name: "defaults when only DATABASE_URL is set",
			env: map[string]string{
				"PORT":              "",
				"APP_ENV":           "",
				"APP_TIMEZONE":      "",
				"DATABASE_URL":      testDatabaseURL,
				"TEST_DATABASE_URL": "",
			},
			wantPort:        "8080",
			wantAppEnv:      "development",
			wantTimezone:    "Asia/Dhaka",
			wantDatabaseURL: testDatabaseURL,
		},
		{
			name: "every value overridden",
			env: map[string]string{
				"PORT":              "9090",
				"APP_ENV":           "production",
				"APP_TIMEZONE":      "UTC",
				"DATABASE_URL":      testDatabaseURL,
				"TEST_DATABASE_URL": "postgres://postgres:secret@localhost:5432/sust_coord_test?sslmode=disable",
			},
			wantPort:            "9090",
			wantAppEnv:          "production",
			wantTimezone:        "UTC",
			wantDatabaseURL:     testDatabaseURL,
			wantTestDatabaseURL: "postgres://postgres:secret@localhost:5432/sust_coord_test?sslmode=disable",
		},
		{
			name: "only PORT overridden, the rest fall back",
			env: map[string]string{
				"PORT":              "3000",
				"APP_ENV":           "",
				"APP_TIMEZONE":      "",
				"DATABASE_URL":      testDatabaseURL,
				"TEST_DATABASE_URL": "",
			},
			wantPort:        "3000",
			wantAppEnv:      "development",
			wantTimezone:    "Asia/Dhaka",
			wantDatabaseURL: testDatabaseURL,
		},
		{
			name: "missing DATABASE_URL is rejected",
			env: map[string]string{
				"PORT":              "",
				"APP_ENV":           "",
				"APP_TIMEZONE":      "",
				"DATABASE_URL":      "",
				"TEST_DATABASE_URL": "",
			},
			wantErr:         true,
			wantErrContains: "DATABASE_URL",
		},
		{
			name: "invalid timezone is rejected",
			env: map[string]string{
				"PORT":              "",
				"APP_ENV":           "",
				"APP_TIMEZONE":      "Mars/Olympus",
				"DATABASE_URL":      testDatabaseURL,
				"TEST_DATABASE_URL": "",
			},
			wantErr:         true,
			wantErrContains: "Mars/Olympus",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			cfg, err := Load()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Load() returned no error, want one")
				}
				if cfg != nil {
					t.Errorf("Load() returned a config alongside the error, want nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("error %q does not mention %q", err, tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() returned an unexpected error: %v", err)
			}
			if cfg.Port != tc.wantPort {
				t.Errorf("Port = %q, want %q", cfg.Port, tc.wantPort)
			}
			if cfg.AppEnv != tc.wantAppEnv {
				t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, tc.wantAppEnv)
			}
			if cfg.Timezone != tc.wantTimezone {
				t.Errorf("Timezone = %q, want %q", cfg.Timezone, tc.wantTimezone)
			}
			if cfg.DatabaseURL != tc.wantDatabaseURL {
				t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tc.wantDatabaseURL)
			}
			if cfg.TestDatabaseURL != tc.wantTestDatabaseURL {
				t.Errorf("TestDatabaseURL = %q, want %q", cfg.TestDatabaseURL, tc.wantTestDatabaseURL)
			}
			if cfg.Location == nil {
				t.Fatal("Location is nil, want a loaded timezone")
			}
			if cfg.Location.String() != tc.wantTimezone {
				t.Errorf("Location = %q, want %q", cfg.Location, tc.wantTimezone)
			}
		})
	}
}

func TestLoadMissingDatabaseURLReturnsSentinelError(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("TEST_DATABASE_URL", "")
	t.Setenv("APP_TIMEZONE", "")
	if _, err := Load(); !errors.Is(err, ErrMissingDatabaseURL) {
		t.Errorf("Load() error = %v, want ErrMissingDatabaseURL", err)
	}
}

func TestNow(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_TIMEZONE", "Asia/Dhaka")
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("TEST_DATABASE_URL", "")
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
