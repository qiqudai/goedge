package main

import (
	"path/filepath"
	"testing"
)

func TestResolveNginxLogsDir(t *testing.T) {
	oldWorkDir := WorkDir
	WorkDir = filepath.Join(t.TempDir(), "runtime")
	t.Cleanup(func() {
		WorkDir = oldWorkDir
	})

	rootLogs := filepath.Join(runtimeRoot(), "logs")
	if got := resolveNginxLogsDir(""); got != rootLogs {
		t.Fatalf("expected default logs dir %q, got %q", rootLogs, got)
	}

	customRelative := filepath.Join(runtimeRoot(), "custom", "logs")
	if got := resolveNginxLogsDir("custom/logs"); got != customRelative {
		t.Fatalf("expected relative logs dir %q, got %q", customRelative, got)
	}

	customAbsolute := filepath.Join(t.TempDir(), "abs-logs")
	if got := resolveNginxLogsDir(customAbsolute); got != customAbsolute {
		t.Fatalf("expected absolute logs dir %q, got %q", customAbsolute, got)
	}
}
