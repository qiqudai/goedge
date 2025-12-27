package config

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var App = &AppConfig{
	Port:      "8080",
	DBDSN:     "root:123456@tcp(127.0.0.1:3306)/cdn_system?charset=utf8mb4&parseTime=True&loc=Local",
	Debug:     false,
	AgentToken: "",
	ClickHouseEnabled: false,
	ClickHouseDSN:     "",
}

type AppConfig struct {
	Port       string `yaml:"port"`
	DBDSN      string `yaml:"db_dsn"`
	Debug      bool   `yaml:"debug"`
	AgentToken string `yaml:"agent_token"`
	ClickHouseEnabled bool   `yaml:"clickhouse_enabled"`
	ClickHouseDSN     string `yaml:"clickhouse_dsn"`
}

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
	port       = flag.String("port", "", "Server port")
	dbDSN      = flag.String("db", "", "Database DSN (e.g. root:pass@tcp(127.0.0.1:3306)/dbname)")
	debugFlag  = flag.Bool("debug", false, "Enable debug logging")
)

func Load() {
	// 1. Initial Flags Parsing
	if !flag.Parsed() {
		flag.Parse()
	}

	// 2. Load from Config File (if exists)
	if data, err := os.ReadFile(*configFile); err == nil {
		if err := yaml.Unmarshal(data, App); err != nil {
			log.Printf("[Warn] Failed to parse %s: %v", *configFile, err)
		} else {
			log.Printf("[Info] Loaded config from %s", *configFile)
		}
	}

	// 3. Override with Flags (if provided)
	if *port != "" {
		App.Port = *port
	}
	if *dbDSN != "" {
		App.DBDSN = *dbDSN
	}
	if *debugFlag {
		App.Debug = true
	}
}
