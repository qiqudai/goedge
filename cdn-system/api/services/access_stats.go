package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
	return "status IN (" + strings.Join(parts, ",") + ")"
}

func bucketExpression(bucket time.Duration) string {
	if bucket >= 24*time.Hour {
		return "toStartOfDay(ts)"
	}
	seconds := int(bucket.Seconds())
	if seconds <= 0 {
		seconds = 60
	}
	return fmt.Sprintf("toStartOfInterval(ts, INTERVAL %d SECOND)", seconds)
}

func BuildBucketSeries(rng StatsRange, buckets []AccessBucket) BucketSeries {
	series := BucketSeries{}
	if rng.Start.IsZero() || rng.End.IsZero() || rng.End.Before(rng.Start) || rng.Bucket <= 0 {
		return series
	}
	bucketMap := make(map[time.Time]AccessBucket, len(buckets))
	for _, bucket := range buckets {
		bucketMap[AlignToBucket(bucket.Bucket, rng.Bucket)] = bucket
	}
	start := AlignToBucket(rng.Start, rng.Bucket)
	end := AlignToBucket(rng.End, rng.Bucket)
	for cur := start; !cur.After(end); cur = cur.Add(rng.Bucket) {
		entry, ok := bucketMap[cur]
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
	if start.IsZero() || end.IsZero() || end.Before(start) || bucket <= 0 {
		return []AccessBucket{}, nil
	}
	if !db.ClickHouseEnabled() {
		return []AccessBucket{}, nil
	}
	bucketExpr := bucketExpression(bucket)
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS bucket,
		count() AS requests,
		sum(bytes) AS bytes,
		countIf(upstream_cache_status = 'HIT') AS hit_count,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_bytes,
		countIf(status >= 400 AND status < 500) AS status_4xx,
		countIf(status >= 500 AND status < 600) AS status_5xx,
		uniqExactIf(remote_addr, %s) AS blocked_ips
		FROM node_access_logs WHERE %s
		GROUP BY bucket ORDER BY bucket`, bucketExpr, blockedStatusCondition(), whereSQL)

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryAccessBucketsHTTP(httpCfg, query, args...)
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
		row.Bucket = bucketTime
		list = append(list, row)
	}
	return list, nil
}

func QueryAccessTotals(start, end time.Time, hostFilter HostFilter) (AccessTotals, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return AccessTotals{}, nil
	}
	if !db.ClickHouseEnabled() {
		return AccessTotals{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT count() AS requests,
		sum(bytes) AS bytes,
		uniqExactIf(remote_addr, %s) AS blocked_ips
		FROM node_access_logs WHERE %s`, blockedStatusCondition(), whereSQL)

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
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

func queryAccessBucketsHTTP(cfg *httpCKConfig, query string, args ...interface{}) ([]AccessBucket, error) {
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
		var row accessBucketRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if row.Bucket == "" {
			continue
		}
		bucketTime, err := time.ParseInLocation(statsTimeLayout, row.Bucket, time.Local)
		if err != nil {
			continue
		}
		list = append(list, AccessBucket{
			Bucket:      bucketTime,
			Requests:    row.Requests,
			Bytes:       row.Bytes,
			HitCount:    row.HitCount,
			OriginBytes: row.OriginBytes,
			Status4xx:   row.Status4xx,
			Status5xx:   row.Status5xx,
			BlockedIPs:  row.BlockedIPs,
		})
	}
	return list, nil
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
		var row accessTotalsRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		totals.Requests = row.Requests
		totals.Bytes = row.Bytes
		totals.BlockedIPs = row.BlockedIPs
		break
	}
	return totals, nil
}

func interpolateQuery(query string, args ...interface{}) string {
	for _, arg := range args {
		replacement := ""
		switch v := arg.(type) {
		case time.Time:
			replacement = fmt.Sprintf("toDateTime('%s')", formatTime(v))
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
		query = strings.Replace(query, "?", replacement, 1)
	}
	return query
}
