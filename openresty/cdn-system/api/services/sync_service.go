package services

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
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
	var version int64 = 1

	// Find the config version record
	err := db.DB.Where("name = ? AND type = ?", configVersionKey, "system").First(&cfg).Error
	if err != nil {
		// Not found, create new
		cfg = models.SysConfig{
			Name:      configVersionKey,
			Type:      "system", // Use 'system' type
			ScopeID:   0,
			ScopeName: "global",
			Value:     "1",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.DB.Create(&cfg)
	} else {
		// Parse existing version
		v, err := strconv.ParseInt(cfg.Value, 10, 64)
		if err == nil {
			version = v + 1
		}
		cfg.Value = strconv.FormatInt(version, 10)
		cfg.UpdatedAt = time.Now()
		db.DB.Save(&cfg)
	}

	NotifyConfigChanged(ConfigChange{
		Version:   version,
		Resource:  resource,
		IDs:       ids,
		Timestamp: cfg.UpdatedAt,
	})
	return version
}

// GetConfigVersion returns the latest global config version.
func GetConfigVersion() int64 {
	var cfg models.SysConfig
	if err := db.DB.Where("name = ? AND type = ?", configVersionKey, "system").First(&cfg).Error; err != nil {
		return 0
	}
	v, _ := strconv.ParseInt(cfg.Value, 10, 64)
	return v
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
