package services

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"

	"github.com/redis/go-redis/v9"
)

const (
	configVersionKey  = "edge_config_version"
	configChangeTopic = "config:changed"
)

type ConfigChange struct {
	Version   int64     `json:"version"`
	Resource  string    `json:"resource"`
	IDs       []int64   `json:"ids,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// BumpConfigVersion increments the global config version and optionally publishes via Redis.
func BumpConfigVersion(resource string, ids []int64) int64 {
	var cfg models.SysConfig
	if err := db.DB.Where("`key` = ?", configVersionKey).First(&cfg).Error; err != nil {
		cfg = models.SysConfig{
			Key:       configVersionKey,
			Version:   1,
			UpdatedAt: time.Now(),
		}
	} else {
		cfg.Version += 1
		cfg.UpdatedAt = time.Now()
	}
	_ = db.DB.Save(&cfg).Error

	NotifyConfigChanged(ConfigChange{
		Version:   cfg.Version,
		Resource:  resource,
		IDs:       ids,
		Timestamp: cfg.UpdatedAt,
	})
	return cfg.Version
}

// GetConfigVersion returns the latest global config version.
func GetConfigVersion() int64 {
	var cfg models.SysConfig
	if err := db.DB.Where("`key` = ?", configVersionKey).First(&cfg).Error; err != nil {
		return 0
	}
	return cfg.Version
}

// NotifyConfigChanged publishes a change event to Redis if configured.
func NotifyConfigChanged(change ConfigChange) {
	addr := strings.TrimSpace(config.App.RedisAddr)
	if addr == "" {
		return
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = rdb.Close() }()

	payload, _ := json.Marshal(change)
	if err := rdb.Publish(context.Background(), configChangeTopic, payload).Err(); err != nil {
		log.Printf("[sync] redis publish failed: %v", err)
	}
}
