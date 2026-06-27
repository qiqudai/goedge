package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
)

type rawAccessLog struct {
	TimeISO8601          string  `json:"time_iso8601"`
	RemoteAddr           string  `json:"remote_addr"`
	ClientCountry        string  `json:"client_country"`
	ClientProvince       string  `json:"client_province"`
	ClientCity           string  `json:"client_city"`
	ClientISP            string  `json:"client_isp"`
	SiteName             string  `json:"site_name"`
	Host                 string  `json:"host"`
	HTTPHost             string  `json:"http_host"`
	SSLServerName        string  `json:"ssl_server_name"`
	Request              string  `json:"request"`
	Status               int     `json:"status"`
	BodyBytesSent        int64   `json:"body_bytes_sent"`
	RequestTime          float64 `json:"request_time"`
	UpstreamAddr         string  `json:"upstream_addr"`
	UpstreamConnectTime  string  `json:"upstream_connect_time"`
	UpstreamHeaderTime   string  `json:"upstream_header_time"`
	UpstreamResponseTime string  `json:"upstream_response_time"`
	UpstreamCacheStatus  string  `json:"upstream_cache_status"`
	BlockSource          string  `json:"block_source"`
	HttpReferer          string  `json:"http_referer"`
	HttpUserAgent        string  `json:"http_user_agent"`
	CDNReqHeaders        string  `json:"cdn_req_headers"`
	Scheme               string  `json:"scheme"`
	SSLProtocol          string  `json:"ssl_protocol"`
	SSLCipher            string  `json:"ssl_cipher"`
}

type rawStreamLog struct {
	TimeISO8601           string `json:"time_iso8601"`
	RemoteAddr            string `json:"remote_addr"`
	Protocol              string `json:"protocol"`
	Status                int    `json:"status"`
	BytesSent             int64  `json:"bytes_sent"`
	BytesReceived         int64  `json:"bytes_received"`
	SessionTime           string `json:"session_time"`
	UpstreamAddr          string `json:"upstream_addr"`
	UpstreamBytesSent     string `json:"upstream_bytes_sent"`
	UpstreamBytesReceived string `json:"upstream_bytes_received"`
	UpstreamConnectTime   string `json:"upstream_connect_time"`
	UpstreamSessionTime   string `json:"upstream_session_time"`
	ServerPort            int    `json:"server_port"`
}

const accessLogMaxInsertAge = 24 * time.Hour

