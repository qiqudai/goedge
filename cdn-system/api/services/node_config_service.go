package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

func UpsertNodeConfigItem(nodeID int64, name string, value string) error {
	if nodeID == 0 || name == "" {
		return nil
	}
	var item models.ConfigItem
	query := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", "node_config", "node", nodeID, name)
	if err := query.First(&item).Error; err == nil {
		updates := map[string]interface{}{
			"value":     value,
			"enable":    true,
			"update_at": time.Now(),
		}
		return query.Model(&models.ConfigItem{}).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	item = models.ConfigItem{
		Name:      name,
		Value:     value,
		Type:      "node_config",
		ScopeName: "node",
		ScopeID:   nodeID,
		Enable:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return db.DB.Create(&item).Error
}

func GetNodeConfigMap(name string) (map[int64]string, error) {
	result := map[int64]string{}
	if name == "" {
		return result, nil
	}
	var items []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND name = ?", "node_config", "node", name).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ScopeID] = item.Value
	}
	return result, nil
}

func GetNodeConfigValue(nodeID int64, name string) (string, error) {
	if nodeID == 0 || name == "" {
		return "", nil
	}
	var item models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", "node_config", "node", nodeID, name).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return item.Value, nil
}
