package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		cleanupStoredLogs()
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
	logPath, offsetPath := getAccessLogPaths()
	if DebugMode && !accessLogPathLogged {
		log.Printf("[Debug] Access log ship path=%s offset=%s", logPath, offsetPath)
		accessLogPathLogged = true
	}
	fi, err := os.Stat(logPath)
	if err != nil {
		return
	}
	offset := loadOffset(offsetPath)
	if offset > fi.Size() {
		offset = 0
	}

	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return
	}

	reader := bufio.NewReader(file)
	lines := make([]string, 0, 200)
	for len(lines) < 200 {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
		if err != nil {
			break
		}
	}
	if len(lines) == 0 {
		return
	}

	newOffset, _ := file.Seek(0, io.SeekCurrent)
	if err := sendAccessLogs(lines); err != nil {
		log.Printf("[Error] Access log ship failed: %v", err)
		return
	}
	saveOffset(offsetPath, newOffset)
}

func shipStreamLogs() {
	logPath, offsetPath := getStreamLogPaths()
	if DebugMode && !streamLogPathLogged {
		log.Printf("[Debug] Stream log ship path=%s offset=%s", logPath, offsetPath)
		streamLogPathLogged = true
	}
	fi, err := os.Stat(logPath)
	if err != nil {
		return
	}
	offset := loadOffset(offsetPath)
	if offset > fi.Size() {
		offset = 0
	}

	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return
	}

	reader := bufio.NewReader(file)
	lines := make([]string, 0, 200)
	for len(lines) < 200 {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
		if err != nil {
			break
		}
	}
	if len(lines) == 0 {
		return
	}

	newOffset, _ := file.Seek(0, io.SeekCurrent)
	if err := sendStreamLogs(lines); err != nil {
		log.Printf("[Error] Stream log ship failed: %v", err)
		return
	}
	saveOffset(offsetPath, newOffset)
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

func loadOffset(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return 0
	}
	offset, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return offset
}

func saveOffset(path string, offset int64) {
	_ = os.WriteFile(path, []byte(strconv.FormatInt(offset, 10)), 0644)
}

func getAccessLogPaths() (string, string) {
	rootDir := runtimeRoot()
	logsDir := filepath.Join(rootDir, "logs")
	localConfigMu.RLock()
	nginx := LocalNginxConfig
	localConfigMu.RUnlock()
	if nginx != nil {
		if dir := strings.TrimSpace(nginx.LogsDir); dir != "" {
			logsDir = dir
			if !filepath.IsAbs(logsDir) {
				logsDir = filepath.Join(rootDir, logsDir)
			}
		}
	}
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, "access.json"), filepath.Join(logsDir, "access.offset")
}

func getStreamLogPaths() (string, string) {
	rootDir := runtimeRoot()
	logsDir := filepath.Join(rootDir, "logs")
	localConfigMu.RLock()
	nginx := LocalNginxConfig
	localConfigMu.RUnlock()
	if nginx != nil {
		if dir := strings.TrimSpace(nginx.LogsDir); dir != "" {
			logsDir = dir
			if !filepath.IsAbs(logsDir) {
				logsDir = filepath.Join(rootDir, logsDir)
			}
		}
	}
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, "stream_access.json"), filepath.Join(logsDir, "stream_access.offset")
}
