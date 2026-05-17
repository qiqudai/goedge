package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
)

var blockedStatusCodes = []int{403, 418, 429, 451, 410}

// AccessBucket aggregates stats for a time bucket.
type AccessBucket struct {
	Bucket      time.Time
	Requests    uint64
	Bytes       uint64
	HitCount    uint64
	OriginBytes uint64
	Status4xx   uint64
	Status5xx   uint64
	BlockedIPs  uint64
}

// AccessTotals aggregates totals across the full range.
type AccessTotals struct {
	Requests   uint64
	Bytes      uint64
	BlockedIPs uint64
}

// BucketSeries aligns bucket data to the expected range buckets.
type BucketSeries struct {
	XAxis       []string
	Requests    []uint64
	Bytes       []uint64
	HitCount    []uint64
	OriginBytes []uint64
	Status4xx   []uint64
	Status5xx   []uint64
	BlockedIPs  []uint64
}

func BlockedStatusCodes() []int {
	return blockedStatusCodes
}

func blockedStatusCondition() string {
	parts := make([]string, 0, len(blockedStatusCodes))
	for _, code := range blockedStatusCodes {
		parts = append(parts, fmt.Sprintf("%d", code))
	}
	return "(status IN (" + strings.Join(parts, ",") + ") AND block_source != 'origin' AND NOT (block_source = '' AND upstream_addr != ''))"
}

// Some local protection paths currently return 503 without upstream interaction.
// Treat these as protection blocks, not origin/server 5xx, to avoid polluting 5xx health metrics.
func real5xxCondition() string {
	return "(status >= 500 AND status < 600 AND (block_source = 'origin' OR (block_source = '' AND NOT (status = 503 AND upstream_addr = '' AND request_time = 0 AND upstream_response_time = 0))))"
}

func bucketExpression(bucket time.Duration) string {
	if bucket >= 24*time.Hour {
		return "toStartOfDay(ts, 'UTC')"
	}
	seconds := int(bucket.Seconds())
	if seconds <= 0 {
		seconds = 60
	}
	return fmt.Sprintf("toStartOfInterval(ts, INTERVAL %d SECOND, 'UTC')", seconds)
}

func BuildBucketSeries(rng StatsRange, buckets []AccessBucket) BucketSeries {
	series := BucketSeries{}
	if rng.Start.IsZero() || rng.End.IsZero() || rng.End.Before(rng.Start) || rng.Bucket <= 0 {
		return series
	}
	bucketSeconds := int64(rng.Bucket.Seconds())
	if bucketSeconds <= 0 {
		bucketSeconds = 60
	}
	bucketMap := make(map[int64]AccessBucket, len(buckets))
	for _, bucket := range buckets {
		key := bucket.Bucket.Unix() / bucketSeconds
		bucketMap[key] = bucket
	}
	start := AlignToBucket(rng.Start, rng.Bucket)
	end := AlignToBucket(rng.End, rng.Bucket)
	for cur := start; !cur.After(end); cur = cur.Add(rng.Bucket) {
		key := cur.Unix() / bucketSeconds
		entry, ok := bucketMap[key]
		if !ok {
			entry = AccessBucket{Bucket: cur}
		}
		series.XAxis = append(series.XAxis, cur.Format(rng.LabelFormat))
		series.Requests = append(series.Requests, entry.Requests)
		series.Bytes = append(series.Bytes, entry.Bytes)
		series.HitCount = append(series.HitCount, entry.HitCount)
		series.OriginBytes = append(series.OriginBytes, entry.OriginBytes)
		series.Status4xx = append(series.Status4xx, entry.Status4xx)
		series.Status5xx = append(series.Status5xx, entry.Status5xx)
		series.BlockedIPs = append(series.BlockedIPs, entry.BlockedIPs)
	}
	return series
}

func QueryAccessBuckets(start, end time.Time, bucket time.Duration, hostFilter HostFilter) ([]AccessBucket, error) {
	return queryAccessBucketsWithExtraCondition(start, end, bucket, hostFilter, "")
}

