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

type HotURLIPItem struct {
	Rank         int             `json:"rank,omitempty"`
	Site         string          `json:"site"`
	URI          string          `json:"uri"`
	Item         string          `json:"item"`
	RequestCount uint64          `json:"request_count"`
	OutBytes     uint64          `json:"out_bytes"`
	OriginBytes  uint64          `json:"origin_bytes"`
	IPs          []HotURLIPCount `json:"ips,omitempty"`
}

type HotURLIPCount struct {
	Rank              int    `json:"rank"`
	IP                string `json:"ip"`
	RequestCount      uint64 `json:"request_count"`
	TotalRequestCount uint64 `json:"total_request_count"`
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
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []RankItem{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	expr, ok := regionRankingExprForType(regionType)
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
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		conditions = append(conditions, expr+" LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS item,
		count() AS request_count,
		sum(bytes) AS out_traffic,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_traffic
		FROM node_access_logs WHERE %s
		GROUP BY %s
		ORDER BY request_count DESC
		LIMIT %d`, expr, whereSQL, expr, limit)

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
		if item == "" {
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

func QueryHotURLRanking(start, end time.Time, hostFilter HostFilter, keyword string, limit int) ([]HotURLIPItem, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []HotURLIPItem{}, nil
	}
	start, end = adjustAccessLogQueryRange(start, end)
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return []HotURLIPItem{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	siteExpr := AccessLogSiteExpr()
	uriExpr := AccessLogURIPathExpr()
	conditions := []string{"ts >= ? AND ts <= ?"}
	args := []interface{}{start, end}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		conditions = append(conditions, "("+siteExpr+" LIKE ? OR uri LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS item_site,
		%s AS uri,
		concat(%s, %s) AS item,
		count() AS request_count,
		sum(bytes) AS out_traffic,
		sumIf(bytes, upstream_cache_status != 'HIT') AS origin_traffic
		FROM node_access_logs WHERE %s
		GROUP BY item_site, uri
		ORDER BY request_count DESC
		LIMIT %d`, siteExpr, uriExpr, siteExpr, uriExpr, whereSQL, limit)

	if httpCfg != nil {
		return queryHotURLRankingHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]HotURLIPItem, 0)
	for rows.Next() {
		var item HotURLIPItem
		if err := rows.Scan(&item.Site, &item.URI, &item.Item, &item.RequestCount, &item.OutBytes, &item.OriginBytes); err != nil {
			continue
		}
		item.Site = strings.TrimSpace(item.Site)
		item.URI = strings.TrimSpace(item.URI)
		item.Item = strings.TrimSpace(item.Item)
		list = append(list, item)
	}
	return list, nil
}

