package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
)

type rawAccessLog struct {
	TimeISO8601          string  `json:"time_iso8601"`
	RemoteAddr           string  `json:"remote_addr"`
	Host                 string  `json:"host"`
	Request              string  `json:"request"`
	Status               int     `json:"status"`
	BodyBytesSent        int64   `json:"body_bytes_sent"`
	RequestTime          float64 `json:"request_time"`
	UpstreamAddr         string  `json:"upstream_addr"`
	UpstreamResponseTime string  `json:"upstream_response_time"`
	UpstreamCacheStatus  string  `json:"upstream_cache_status"`
	HttpReferer          string  `json:"http_referer"`
	HttpUserAgent        string  `json:"http_user_agent"`
	Scheme               string  `json:"scheme"`
	SSLProtocol          string  `json:"ssl_protocol"`
	SSLCipher            string  `json:"ssl_cipher"`
}

func InsertAccessLogs(nodeID, nodeIP string, lines []string) int {
	if !db.ClickHouseEnabled() || len(lines) == 0 {
		return 0
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_access_logs
		(ts, node_id, node_ip, remote_addr, host, method, uri, status, bytes, request_time, upstream_addr, upstream_response_time, upstream_cache_status, http_referer, http_user_agent, scheme, ssl_protocol, ssl_cipher, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0
	}
	defer stmt.Close()

	inserted := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw rawAccessLog
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		method, uri := parseRequest(raw.Request)
		ts := parseISOTime(raw.TimeISO8601)
		upstreamRT := parseFloatFirst(raw.UpstreamResponseTime)
		_, _ = stmt.Exec(
			ts,
			nodeID,
			nodeIP,
			raw.RemoteAddr,
			raw.Host,
			method,
			uri,
			raw.Status,
			raw.BodyBytesSent,
			raw.RequestTime,
			raw.UpstreamAddr,
			upstreamRT,
			raw.UpstreamCacheStatus,
			raw.HttpReferer,
			raw.HttpUserAgent,
			raw.Scheme,
			raw.SSLProtocol,
			raw.SSLCipher,
			line,
		)
		inserted++
	}
	return inserted
}

func InsertMetrics(nodeID, nodeIP string, content string) int {
	if !db.ClickHouseEnabled() || strings.TrimSpace(content) == "" {
		return 0
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_metrics
		(ts, node_id, node_ip, metric, labels, value)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(strings.NewReader(content))
	inserted := 0
	now := time.Now()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		metric, labels, value, ok := parseMetricLine(line)
		if !ok {
			continue
		}
		_, _ = stmt.Exec(now, nodeID, nodeIP, metric, labels, value)
		inserted++
	}
	return inserted
}

func InsertEventLogs(nodeID, nodeIP, eventType string, payloads []string) int {
	if !db.ClickHouseEnabled() || len(payloads) == 0 {
		return 0
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_events
		(ts, node_id, node_ip, event_type, payload)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0
	}
	defer stmt.Close()

	now := time.Now()
	inserted := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}
		_, _ = stmt.Exec(now, nodeID, nodeIP, eventType, payload)
		inserted++
	}
	return inserted
}

func parseRequest(request string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(request), " ", 3)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func parseISOTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now()
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts
	}
	return time.Now()
}

func parseFloatFirst(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return 0
	}
	if strings.Contains(value, ",") {
		value = strings.Split(value, ",")[0]
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
		return f
	}
	return 0
}

func parseMetricLine(line string) (string, string, float64, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", 0, false
	}
	metricPart := parts[0]
	valuePart := parts[1]
	metric := metricPart
	labels := ""
	if idx := strings.Index(metricPart, "{"); idx >= 0 {
		metric = metricPart[:idx]
		labels = strings.TrimSuffix(metricPart[idx+1:], "}")
	}
	value, err := strconv.ParseFloat(valuePart, 64)
	if err != nil {
		return "", "", 0, false
	}
	return metric, labels, value, true
}

func SplitLinesWithLimit(input string, limit int) []string {
	if limit <= 0 {
		limit = 1000
	}
	scanner := bufio.NewScanner(bytes.NewBufferString(input))
	out := make([]string, 0, limit)
	for scanner.Scan() {
		if len(out) >= limit {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
