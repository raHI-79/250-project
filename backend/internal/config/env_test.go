package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("could not write the test env file: %v", err)
	}
	return path
}

func unsetAfterTest(t *testing.T, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		for _, key := range keys {
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("could not unset %s: %v", key, err)
			}
		}
	})
}

func TestLoadEnvFileSetsMissingVariables(t *testing.T) {
	unsetAfterTest(t,
		"ENVTEST_PLAIN",
		"ENVTEST_SPACED",
		"ENVTEST_QUOTED",
		"ENVTEST_SINGLE",
		"ENVTEST_EXPORTED",
		"ENVTEST_URL",
		"ENVTEST_EQUALS",
		"ENVTEST_EMPTY",
		"ENVTEST_COMMENT",
	)
	path := writeEnvFile(t, "ENVTEST_PLAIN=plain\n"+
		"\n"+
		"   \n"+
		"ENVTEST_SPACED =  spaced  \n"+
		"#ENVTEST_COMMENT=ignored\n"+
		"ENVTEST_QUOTED=\"quoted value\"\n"+
		"ENVTEST_SINGLE='single value'\n"+
		"export ENVTEST_EXPORTED=exported\n"+
		"ENVTEST_URL=postgres://postgres:secret@localhost:5432/db?sslmode=disable\n"+
		"ENVTEST_EQUALS=a=b=c\n"+
		"ENVTEST_EMPTY=\n"+
		"this line has no equals sign\n")
	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() returned an unexpected error: %v", err)
	}
	want := map[string]string{
		"ENVTEST_PLAIN":    "plain",
		"ENVTEST_SPACED":   "spaced",
		"ENVTEST_QUOTED":   "quoted value",
		"ENVTEST_SINGLE":   "single value",
		"ENVTEST_EXPORTED": "exported",
		"ENVTEST_URL":      "postgres://postgres:secret@localhost:5432/db?sslmode=disable",
		"ENVTEST_EQUALS":   "a=b=c",
		"ENVTEST_EMPTY":    "",
	}
	for key, wantValue := range want {
		got, found := os.LookupEnv(key)
		if !found {
			t.Errorf("%s was not set", key)
			continue
		}
		if got != wantValue {
			t.Errorf("%s = %q, want %q", key, got, wantValue)
		}
	}
	if _, found := os.LookupEnv("ENVTEST_COMMENT"); found {
		t.Error("ENVTEST_COMMENT was set, want comment lines to be skipped")
	}
}

func TestLoadEnvFileKeepsRealEnvironment(t *testing.T) {
	t.Setenv("ENVTEST_EXISTING", "from-environment")
	path := writeEnvFile(t, "ENVTEST_EXISTING=from-file\n")
	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() returned an unexpected error: %v", err)
	}
	if got := os.Getenv("ENVTEST_EXISTING"); got != "from-environment" {
		t.Errorf("ENVTEST_EXISTING = %q, want %q", got, "from-environment")
	}
}

func TestLoadEnvFileMissingFileIsNotAnError(t *testing.T) {
	if err := LoadEnvFile(filepath.Join(t.TempDir(), "no-such-file")); err != nil {
		t.Errorf("LoadEnvFile() returned %v, want nil for a missing file", err)
	}
}

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{name: "simple pair", line: "KEY=value", wantKey: "KEY", wantValue: "value", wantOK: true},
		{name: "surrounding spaces", line: "  KEY = value  ", wantKey: "KEY", wantValue: "value", wantOK: true},
		{name: "empty value", line: "KEY=", wantKey: "KEY", wantValue: "", wantOK: true},
		{name: "value keeps later equals signs", line: "KEY=a=b", wantKey: "KEY", wantValue: "a=b", wantOK: true},
		{name: "double quotes are removed", line: `KEY="value"`, wantKey: "KEY", wantValue: "value", wantOK: true},
		{name: "blank line", line: "   "},
		{name: "comment line", line: "# KEY=value"},
		{name: "no equals sign", line: "KEY"},
		{name: "missing key", line: "=value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, value, ok := parseEnvLine(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %t, want %t", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if key != tc.wantKey {
				t.Errorf("key = %q, want %q", key, tc.wantKey)
			}
			if value != tc.wantValue {
				t.Errorf("value = %q, want %q", value, tc.wantValue)
			}
		})
	}
}
