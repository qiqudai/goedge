package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type RegionMeta struct {
	L2CheckPort int `json:"l2_check_port"`
	SortOrder   int `json:"sort_order"`
}

func LoadRegionMetaMap() map[string]RegionMeta {
	var item models.ConfigItem
	err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		First(&item).Error
	if err != nil {
		return map[string]RegionMeta{}
	}
	if item.Value == "" {
		return map[string]RegionMeta{}
	}
	metaMap := map[string]RegionMeta{}
	if jsonErr := json.Unmarshal([]byte(item.Value), &metaMap); jsonErr != nil {
		return map[string]RegionMeta{}
	}
	return metaMap
}

func SaveRegionMetaMap(metaMap map[string]RegionMeta) error {
	b, err := json.Marshal(metaMap)
	if err != nil {
		return err
	}
	var item models.ConfigItem
	err = db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newItem := models.ConfigItem{
				Name:      "region_meta",
				Value:     string(b),
				Type:      "system",
				ScopeID:   0,
				ScopeName: "global",
				Enable:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return db.DB.Create(&newItem).Error
		}
		return err
	}
	return db.DB.Model(&models.ConfigItem{}).
		Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "region_meta", "system", "global", 0).
		Updates(map[string]interface{}{
			"value":     string(b),
			"update_at": time.Now(),
		}).Error
}

func ResolveRegionL2CheckPort(metaMap map[string]RegionMeta, regionID *int64) int {
	if regionID == nil || *regionID == 0 {
		return 80
	}
	key := strconv.FormatInt(*regionID, 10)
	if meta, ok := metaMap[key]; ok && meta.L2CheckPort > 0 {
		return meta.L2CheckPort
	}
	return 80
}
