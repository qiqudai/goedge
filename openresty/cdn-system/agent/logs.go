package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	ticker := time.NewTicker(LOG_SHIP_INT)
	for range ticker.C {
		shipAccessLogs()
	}
}

func startMetricsShip() {
	ticker := time.NewTicker(METRICS_INT)
	for range ticker.C {
		shipMetrics()
	}
}

func startLogCleanup() {
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		cleanupStoredLogs()
	}
}

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
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "access.json" || name == "access.offset" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(expireBefore) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

func getLogStorageSettings() (string, int) {
	localConfigMu.RLock()
	resources := LocalResources
	localConfigMu.RUnlock()

	dir := filepath.Join(WorkDir, "logs")
	hours := 0
	if resources != nil {
		if strings.TrimSpace(resources.Website.LogStorageDir) != "" {
			dir = resources.Website.LogStorageDir
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(WorkDir, dir)
			}
		}
		if resources.Website.LogStorageHours > 0 {
			hours = resources.Website.LogStorageHours
		}
	}
	return dir, hours
}

func shipAccessLogs() {
	logPath := filepath.Join(WorkDir, "logs", "access.json")
	offsetPath := filepath.Join(WorkDir, "logs", "access.offset")
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
	payload := map[string]interface{}{
		"node_id": NodeID,
		"node_ip": "",
		"lines":   lines,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/logs/access", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	readBody := DebugMode
	respBody, status, err := doRequest(req, 10*time.Second, readBody)
	if err != nil {
		log.Printf("[Error] Access log ship failed: %v", err)
		return
	}
	debugLogInteraction("POST", req.URL.String(), status, body, respBody)
	if status == 200 {
		saveOffset(offsetPath, newOffset)
	}
}

func shipMetrics() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:9100/metrics", nil)
	body, status, err := doRequest(req, 5*time.Second, true)
	if err != nil || status != 200 {
		return
	}
	payload := map[string]interface{}{
		"node_id": NodeID,
		"node_ip": "",
		"content": string(body),
	}
	jsonBody, _ := json.Marshal(payload)
	postReq, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/logs/metrics", bytes.NewBuffer(jsonBody))
	postReq.Header.Set("Authorization", "Bearer "+AuthToken)
	postReq.Header.Set("Content-Type", "application/json")

	readBody := DebugMode
	respBody, status, err := doRequest(postReq, 10*time.Second, readBody)
	if err != nil {
		log.Printf("[Error] Metrics ship failed: %v", err)
		return
	}
	debugLogInteraction("POST", postReq.URL.String(), status, jsonBody, respBody)
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