func InsertAccessLogs(nodeID, nodeIP string, lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	lines = filterAccessLogLinesForInsert(lines)
	if len(lines) == 0 {
		return 0
	}

	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		log.Printf("[CK] Access logs skipped: ClickHouse unavailable (native/http)")
		return 0
	}

	if httpCfg != nil {
		rows := make([]map[string]interface{}, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var raw rawAccessLog
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				continue
			}
			if isInternalMetricsAccess(raw) {
				continue
			}
			method, uri := parseRequest(raw.Request)
			ts := formatTime(parseISOTime(raw.TimeISO8601))
			upstreamCT := parseFloatFirst(raw.UpstreamConnectTime)
			upstreamHT := parseFloatFirst(raw.UpstreamHeaderTime)
			upstreamRT := parseFloatFirst(raw.UpstreamResponseTime)
			cacheStatus := normalizeCacheStatus(raw.UpstreamCacheStatus)
			blockSource := normalizeBlockSource(raw)
			host := effectiveAccessHost(raw)
			slowReason, slowAdvice := DiagnoseAccessLogSlowReason(DiagnoseInput{
				RequestTime:          raw.RequestTime,
				UpstreamConnectTime:  upstreamCT,
				UpstreamHeaderTime:   upstreamHT,
				UpstreamResponseTime: upstreamRT,
				UpstreamCacheStatus:  cacheStatus,
				Status:               raw.Status,
				Scheme:               raw.Scheme,
				SSLProtocol:          raw.SSLProtocol,
			})
			rows = append(rows, map[string]interface{}{
				"ts":                     ts,
				"node_id":                nodeID,
				"node_ip":                nodeIP,
				"remote_addr":            raw.RemoteAddr,
				"client_country":         raw.ClientCountry,
				"client_province":        raw.ClientProvince,
				"client_city":            raw.ClientCity,
				"client_isp":             raw.ClientISP,
				"site_name":              raw.SiteName,
				"host":                   host,
				"method":                 method,
				"uri":                    uri,
				"status":                 raw.Status,
				"bytes":                  raw.BodyBytesSent,
				"request_time":           raw.RequestTime,
				"upstream_addr":          raw.UpstreamAddr,
				"upstream_connect_time":  upstreamCT,
				"upstream_header_time":   upstreamHT,
				"upstream_response_time": upstreamRT,
				"upstream_cache_status":  cacheStatus,
				"block_source":           blockSource,
				"slow_reason":            slowReason,
				"slow_advice":            slowAdvice,
				"http_referer":           raw.HttpReferer,
				"http_user_agent":        raw.HttpUserAgent,
				"scheme":                 raw.Scheme,
				"ssl_protocol":           raw.SSLProtocol,
				"ssl_cipher":             raw.SSLCipher,
				"raw":                    line,
			})
		}
		return insertHTTPRows(httpCfg, "node_access_logs", rows)
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_access_logs
		(ts, node_id, node_ip, remote_addr, client_country, client_province, client_city, client_isp, site_name, host, method, uri, status, bytes, request_time, upstream_addr, upstream_connect_time, upstream_header_time, upstream_response_time, upstream_cache_status, block_source, slow_reason, slow_advice, http_referer, http_user_agent, scheme, ssl_protocol, ssl_cipher, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("[CK] Prepare access logs failed: %v", err)
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
		if isInternalMetricsAccess(raw) {
			continue
		}
		method, uri := parseRequest(raw.Request)
		ts := parseISOTime(raw.TimeISO8601)
		upstreamCT := parseFloatFirst(raw.UpstreamConnectTime)
		upstreamHT := parseFloatFirst(raw.UpstreamHeaderTime)
		upstreamRT := parseFloatFirst(raw.UpstreamResponseTime)
		cacheStatus := normalizeCacheStatus(raw.UpstreamCacheStatus)
		blockSource := normalizeBlockSource(raw)
		host := effectiveAccessHost(raw)
		slowReason, slowAdvice := DiagnoseAccessLogSlowReason(DiagnoseInput{
			RequestTime:          raw.RequestTime,
			UpstreamConnectTime:  upstreamCT,
			UpstreamHeaderTime:   upstreamHT,
			UpstreamResponseTime: upstreamRT,
			UpstreamCacheStatus:  cacheStatus,
			Status:               raw.Status,
			Scheme:               raw.Scheme,
			SSLProtocol:          raw.SSLProtocol,
		})
		if _, err := stmt.Exec(
			ts,
			nodeID,
			nodeIP,
			raw.RemoteAddr,
			raw.ClientCountry,
			raw.ClientProvince,
			raw.ClientCity,
			raw.ClientISP,
			raw.SiteName,
			host,
			method,
			uri,
			raw.Status,
			raw.BodyBytesSent,
			raw.RequestTime,
			raw.UpstreamAddr,
			upstreamCT,
			upstreamHT,
			upstreamRT,
			cacheStatus,
			blockSource,
			slowReason,
			slowAdvice,
			raw.HttpReferer,
			raw.HttpUserAgent,
			raw.Scheme,
			raw.SSLProtocol,
			raw.SSLCipher,
			line,
		); err != nil {
			log.Printf("[CK] Insert access log failed: %v", err)
			continue
		}
		inserted++
	}
	return inserted
}

func isInternalMetricsAccess(raw rawAccessLog) bool {
	host := strings.TrimSpace(strings.ToLower(effectiveAccessHost(raw)))
	site := strings.TrimSpace(strings.ToLower(raw.SiteName))
	if !(host == "127.0.0.1" || host == "localhost" || strings.HasPrefix(site, "localhost:9100")) {
		return false
	}
	_, uri := parseRequest(raw.Request)
	if uri == "" {
		return false
	}
	return strings.HasPrefix(uri, "/metrics")
}

func effectiveAccessHost(raw rawAccessLog) string {
	for _, value := range []string{
		raw.HTTPHost,
		hostFromReqHeaders(raw.CDNReqHeaders),
		raw.SSLServerName,
		raw.Host,
	} {
		if host := normalizeAccessHost(value); host != "" {
			return host
		}
	}
	return strings.TrimSpace(raw.Host)
}

