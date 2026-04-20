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

// BlockedCurrentRow is a distinct IP+host block summary.
type BlockedCurrentRow struct {
	Host      string
	IP        string
	Country   string
	Province  string
	Status    int
	BlockTime time.Time
}

// BlockedHistoryRow is a single block event.
type BlockedHistoryRow struct {
	Host      string
	IP        string
	Country   string
	Province  string
	Status    int
	BlockTime time.Time
}

// BlockedStatRow aggregates block counts per host.
type BlockedStatRow struct {
	Host  string
	Count uint64
}

func QueryBlockedCurrent(start, end time.Time, hostFilter HostFilter, ipFilter string, limit, offset int) ([]BlockedCurrentRow, uint64, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []BlockedCurrentRow{}, 0, nil
	}
	if !db.ClickHouseEnabled() {
		return []BlockedCurrentRow{}, 0, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?", blockedStatusCondition()}
	args := []interface{}{start, end}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	ipFilter = strings.TrimSpace(ipFilter)
	if ipFilter != "" {
		conditions = append(conditions, "remote_addr = ?")
		args = append(args, ipFilter)
	}
	whereSQL := strings.Join(conditions, " AND ")
	countSQL := fmt.Sprintf("SELECT uniqExact((host, remote_addr)) FROM node_access_logs WHERE %s", whereSQL)
	querySQL := fmt.Sprintf(`SELECT host, remote_addr,
		argMax(%s, ts) AS client_country,
		argMax(%s, ts) AS client_province,
		max(ts) AS block_time,
		any(status) AS status
		FROM node_access_logs WHERE %s
		GROUP BY host, remote_addr
		ORDER BY block_time DESC
		LIMIT ? OFFSET ?`, AccessLogClientCountryExpr(), AccessLogClientProvinceExpr(), whereSQL)

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryBlockedCurrentHTTP(httpCfg, countSQL, querySQL, args, limit, offset)
	}
	var total uint64
	if err := db.CK.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return []BlockedCurrentRow{}, 0, err
	}
	args = append(args, limit, offset)
	rows, err := db.CK.Query(querySQL, args...)
	if err != nil {
		return []BlockedCurrentRow{}, total, err
	}
	defer rows.Close()
	list := make([]BlockedCurrentRow, 0)
	for rows.Next() {
		var host, ip, country, province string
		var ts time.Time
		var status int
		if err := rows.Scan(&host, &ip, &country, &province, &ts, &status); err != nil {
			continue
		}
		list = append(list, BlockedCurrentRow{
			Host:      strings.TrimSpace(host),
			IP:        strings.TrimSpace(ip),
			Country:   strings.TrimSpace(country),
			Province:  strings.TrimSpace(province),
			Status:    status,
			BlockTime: ts,
		})
	}
	return list, total, nil
}

func QueryBlockedHistory(start, end time.Time, hostFilter HostFilter, ipFilter string, limit, offset int) ([]BlockedHistoryRow, uint64, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []BlockedHistoryRow{}, 0, nil
	}
	if !db.ClickHouseEnabled() {
		return []BlockedHistoryRow{}, 0, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?", blockedStatusCondition()}
	args := []interface{}{start, end}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	ipFilter = strings.TrimSpace(ipFilter)
	if ipFilter != "" {
		conditions = append(conditions, "remote_addr = ?")
		args = append(args, ipFilter)
	}
	whereSQL := strings.Join(conditions, " AND ")
	countSQL := fmt.Sprintf("SELECT count() FROM node_access_logs WHERE %s", whereSQL)
	querySQL := fmt.Sprintf(`SELECT ts, host, remote_addr,
		%s AS client_country,
		%s AS client_province,
		status
		FROM node_access_logs WHERE %s
		ORDER BY ts DESC
		LIMIT ? OFFSET ?`, AccessLogClientCountryExpr(), AccessLogClientProvinceExpr(), whereSQL)

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryBlockedHistoryHTTP(httpCfg, countSQL, querySQL, args, limit, offset)
	}
	var total uint64
	if err := db.CK.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return []BlockedHistoryRow{}, 0, err
	}
	args = append(args, limit, offset)
	rows, err := db.CK.Query(querySQL, args...)
	if err != nil {
		return []BlockedHistoryRow{}, total, err
	}
	defer rows.Close()
	list := make([]BlockedHistoryRow, 0)
	for rows.Next() {
		var ts time.Time
		var host, ip, country, province string
		var status int
		if err := rows.Scan(&ts, &host, &ip, &country, &province, &status); err != nil {
			continue
		}
		list = append(list, BlockedHistoryRow{
			Host:      strings.TrimSpace(host),
			IP:        strings.TrimSpace(ip),
			Country:   strings.TrimSpace(country),
			Province:  strings.TrimSpace(province),
			Status:    status,
			BlockTime: ts,
		})
	}
	return list, total, nil
}

