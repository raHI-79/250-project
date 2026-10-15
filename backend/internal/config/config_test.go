package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		wantPort     string
		wantAppEnv   string
		wantTimezone string
		wantErr      bool
	}{
		{
			name:         "defaults when nothing is set",
			env:          map[string]string{"PORT": "", "APP_ENV": "", "APP_TIMEZONE": ""},
			wantPort:     "8080",
			wantAppEnv:   "development",
			wantTimezone: "Asia/Dhaka",
		},
		{
			name: "all three values overridden",
			env: map[string]string{
				"PORT":         "9090",
				"APP_ENV":      "production",
				"APP_TIMEZONE": "UTC",
			},
			wantPort:     "9090",
			wantAppEnv:   "production",
			wantTimezone: "UTC",
		},
		{
			name:         "only PORT overridden, the rest fall back",
			env:          map[string]string{"PORT": "3000", "APP_ENV": "", "APP_TIMEZONE": ""},
			wantPort:     "3000",
			wantAppEnv:   "development",
			wantTimezone: "Asia/Dhaka",
		},
		{
			name:    "invalid timezone is rejected",
			env:     map[string]string{"PORT": "", "APP_ENV": "", "APP_TIMEZONE": "Mars/Olympus"},
			wantErr: true,
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
				if !strings.Contains(err.Error(), "Mars/Olympus") {
					t.Errorf("error %q does not mention the invalid timezone", err)
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
			if cfg.Location == nil {
				t.Fatal("Location is nil, want a loaded timezone")
			}
			if cfg.Location.String() != tc.wantTimezone {
				t.Errorf("Location = %q, want %q", cfg.Location, tc.wantTimezone)
			}
		})
	}
}

func TestNow(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_TIMEZONE", "Asia/Dhaka")
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
