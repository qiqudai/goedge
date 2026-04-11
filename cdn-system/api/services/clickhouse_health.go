package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"cdn-api/config"
	"cdn-api/db"
)

var requiredClickHouseTables = []string{
	"node_access_logs",
	"node_stream_logs",
	"node_metrics",
	"node_events",
}

type ClickHouseHealthReport struct {
	OK           bool
	Errors       []string
	Database     string
	MissingTable []string
}

func CheckClickHouseHealth() ClickHouseHealthReport {
	report := ClickHouseHealthReport{
		OK:     false,
		Errors: []string{},
	}

	if !config.App.ClickHouseEnabled {
		report.Errors = append(report.Errors, "ClickHouse 未启用 (clickhouse_enabled=false)")
		return report
	}

	dsn := strings.TrimSpace(config.App.ClickHouseDSN)
	if dsn == "" {
		report.Errors = append(report.Errors, "ClickHouse DSN 为空")
		return report
	}

	if httpCfg := buildHTTPConfig(); httpCfg != nil {
		report.Database = strings.TrimSpace(httpCfg.database)
		if report.Database == "" {
			report.Database = "default"
		}
		return checkClickHouseViaHTTP(httpCfg, report)
	}

	report.Database = resolveCKDatabaseFromDSN(dsn)
	return checkClickHouseViaNative(report)
}

func checkClickHouseViaNative(report ClickHouseHealthReport) ClickHouseHealthReport {
	if db.CK == nil {
		report.Errors = append(report.Errors, "ClickHouse native 连接未建立")
		return report
	}
	if err := db.CK.Ping(); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("ClickHouse 连接失败: %v", err))
		return report
	}

	if !nativeDatabaseExists(report.Database) {
		report.Errors = append(report.Errors, fmt.Sprintf("库不存在: %s", report.Database))
		return report
	}

	existing := nativeTableNames(report.Database)
	missing := findMissingTables(existing, requiredClickHouseTables)
	if len(missing) > 0 {
		report.MissingTable = missing
		report.Errors = append(report.Errors, fmt.Sprintf("缺少表: %s", strings.Join(missing, ", ")))
		return report
	}

	report.OK = true
	return report
}

func nativeDatabaseExists(database string) bool {
	if db.CK == nil {
		return false
	}
	var count uint64
	if err := db.CK.QueryRow("SELECT count() FROM system.databases WHERE name = ?", database).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func nativeTableNames(database string) map[string]struct{} {
	out := map[string]struct{}{}
	if db.CK == nil {
		return out
	}
	rows, err := db.CK.Query("SELECT name FROM system.tables WHERE database = ?", database)
	if err != nil {
		return out
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out
}

func checkClickHouseViaHTTP(cfg *httpCKConfig, report ClickHouseHealthReport) ClickHouseHealthReport {
	if _, err := queryClickHouseHTTP(cfg, "SELECT 1 FORMAT JSONEachRow"); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("ClickHouse 连接失败: %v", err))
		return report
	}

	dbName := strings.TrimSpace(report.Database)
	if dbName == "" {
		dbName = "default"
	}

	dbSQL := fmt.Sprintf("SELECT count() AS c FROM system.databases WHERE name = %s FORMAT JSONEachRow", quoteClickHouseString(dbName))
	body, err := queryClickHouseHTTP(cfg, dbSQL)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("检查库失败: %v", err))
		return report
	}
	if parseCountFromJSONEachRow(body, "c") == 0 {
		report.Errors = append(report.Errors, fmt.Sprintf("库不存在: %s", dbName))
		return report
	}

	tablesSQL := fmt.Sprintf(
		"SELECT name FROM system.tables WHERE database = %s FORMAT JSONEachRow",
		quoteClickHouseString(dbName),
	)
	body, err = queryClickHouseHTTP(cfg, tablesSQL)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("检查表失败: %v", err))
		return report
	}
	existing := parseNamesFromJSONEachRow(body, "name")
	missing := findMissingTables(existing, requiredClickHouseTables)
	if len(missing) > 0 {
		report.MissingTable = missing
		report.Errors = append(report.Errors, fmt.Sprintf("缺少表: %s", strings.Join(missing, ", ")))
		return report
	}

	report.OK = true
	return report
}

func parseCountFromJSONEachRow(body []byte, key string) uint64 {
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
		return toUint64Any(raw[key])
	}
	return 0
}

func parseNamesFromJSONEachRow(body []byte, key string) map[string]struct{} {
	names := map[string]struct{}{}
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
		name := strings.TrimSpace(toStringAny(raw[key]))
		if name == "" {
			continue
		}
		names[name] = struct{}{}
	}
	return names
}

func findMissingTables(existing map[string]struct{}, required []string) []string {
	missing := make([]string, 0)
	for _, table := range required {
		if _, ok := existing[table]; !ok {
			missing = append(missing, table)
		}
	}
	return missing
}

func resolveCKDatabaseFromDSN(dsn string) string {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "default"
	}
	name := strings.Trim(strings.TrimSpace(u.Path), "/")
	if name == "" {
		name = strings.TrimSpace(u.Query().Get("database"))
	}
	if name == "" {
		return "default"
	}
	return name
}
