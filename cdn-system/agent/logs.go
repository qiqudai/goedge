package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logShipMu        sync.Mutex
	accessLogDeliver = sendAccessLogs
	streamLogDeliver = sendStreamLogs
)

func startAccessLogShip() {
	shipAccessLogs()
	shipStreamLogs()
	ticker := time.NewTicker(LOG_SHIP_INT)
	for range ticker.C {
		shipAccessLogs()
		shipStreamLogs()
	}
}

func startMetricsShip() {
	ticker := time.NewTicker(METRICS_INT)
	for range ticker.C {
		shipMetrics()
	}
}

func startLogCleanup() {
	cleanupStoredLogs()
	handleLowDiskCleanup()
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		cleanupStoredLogs()
		handleLowDiskCleanup()
	}
}

var accessLogPathLogged bool
var streamLogPathLogged bool

func cleanupStoredLogs() {
	dir, hours := getLogStorageSettings()
	if hours <= 0 {
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[Warn] Log cleanup mkdir failed: %v", err)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[Warn] Log cleanup read dir failed: %v", err)
		return
	}

	expireBefore := time.Now().Add(-time.Duration(hours) * time.Hour)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "access.json" || name == "access.offset" || name == "stream_access.json" || name == "stream_access.offset" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(expireBefore) {
			if err := os.Remove(filepath.Join(dir, name)); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		log.Printf("[Info] Log cleanup removed %d file(s) from %s", removed, dir)
	}
}

func getLogStorageSettings() (string, int) {
	localConfigMu.RLock()
	resources := LocalResources
	localConfigMu.RUnlock()

	rootDir := runtimeRoot()
	dir := filepath.Join(rootDir, "logs")
	hours := 0
	if resources != nil {
		if strings.TrimSpace(resources.Website.LogStorageDir) != "" {
			dir = resources.Website.LogStorageDir
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(rootDir, dir)
			}
		}
		if resources.Website.LogStorageHours > 0 {
			hours = resources.Website.LogStorageHours
		}
	}
	return dir, hours
}

func shipAccessLogs() {
	logShipMu.Lock()
	defer logShipMu.Unlock()

	logPath, offsetPath := getAccessLogPaths()
	if DebugMode && !accessLogPathLogged {
		log.Printf("[Debug] Access log ship path=%s offset=%s", logPath, offsetPath)
		accessLogPathLogged = true
	}
	shipLogBatch(logPath, offsetPath, accessLogDeliver)
}

func shipStreamLogs() {
	logShipMu.Lock()
	defer logShipMu.Unlock()

	logPath, offsetPath := getStreamLogPaths()
	if DebugMode && !streamLogPathLogged {
		log.Printf("[Debug] Stream log ship path=%s offset=%s", logPath, offsetPath)
		streamLogPathLogged = true
	}
	shipLogBatch(logPath, offsetPath, streamLogDeliver)
}

const (
	// logShipBatchLines bounds how many lines we put into a single delivery
	// (one WS frame / HTTP body) to keep memory and frame size reasonable.
	logShipBatchLines = 1000
	// logShipMaxBatchesPerTick bounds how much we drain per tick so a single
	// shipper run cannot monopolize CPU on a pathological backlog, while still
	// providing huge headroom (1000 * 500 = 500k lines / tick) so a busy node
	// can stay caught up instead of falling permanently behind.
	logShipMaxBatchesPerTick = 500
)

// shipLogBatch drains new log lines from logPath starting at the persisted byte
// offset and delivers them in bounded batches. It keeps draining within a tick
// until it reaches EOF (or the per-tick batch cap), so a high-traffic node does
// not accumulate an unbounded backlog. De-duplication relies solely on the byte
// offset (plus inode/size reconciliation for truncation/rotation); the API layer
// is the authoritative guard against stale replays.
func shipLogBatch(logPath, offsetPath string, deliver func([]string) error) {
	for batch := 0; batch < logShipMaxBatchesPerTick; batch++ {
		fi, err := os.Stat(logPath)
		if err != nil {
			return
		}

		state := loadLogOffsetState(offsetPath)
		offset := resolveLogReadOffset(state, fi)

		file, err := os.Open(logPath)
		if err != nil {
			return
		}
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			file.Close()
			return
		}

		reader := bufio.NewReader(file)
		lines := make([]string, 0, logShipBatchLines)
		var consumed int64
		reachedEOF := false
		for len(lines) < logShipBatchLines {
			line, rerr := reader.ReadString('\n')
			if rerr != nil {
				// Trailing fragment without a newline: leave it unconsumed so
				// the completed line is shipped on a later tick.
				reachedEOF = true
				break
			}
			consumed += int64(len(line))
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		file.Close()

		if len(lines) == 0 {
			// Caught up: reconcile offset bookkeeping (handles truncation/rotation
			// detected via resolveLogReadOffset) without losing LastTS.
			if offset != state.Offset || fi.Size() != state.Size || fileInfoInode(fi) != state.Inode {
				saveLogOffsetState(offsetPath, logOffsetState{
					Offset: offset,
					Inode:  fileInfoInode(fi),
					Size:   fi.Size(),
					LastTS: state.LastTS,
				})
			}
			return
		}

		lines = normalizeLogTimesToUTC(lines)
		if err := deliver(lines); err != nil {
			log.Printf("[Error] Log ship failed: %v", err)
			return
		}
		saveLogOffsetState(offsetPath, logOffsetState{
			Offset: offset + consumed,
			Inode:  fileInfoInode(fi),
			Size:   fi.Size(),
			LastTS: maxLogLineTimestamp(lines, state.LastTS),
		})

		if reachedEOF {
			return
		}
	}
}

func shipMetrics() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:9100/metrics", nil)
	body, status, err := doRequest(req, 5*time.Second, true)
	if err != nil || status != 200 {
		return
	}
	if err := sendMetrics(string(body)); err != nil {
		log.Printf("[Error] Metrics ship failed: %v", err)
	}
}

func currentNginxLogsDir() string {
	localConfigMu.RLock()
	nginx := LocalNginxConfig
	localConfigMu.RUnlock()
	if nginx == nil {
		return resolveNginxLogsDir("")
	}
	return resolveNginxLogsDir(nginx.LogsDir)
}

func getAccessLogPaths() (string, string) {
	logsDir := currentNginxLogsDir()
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, "access.json"), filepath.Join(logsDir, "access.offset")
}

func getStreamLogPaths() (string, string) {
	logsDir := currentNginxLogsDir()
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, "stream_access.json"), filepath.Join(logsDir, "stream_access.offset")
}

func normalizeLogTimesToUTC(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, normalizeLogLineTimeToUTC(line))
	}
	return out
}

func normalizeLogLineTimeToUTC(line string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return line
	}
	raw, ok := payload["time_iso8601"]
	if !ok {
		return line
	}
	rawTime, ok := raw.(string)
	if !ok {
		return line
	}
	rawTime = strings.TrimSpace(rawTime)
	if rawTime == "" {
		return line
	}
	parsed, err := parseISO8601Time(rawTime)
	if err != nil {
		return line
	}
	payload["time_iso8601"] = parsed.UTC().Format(time.RFC3339)
	buf, err := json.Marshal(payload)
	if err != nil {
		return line
	}
	return string(buf)
}

func parseISO8601Time(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, os.ErrInvalid
}