func QueryAccessBucketsRealTraffic(start, end time.Time, bucket time.Duration, hostFilter HostFilter) ([]AccessBucket, error) {
	return queryAccessBucketsWithExtraCondition(start, end, bucket, hostFilter, AccessLogRealSiteTrafficCondition())
}

func queryAccessBucketsWithExtraCondition(start, end time.Time, bucket time.Duration, hostFilter HostFilter, extraCondition string) ([]AccessBucket, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) || bucket <= 0 {
		return []AccessBucket{}, nil
	}
	start, end, bucketDisplayShift := accessLogQueryWindow(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []AccessBucket{}, nil
	}
	bucketExpr := bucketExpression(bucket)
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	extraCondition = strings.TrimSpace(extraCondition)
	if extraCondition != "" {
		conditions = append(conditions, extraCondition)
	}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS bucket,
		count() AS requests,
		sum(bytes) AS out_bytes,
		countIf(upstream_cache_status = 'HIT') AS hit_count,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_bytes,
		countIf(status >= 400 AND status < 500) AS status_4xx,
		countIf(%s) AS status_5xx,
		uniqExactIf(remote_addr, %s) AS blocked_ips
		FROM node_access_logs WHERE %s
		GROUP BY bucket ORDER BY bucket`, bucketExpr, real5xxCondition(), blockedStatusCondition(), whereSQL)

	if httpCfg != nil {
		return queryAccessBucketsHTTP(httpCfg, bucketDisplayShift, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AccessBucket, 0)
	for rows.Next() {
		var bucketTime time.Time
		var row AccessBucket
		if err := rows.Scan(
			&bucketTime,
			&row.Requests,
			&row.Bytes,
			&row.HitCount,
			&row.OriginBytes,
			&row.Status4xx,
			&row.Status5xx,
			&row.BlockedIPs,
		); err != nil {
			continue
		}
		row.Bucket = bucketTime.Add(bucketDisplayShift)
		list = append(list, row)
	}
	return list, nil
}

func QueryAccessTotals(start, end time.Time, hostFilter HostFilter) (AccessTotals, error) {
	return queryAccessTotalsWithExtraCondition(start, end, hostFilter, "")
}

func QueryAccessTotalsRealTraffic(start, end time.Time, hostFilter HostFilter) (AccessTotals, error) {
	return queryAccessTotalsWithExtraCondition(start, end, hostFilter, AccessLogRealSiteTrafficCondition())
}

func queryAccessTotalsWithExtraCondition(start, end time.Time, hostFilter HostFilter, extraCondition string) (AccessTotals, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return AccessTotals{}, nil
	}
	start, end = adjustAccessLogQueryRange(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return AccessTotals{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	extraCondition = strings.TrimSpace(extraCondition)
	if extraCondition != "" {
		conditions = append(conditions, extraCondition)
	}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT count() AS requests,
		sum(bytes) AS bytes,
		uniqExactIf(remote_addr, %s) AS blocked_ips
		FROM node_access_logs WHERE %s`, blockedStatusCondition(), whereSQL)

	if httpCfg != nil {
		return queryAccessTotalsHTTP(httpCfg, query, args...)
	}
	var totals AccessTotals
	if err := db.CK.QueryRow(query, args...).Scan(&totals.Requests, &totals.Bytes, &totals.BlockedIPs); err != nil {
		return AccessTotals{}, err
	}
	return totals, nil
}

type accessBucketRow struct {
	Bucket      string `json:"bucket"`
	Requests    uint64 `json:"requests"`
	Bytes       uint64 `json:"bytes"`
	HitCount    uint64 `json:"hit_count"`
	OriginBytes uint64 `json:"origin_bytes"`
	Status4xx   uint64 `json:"status_4xx"`
	Status5xx   uint64 `json:"status_5xx"`
	BlockedIPs  uint64 `json:"blocked_ips"`
}

type accessTotalsRow struct {
	Requests   uint64 `json:"requests"`
	Bytes      uint64 `json:"bytes"`
	BlockedIPs uint64 `json:"blocked_ips"`
}

