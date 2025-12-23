package config

import "os"

type AppConfig struct {
	Port      string
	DBDSN     string
	RedisAddr string
}

var App = &AppConfig{
	Port:      "8080",
	DBDSN:     "root:123456@tcp(127.0.0.1:3306)/cdnfy?charset=utf8mb4&parseTime=True&loc=Local",
	RedisAddr: "127.0.0.1:6379",
}

func Load() {
	// Simple Env Loader for Docker/K8s/HA compatibility
	if port := os.Getenv("APP_PORT"); port != "" {
		App.Port = port
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		App.DBDSN = dsn
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		App.RedisAddr = redisAddr
	}
}
