package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	installProgressType      = "install_progress"
	installProgressScopeName = "node"
)

type InstallProgress struct {
	NodeID       int64     `json:"node_id"`
	Stage        string    `json:"stage"`
	Percent      int       `json:"percent"`
	CurrentBytes int64     `json:"current_bytes"`
	TotalBytes   int64     `json:"total_bytes"`
	Error        string    `json:"error,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func UpdateInstallProgress(nodeID int64, stage string, percent int, currentBytes, totalBytes int64, errMsg string) error {
	if nodeID <= 0 || db.DB == nil {
		return nil
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	payload := InstallProgress{
		NodeID:       nodeID,
		Stage:        stage,
		Percent:      percent,
		CurrentBytes: currentBytes,
		TotalBytes:   totalBytes,
		Error:        errMsg,
		UpdatedAt:    time.Now(),
	}
	raw, _ := json.Marshal(payload)
	name := installProgressKey(nodeID)
	now := time.Now()

	var existing models.SysConfig
	err := db.DB.Where("name = ? AND type = ? AND scope_id = ? AND scope_name = ?", name, installProgressType, nodeID, installProgressScopeName).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record := models.SysConfig{
				Name:      name,
				Value:     string(raw),
				Type:      installProgressType,
				ScopeID:   int(nodeID),
				ScopeName: installProgressScopeName,
				CreatedAt: now,
				UpdatedAt: now,
				Enable:    true,
			}
			return db.DB.Create(&record).Error
		}
		return err
	}
	updates := map[string]interface{}{
		"value":      string(raw),
		"scope_id":   int(nodeID),
		"scope_name": installProgressScopeName,
		"enable":     true,
		"update_at":  now,
	}
	return db.DB.Model(&models.SysConfig{}).
		Where("name = ? AND type = ? AND scope_id = ? AND scope_name = ?", name, installProgressType, nodeID, installProgressScopeName).
		Updates(updates).Error
}

func FetchInstallProgress(nodeIDs []int64) (map[int64]InstallProgress, error) {
	out := make(map[int64]InstallProgress)
	if db.DB == nil || len(nodeIDs) == 0 {
		return out, nil
	}
	keys := make([]string, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if id > 0 {
			keys = append(keys, installProgressKey(id))
		}
	}
	if len(keys) == 0 {
		return out, nil
	}
	var rows []models.SysConfig
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id IN ?", installProgressType, installProgressScopeName, nodeIDs).Find(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		var payload InstallProgress
		if err := json.Unmarshal([]byte(row.Value), &payload); err != nil {
			continue
		}
		if payload.NodeID > 0 {
			out[payload.NodeID] = payload
		}
	}
	return out, nil
}

func installProgressKey(nodeID int64) string {
	return fmt.Sprintf("node-install-progress-%d", nodeID)
}
