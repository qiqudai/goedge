package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupStoredLogs_RemovesExpiredFiles(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "log_storage")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	oldFile := filepath.Join(logDir, "old.log")
	newFile := filepath.Join(logDir, "new.log")
	accessJSON := filepath.Join(logDir, "access.json")
	accessOffset := filepath.Join(logDir, "access.offset")

	for _, path := range []string{oldFile, newFile, accessJSON, accessOffset} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	past := time.Now().Add(-2 * time.Hour)
	recent := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(oldFile, past, past); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}
	if err := os.Chtimes(accessJSON, past, past); err != nil {
		t.Fatalf("chtimes access.json: %v", err)
	}
	if err := os.Chtimes(accessOffset, past, past); err != nil {
		t.Fatalf("chtimes access.offset: %v", err)
	}
	if err := os.Chtimes(newFile, recent, recent); err != nil {
		t.Fatalf("chtimes new: %v", err)
	}

	previousWorkDir := WorkDir
	localConfigMu.RLock()
	previousResources := LocalResources
	localConfigMu.RUnlock()
	t.Cleanup(func() {
		WorkDir = previousWorkDir
		localConfigMu.Lock()
		LocalResources = previousResources
		localConfigMu.Unlock()
	})

	WorkDir = tempDir
	localConfigMu.Lock()
	LocalResources = &edgeResources{
		Website: edgeWebsiteResources{
			LogStorageDir:   logDir,
			LogStorageHours: 1,
		},
	}
	localConfigMu.Unlock()

	cleanupStoredLogs()

	if _, err := os.Stat(oldFile); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old file removed, stat err=%v", err)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("expected new file kept, stat err=%v", err)
	}
	if _, err := os.Stat(accessJSON); err != nil {
		t.Fatalf("expected access.json kept, stat err=%v", err)
	}
	if _, err := os.Stat(accessOffset); err != nil {
		t.Fatalf("expected access.offset kept, stat err=%v", err)
	}
}

