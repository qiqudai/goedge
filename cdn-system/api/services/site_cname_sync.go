package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"strings"
)

func ResyncSiteCnameForSite(site models.Site) {
	if !shouldSyncSiteCname(site) {
		return
	}
	groupID := resolveGroupIDFromSite(site)
	if groupID != 0 {
		ResyncGroupLineCnames(groupID)
	}

	backupGroup := site.BackupNodeGroupID
	enableBackup := site.EnableBackupGroup
	if !enableBackup && site.UserPackageID != 0 {
		var pkg models.UserPackage
		if err := db.DB.Select("backup_node_group", "enable_backup_group").
			Where("id = ?", site.UserPackageID).
			First(&pkg).Error; err == nil {
			if backupGroup == 0 {
				backupGroup = pkg.BackupNodeGroup
			}
			enableBackup = pkg.EnableBackup
		}
	}
	if enableBackup && backupGroup != 0 {
		ResyncGroupLineCnames(backupGroup)
	}
}

func shouldSyncSiteCname(site models.Site) bool {
	if site.UserPackageID == 0 {
		return false
	}
	mode := strings.TrimSpace(site.CnameMode)
	if mode != "" {
		return mode != "package"
	}
	var pkg models.UserPackage
	if err := db.DB.Select("cname_mode").Where("id = ?", site.UserPackageID).First(&pkg).Error; err != nil {
		return true
	}
	return strings.TrimSpace(pkg.CnameMode) != "package"
}

func ResyncGroupLineCnames(groupID int64) {
	if groupID == 0 || db.DB == nil {
		return
	}
	var lines []models.Line
	if err := db.DB.Select("line_id", "line_name").
		Where("node_group_id = ?", groupID).
		Find(&lines).Error; err != nil {
		return
	}
	lineMap := map[string]string{}
	for _, line := range lines {
		lineID := strings.TrimSpace(line.LineID)
		if lineID == "" {
			lineID = "default"
		}
		lineName := strings.TrimSpace(line.LineName)
		if lineName == "" {
			lineName = lineID
		}
		if _, ok := lineMap[lineID]; !ok {
			lineMap[lineID] = lineName
		}
	}
	for lineID, lineName := range lineMap {
		_ = SyncPackageCnameForLineChange(groupID, lineID, lineName, nil, "resync")
	}
}

func resolveGroupIDFromSite(site models.Site) int64 {
	if site.NodeGroupID != 0 {
		return site.NodeGroupID
	}
	if site.UserPackageID == 0 {
		return 0
	}
	var pkg models.UserPackage
	if err := db.DB.Select("node_group_id").Where("id = ?", site.UserPackageID).First(&pkg).Error; err != nil {
		return 0
	}
	return pkg.NodeGroupID
}
