package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"cdn-api/db"
)

var accessLogTimeSkewCache struct {
	mu       sync.Mutex
	offset   time.Duration
	cachedAt time.Time
}

func adjustAccessLogQueryRange(start, end time.Time) (time.Time, time.Time) {
	adjustedStart, adjustedEnd, _ := accessLogQueryWindow(start, end)
	return adjustedStart, adjustedEnd
}

func accessLogQueryWindow(start, end time.Time) (time.Time, time.Time, time.Duration) {
	skew := detectAccessLogTimeSkew()
	adjustedStart, adjustedEnd, displayShift := adjustAccessLogQueryRangeForSkew(start, end, time.Now(), skew)
	return adjustedStart, adjustedEnd, displayShift
}

func adjustAccessLogQueryRangeForSkew(start, end, now time.Time, skew time.Duration) (time.Time, time.Time, time.Duration) {
	if start.IsZero() || end.IsZero() || end.Before(start) || skew == 0 {
		return start, end, 0
	}
	if end.Sub(start) > 2*time.Hour {
		return start, end, 0
	}
	if end.Before(now.Add(-15*time.Minute)) || end.After(now.Add(15*time.Minute)) {
		return start, end, 0
	}
	return start.Add(skew), end.Add(skew), -skew
}

func detectAccessLogTimeSkew() time.Duration {
	accessLogTimeSkewCache.mu.Lock()
	defer accessLogTimeSkewCache.mu.Unlock()

	if time.Since(accessLogTimeSkewCache.cachedAt) < time.Minute {
		return accessLogTimeSkewCache.offset
	}

	offset := queryAccessLogTimeSkew()
	accessLogTimeSkewCache.offset = offset
	accessLogTimeSkewCache.cachedAt = time.Now()
	if offset != 0 {
		log.Printf("[CK] detected node_access_logs time skew: %s", offset.String())
	}
	return offset
}

func queryAccessLogTimeSkew() time.Duration {
	httpCfg := buildHTTPConfig()
	if httpCfg != nil {
		query := "SELECT toUnixTimestamp(max(ts)) AS max_ts, toUnixTimestamp(now()) AS now_ts FROM node_access_logs FORMAT JSONEachRow"
		body, err := queryClickHouseHTTP(httpCfg, query)
		if err != nil {
			return 0
		}
		scanner := bufio.NewScanner(bytes.NewReader(body))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var anyRow map[string]interface{}
			if err := json.Unmarshal([]byte(line), &anyRow); err != nil {
				continue
			}
			maxTS := toInt64(anyRow["max_ts"])
			nowTS := toInt64(anyRow["now_ts"])
			return normalizeSkew(maxTS, nowTS)
		}
		return 0
	}

	if !db.ClickHouseEnabled() || db.CK == nil {
		return 0
	}
	var maxTS, nowTS int64
	if err := db.CK.QueryRow("SELECT toUnixTimestamp(max(ts)), toUnixTimestamp(now()) FROM node_access_logs").Scan(&maxTS, &nowTS); err != nil {
		return 0
	}
	return normalizeSkew(maxTS, nowTS)
}

func normalizeSkew(maxTS, nowTS int64) time.Duration {
	if maxTS == 0 || nowTS == 0 {
		return 0
	}
	skew := time.Duration(maxTS-nowTS) * time.Second
	if skew < -14*time.Hour || skew > 14*time.Hour {
		return 0
	}
	if skew > -90*time.Second && skew < 90*time.Second {
		return 0
	}
	return skew
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
	}
	return 0
}
