package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cdn-api/db"
)

type ForwardTrafficBucket struct {
	Bucket     time.Time
	TotalBytes uint64
}

type ForwardPortRank struct {
	Port        int
	Protocol    string
	Connections uint64
	TotalBytes  uint64
}

func QueryForwardTrafficBuckets(start, end time.Time, bucketMinutes int, port int, protocol string) ([]ForwardTrafficBucket, error) {
	return queryForwardTrafficBuckets(start, end, bucketMinutes, port, protocol, nil)
}

func QueryForwardTrafficBucketsWithPorts(start, end time.Time, bucketMinutes int, port int, protocol string, allowedPorts []int) ([]ForwardTrafficBucket, error) {
	return queryForwardTrafficBuckets(start, end, bucketMinutes, port, protocol, allowedPorts)
}

func queryForwardTrafficBuckets(start, end time.Time, bucketMinutes int, port int, protocol string, allowedPorts []int) ([]ForwardTrafficBucket, error) {
	if bucketMinutes <= 0 {
		bucketMinutes = 1
	}
	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryForwardTrafficBucketsHTTP(httpCfg, start, end, bucketMinutes, port, protocol, allowedPorts)
	}
	if !db.ClickHouseEnabled() {
		return []ForwardTrafficBucket{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	if port > 0 {
		conditions = append(conditions, "server_port = ?")
		args = append(args, port)
	}
	if len(allowedPorts) > 0 {
		placeholders := make([]string, 0, len(allowedPorts))
		for _, p := range allowedPorts {
			placeholders = append(placeholders, "?")
			args = append(args, p)
		}
		conditions = append(conditions, "server_port IN ("+strings.Join(placeholders, ",")+")")
	}
	if protocol != "" {
		conditions = append(conditions, "protocol = ?")
		args = append(args, protocol)
	}
	query := fmt.Sprintf(`SELECT toStartOfInterval(ts, INTERVAL %d MINUTE) AS bucket,
		sum(bytes_sent + bytes_received) AS total_bytes
		FROM node_stream_logs WHERE %s
		GROUP BY bucket ORDER BY bucket`, bucketMinutes, strings.Join(conditions, " AND "))
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	buckets := make([]ForwardTrafficBucket, 0)
	for rows.Next() {
		var bucket time.Time
		var total uint64
		if err := rows.Scan(&bucket, &total); err != nil {
			continue
		}
		buckets = append(buckets, ForwardTrafficBucket{Bucket: bucket, TotalBytes: total})
	}
	return buckets, nil
}

func QueryForwardPortRanking(start, end time.Time, limit int) ([]ForwardPortRank, error) {
	return queryForwardPortRanking(start, end, limit, nil)
}

func QueryForwardPortRankingWithPorts(start, end time.Time, limit int, allowedPorts []int) ([]ForwardPortRank, error) {
	return queryForwardPortRanking(start, end, limit, allowedPorts)
}

func queryForwardPortRanking(start, end time.Time, limit int, allowedPorts []int) ([]ForwardPortRank, error) {
	if limit <= 0 {
		limit = 20
	}
	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryForwardPortRankingHTTP(httpCfg, start, end, limit, allowedPorts)
	}
	if !db.ClickHouseEnabled() {
		return []ForwardPortRank{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	if len(allowedPorts) > 0 {
		placeholders := make([]string, 0, len(allowedPorts))
		for _, p := range allowedPorts {
			placeholders = append(placeholders, "?")
			args = append(args, p)
		}
		conditions = append(conditions, "server_port IN ("+strings.Join(placeholders, ",")+")")
	}
	query := fmt.Sprintf(`SELECT server_port, protocol, count() AS connections,
		sum(bytes_sent + bytes_received) AS total_bytes
		FROM node_stream_logs
		WHERE %s
		GROUP BY server_port, protocol
		ORDER BY total_bytes DESC
		LIMIT ?`, strings.Join(conditions, " AND "))
	args = append(args, limit)
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]ForwardPortRank, 0, limit)
	for rows.Next() {
		var port uint16
		var protocol string
		var connections uint64
		var total uint64
		if err := rows.Scan(&port, &protocol, &connections, &total); err != nil {
			continue
		}
		list = append(list, ForwardPortRank{
			Port:        int(port),
			Protocol:    protocol,
			Connections: connections,
			TotalBytes:  total,
		})
	}
	return list, nil
}

