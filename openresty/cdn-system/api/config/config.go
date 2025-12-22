package config

import "os"

type AppConfig struct {
	Port      string
	DBDSN     string
	RedisAddr string
}

var App = &AppConfig{
	Port: "8080",
	DBDSN: "root:123456@tcp(127.0.0.1:3306)/cdn_system?charset=utf8mb4&parseTime=True&loc=Local",
}

func Load() {
	// Simple Env Loader for Docker/K8s/HA compatibility
	if port := os.Getenv("APP_PORT"); port != "" {
		App.Port = port
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		App.DBDSN = dsn
	}
}