func QueryHotURLTopIPs(start, end time.Time, hostFilter HostFilter, urls []HotURLIPItem, perURLLimit int) (map[string][]HotURLIPCount, error) {
	if len(urls) == 0 || start.IsZero() || end.IsZero() || end.Before(start) {
		return map[string][]HotURLIPCount{}, nil
	}
	if perURLLimit <= 0 {
		perURLLimit = 100
	}
	if perURLLimit > 100 {
		perURLLimit = 100
	}
	rangeStart, rangeEnd := adjustAccessLogQueryRange(start, end)
	windowEnd := end
	windowStart := end.Add(-60 * time.Second)
	windowStart, windowEnd = adjustAccessLogQueryRange(windowStart, windowEnd)

	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
		return map[string][]HotURLIPCount{}, nil
	}
	siteExpr := AccessLogSiteExpr()
	uriExpr := AccessLogURIPathExpr()
	conditions := []string{"ts >= ? AND ts <= ?", "remote_addr != ''"}
	args := []interface{}{windowStart, windowEnd, rangeStart, rangeEnd}
	conditions = append(conditions, AccessLogRealSiteTrafficCondition())
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	pairCond, pairArgs := hotURLPairCondition(siteExpr, uriExpr, urls)
	if pairCond == "" {
		return map[string][]HotURLIPCount{}, nil
	}
	conditions = append(conditions, pairCond)
	args = append(args, pairArgs...)
	whereSQL := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`SELECT %s AS item_site,
		%s AS uri,
		remote_addr AS ip,
		count() AS total_request_count,
		countIf(ts >= ? AND ts <= ?) AS request_count
		FROM node_access_logs WHERE %s
		GROUP BY item_site, uri, ip
		ORDER BY item_site, uri, total_request_count DESC
		LIMIT %d BY item_site, uri`, siteExpr, uriExpr, whereSQL, perURLLimit)

	if httpCfg != nil {
		return queryHotURLTopIPsHTTP(httpCfg, query, args...)
	}
	rows, err := db.CK.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grouped := map[string][]HotURLIPCount{}
	for rows.Next() {
		var site, uri, ip string
		var totalCount, recentCount uint64
		if err := rows.Scan(&site, &uri, &ip, &totalCount, &recentCount); err != nil {
			continue
		}
		key := hotURLKey(strings.TrimSpace(site), strings.TrimSpace(uri))
		grouped[key] = append(grouped[key], HotURLIPCount{
			Rank:              len(grouped[key]) + 1,
			IP:                strings.TrimSpace(ip),
			RequestCount:      recentCount,
			TotalRequestCount: totalCount,
		})
	}
	return grouped, nil
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
		uriExpr := AccessLogURIPathExpr()
		return rankingSpec{ItemExpr: "concat(" + AccessLogSiteExpr() + ", " + uriExpr + ")", GroupBy: AccessLogSiteExpr() + ", " + uriExpr, KeywordCond: "(" + AccessLogSiteExpr() + " LIKE ? OR " + uriExpr + " LIKE ? OR uri LIKE ?)", KeywordArgs: 3}, true
	case "ip":
		return rankingSpec{ItemExpr: "remote_addr", GroupBy: "remote_addr", KeywordCond: "remote_addr LIKE ?", KeywordArgs: 1}, true
	case "referer":
		return rankingSpec{ItemExpr: AccessLogRefererExpr(), GroupBy: AccessLogRefererExpr(), KeywordCond: AccessLogRefererExpr() + " LIKE ?", KeywordArgs: 1, NormalizeNil: true}, true
	default:
		return rankingSpec{}, false
	}
}

func regionRankingExprForType(regionType string) (string, bool) {
	switch regionType {
	case "country":
		return AccessLogClientCountryExpr(), true
	case "province":
		return AccessLogClientProvinceExpr(), true
	default:
		return "", false
	}
}

func hotURLKey(site, uri string) string {
	return site + "\x00" + uri
}

func hotURLPairCondition(siteExpr, uriExpr string, urls []HotURLIPItem) (string, []interface{}) {
	parts := make([]string, 0, len(urls))
	args := make([]interface{}, 0, len(urls)*2)
	seen := map[string]struct{}{}
	for _, item := range urls {
		site := strings.TrimSpace(item.Site)
		uri := strings.TrimSpace(item.URI)
		if site == "" {
			continue
		}
		key := hotURLKey(site, uri)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, "("+siteExpr+" = ? AND "+uriExpr+" = ?)")
		args = append(args, site, uri)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func queryHotURLRankingHTTP(cfg *httpCKConfig, query string, args ...interface{}) ([]HotURLIPItem, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	list := make([]HotURLIPItem, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		list = append(list, HotURLIPItem{
			Site:         strings.TrimSpace(toStringAny(raw["item_site"])),
			URI:          strings.TrimSpace(toStringAny(raw["uri"])),
			Item:         strings.TrimSpace(toStringAny(raw["item"])),
			RequestCount: toUint64Any(raw["request_count"]),
			OutBytes:     toUint64Any(raw["out_traffic"]),
			OriginBytes:  toUint64Any(raw["origin_traffic"]),
		})
	}
	return list, nil
}

func queryHotURLTopIPsHTTP(cfg *httpCKConfig, query string, args ...interface{}) (map[string][]HotURLIPCount, error) {
	query = interpolateQuery(query, args...)
	query = query + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, query)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	grouped := map[string][]HotURLIPCount{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		key := hotURLKey(strings.TrimSpace(toStringAny(raw["item_site"])), strings.TrimSpace(toStringAny(raw["uri"])))
		grouped[key] = append(grouped[key], HotURLIPCount{
			Rank:              len(grouped[key]) + 1,
			IP:                strings.TrimSpace(toStringAny(raw["ip"])),
			RequestCount:      toUint64Any(raw["request_count"]),
			TotalRequestCount: toUint64Any(raw["total_request_count"]),
		})
	}
	return grouped, nil
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
