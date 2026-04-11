package services

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cdn-api/db"
)

func QueryNodeBandwidthPeakMbps(start, end time.Time) (float64, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0, nil
	}
	bucketSeconds := resolveNodeBandwidthBucketSeconds(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return 0, nil
	}

	query := fmt.Sprintf(`
SELECT max(total_mbps) AS peak_mbps
FROM (
  SELECT bucket, sum(bytes_per_sec) * 8 / 1000000 AS total_mbps
  FROM (
    SELECT
      toStartOfInterval(ts, INTERVAL %d SECOND) AS bucket,
      node_id, metric, labels,
      greatest(argMax(value, ts) - argMin(value, ts), 0) /
        greatest(toUnixTimestamp(max(ts)) - toUnixTimestamp(min(ts)), 1) AS bytes_per_sec
    FROM node_metrics
    WHERE metric IN ('node_network_receive_bytes_total', 'node_network_transmit_bytes_total')
      AND ts >= ? AND ts <= ?
      AND labels NOT LIKE ?
    GROUP BY bucket, node_id, metric, labels
  )
  GROUP BY bucket
)`, bucketSeconds)

	args := []interface{}{start, end, `%device="lo"%`}
	if httpCfg != nil {
		return queryNodeBandwidthPeakHTTP(httpCfg, query, args...)
	}

	var peak sql.NullFloat64
	if err := db.CK.QueryRow(query, args...).Scan(&peak); err != nil {
		return 0, err
	}
	if !peak.Valid || peak.Float64 < 0 {
		return 0, nil
	}
	return peak.Float64, nil
}

func queryNodeBandwidthPeakHTTP(cfg *httpCKConfig, query string, args ...interface{}) (float64, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return 0, err
	}
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
		peak := toFloat64Any(raw["peak_mbps"])
		if peak < 0 {
			return 0, nil
		}
		return peak, nil
	}
	return 0, nil
}

func resolveNodeBandwidthBucketSeconds(start, end time.Time) int {
	total := end.Sub(start)
	if total <= 48*time.Hour {
		return 60
	}
	if total <= 14*24*time.Hour {
		return 300
	}
	if total <= 45*24*time.Hour {
		return 3600
	}
	return 3600
}
