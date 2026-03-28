package services

import (
	"cdn-api/db"
	"cdn-api/models"
)

func resolveScopedConfigSyncTargets(resource string, ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	groupIDs := resolveGroupIDsForResource(resource, ids)
	if len(groupIDs) == 0 {
		return nil
	}
	scope := ResolveSyncSiteConfigScope()
	targetGroupIDs := groupIDs
	if scope == "region" {
		targetGroupIDs = resolveGroupIDsByRegion(groupIDs)
	}
	if len(targetGroupIDs) == 0 {
		return nil
	}
	var nodeIDs []int64
	_ = db.DB.Model(&models.Line{}).
		Select("distinct node_id").
		Where("node_group_id IN ?", targetGroupIDs).
		Where("node_id <> 0").
		Pluck("node_id", &nodeIDs).Error
	return uniqueInt64List(nodeIDs)
}

func resolveGroupIDsForResource(resource string, ids []int64) []int64 {
	switch resource {
	case "site":
		var groupIDs []int64
		_ = db.DB.Model(&models.Site{}).
			Select("distinct node_group_id").
			Where("id IN ?", ids).
			Where("node_group_id <> 0").
			Pluck("node_group_id", &groupIDs).Error
		return uniqueInt64List(groupIDs)
	case "forward":
		var groupIDs []int64
		_ = db.DB.Model(&models.Forward{}).
			Select("distinct node_group_id").
			Where("id IN ?", ids).
			Where("node_group_id <> 0").
			Pluck("node_group_id", &groupIDs).Error
		return uniqueInt64List(groupIDs)
	default:
		return nil
	}
}

func resolveGroupIDsByRegion(groupIDs []int64) []int64 {
	if len(groupIDs) == 0 {
		return nil
	}
	type groupRegion struct {
		ID       int64  `gorm:"column:id"`
		RegionID *int64 `gorm:"column:region_id"`
	}
	var rows []groupRegion
	_ = db.DB.Model(&models.NodeGroup{}).
		Select("id", "region_id").
		Where("id IN ?", groupIDs).
		Scan(&rows).Error
	if len(rows) == 0 {
		return nil
	}
	regionIDs := make([]int64, 0)
	targetGroupIDs := make([]int64, 0)
	for _, row := range rows {
		if row.RegionID == nil || *row.RegionID == 0 {
			targetGroupIDs = append(targetGroupIDs, row.ID)
			continue
		}
		regionIDs = append(regionIDs, *row.RegionID)
	}
	regionIDs = uniqueInt64List(regionIDs)
	if len(regionIDs) == 0 {
		return uniqueInt64List(targetGroupIDs)
	}
	var regionGroupIDs []int64
	_ = db.DB.Model(&models.NodeGroup{}).
		Select("distinct id").
		Where("region_id IN ?", regionIDs).
		Pluck("id", &regionGroupIDs).Error
	targetGroupIDs = append(targetGroupIDs, regionGroupIDs...)
	return uniqueInt64List(targetGroupIDs)
}
