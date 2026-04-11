package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
)

// RankItem aggregates request/traffic metrics for rankings.
type RankItem struct {
	Item         string
	RequestCount uint64
	OutBytes     uint64
	OriginBytes  uint64
}

// LatencyRankItem describes latency ranking.
type LatencyRankItem struct {
	Rank         int     `json:"rank"`
	Item         string  `json:"item"`
	RequestCount int     `json:"request_count"`
	AvgTime      float64 `json:"avg_time"`
	MaxTime      float64 `json:"max_time"`
	MinTime      float64 `json:"min_time"`
	P95Time      float64 `json:"p95_time"`
}

type rankingSpec struct {
	ItemExpr     string
	GroupBy      string
	KeywordCond  string
	KeywordArgs  int
	NormalizeNil bool
}

func QueryAccessRanking(rankType string, start, end time.Time, hostFilter HostFilter, keyword string, limit int) ([]RankItem, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []RankItem{}, nil
	}
	start, end = adjustAccessLogQueryRange(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []RankItem{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	spec, ok := rankingSpecForType(rankType)
	if !ok {
		return []RankItem{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" && spec.KeywordCond != "" {
		conditions = append(conditions, spec.KeywordCond)
		for i := 0; i < spec.KeywordArgs; i++ {
			args = append(args, "%"+keyword+"%")
		}
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS item,
		count() AS request_count,
		sum(bytes) AS out_traffic,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_traffic
		FROM node_access_logs WHERE %s
		GROUP BY %s
		ORDER BY request_count DESC
		LIMIT %d`, spec.ItemExpr, whereSQL, spec.GroupBy, limit)

	if httpCfg != nil {
		return queryAccessRankingHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]RankItem, 0)
	for rows.Next() {
		var item string
		var reqCount, outBytes, originBytes uint64
		if err := rows.Scan(&item, &reqCount, &outBytes, &originBytes); err != nil {
			continue
		}
		item = strings.TrimSpace(item)
		if item == "" && spec.NormalizeNil {
			item = "-"
		}
		list = append(list, RankItem{
			Item:         item,
			RequestCount: reqCount,
			OutBytes:     outBytes,
			OriginBytes:  originBytes,
		})
	}
	return list, nil
}

func QueryRegionRanking(regionType string, start, end time.Time, hostFilter HostFilter, keyword string, limit int) ([]RankItem, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []RankItem{}, nil
	}
	start, end = adjustAccessLogQueryRange(start, end)
	if !db.ClickHouseEnabled() && buildHTTPConfig() == nil {
		return []RankItem{}, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	items, err := queryRegionIPAggregates(whereSQL, args, limit)
	if err != nil {
		return nil, err
	}
	return buildRegionRankingFromIP(items, regionType, keyword, limit, LookupIPRegion), nil
}

func queryRegionIPAggregates(whereSQL string, args []interface{}, limit int) ([]RankItem, error) {
	ipSampleLimit := resolveRegionIPSampleLimit(limit)
	expressions := []string{"remote_addr", "client_ip", "realip_remote_addr"}
	var lastErr error
	for _, ipExpr := range expressions {
		query := fmt.Sprintf(`SELECT %s AS ip,
			count() AS request_count,
			sum(bytes) AS out_traffic,
			sumIf(bytes, upstream_cache_status != 'HIT') AS origin_traffic
			FROM node_access_logs WHERE %s
			GROUP BY ip
			ORDER BY request_count DESC
			LIMIT %d`, ipExpr, whereSQL, ipSampleLimit)
		items, err := queryAccessIPAggregates(query, args...)
		if err == nil {
			return items, nil
		}
		lastErr = err
		if !isUnknownIdentifierError(err) {
			// Non-column errors usually won't be fixed by switching expressions.
			return nil, err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return []RankItem{}, nil
}

func buildRegionRankingFromIP(items []RankItem, regionType, keyword string, limit int, lookup func(string) (string, string)) []RankItem {
	regionMap := make(map[string]*RankItem)
	cache := make(map[string]string)
	for _, row := range items {
		region := resolveRegionForIPWithLookup(row.Item, regionType, cache, lookup)
		if region == "" {
			region = "-"
		}
		entry := regionMap[region]
		if entry == nil {
			entry = &RankItem{Item: region}
			regionMap[region] = entry
		}
		entry.RequestCount += row.RequestCount
		entry.OutBytes += row.OutBytes
		entry.OriginBytes += row.OriginBytes
	}
	list := make([]RankItem, 0, len(regionMap))
	for _, entry := range regionMap {
		if keyword = strings.TrimSpace(keyword); keyword != "" {
			if !strings.Contains(entry.Item, keyword) {
				continue
			}
		}
		list = append(list, *entry)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].RequestCount > list[j].RequestCount
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}

func resolveRegionIPSampleLimit(limit int) int {
	// Keep region ranking fast/stable under high-cardinality IP data.
	// 5k works well for top lists while avoiding full result scans to app side.
	base := 5000
	if limit <= 0 {
		return base
	}
	candidate := limit * 300
	if candidate < base {
		candidate = base
	}
	if candidate > 50000 {
		candidate = 50000
	}
	return candidate
}

func isUnknownIdentifierError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown identifier") ||
		strings.Contains(msg, "code: 47") ||
		strings.Contains(msg, "missing columns")
}

func QueryLatencyRanking(start, end time.Time, hostFilter HostFilter, keyword string, limit int) []LatencyRankItem {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []LatencyRankItem{}
	}
	start, end = adjustAccessLogQueryRange(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []LatencyRankItem{}
	}
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{"ts >= ? AND ts <= ? AND request_time > 0"}
	args := []interface{}{start, end}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		conditions = append(conditions, "("+AccessLogSiteExpr()+" LIKE ? OR uri LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS item_host, uri, count() AS request_count,
		avg(request_time) AS avg_time,
		max(request_time) AS max_time,
		min(request_time) AS min_time,
		quantile(0.95)(request_time) AS p95_time
		FROM node_access_logs WHERE %s
		GROUP BY item_host, uri
		ORDER BY avg_time DESC
		LIMIT %d`, AccessLogSiteExpr(), whereSQL, limit)

	if httpCfg != nil {
		return queryLatencyRankingHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return []LatencyRankItem{}
	}
	defer rows.Close()
	list := make([]LatencyRankItem, 0)
	rank := 1
	for rows.Next() {
		var host, uri string
		var reqCount uint64
		var avgTime, maxTime, minTime, p95Time float64
		if err := rows.Scan(&host, &uri, &reqCount, &avgTime, &maxTime, &minTime, &p95Time); err != nil {
			continue
		}
		item := strings.TrimSpace(host)
		if uri != "" {
			item = item + uri
		}
		list = append(list, LatencyRankItem{
			Rank:         rank,
			Item:         item,
			RequestCount: int(reqCount),
			AvgTime:      RoundFloat(avgTime*1000, 2),
			MaxTime:      RoundFloat(maxTime*1000, 2),
			MinTime:      RoundFloat(minTime*1000, 2),
			P95Time:      RoundFloat(p95Time*1000, 2),
		})
		rank++
	}
	return list
}

func QueryNodeTrafficRanking(start, end time.Time, hostFilter HostFilter, limit int) ([]RankItem, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []RankItem{}, nil
	}
	start, end = adjustAccessLogQueryRange(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []RankItem{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT node_id AS item,
		count() AS request_count,
		sum(bytes) AS out_traffic,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_traffic
		FROM node_access_logs WHERE %s
		GROUP BY node_id
		ORDER BY out_traffic DESC
		LIMIT %d`, whereSQL, limit)

	if httpCfg != nil {
		return queryAccessRankingHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]RankItem, 0)
	for rows.Next() {
		var item string
		var reqCount, outBytes, originBytes uint64
		if err := rows.Scan(&item, &reqCount, &outBytes, &originBytes); err != nil {
			continue
		}
		list = append(list, RankItem{
			Item:         strings.TrimSpace(item),
			RequestCount: reqCount,
			OutBytes:     outBytes,
			OriginBytes:  originBytes,
		})
	}
	return list, nil
}

func rankingSpecForType(rankType string) (rankingSpec, bool) {
	switch rankType {
	case "domain":
		return rankingSpec{ItemExpr: AccessLogSiteExpr(), GroupBy: AccessLogSiteExpr(), KeywordCond: AccessLogSiteExpr() + " LIKE ?", KeywordArgs: 1}, true
	case "url":
		return rankingSpec{ItemExpr: "concat(" + AccessLogSiteExpr() + ", uri)", GroupBy: AccessLogSiteExpr() + ", uri", KeywordCond: "(" + AccessLogSiteExpr() + " LIKE ? OR uri LIKE ?)", KeywordArgs: 2}, true
	case "ip":
		return rankingSpec{ItemExpr: "remote_addr", GroupBy: "remote_addr", KeywordCond: "remote_addr LIKE ?", KeywordArgs: 1}, true
	case "referer":
		return rankingSpec{ItemExpr: AccessLogRefererExpr(), GroupBy: AccessLogRefererExpr(), KeywordCond: AccessLogRefererExpr() + " LIKE ?", KeywordArgs: 1, NormalizeNil: true}, true
	default:
		return rankingSpec{}, false
	}
}

func resolveRegionForIP(ip, regionType string, cache map[string]string) string {
	return resolveRegionForIPWithLookup(ip, regionType, cache, LookupIPRegion)
}

func resolveRegionForIPWithLookup(ip, regionType string, cache map[string]string, lookup func(string) (string, string)) string {
	if ip == "" {
		return ""
	}
	if cached, ok := cache[ip+"|"+regionType]; ok {
		return cached
	}
	if lookup == nil {
		lookup = LookupIPRegion
	}
	country, province := lookup(ip)
	region := ""
	if regionType == "country" {
		region = country
	} else {
		region = province
		if region == "" {
			region = country
		}
	}
	cache[ip+"|"+regionType] = region
	return region
}

func queryAccessRankingHTTP(cfg *httpCKConfig, query string, args ...interface{}) ([]RankItem, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]RankItem, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		item := strings.TrimSpace(toStringAny(raw["item"]))
		if item == "" {
			item = "-"
		}
		list = append(list, RankItem{
			Item:         item,
			RequestCount: toUint64Any(raw["request_count"]),
			OutBytes:     toUint64Any(raw["out_traffic"]),
			OriginBytes:  toUint64Any(raw["origin_traffic"]),
		})
	}
	return list, nil
}

func queryAccessIPAggregates(query string, args ...interface{}) ([]RankItem, error) {
	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryAccessIPAggregatesHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		log.Printf("[ranking] region ip aggregate query failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	list := make([]RankItem, 0)
	for rows.Next() {
		var ip string
		var reqCount, outBytes, originBytes uint64
		if err := rows.Scan(&ip, &reqCount, &outBytes, &originBytes); err != nil {
			continue
		}
		list = append(list, RankItem{
			Item:         strings.TrimSpace(ip),
			RequestCount: reqCount,
			OutBytes:     outBytes,
			OriginBytes:  originBytes,
		})
	}
	return list, nil
}

func queryAccessIPAggregatesHTTP(cfg *httpCKConfig, query string, args ...interface{}) ([]RankItem, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]RankItem, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		list = append(list, RankItem{
			Item:         strings.TrimSpace(toStringAny(raw["ip"])),
			RequestCount: toUint64Any(raw["request_count"]),
			OutBytes:     toUint64Any(raw["out_traffic"]),
			OriginBytes:  toUint64Any(raw["origin_traffic"]),
		})
	}
	return list, nil
}

func queryLatencyRankingHTTP(cfg *httpCKConfig, query string, args ...interface{}) []LatencyRankItem {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return []LatencyRankItem{}
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]LatencyRankItem, 0)
	rank := 1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		host := strings.TrimSpace(toStringAny(raw["item_host"]))
		if host == "" {
			host = strings.TrimSpace(toStringAny(raw["host"]))
		}
		uri := strings.TrimSpace(toStringAny(raw["uri"]))
		item := host
		if uri != "" {
			item = item + uri
		}
		list = append(list, LatencyRankItem{
			Rank:         rank,
			Item:         item,
			RequestCount: int(toUint64Any(raw["request_count"])),
			AvgTime:      RoundFloat(toFloat64Any(raw["avg_time"])*1000, 2),
			MaxTime:      RoundFloat(toFloat64Any(raw["max_time"])*1000, 2),
			MinTime:      RoundFloat(toFloat64Any(raw["min_time"])*1000, 2),
			P95Time:      RoundFloat(toFloat64Any(raw["p95_time"])*1000, 2),
		})
		rank++
	}
	return list
}

func toFloat64Any(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint64:
		return float64(t)
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return 0
}
