package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"log"
	"strings"
)

type lineKey struct {
	ID   string
	Name string
}

type lineNodes struct {
	NodeIDs []int64
	Enabled bool
}

// SyncPackageCnameForNodes syncs package CNAME DNS records when nodes change.
// action: add, delete, resync
func SyncPackageCnameForNodes(nodeIDs []int64, action string) error {
	if db.DB == nil {
		return nil
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return nil
	}
	nodeIDs = uniquePackageIDs(nodeIDs)
	if len(nodeIDs) == 0 {
		return nil
	}

	var subIDs []int64
	_ = db.DB.Model(&models.Node{}).Where("pid IN ?", nodeIDs).Pluck("id", &subIDs).Error
	nodeIDs = uniquePackageIDs(append(nodeIDs, subIDs...))

	log.Printf("[DNS] package cname sync start action=%s nodes=%v", action, nodeIDs)

	var lines []models.Line
	if err := db.DB.Select("node_group_id", "line_id", "line_name", "node_id", "node_ip_id", "enable").
		Where("node_id IN ? OR node_ip_id IN ?", nodeIDs, nodeIDs).
		Find(&lines).Error; err != nil {
		return err
	}
	if len(lines) == 0 {
		log.Printf("[DNS] package cname sync skip: no lines for nodes=%v", nodeIDs)
		return nil
	}

	groupLineNodes := make(map[int64]map[lineKey]*lineNodes)
	for _, line := range lines {
		if action != "delete" && !line.Enable {
			continue
		}
		key := lineKey{ID: strings.TrimSpace(line.LineID), Name: strings.TrimSpace(line.LineName)}
		if _, ok := groupLineNodes[line.NodeGroupID]; !ok {
			groupLineNodes[line.NodeGroupID] = make(map[lineKey]*lineNodes)
		}
		item := groupLineNodes[line.NodeGroupID][key]
		if item == nil {
			item = &lineNodes{Enabled: line.Enable}
			groupLineNodes[line.NodeGroupID][key] = item
		}
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID != 0 {
			item.NodeIDs = append(item.NodeIDs, nodeID)
		}
	}

	groupIDs := make([]int64, 0, len(groupLineNodes))
	for id := range groupLineNodes {
		groupIDs = append(groupIDs, id)
	}
	if len(groupIDs) == 0 {
		log.Printf("[DNS] package cname sync skip: no groups for nodes=%v", nodeIDs)
		return nil
	}
	siteInfos, domainMap, err := loadSiteCnameInfos(groupIDs)
	if err != nil {
		return err
	}
	if len(siteInfos) == 0 {
		log.Printf("[DNS] package cname sync skip: no sites for groups=%v", groupIDs)
		return nil
	}
	if len(domainMap) == 0 {
		log.Printf("[DNS] package cname sync skip: cname domain map empty for groups=%v", groupIDs)
		return nil
	}

	for _, info := range siteInfos {
		if info.Hostname == "" || info.DomainKey == "" {
			continue
		}
		domainInfo, ok := domainMap[info.DomainKey]
		if !ok {
			log.Printf("[DNS] package cname sync skip: site=%d domain=%s not found", info.SiteID, info.DomainKey)
			continue
		}
		targetGroupSet := map[int64]struct{}{}
		if info.PrimaryGroup != 0 {
			if _, ok := groupLineNodes[info.PrimaryGroup]; ok {
				targetGroupSet[info.PrimaryGroup] = struct{}{}
			}
		}
		if info.EnableBackup && info.BackupGroup != 0 {
			if _, ok := groupLineNodes[info.BackupGroup]; ok {
				targetGroupSet[info.BackupGroup] = struct{}{}
			}
		}
		if len(targetGroupSet) == 0 && info.PrimaryGroup == 0 && info.BackupGroup == 0 {
			for gid := range groupLineNodes {
				targetGroupSet[gid] = struct{}{}
			}
		}
		if len(targetGroupSet) == 0 {
			continue
		}
		for gid := range targetGroupSet {
			linesForGroup := groupLineNodes[gid]
			if len(linesForGroup) == 0 {
				continue
			}
			for key, item := range linesForGroup {
				nodeList := uniquePackageIDs(item.NodeIDs)
				if len(nodeList) == 0 {
					continue
				}
				if err := dns.SyncPackageLineRecords(domainInfo, info.Hostname, gid, key.ID, key.Name, action, nodeList); err != nil {
					log.Printf("[DNS] package cname sync failed site=%d group=%d host=%s.%s line=%s err=%v", info.SiteID, gid, info.Hostname, info.DomainKey, key.ID, err)
				}
			}
		}
	}

	log.Printf("[DNS] package cname sync finished action=%s nodes=%v", action, nodeIDs)
	return nil
}

