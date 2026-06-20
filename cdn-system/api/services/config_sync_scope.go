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
		return resolveSiteConfigGroupIDs(ids)
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

func resolveSiteConfigGroupIDs(ids []int64) []int64 {
	ids = uniqueInt64List(ids)
	if len(ids) == 0 {
		return nil
	}
	var sites []models.Site
	if err := db.DB.Select("id", "user_package", "node_group_id", "backup_node_group", "enable_backup_group").
		Where("id IN ?", ids).
		Find(&sites).Error; err != nil {
		return nil
	}
	if len(sites) == 0 {
		return nil
	}
	packMap, err := loadUserPackageMap(sites)
	if err != nil {
		return nil
	}
	planIDSet := map[int64]struct{}{}
	for _, pkg := range packMap {
		if pkg.PackageID != 0 {
			planIDSet[int64(pkg.PackageID)] = struct{}{}
		}
	}
	planIDs := make([]int64, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}
	planMap := loadPlanGroupMap(planIDs)

	groupIDs := make([]int64, 0, len(sites)*2)
	for _, site := range sites {
		pkg := packMap[site.UserPackageID]
		primary, backup, enableBackup := resolveSiteGroups(site, pkg, planMap[int64(pkg.PackageID)])
		if primary != 0 {
			groupIDs = append(groupIDs, primary)
		}
		if enableBackup && backup != 0 {
			groupIDs = append(groupIDs, backup)
		}
	}
	return uniqueInt64List(groupIDs)
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