func hostFromReqHeaders(rawHeaders string) string {
	rawHeaders = strings.TrimSpace(rawHeaders)
	if rawHeaders == "" {
		return ""
	}
	var headers map[string]interface{}
	if err := json.Unmarshal([]byte(rawHeaders), &headers); err != nil {
		return ""
	}
	for key, value := range headers {
		if strings.EqualFold(strings.TrimSpace(key), "host") {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func normalizeAccessHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || host == "-" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if idx := strings.Index(host, "]"); idx > 0 {
			return strings.ToLower(host[1:idx])
		}
	}
	if strings.Count(host, ":") == 1 {
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}
	}
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

func InsertStreamLogs(nodeID, nodeIP string, lines []string) int {
	if len(lines) == 0 {
		return 0
	}

	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		log.Printf("[CK] Stream logs skipped: ClickHouse unavailable (native/http)")
		return 0
	}

	if httpCfg != nil {
		rows := make([]map[string]interface{}, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var raw rawStreamLog
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				continue
			}
			sessionTime := parseFloatFirst(raw.SessionTime)
			upstreamBytesSent := parseInt64First(raw.UpstreamBytesSent)
			upstreamBytesReceived := parseInt64First(raw.UpstreamBytesReceived)
			upstreamConnectTime := parseFloatFirst(raw.UpstreamConnectTime)
			upstreamSessionTime := parseFloatFirst(raw.UpstreamSessionTime)
			rows = append(rows, map[string]interface{}{
				"ts":                      formatTime(parseISOTime(raw.TimeISO8601)),
				"node_id":                 nodeID,
				"node_ip":                 nodeIP,
				"remote_addr":             raw.RemoteAddr,
				"server_port":             raw.ServerPort,
				"protocol":                raw.Protocol,
				"status":                  raw.Status,
				"bytes_sent":              raw.BytesSent,
				"bytes_received":          raw.BytesReceived,
				"session_time":            sessionTime,
				"upstream_addr":           raw.UpstreamAddr,
				"upstream_bytes_sent":     upstreamBytesSent,
				"upstream_bytes_received": upstreamBytesReceived,
				"upstream_connect_time":   upstreamConnectTime,
				"upstream_session_time":   upstreamSessionTime,
				"raw":                     line,
			})
		}
		return insertHTTPRows(httpCfg, "node_stream_logs", rows)
	}

	stmt, err := db.CK.Prepare(`INSERT INTO node_stream_logs
		(ts, node_id, node_ip, remote_addr, server_port, protocol, status, bytes_sent, bytes_received, session_time, upstream_addr, upstream_bytes_sent, upstream_bytes_received, upstream_connect_time, upstream_session_time, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("[CK] Prepare stream logs failed: %v", err)
		return 0
	}
	defer stmt.Close()

	inserted := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw rawStreamLog
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		sessionTime := parseFloatFirst(raw.SessionTime)
		upstreamBytesSent := parseInt64First(raw.UpstreamBytesSent)
		upstreamBytesReceived := parseInt64First(raw.UpstreamBytesReceived)
		upstreamConnectTime := parseFloatFirst(raw.UpstreamConnectTime)
		upstreamSessionTime := parseFloatFirst(raw.UpstreamSessionTime)
		ts := parseISOTime(raw.TimeISO8601)
		if _, err := stmt.Exec(
			ts,
			nodeID,
			nodeIP,
			raw.RemoteAddr,
			raw.ServerPort,
			raw.Protocol,
			raw.Status,
			raw.BytesSent,
			raw.BytesReceived,
			sessionTime,
			raw.UpstreamAddr,
			upstreamBytesSent,
			upstreamBytesReceived,
			upstreamConnectTime,
			upstreamSessionTime,
			line,
		); err != nil {
			log.Printf("[CK] Insert stream log failed: %v", err)
			continue
		}
		inserted++
	}
	return inserted
}

func InsertMetrics(nodeID, nodeIP string, content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}

	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		log.Printf("[CK] Metrics skipped: ClickHouse unavailable (native/http)")
		return 0
	}

	if httpCfg != nil {
		scanner := bufio.NewScanner(strings.NewReader(content))
		rows := make([]map[string]interface{}, 0, 100)
		now := formatTime(time.Now().UTC())
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			metric, labels, value, ok := parseMetricLine(line)
			if !ok {
				continue
			}
			rows = append(rows, map[string]interface{}{
				"ts":      now,
				"node_id": nodeID,
				"node_ip": nodeIP,
				"metric":  metric,
				"labels":  labels,
				"value":   value,
			})
		}
		return insertHTTPRows(httpCfg, "node_metrics", rows)
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_metrics
		(ts, node_id, node_ip, metric, labels, value)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("[CK] Prepare metrics failed: %v", err)
		return 0
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(strings.NewReader(content))
	inserted := 0
	now := time.Now().UTC()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		metric, labels, value, ok := parseMetricLine(line)
		if !ok {
			continue
		}
		if _, err := stmt.Exec(now, nodeID, nodeIP, metric, labels, value); err != nil {
			log.Printf("[CK] Insert metric failed: %v", err)
			continue
		}
		inserted++
	}
	return inserted
}

func InsertEventLogs(nodeID, nodeIP, eventType string, payloads []string) int {
	if len(payloads) == 0 {
		return 0
	}

	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		log.Printf("[CK] Events skipped: ClickHouse unavailable (native/http)")
		return 0
	}

	if httpCfg != nil {
		rows := make([]map[string]interface{}, 0, len(payloads))
		now := formatTime(time.Now().UTC())
		for _, payload := range payloads {
			payload = strings.TrimSpace(payload)
			if payload == "" {
				continue
			}
			rows = append(rows, map[string]interface{}{
				"ts":         now,
				"node_id":    nodeID,
				"node_ip":    nodeIP,
				"event_type": eventType,
				"payload":    payload,
			})
		}
		return insertHTTPRows(httpCfg, "node_events", rows)
	}
	stmt, err := db.CK.Prepare(`INSERT INTO node_events
		(ts, node_id, node_ip, event_type, payload)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("[CK] Prepare events failed: %v", err)
		return 0
	}
	defer stmt.Close()

	now := time.Now().UTC()
	inserted := 0
	for _, payload := range payloads {
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue
		}
		if _, err := stmt.Exec(now, nodeID, nodeIP, eventType, payload); err != nil {
			log.Printf("[CK] Insert event failed: %v", err)
			continue
		}
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