// SyncPackageCnameForLineChange syncs package DNS records when a line assignment changes.
// action: add, delete
func SyncPackageCnameForLineChange(groupID int64, lineID, lineName string, nodeIDs []int64, action string) error {
	if db.DB == nil {
		return nil
	}
	if groupID == 0 {
		return nil
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return nil
	}
	switch action {
	case "enable":
		action = "add"
	case "disable":
		action = "delete"
	}
	if action == "resync" {
		nodeIDs = loadLineNodeIDs(groupID, lineID)
	} else {
		nodeIDs = uniquePackageIDs(nodeIDs)
	}

	log.Printf("[DNS] package line sync start group=%d line=%s action=%s nodes=%v", groupID, lineID, action, nodeIDs)
	siteInfos, domainMap, err := loadSiteCnameInfos([]int64{groupID})
	if err != nil {
		return err
	}
	if len(siteInfos) == 0 {
		log.Printf("[DNS] package line sync skip: no sites for group=%d", groupID)
		return nil
	}
	if len(domainMap) == 0 {
		log.Printf("[DNS] package line sync skip: domain map empty for group=%d", groupID)
		return nil
	}

	for _, info := range siteInfos {
		if info.Hostname == "" || info.DomainKey == "" {
			continue
		}
		if info.PrimaryGroup != groupID && !(info.EnableBackup && info.BackupGroup == groupID) {
			if info.PrimaryGroup == 0 && info.BackupGroup == 0 {
				// Fallback: no group binding, allow.
			} else {
				continue
			}
		}
		domainInfo, ok := domainMap[info.DomainKey]
		if !ok {
			log.Printf("[DNS] package line sync skip: site=%d domain=%s not found", info.SiteID, info.DomainKey)
			continue
		}
		if err := dns.SyncPackageLineRecords(domainInfo, info.Hostname, groupID, lineID, lineName, action, nodeIDs); err != nil {
			log.Printf("[DNS] package line sync failed site=%d group=%d host=%s.%s line=%s err=%v", info.SiteID, groupID, info.Hostname, info.DomainKey, lineID, err)
		}
	}

	log.Printf("[DNS] package line sync finished group=%d line=%s action=%s", groupID, lineID, action)
	return nil
}