func queryAccessBucketsHTTP(cfg *httpCKConfig, bucketDisplayShift time.Duration, query string, args ...interface{}) ([]AccessBucket, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]AccessBucket, 0)
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
		bucketTime, err := parseCKTimeString(bucketRaw)
		if err != nil {
			continue
		}
		list = append(list, AccessBucket{
			Bucket:      bucketTime.Add(bucketDisplayShift),
			Requests:    toUint64Any(raw["requests"]),
			Bytes:       pickUint64Any(raw, "out_bytes", "bytes"),
			HitCount:    toUint64Any(raw["hit_count"]),
			OriginBytes: toUint64Any(raw["origin_bytes"]),
			Status4xx:   toUint64Any(raw["status_4xx"]),
			Status5xx:   toUint64Any(raw["status_5xx"]),
			BlockedIPs:  toUint64Any(raw["blocked_ips"]),
		})
	}
	return list, nil
}

func parseCKTimeString(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	// ClickHouse HTTP responses in this project carry UTC wall-clock strings
	// (without timezone suffix) after UTC migration. Parse those layouts in UTC
	// first so bucket alignment remains correct across API host timezones.
	naiveLayouts := []string{
		statsTimeLayout,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000000",
	}
	for _, layout := range naiveLayouts {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return t, nil
		}
	}

	offsetLayouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range offsetLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", raw)
}

func queryAccessTotalsHTTP(cfg *httpCKConfig, query string, args ...interface{}) (AccessTotals, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return AccessTotals{}, err
	}
	var totals AccessTotals
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		totals.Requests = toUint64Any(raw["requests"])
		totals.Bytes = toUint64Any(raw["bytes"])
		totals.BlockedIPs = toUint64Any(raw["blocked_ips"])
		break
	}
	return totals, nil
}

func toStringAny(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toUint64Any(v interface{}) uint64 {
	switch t := v.(type) {
	case uint64:
		return t
	case int64:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case int:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case float64:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			if i < 0 {
				return 0
			}
			return uint64(i)
		}
		if f, err := strconv.ParseFloat(t.String(), 64); err == nil {
			if f < 0 {
				return 0
			}
			return uint64(f)
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		if i, err := strconv.ParseUint(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			if f < 0 {
				return 0
			}
			return uint64(f)
		}
	}
	return 0
}

func pickUint64Any(m map[string]interface{}, keys ...string) uint64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return toUint64Any(v)
		}
	}
	return 0
}

func interpolateQuery(query string, args ...interface{}) string {
	nextPlaceholder := func(sql string) int {
		inSingle := false
		inDouble := false
		for i := 0; i < len(sql); i++ {
			ch := sql[i]
			switch ch {
			case '\\':
				if inSingle || inDouble {
					i++
				}
			case '\'':
				if !inDouble {
					if inSingle && i+1 < len(sql) && sql[i+1] == '\'' {
						i++
						continue
					}
					inSingle = !inSingle
				}
			case '"':
				if !inSingle {
					if inDouble && i+1 < len(sql) && sql[i+1] == '"' {
						i++
						continue
					}
					inDouble = !inDouble
				}
			case '?':
				if !inSingle && !inDouble {
					return i
				}
			}
		}
		return -1
	}

	for _, arg := range args {
		replacement := ""
		switch v := arg.(type) {
		case time.Time:
			// Use epoch seconds with explicit UTC timezone so the query range matches
			// the UTC wall-clock strings stored in the ts column, regardless of the
			// ClickHouse server's local timezone setting.
			replacement = fmt.Sprintf("toDateTime(%d, 'UTC')", v.Unix())
		case string:
			replacement = quoteClickHouseString(v)
		case int:
			replacement = fmt.Sprintf("%d", v)
		case int64:
			replacement = fmt.Sprintf("%d", v)
		case uint64:
			replacement = fmt.Sprintf("%d", v)
		case float64:
			replacement = fmt.Sprintf("%f", v)
		default:
			replacement = fmt.Sprintf("%v", v)
		}
		idx := nextPlaceholder(query)
		if idx < 0 {
			break
		}
		query = query[:idx] + replacement + query[idx+1:]
	}
	return query
}
