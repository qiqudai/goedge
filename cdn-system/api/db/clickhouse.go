package db

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cdn-api/config"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

var CK *sql.DB

func InitClickHouse() {
	if !config.App.ClickHouseEnabled || config.App.ClickHouseDSN == "" {
		return
	}
	dsn := strings.TrimSpace(config.App.ClickHouseDSN)
	if isHTTPClickHouseDSN(dsn) {
		if err := ensureClickHouseSchemaHTTP(dsn); err != nil {
			log.Printf("[CK] HTTP schema init failed: %v", err)
			return
		}
		log.Println("[CK] ClickHouse schema ready (HTTP mode)")
		return
	}

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		log.Printf("[CK] Failed to open ClickHouse: %v", err)
		return
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxIdleTime(time.Minute)
	db.SetConnMaxLifetime(time.Minute * 3)

	if err := db.Ping(); err != nil {
		if isUnknownDatabaseErr(err) {
			targetDB := clickHouseDBNameFromDSN(dsn)
			if e := ensureClickHouseDatabaseNative(dsn, targetDB); e != nil {
				log.Printf("[CK] Create database failed: %v", e)
				return
			}
			if err = db.Ping(); err != nil {
				log.Printf("[CK] Ping failed after creating database: %v", err)
				return
			}
		} else {
			log.Printf("[CK] Ping failed: %v", err)
			return
		}
	}

	if err := ensureClickHouseTables(db); err != nil {
		log.Printf("[CK] Ensure tables failed: %v", err)
		return
	}

	CK = db
	log.Println("[CK] ClickHouse ready")
}

func ClickHouseEnabled() bool {
	return config.App.ClickHouseEnabled && CK != nil
}

func ensureClickHouseTables(db *sql.DB) error {
	stmts := clickHouseTableStmts()
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func clickHouseTableStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS node_access_logs (
			ts DateTime,
			node_id String,
			node_ip String,
			remote_addr String,
			client_country String,
			client_province String,
			client_city String,
			client_isp String,
			site_name String,
			host String,
			method String,
			uri String,
			status UInt16,
			bytes UInt64,
			request_time Float64,
			upstream_addr String,
			upstream_response_time Float64,
			upstream_cache_status String,
			http_referer String,
			http_user_agent String,
			scheme String,
			ssl_protocol String,
			ssl_cipher String,
			raw String
		) ENGINE = MergeTree
		PARTITION BY toDate(ts)
		ORDER BY (host, node_id, ts)`,
		`ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS client_country String AFTER remote_addr`,
		`ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS client_province String AFTER client_country`,
		`ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS client_city String AFTER client_province`,
		`ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS client_isp String AFTER client_city`,
		`ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS site_name String AFTER remote_addr`,
		`CREATE TABLE IF NOT EXISTS node_stream_logs (
			ts DateTime,
			node_id String,
			node_ip String,
			remote_addr String,
			server_port UInt16,
			protocol String,
			status UInt16,
			bytes_sent UInt64,
			bytes_received UInt64,
			session_time Float64,
			upstream_addr String,
			upstream_bytes_sent UInt64,
			upstream_bytes_received UInt64,
			upstream_connect_time Float64,
			upstream_session_time Float64,
			raw String
		) ENGINE = MergeTree
		PARTITION BY toDate(ts)
		ORDER BY (server_port, node_id, ts)`,
		`CREATE TABLE IF NOT EXISTS node_metrics (
			ts DateTime,
			node_id String,
			node_ip String,
			metric String,
			labels String,
			value Float64
		) ENGINE = MergeTree
		PARTITION BY toDate(ts)
		ORDER BY (metric, node_id, ts)`,
		`CREATE TABLE IF NOT EXISTS node_events (
			ts DateTime,
			node_id String,
			node_ip String,
			event_type String,
			payload String
		) ENGINE = MergeTree
		PARTITION BY toDate(ts)
		ORDER BY (event_type, node_id, ts)`,
	}
}

func isHTTPClickHouseDSN(dsn string) bool {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func clickHouseDBNameFromDSN(dsn string) string {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "default"
	}
	name := strings.TrimSpace(strings.Trim(u.Path, "/"))
	if name == "" {
		name = strings.TrimSpace(u.Query().Get("database"))
	}
	if name == "" {
		name = "default"
	}
	return sanitizeCKIdentifier(name)
}

func sanitizeCKIdentifier(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return "default"
	}
	return name
}

func isUnknownDatabaseErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown database")
}

func clickHouseAdminDSN(dsn string) string {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return dsn
	}
	u.Path = "/default"
	q := u.Query()
	q.Set("database", "default")
	u.RawQuery = q.Encode()
	return u.String()
}

func ensureClickHouseDatabaseNative(dsn, dbName string) error {
	adminDSN := clickHouseAdminDSN(dsn)
	adminDB, err := sql.Open("clickhouse", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.Ping(); err != nil {
		return err
	}
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	return err
}

type ckHTTPConn struct {
	baseURL  string
	database string
	user     string
	pass     string
}

func parseHTTPCKDSN(dsn string) (*ckHTTPConn, error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return nil, err
	}
	cfg := &ckHTTPConn{
		baseURL:  u.Scheme + "://" + u.Host,
		database: clickHouseDBNameFromDSN(dsn),
	}
	if u.User != nil {
		cfg.user = u.User.Username()
		cfg.pass, _ = u.User.Password()
	}
	return cfg, nil
}

func executeHTTPCKQuery(cfg *ckHTTPConn, query string, dbOverride string) error {
	dbName := cfg.database
	if strings.TrimSpace(dbOverride) != "" {
		dbName = dbOverride
	}
	params := url.Values{}
	params.Set("query", query)
	if dbName != "" {
		params.Set("database", dbName)
	}
	req, err := http.NewRequest("POST", cfg.baseURL+"/?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	if cfg.user != "" {
		req.SetBasicAuth(cfg.user, cfg.pass)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func ensureClickHouseSchemaHTTP(dsn string) error {
	cfg, err := parseHTTPCKDSN(dsn)
	if err != nil {
		return err
	}
	dbName := sanitizeCKIdentifier(cfg.database)
	if err := executeHTTPCKQuery(cfg, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName), "default"); err != nil {
		return err
	}
	for _, stmt := range clickHouseTableStmts() {
		if err := executeHTTPCKQuery(cfg, stmt, dbName); err != nil {
			return err
		}
	}
	return nil
}
