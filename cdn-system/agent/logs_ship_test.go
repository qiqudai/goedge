package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolveLogReadOffsetResumesFromEOFAfterTruncate(t *testing.T) {
	fi := fakeFileInfo{size: 128, inode: 42}
	state := logOffsetState{Offset: 4096, Inode: 42, Size: 4096}
	got := resolveLogReadOffset(state, fi)
	if got != 128 {
		t.Fatalf("expected EOF offset 128 after truncate, got %d", got)
	}
}

func TestResolveLogReadOffsetResumesFromEOFAfterRotation(t *testing.T) {
	fi := fakeFileInfo{size: 256, inode: 0}
	state := logOffsetState{Offset: 600, Inode: 0, Size: 500}
	got := resolveLogReadOffset(state, fi)
	if got != 256 {
		t.Fatalf("expected EOF offset 256 after shrink, got %d", got)
	}
}

func TestFilterLinesAfterTimestampSkipsReplay(t *testing.T) {
	base := time.Date(2026, 6, 26, 1, 30, 0, 0, time.UTC)
	lines := []string{
		fmt.Sprintf(`{"time_iso8601":"%s","host":"old.example"}`, base.Format(time.RFC3339)),
		fmt.Sprintf(`{"time_iso8601":"%s","host":"new.example"}`, base.Add(time.Second).Format(time.RFC3339)),
	}
	filtered := filterLinesAfterTimestamp(lines, base.Format(time.RFC3339))
	if len(filtered) != 1 {
		t.Fatalf("expected 1 line after replay filter, got %d", len(filtered))
	}
	if !contains(filtered[0], "new.example") {
		t.Fatalf("unexpected filtered line: %s", filtered[0])
	}
}

// TestShipLogBatchDrainsBacklogBeyondSingleBatch verifies the shipper drains a
// backlog larger than a single batch within one run, so a high-traffic node can
// stay caught up instead of falling permanently behind (the throughput root
// cause behind repeated stats outages on the busiest node).
func TestShipLogBatchDrainsBacklogBeyondSingleBatch(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.json")
	offsetPath := filepath.Join(dir, "access.offset")

	total := logShipBatchLines*2 + 137 // spans 3 batches
	var builder strings.Builder
	base := time.Now().UTC()
	for i := 0; i < total; i++ {
		ts := base.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		builder.WriteString(fmt.Sprintf("{\"time_iso8601\":\"%s\",\"host\":\"host-%d.example\"}\n", ts, i))
	}
	if err := os.WriteFile(logPath, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var delivered atomic.Int32
	var batches atomic.Int32
	shipLogBatch(logPath, offsetPath, func(lines []string) error {
		batches.Add(1)
		delivered.Add(int32(len(lines)))
		return nil
	})

	if int(delivered.Load()) != total {
		t.Fatalf("expected all %d lines drained, got %d", total, delivered.Load())
	}
	if batches.Load() < 3 {
		t.Fatalf("expected backlog drained across >=3 batches, got %d", batches.Load())
	}
	fi, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	state := loadLogOffsetState(offsetPath)
	if state.Offset != fi.Size() {
		t.Fatalf("expected offset at EOF %d after drain, got %d", fi.Size(), state.Offset)
	}
}

// TestShipLogBatchKeepsSameSecondLines guards against silently dropping lines
// that share the same 1-second timestamp (busy nodes emit many requests per
// second). The shipper must rely on byte offsets, not a strict timestamp filter.
func TestShipLogBatchKeepsSameSecondLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.json")
	offsetPath := filepath.Join(dir, "access.offset")

	ts := time.Now().UTC().Format(time.RFC3339)
	var builder strings.Builder
	for i := 0; i < 50; i++ {
		builder.WriteString(fmt.Sprintf("{\"time_iso8601\":\"%s\",\"host\":\"host-%d.example\"}\n", ts, i))
	}
	if err := os.WriteFile(logPath, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var delivered atomic.Int32
	shipLogBatch(logPath, offsetPath, func(lines []string) error {
		delivered.Add(int32(len(lines)))
		return nil
	})

	if delivered.Load() != 50 {
		t.Fatalf("expected all 50 same-second lines shipped, got %d", delivered.Load())
	}
}

func TestShipLogBatchDoesNotAdvanceOffsetOnDeliveryFailure(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.json")
	offsetPath := filepath.Join(dir, "access.offset")
	line := fmt.Sprintf("{\"time_iso8601\":\"%s\",\"host\":\"retry.example\"}\n", time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(logPath, []byte(line), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	shipLogBatch(logPath, offsetPath, func(lines []string) error {
		return errors.New("delivery failed")
	})
	state := loadLogOffsetState(offsetPath)
	if state.Offset != 0 {
		t.Fatalf("expected offset unchanged after failure, got %+v", state)
	}
}

func TestShipLogBatchConcurrentSafe(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.json")
	offsetPath := filepath.Join(dir, "access.offset")

	var builder string
	for i := 0; i < 400; i++ {
		ts := time.Now().UTC().Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339)
		builder += fmt.Sprintf("{\"time_iso8601\":\"%s\",\"host\":\"host-%d.example\"}\n", ts, i)
	}
	if err := os.WriteFile(logPath, []byte(builder), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var delivered atomic.Int32
	deliver := func(lines []string) error {
		delivered.Add(int32(len(lines)))
		return nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logShipMu.Lock()
			shipLogBatch(logPath, offsetPath, deliver)
			logShipMu.Unlock()
		}()
	}
	wg.Wait()

	if delivered.Load() == 0 {
		t.Fatalf("expected some lines delivered")
	}
	state := loadLogOffsetState(offsetPath)
	if state.Offset <= 0 {
		t.Fatalf("expected offset advanced under concurrency, got %+v", state)
	}
}

type fakeFileInfo struct {
	size  int64
	inode uint64
}

func (f fakeFileInfo) Name() string       { return "access.json" }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() os.FileMode  { return 0o644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Now() }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() interface{} {
	return &struct {
		Ino uint64
	}{Ino: f.inode}
}

func TestReconcileLogOffsetAtEOFResetsStaleState(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.json")
	offsetPath := filepath.Join(dir, "access.offset")
	if err := os.WriteFile(logPath, []byte(`{"time_iso8601":"2026-06-26T01:00:00Z"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(offsetPath, []byte(`{"offset":0,"last_ts":"2026-06-26T01:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	reconcileLogOffsetAtEOF(logPath, offsetPath)

	state := loadLogOffsetState(offsetPath)
	fi, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if state.Offset != fi.Size() {
		t.Fatalf("expected offset at EOF %d, got %d", fi.Size(), state.Offset)
	}
	if state.LastTS != "" {
		t.Fatalf("expected last_ts cleared, got %q", state.LastTS)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
