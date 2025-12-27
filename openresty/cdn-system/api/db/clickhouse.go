package db

import (
	"database/sql"
	"log"
	"time"

	"cdn-api/config"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

var CK *sql.DB

func InitClickHouse() {
	if !config.App.ClickHouseEnabled || config.App.ClickHouseDSN == "" {
		return
	}

	db, err := sql.Open("clickhouse", config.App.ClickHouseDSN)
	if err != nil {
		log.Printf("[CK] Failed to open ClickHouse: %v", err)
		return
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxIdleTime(time.Minute)
	db.SetConnMaxLifetime(time.Minute * 3)

	if err := db.Ping(); err != nil {
		log.Printf("[CK] Ping failed: %v", err)
		return
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
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS node_access_logs (
			ts DateTime,
			node_id String,
			node_ip String,
			remote_addr String,
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
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
