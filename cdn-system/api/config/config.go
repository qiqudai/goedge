package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var App = &AppConfig{
	Port:               "8080",
	DBDSN:              "root:123456@tcp(127.0.0.1:3306)/cdn_system?charset=utf8mb4&parseTime=True&loc=Local",
	Debug:              false,
	AgentToken:         "",
	ClickHouseEnabled:  false,
	ClickHouseDSN:      "",
	AcmeEmail:          "",
	AcmeWebroot:        "./acme",
	AcmeAccountDir:     "./acme/accounts",
	SecretKey:          "0123456789abcdef0123456789abcdef",
	JWTSecret:          "",
	CORSAllowedOrigins: "",
}

type AppConfig struct {
	Port               string `yaml:"port"`
	DBDSN              string `yaml:"db_dsn"`
	Debug              bool   `yaml:"debug"`
	AgentToken         string `yaml:"agent_token"`
	ClickHouseEnabled  bool   `yaml:"clickhouse_enabled"`
	ClickHouseDSN      string `yaml:"clickhouse_dsn"`
	AcmeEmail          string `yaml:"acme_email"`
	AcmeWebroot        string `yaml:"acme_webroot"`
	AcmeAccountDir     string `yaml:"acme_account_dir"`
	SecretKey          string `yaml:"secret_key"`
	JWTSecret          string `yaml:"jwt_secret"`
	CORSAllowedOrigins string `yaml:"cors_allowed_origins"`
}

var (
	configFile         = flag.String("config", "config.yaml", "Path to configuration file")
	port               = flag.String("port", "", "Server port")
	dbDSN              = flag.String("db", "", "Database DSN (e.g. root:pass@tcp(127.0.0.1:3306)/dbname)")
	debugFlag          = flag.Bool("debug", false, "Enable debug logging")
	resolvedConfigPath = "config.yaml"
)

func Load() {
	// 1. Initial Flags Parsing
	if !flag.Parsed() {
		flag.Parse()
	}

	// 2. Resolve config path and load (if exists)
	path := resolveConfigPath(*configFile)
	resolvedConfigPath = path
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, App); err != nil {
			log.Printf("[Warn] Failed to parse %s: %v", path, err)
		} else {
			log.Printf("[Info] Loaded config from %s", path)
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

func ConfigPath() string {
	if resolvedConfigPath != "" {
		return resolvedConfigPath
	}
	if configFile != nil && *configFile != "" {
		return *configFile
	}
	return "config.yaml"
}

func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}

func resolveConfigPath(raw string) string {
	raw = filepath.Clean(raw)
	if raw == "" {
		raw = "config.yaml"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	if _, err := os.Stat(raw); err == nil {
		return raw
	}
	exe, err := os.Executable()
	if err != nil {
		return raw
	}
	exePath := filepath.Join(filepath.Dir(exe), raw)
	if _, err := os.Stat(exePath); err == nil {
		return exePath
	}
	return raw
}