func loadLineNodeIDs(groupID int64, lineID string) []int64 {
	if groupID == 0 || db.DB == nil {
		return []int64{}
	}
	var lines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id = ? AND line_id = ? AND enable = ?", groupID, lineID, true).
		Find(&lines).Error; err != nil {
		return []int64{}
	}
	nodeIDs := make([]int64, 0, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID != 0 {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	return uniquePackageIDs(nodeIDs)
}

func resyncPackageHostname(domain models.CnameDomain, hostname string, groupID int64) error {
	if groupID == 0 {
		return nil
	}
	var lines []models.Line
	if err := db.DB.Where("node_group_id = ?", groupID).Find(&lines).Error; err != nil {
		return err
	}
	if len(lines) == 0 {
		return nil
	}

	lineMap := make(map[lineKey][]int64)
	allNodeIDs := make([]int64, 0)
	for _, line := range lines {
		if !line.Enable {
			continue
		}
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID == 0 {
			continue
		}
		key := lineKey{ID: strings.TrimSpace(line.LineID), Name: strings.TrimSpace(line.LineName)}
		lineMap[key] = append(lineMap[key], nodeID)
		allNodeIDs = append(allNodeIDs, nodeID)
	}
	if len(lineMap) == 0 {
		return nil
	}

	enabled := map[int64]bool{}
	if len(allNodeIDs) > 0 {
		var nodes []models.Node
		_ = db.DB.Select("id", "enable").Where("id IN ?", uniquePackageIDs(allNodeIDs)).Find(&nodes).Error
		for _, node := range nodes {
			if node.Enable {
				enabled[node.ID] = true
			}
		}
	}

	for key, nodeIDs := range lineMap {
		filtered := make([]int64, 0, len(nodeIDs))
		for _, id := range uniquePackageIDs(nodeIDs) {
			if enabled[id] {
				filtered = append(filtered, id)
			}
		}
		if err := dns.SyncPackageLineRecords(domain, hostname, groupID, key.ID, key.Name, "resync", filtered); err != nil {
			return err
		}
	}
	return nil
}

func resolvePackageHostname(pack models.UserPackage, domain string) string {
	host := strings.TrimSpace(pack.CnameHostname)
	if host == "" {
		host = strings.TrimSpace(pack.RecordID)
	}
	host = normalizePackageDomain(host)
	root := normalizePackageDomain(domain)
	if host == "" {
		return ""
	}
	if root != "" {
		suffix := "." + root
		if host == root {
			host = "@"
		} else if strings.HasSuffix(host, suffix) {
			host = strings.TrimSuffix(host, suffix)
		}
	}
	return strings.TrimSuffix(host, ".")
}

func normalizePackageDomain(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.TrimRight(host, ".")
}

func uniquePackageIDs(items []int64) []int64 {
	if len(items) == 0 {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	result := make([]int64, 0, len(items))
	for _, id := range items {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func loadPackageGroupsFromSites(groupIDs []int64) map[int64]map[int64]struct{} {
	result := map[int64]map[int64]struct{}{}
	if db.DB == nil || len(groupIDs) == 0 {
		return result
	}
	groupSet := map[int64]struct{}{}
	for _, id := range groupIDs {
		if id != 0 {
			groupSet[id] = struct{}{}
		}
	}
	if len(groupSet) == 0 {
		return result
	}
	var rows []struct {
		UserPackageID     int64 `gorm:"column:user_package"`
		NodeGroupID       int64 `gorm:"column:node_group_id"`
		BackupNodeGroupID int64 `gorm:"column:backup_node_group"`
	}
	if err := db.DB.Model(&models.Site{}).
		Select("user_package, node_group_id, backup_node_group").
		Where("node_group_id IN ? OR backup_node_group IN ?", groupIDs, groupIDs).
		Find(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		if row.UserPackageID == 0 {
			continue
		}
		if _, ok := result[row.UserPackageID]; !ok {
			result[row.UserPackageID] = map[int64]struct{}{}
		}
		if row.NodeGroupID != 0 {
			if _, ok := groupSet[row.NodeGroupID]; ok {
				result[row.UserPackageID][row.NodeGroupID] = struct{}{}
			}
		}
		if row.BackupNodeGroupID != 0 {
			if _, ok := groupSet[row.BackupNodeGroupID]; ok {
				result[row.UserPackageID][row.BackupNodeGroupID] = struct{}{}
			}
		}
	}
	return result
}

type planGroup struct {
	NodeGroupID     int64
	BackupNodeGroup int64
}

func loadPlanIDsByGroups(groupIDs []int64) []int64 {
	if db.DB == nil || len(groupIDs) == 0 {
		return []int64{}
	}
	var ids []int64
	if err := db.DB.Model(&models.Package{}).
		Where("node_group_id IN ? OR backup_node_group IN ?", groupIDs, groupIDs).
		Pluck("id", &ids).Error; err != nil {
		return []int64{}
	}
	return uniquePackageIDs(ids)
}

func loadPlanGroupMap(packageIDs []int64) map[int64]planGroup {
	result := map[int64]planGroup{}
	if db.DB == nil || len(packageIDs) == 0 {
		return result
	}
	var rows []struct {
		ID              int64 `gorm:"column:id"`
		NodeGroupID     int64 `gorm:"column:node_group_id"`
		BackupNodeGroup int64 `gorm:"column:backup_node_group"`
	}
	if err := db.DB.Model(&models.Package{}).
		Select("id, node_group_id, backup_node_group").
		Where("id IN ?", packageIDs).
		Find(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		result[row.ID] = planGroup{
			NodeGroupID:     row.NodeGroupID,
			BackupNodeGroup: row.BackupNodeGroup,
		}
	}
	return result
}

type siteCnameInfo struct {
	SiteID       int64
	Hostname     string
	DomainKey    string
	PrimaryGroup int64
	BackupGroup  int64
	EnableBackup bool
}

func loadSiteCnameInfos(groupIDs []int64) ([]siteCnameInfo, map[string]models.CnameDomain, error) {
	if db.DB == nil || len(groupIDs) == 0 {
		return nil, nil, nil
	}

	planIDsForGroups := loadPlanIDsByGroups(groupIDs)
	packMap := make(map[int64]models.UserPackage)
	var packs []models.UserPackage
	if err := db.DB.Where("node_group_id IN ? OR backup_node_group IN ?", groupIDs, groupIDs).Find(&packs).Error; err != nil {
		return nil, nil, err
	}
	for _, pack := range packs {
		packMap[pack.ID] = pack
	}
	if len(planIDsForGroups) > 0 {
		var planPacks []models.UserPackage
		if err := db.DB.Where("package IN ?", planIDsForGroups).Find(&planPacks).Error; err != nil {
			return nil, nil, err
		}
		for _, pack := range planPacks {
			if _, ok := packMap[pack.ID]; !ok {
				packMap[pack.ID] = pack
			}
		}
	}

	packIDs := make([]int64, 0, len(packMap))
	for id := range packMap {
		packIDs = append(packIDs, id)
	}

	query := db.DB.Model(&models.Site{}).Where("node_group_id IN ? OR backup_node_group IN ?", groupIDs, groupIDs)
	if len(packIDs) > 0 {
		query = query.Or("user_package IN ?", packIDs)
	}
	var sites []models.Site
	if err := query.Find(&sites).Error; err != nil {
		return nil, nil, err
	}
	if len(sites) == 0 {
		return nil, nil, nil
	}

	missingPackIDs := make([]int64, 0)
	for _, site := range sites {
		if site.UserPackageID == 0 {
			continue
		}
		if _, ok := packMap[site.UserPackageID]; ok {
			continue
		}
		missingPackIDs = append(missingPackIDs, site.UserPackageID)
	}
	if len(missingPackIDs) > 0 {
		var extraPacks []models.UserPackage
		if err := db.DB.Where("id IN ?", uniquePackageIDs(missingPackIDs)).Find(&extraPacks).Error; err != nil {
			return nil, nil, err
		}
		for _, pack := range extraPacks {
			packMap[pack.ID] = pack
		}
	}

	planIDSet := map[int64]struct{}{}
	for _, pack := range packMap {
		if pack.PackageID != 0 {
			planIDSet[int64(pack.PackageID)] = struct{}{}
		}
	}
	planIDs := make([]int64, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}
	planGroupMap := loadPlanGroupMap(planIDs)

	domainSet := make(map[string]struct{})
	infos := make([]siteCnameInfo, 0, len(sites))
	for _, site := range sites {
		pkg := packMap[site.UserPackageID]
		domainKey, host := resolveSiteCnameTarget(site, pkg)
		if domainKey == "" || host == "" {
			continue
		}
		primary, backup, enableBackup := resolveSiteGroups(site, pkg, planGroupMap[int64(pkg.PackageID)])
		infos = append(infos, siteCnameInfo{
			SiteID:       site.ID,
			Hostname:     host,
			DomainKey:    domainKey,
			PrimaryGroup: primary,
			BackupGroup:  backup,
			EnableBackup: enableBackup,
		})
		domainSet[domainKey] = struct{}{}
	}
	if len(domainSet) == 0 {
		return infos, map[string]models.CnameDomain{}, nil
	}

	domainList := make([]string, 0, len(domainSet))
	for d := range domainSet {
		domainList = append(domainList, d)
	}
	var domainRows []models.CnameDomain
	if err := db.DB.Where("domain IN ?", domainList).Find(&domainRows).Error; err != nil {
		return infos, nil, err
	}
	domainMap := make(map[string]models.CnameDomain, len(domainRows))
	for _, row := range domainRows {
		key := normalizePackageDomain(row.Domain)
		if key != "" {
			domainMap[key] = row
		}
	}
	return infos, domainMap, nil
}

func resolveSiteGroups(site models.Site, pkg models.UserPackage, plan planGroup) (int64, int64, bool) {
	primary := site.NodeGroupID
	if primary == 0 {
		primary = pkg.NodeGroupID
	}
	if primary == 0 {
		primary = plan.NodeGroupID
	}
	enableBackup := site.EnableBackupGroup
	if !enableBackup {
		enableBackup = pkg.EnableBackup
	}
	backup := int64(0)
	if enableBackup {
		backup = site.BackupNodeGroupID
		if backup == 0 {
			backup = pkg.BackupNodeGroup
		}
		if backup == 0 {
			backup = plan.BackupNodeGroup
		}
	}
	return primary, backup, enableBackup
}

func resolveSiteCnameTarget(site models.Site, pkg models.UserPackage) (string, string) {
	siteMode := strings.TrimSpace(site.CnameMode)
	pkgMode := strings.TrimSpace(pkg.CnameMode)

	if siteMode == "package" || (siteMode == "" && pkgMode == "package") {
		domainKey := normalizePackageDomain(pkg.CnameDomain)
		host := normalizePackageDomain(pkg.CnameHostname)
		if host == "" {
			host = normalizePackageDomain(pkg.RecordID)
		}
		if host == "" {
			return domainKey, ""
		}
		if domainKey == "" {
			root, name := splitRootDomain(host)
			if root == "" || name == "" {
				return "", ""
			}
			return normalizePackageDomain(root), name
		}
		suffix := "." + domainKey
		if host == domainKey {
			host = "@"
		} else if strings.HasSuffix(host, suffix) {
			host = strings.TrimSuffix(host, suffix)
		}
		host = strings.TrimSuffix(host, ".")
		return domainKey, host
	}

	domainKey := normalizePackageDomain(site.CnameDomain)
	if domainKey == "" {
		domainKey = normalizePackageDomain(pkg.CnameDomain)
	}
	full := normalizePackageDomain(site.CnameHostname)
	if full == "" && len(site.Domains) > 0 && domainKey != "" {
		full = normalizePackageDomain(site.Domains[0] + "." + domainKey)
	}
	if full == "" {
		return domainKey, ""
	}
	if domainKey == "" {
		root, name := splitRootDomain(full)
		if root == "" || name == "" {
			return "", ""
		}
		return normalizePackageDomain(root), name
	}
	host := full
	suffix := "." + domainKey
	if full == domainKey {
		host = "@"
	} else if strings.HasSuffix(full, suffix) {
		host = strings.TrimSuffix(full, suffix)
	}
	host = strings.TrimSuffix(host, ".")
	return domainKey, host
}
