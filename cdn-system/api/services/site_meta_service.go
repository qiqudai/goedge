package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	siteMetaType         = "site_meta"
	siteMetaScopeName    = "site"
	siteMetaSiteTypeName = "site_type"
)

func LoadSiteTypeMeta(siteID int64) string {
	if siteID <= 0 {
		return ""
	}
	var item models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", siteMetaType, siteMetaScopeName, siteID, siteMetaSiteTypeName).
		First(&item).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(item.Value)
}

func LoadSiteTypeMetaMap(siteIDs []int64) map[int64]string {
	if len(siteIDs) == 0 {
		return nil
	}
	var items []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id IN ? AND name = ?", siteMetaType, siteMetaScopeName, siteIDs, siteMetaSiteTypeName).
		Find(&items).Error; err != nil {
		return nil
	}
	result := make(map[int64]string, len(items))
	for _, item := range items {
		if item.ScopeID == 0 {
			continue
		}
		value := strings.TrimSpace(item.Value)
		if value == "" {
			continue
		}
		result[item.ScopeID] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func UpsertSiteTypeMeta(siteID int64, siteType string) error {
	if siteID <= 0 {
		return nil
	}
	value := strings.TrimSpace(siteType)
	if value == "" {
		return nil
	}

	var existing models.ConfigItem
	err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", siteMetaType, siteMetaScopeName, siteID, siteMetaSiteTypeName).
		First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			item := models.ConfigItem{
				Type:      siteMetaType,
				ScopeName: siteMetaScopeName,
				ScopeID:   siteID,
				Name:      siteMetaSiteTypeName,
				Value:     value,
				Enable:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return db.DB.Create(&item).Error
		}
		return err
	}

	updates := map[string]interface{}{
		"value":     value,
		"enable":    true,
		"update_at": time.Now(),
	}
	return db.DB.Model(&models.ConfigItem{}).
		Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", siteMetaType, siteMetaScopeName, siteID, siteMetaSiteTypeName).
		Updates(updates).Error
}
