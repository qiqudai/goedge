package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkerRlimitNofileTracksWorkerConnections(t *testing.T) {
	cfg := &edgeNginxConfig{WorkerConnections: 51200}
	workerConnections := resolveWorkerConnections(cfg)
	got := resolveWorkerRlimitNofile(cfg, workerConnections)
	want := workerConnections * 4
	if got != want {
		t.Fatalf("resolveWorkerRlimitNofile = %d, want %d", got, want)
	}
}

func TestResolveWorkerRlimitNofileCapsAtSystemdLimit(t *testing.T) {
	cfg := &edgeNginxConfig{WorkerConnections: 400000}
	got := resolveWorkerRlimitNofile(cfg, resolveWorkerConnections(cfg))
	if got != 1048576 {
		t.Fatalf("resolveWorkerRlimitNofile cap = %d, want 1048576", got)
	}
}

func TestWriteMainConfigEmitsWorkerRlimitNofile(t *testing.T) {
	prevWorkDir := WorkDir
	WorkDir = t.TempDir()
	t.Cleanup(func() { WorkDir = prevWorkDir })
	if err := os.MkdirAll(filepath.Join(runtimeRoot(), "conf", "dynamic"), 0o755); err != nil {
		t.Fatalf("mkdir dynamic conf failed: %v", err)
	}

	cfg := &edgeNginxConfig{WorkerConnections: 51200}
	if err := writeMainConfig(cfg); err != nil {
		t.Fatalf("writeMainConfig failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(runtimeRoot(), "conf", "dynamic", "main.conf"))
	if err != nil {
		t.Fatalf("read generated main.conf failed: %v", err)
	}
	content := string(data)
	want := fmt.Sprintf("worker_rlimit_nofile %d;", resolveWorkerRlimitNofile(cfg, resolveWorkerConnections(cfg)))
	if !strings.Contains(content, want) {
		t.Fatalf("main.conf missing %q:\n%s", want, content)
	}
}
