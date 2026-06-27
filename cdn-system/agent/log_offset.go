package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type logOffsetState struct {
	Offset int64  `json:"offset"`
	Inode  uint64 `json:"inode,omitempty"`
	Size   int64  `json:"size,omitempty"`
	LastTS string `json:"last_ts,omitempty"`
}

func fileInfoInode(fi os.FileInfo) uint64 {
	if fi == nil {
		return 0
	}
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return stat.Ino
	}
	return 0
}

func loadLogOffsetState(path string) logOffsetState {
	data, err := os.ReadFile(path)
	if err != nil {
		return logOffsetState{}
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return logOffsetState{}
	}
	if !strings.HasPrefix(value, "{") {
		offset, err := strconv.ParseInt(value, 10, 64)
		if err != nil || offset < 0 {
			return logOffsetState{}
		}
		return logOffsetState{Offset: offset}
	}
	var state logOffsetState
	if err := json.Unmarshal(data, &state); err != nil {
		return logOffsetState{}
	}
	if state.Offset < 0 {
		state.Offset = 0
	}
	return state
}

func saveLogOffsetState(path string, state logOffsetState) {
	if state.Offset < 0 {
		state.Offset = 0
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0644)
}

func resolveLogReadOffset(state logOffsetState, fi os.FileInfo) int64 {
	if fi == nil {
		return 0
	}
	size := fi.Size()
	inode := fileInfoInode(fi)
	if size == 0 {
		return 0
	}
	if state.Offset > size {
		return size
	}
	if state.Inode != 0 && inode != 0 && state.Inode != inode {
		return size
	}
	if state.Offset < 0 {
		return 0
	}
	return state.Offset
}

func filterLinesAfterTimestamp(lines []string, lastTS string) []string {
	lastTS = strings.TrimSpace(lastTS)
	if lastTS == "" {
		return lines
	}
	prev, err := time.Parse(time.RFC3339, lastTS)
	if err != nil {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		ts := parseLogLineTimestamp(line)
		if ts.IsZero() || ts.After(prev) {
			out = append(out, line)
		}
	}
	return out
}

func parseLogLineTimestamp(line string) time.Time {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return time.Time{}
	}
	raw, ok := payload["time_iso8601"]
	if !ok {
		return time.Time{}
	}
	rawTime, ok := raw.(string)
	if !ok {
		return time.Time{}
	}
	parsed, err := parseISO8601Time(strings.TrimSpace(rawTime))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func reconcileLogOffsetsOnStartup() {
	reconcileLogOffsetAtEOF(getAccessLogPaths())
	reconcileLogOffsetAtEOF(getStreamLogPaths())
}

func reconcileLogOffsetAtEOF(logPath, offsetPath string) {
	fi, err := os.Stat(logPath)
	if err != nil {
		return
	}
	size := fi.Size()
	inode := fileInfoInode(fi)
	saveLogOffsetState(offsetPath, logOffsetState{
		Offset: size,
		Inode:  inode,
		Size:   size,
		LastTS: "",
	})
}

func maxLogLineTimestamp(lines []string, current string) string {
	maxTS := time.Time{}
	if current = strings.TrimSpace(current); current != "" {
		if parsed, err := time.Parse(time.RFC3339, current); err == nil {
			maxTS = parsed.UTC()
		}
	}
	for _, line := range lines {
		ts := parseLogLineTimestamp(line)
		if ts.After(maxTS) {
			maxTS = ts
		}
	}
	if maxTS.IsZero() {
		return current
	}
	return maxTS.UTC().Format(time.RFC3339)
}