func filterAccessLogLinesForInsert(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	cutoff := time.Now().UTC().Add(-accessLogMaxInsertAge)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ts := parseAccessLogLineTime(line)
		if !ts.IsZero() && ts.Before(cutoff) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func parseAccessLogLineTime(line string) time.Time {
	var raw struct {
		TimeISO8601 string `json:"time_iso8601"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return time.Time{}
	}
	return parseISOTime(raw.TimeISO8601)
}

func parseISOTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UTC()
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts.UTC()
	}
	return time.Now().UTC()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	// Return Unix timestamp as integer string so ClickHouse stores it as UTC
	// regardless of the server's local timezone setting.
	return fmt.Sprintf("%d", value.UTC().Unix())
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

func parseInt64First(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return 0
	}
	if strings.Contains(value, ",") {
		value = strings.Split(value, ",")[0]
	}
	if num, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
		return num
	}
	if num, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64); err == nil {
		return int64(num)
	}
	return 0
}

func normalizeBlockSource(raw rawAccessLog) string {
	sourceRaw := strings.TrimSpace(raw.BlockSource)
	source := strings.ToLower(sourceRaw)
	switch source {
	case "anti_cc", "ip_block", "waf", "cc", "cc_rate_limit", "cc_guard", "local_protection", "origin":
		return source
	}
	// Keep structured source untouched (type/rule/rule_id/config...) for attribution.
	if strings.Contains(sourceRaw, "=") {
		return sourceRaw
	}
	// Fallback inference for older agents that do not emit block_source.
	if strings.TrimSpace(raw.UpstreamAddr) != "" || parseFloatFirst(raw.UpstreamResponseTime) > 0 {
		return "origin"
	}
	if raw.Status == 418 || raw.Status == 429 || raw.Status == 403 || raw.Status == 515 {
		return "local_protection"
	}
	if raw.Status == 503 && raw.RequestTime == 0 {
		return "local_protection"
	}
	return ""
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

type httpCKConfig struct {
	baseURL  string
	user     string
	pass     string
	database string
}

func buildHTTPConfig() *httpCKConfig {
	dsn := strings.TrimSpace(config.App.ClickHouseDSN)
	if dsn == "" {
		return nil
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil
	}
	dbName := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if dbName == "" {
		dbName = parsed.Query().Get("database")
	}
	user := ""
	pass := ""
	if parsed.User != nil {
		user = parsed.User.Username()
		pass, _ = parsed.User.Password()
	}
	baseURL := parsed.Scheme + "://" + parsed.Host
	return &httpCKConfig{
		baseURL:  baseURL,
		user:     user,
		pass:     pass,
		database: dbName,
	}
}

func insertHTTPRows(cfg *httpCKConfig, table string, rows []map[string]interface{}) int {
	if cfg == nil || len(rows) == 0 {
		return 0
	}
	query := "INSERT INTO " + table + " FORMAT JSONEachRow"
	params := url.Values{}
	params.Set("query", query)
	if cfg.database != "" {
		params.Set("database", cfg.database)
	}
	endpoint := cfg.baseURL + "/?" + params.Encode()

	var body bytes.Buffer
	for _, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			continue
		}
		body.Write(line)
		body.WriteByte('\n')
	}

	req, err := http.NewRequest("POST", endpoint, &body)
	if err != nil {
		log.Printf("[CK] HTTP insert build failed: %v", err)
		return 0
	}
	if cfg.user != "" {
		req.SetBasicAuth(cfg.user, cfg.pass)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[CK] HTTP insert failed: %v", err)
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[CK] HTTP insert status: %s", resp.Status)
		return 0
	}
	return len(rows)
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