func QueryBlockedStats(start, end time.Time, hostFilter HostFilter, limit, offset int) ([]BlockedStatRow, uint64, error) {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return []BlockedStatRow{}, 0, nil
	}
	if !db.ClickHouseEnabled() {
		return []BlockedStatRow{}, 0, nil
	}
	conditions := []string{"ts >= ? AND ts <= ?", blockedStatusCondition()}
	args := []interface{}{start, end}
	if clause, clauseArgs := hostFilter.SQLCondition(); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	whereSQL := strings.Join(conditions, " AND ")
	countSQL := fmt.Sprintf("SELECT uniqExact(host) FROM node_access_logs WHERE %s", whereSQL)
	querySQL := fmt.Sprintf(`SELECT host, count() AS cnt
		FROM node_access_logs WHERE %s
		GROUP BY host
		ORDER BY cnt DESC
		LIMIT ? OFFSET ?`, whereSQL)

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		return queryBlockedStatsHTTP(httpCfg, countSQL, querySQL, args, limit, offset)
	}
	var total uint64
	if err := db.CK.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return []BlockedStatRow{}, 0, err
	}
	args = append(args, limit, offset)
	rows, err := db.CK.Query(querySQL, args...)
	if err != nil {
		return []BlockedStatRow{}, total, err
	}
	defer rows.Close()
	list := make([]BlockedStatRow, 0)
	for rows.Next() {
		var host string
		var count uint64
		if err := rows.Scan(&host, &count); err != nil {
			continue
		}
		list = append(list, BlockedStatRow{Host: strings.TrimSpace(host), Count: count})
	}
	return list, total, nil
}

func queryBlockedCurrentHTTP(cfg *httpCKConfig, countSQL, querySQL string, args []interface{}, limit, offset int) ([]BlockedCurrentRow, uint64, error) {
	countSQL = interpolateQuery(countSQL, args...)
	countSQL = countSQL + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, countSQL)
	if err != nil {
		return []BlockedCurrentRow{}, 0, err
	}
	var total uint64
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]uint64
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		for _, value := range row {
			total = value
			break
		}
		break
	}
	querySQL = interpolateQuery(querySQL, append(args, limit, offset)...)
	querySQL = querySQL + "\nFORMAT JSONEachRow"
	body, err = queryClickHouseHTTP(cfg, querySQL)
	if err != nil {
		return []BlockedCurrentRow{}, total, err
	}
	list := make([]BlockedCurrentRow, 0)
	scanner = bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		blockRaw := strings.TrimSpace(toStringAny(raw["block_time"]))
		if blockRaw == "" {
			continue
		}
		blockTime, err := parseCKTimeString(blockRaw)
		if err != nil {
			continue
		}
		list = append(list, BlockedCurrentRow{
			Host:      strings.TrimSpace(toStringAny(raw["host"])),
			IP:        strings.TrimSpace(toStringAny(raw["remote_addr"])),
			Country:   strings.TrimSpace(toStringAny(raw["client_country"])),
			Province:  strings.TrimSpace(toStringAny(raw["client_province"])),
			Status:    int(toInt64(raw["status"])),
			BlockTime: blockTime,
		})
	}
	return list, total, nil
}

func queryBlockedHistoryHTTP(cfg *httpCKConfig, countSQL, querySQL string, args []interface{}, limit, offset int) ([]BlockedHistoryRow, uint64, error) {
	countSQL = interpolateQuery(countSQL, args...)
	countSQL = countSQL + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, countSQL)
	if err != nil {
		return []BlockedHistoryRow{}, 0, err
	}
	var total uint64
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]uint64
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		for _, value := range row {
			total = value
			break
		}
		break
	}
	querySQL = interpolateQuery(querySQL, append(args, limit, offset)...)
	querySQL = querySQL + "\nFORMAT JSONEachRow"
	body, err = queryClickHouseHTTP(cfg, querySQL)
	if err != nil {
		return []BlockedHistoryRow{}, total, err
	}
	list := make([]BlockedHistoryRow, 0)
	scanner = bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		blockRaw := strings.TrimSpace(toStringAny(raw["ts"]))
		if blockRaw == "" {
			continue
		}
		blockTime, err := parseCKTimeString(blockRaw)
		if err != nil {
			continue
		}
		list = append(list, BlockedHistoryRow{
			Host:      strings.TrimSpace(toStringAny(raw["host"])),
			IP:        strings.TrimSpace(toStringAny(raw["remote_addr"])),
			Country:   strings.TrimSpace(toStringAny(raw["client_country"])),
			Province:  strings.TrimSpace(toStringAny(raw["client_province"])),
			Status:    int(toInt64(raw["status"])),
			BlockTime: blockTime,
		})
	}
	return list, total, nil
}

func queryBlockedStatsHTTP(cfg *httpCKConfig, countSQL, querySQL string, args []interface{}, limit, offset int) ([]BlockedStatRow, uint64, error) {
	countSQL = interpolateQuery(countSQL, args...)
	countSQL = countSQL + "\nFORMAT JSONEachRow"
	body, err := queryClickHouseHTTP(cfg, countSQL)
	if err != nil {
		return []BlockedStatRow{}, 0, err
	}
	var total uint64
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]uint64
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		for _, value := range row {
			total = value
			break
		}
		break
	}
	querySQL = interpolateQuery(querySQL, append(args, limit, offset)...)
	querySQL = querySQL + "\nFORMAT JSONEachRow"
	body, err = queryClickHouseHTTP(cfg, querySQL)
	if err != nil {
		return []BlockedStatRow{}, total, err
	}
	list := make([]BlockedStatRow, 0)
	scanner = bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		list = append(list, BlockedStatRow{
			Host:  strings.TrimSpace(toStringAny(raw["host"])),
			Count: toUint64Any(raw["cnt"]),
		})
	}
	return list, total, nil
}
