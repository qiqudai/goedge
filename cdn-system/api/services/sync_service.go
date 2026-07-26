package services

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
)

const (
	configVersionKey  = "edge_config_version"
	configChangeTopic = "config:changed"
	// ConfigResourceCNAME is deliberately delivered to every enabled primary
	// node. A CNAME root/prefix change changes the generated virtual-host
	// configuration, so limiting it to a current site-line scope can leave
	// stale configurations on other nodes.
	ConfigResourceCNAME = "cname"
	// ConfigResourceUserPackage is delivered to every enabled primary node.
	// A sold-package change affects package eligibility and limits independently
	// of the site's currently assigned line group.
	ConfigResourceUserPackage = "user_package"
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

// BumpCnameConfigVersion queues CNAME configuration for every enabled primary
// node. It is intentionally separate from normal site-scoped configuration
// changes so unrelated site updates retain their existing scoped delivery.
func BumpCnameConfigVersion(siteIDs []int64) int64 {
	return BumpConfigVersion(ConfigResourceCNAME, siteIDs)
}

// BumpUserPackageConfigVersion queues sold-package configuration for every
// enabled primary node. It deliberately does not use site-scoped delivery:
// nodes need the latest package limits and expiry before a site is reassigned.
func BumpUserPackageConfigVersion(userPackageIDs []int64) int64 {
	return BumpConfigVersion(ConfigResourceUserPackage, userPackageIDs)
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
	if strings.EqualFold(change.Resource, "site") || strings.EqualFold(change.Resource, "forward") {
		if handled := createScopedConfigSyncTasks(change); handled {
			return
		}
	}
	createConfigSyncTask(change, nil)
}

func createScopedConfigSyncTasks(change ConfigChange) bool {
	if len(change.IDs) == 0 {
		return false
	}
	limit := ResolveMaxSiteStreamSyncOneTime()
	if limit <= 0 {
		limit = 1000
	}
	created := false
	for i := 0; i < len(change.IDs); i += limit {
		end := i + limit
		if end > len(change.IDs) {
			end = len(change.IDs)
		}
		chunk := change.IDs[i:end]
		chg := change
		chg.IDs = chunk
		nodes := resolveScopedConfigSyncTargets(strings.ToLower(strings.TrimSpace(change.Resource)), chunk)
		if len(nodes) == 0 {
			continue
		}
		createConfigSyncTask(chg, nodes)
		created = true
	}
	return created
}

func createConfigSyncTask(change ConfigChange, nodeIDs []int64) {
	data, _ := json.Marshal(change)
	now := time.Now()
	task := models.Task{
		Type:     "config_sync",
		State:    "waiting",
		Enable:   true,
		Data:     string(data),
		CreateAt: now,
		StartAt:  &now,
		EndAt:    &now,
		RetryAt:  &now,
	}
	if len(nodeIDs) == 0 {
		if strings.EqualFold(change.Resource, ConfigResourceCNAME) ||
			strings.EqualFold(change.Resource, ConfigResourceUserPackage) {
			nodeIDs = enabledPrimaryNodeIDs()
		} else {
			nodeIDs = ConnectedNodeIDs()
		}
	}
	if len(nodeIDs) > 0 {
		targets := NewTaskTargets(nodeIDs)
		task.TargetsJSON = targets.Marshal()
	}
	if err := db.DB.Create(&task).Error; err == nil {
		TriggerDispatchPending()
	}
}

func enabledPrimaryNodeIDs() []int64 {
	if db.DB == nil {
		return nil
	}
	var nodeIDs []int64
	if err := db.DB.Model(&models.Node{}).
		Where("pid = 0 AND enable = ?", true).
		Pluck("id", &nodeIDs).Error; err != nil {
		return nil
	}
	return uniqueInt64List(nodeIDs)
}

func TriggerNodeConfigSync(nodeID int64) {
	if nodeID == 0 {
		return
	}
	change := ConfigChange{
		Version:   GetConfigVersion(),
		Resource:  "node_config",
		IDs:       []int64{nodeID},
		Timestamp: time.Now(),
	}
	createConfigSyncTask(change, []int64{nodeID})
}
