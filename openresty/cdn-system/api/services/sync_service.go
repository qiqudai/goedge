package services

import (
	"encoding/json"
	"strconv"
	"time"

	"cdn-api/db"
	"cdn-api/models"
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

// BumpConfigVersion increments the global config version.
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
		// Manual update using WHERE because config table has no primary key ID
		db.DB.Model(&models.SysConfig{}).Where("name = ? AND type = ?", configVersionKey, "system").Updates(map[string]interface{}{
			"value":     cfg.Value,
			"update_at": cfg.UpdatedAt,
		})
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

// NotifyConfigChanged is a no-op (sync is handled by agent pull over API).
func NotifyConfigChanged(change ConfigChange) {
	data, _ := json.Marshal(change)
	now := time.Now()
	task := models.Task{
		Type:     "config_sync",
		State:    "waiting",
		Enable:   true,
		Data:     string(data),
		CreateAt: now,
		StartAt:  &now, // 设置开始时间为创建时间，避免MySQL严格模式下的日期错误
		EndAt:    &now, // 设置结束时间为创建时间
		RetryAt:  &now, // Ensure it's picked up immediately
	}
	if err := db.DB.Create(&task).Error; err == nil {
		TriggerDispatchPending()
	}
}