func queryForwardTrafficBucketsHTTP(cfg *httpCKConfig, start, end time.Time, bucketMinutes int, port int, protocol string, allowedPorts []int) ([]ForwardTrafficBucket, error) {
	startStr := formatTime(start)
	endStr := formatTime(end)
	conditions := []string{
		fmt.Sprintf("ts >= toDateTime('%s') AND ts <= toDateTime('%s')", startStr, endStr),
	}
	if port > 0 {
		conditions = append(conditions, fmt.Sprintf("server_port = %d", port))
	}
	if len(allowedPorts) > 0 {
		parts := make([]string, 0, len(allowedPorts))
		for _, p := range allowedPorts {
			parts = append(parts, fmt.Sprintf("%d", p))
		}
		conditions = append(conditions, "server_port IN ("+strings.Join(parts, ",")+")")
	}
	if protocol != "" {
		conditions = append(conditions, "protocol = "+quoteClickHouseString(protocol))
	}
	query := fmt.Sprintf(`SELECT toStartOfInterval(ts, INTERVAL %d MINUTE) AS bucket,
		sum(bytes_sent + bytes_received) AS total_bytes
		FROM node_stream_logs WHERE %s
		GROUP BY bucket ORDER BY bucket
		FORMAT JSONEachRow`, bucketMinutes, strings.Join(conditions, " AND "))
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	buckets := make([]ForwardTrafficBucket, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		bucketRaw := strings.TrimSpace(toStringAny(raw["bucket"]))
		if bucketRaw == "" {
			continue
		}
		bucket, err := parseCKTimeString(bucketRaw)
		if err != nil {
			continue
		}
		buckets = append(buckets, ForwardTrafficBucket{
			Bucket:     bucket,
			TotalBytes: toUint64Any(raw["total_bytes"]),
		})
	}
	return buckets, nil
}

func queryForwardPortRankingHTTP(cfg *httpCKConfig, start, end time.Time, limit int, allowedPorts []int) ([]ForwardPortRank, error) {
	startStr := formatTime(start)
	endStr := formatTime(end)
	conditions := []string{
		fmt.Sprintf("ts >= toDateTime('%s') AND ts <= toDateTime('%s')", startStr, endStr),
	}
	if len(allowedPorts) > 0 {
		parts := make([]string, 0, len(allowedPorts))
		for _, p := range allowedPorts {
			parts = append(parts, fmt.Sprintf("%d", p))
		}
		conditions = append(conditions, "server_port IN ("+strings.Join(parts, ",")+")")
	}
	query := fmt.Sprintf(`SELECT server_port, protocol, count() AS connections,
		sum(bytes_sent + bytes_received) AS total_bytes
		FROM node_stream_logs
		WHERE %s
		GROUP BY server_port, protocol
		ORDER BY total_bytes DESC
		LIMIT %d
		FORMAT JSONEachRow`, strings.Join(conditions, " AND "), limit)
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]ForwardPortRank, 0, limit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		list = append(list, ForwardPortRank{
			Port:        int(toInt64(raw["server_port"])),
			Protocol:    strings.TrimSpace(toStringAny(raw["protocol"])),
			Connections: toUint64Any(raw["connections"]),
			TotalBytes:  toUint64Any(raw["total_bytes"]),
		})
	}
	return list, nil
}

func queryClickHouseHTTP(cfg *httpCKConfig, query string) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("clickhouse http config missing")
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("output_format_json_quote_64bit_integers", "0")
	if cfg.database != "" {
		params.Set("database", cfg.database)
	}
	endpoint := cfg.baseURL + "/?" + params.Encode()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if cfg.user != "" {
		req.SetBasicAuth(cfg.user, cfg.pass)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 512 {
			detail = detail[:512]
		}
		if detail == "" {
			return nil, fmt.Errorf("clickhouse http status %s", resp.Status)
		}
		return nil, fmt.Errorf("clickhouse http status %s: %s", resp.Status, detail)
	}
	return body, nil
}
