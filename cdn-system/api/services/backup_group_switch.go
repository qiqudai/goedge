package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	backupSwitchInterval    = 10 * time.Second
	backupRecoverSuccesses  = 10
	backupDownThreshold     = 1
	backupGroupOnlineWindow = 30 * time.Second
)

type siteBackupState struct {
	ActiveGroup int64
	DownCount   int
	UpCount     int
}

var backupGroupState = struct {
	mu    sync.Mutex
	sites map[int64]*siteBackupState
}{
	sites: map[int64]*siteBackupState{},
}

func StartBackupGroupSwitchWorker() {
	go func() {
		runBackupGroupSwitch()
		ticker := time.NewTicker(backupSwitchInterval)
		defer ticker.Stop()
		for range ticker.C {
			runBackupGroupSwitch()
		}
	}()
}

func runBackupGroupSwitch() {
	if db.DB == nil {
		return
	}
	siteInfos, domainMap, groupIDs, err := loadBackupSwitchSites()
	if err != nil || len(siteInfos) == 0 {
		return
	}
	groupOnline := loadGroupOnlineStatus(groupIDs)
	lineMap := loadGroupLineMap(groupIDs)

	siteSet := map[int64]struct{}{}
	for _, info := range siteInfos {
		siteSet[info.SiteID] = struct{}{}
	}
	backupGroupState.mu.Lock()
	for id := range backupGroupState.sites {
		if _, ok := siteSet[id]; !ok {
			delete(backupGroupState.sites, id)
		}
	}
	backupGroupState.mu.Unlock()

	for _, info := range siteInfos {
		domain, ok := domainMap[info.DomainKey]
		if !ok {
			continue
		}
		primaryOnline := groupOnline[info.PrimaryGroup]
		backupOnline := groupOnline[info.BackupGroup]

		var targetGroup int64
		backupGroupState.mu.Lock()
		state := backupGroupState.sites[info.SiteID]
		if state == nil {
			state = &siteBackupState{ActiveGroup: info.PrimaryGroup}
			backupGroupState.sites[info.SiteID] = state
		}
		if state.ActiveGroup == 0 {
			state.ActiveGroup = info.PrimaryGroup
		}
		if state.ActiveGroup == info.PrimaryGroup {
			if primaryOnline {
				state.DownCount = 0
			} else {
				state.DownCount++
			}
			if state.DownCount >= backupDownThreshold && backupOnline {
				state.ActiveGroup = info.BackupGroup
				state.DownCount = 0
				state.UpCount = 0
				targetGroup = info.BackupGroup
			}
		} else if state.ActiveGroup == info.BackupGroup {
			if primaryOnline {
				state.UpCount++
			} else {
				state.UpCount = 0
			}
			if state.UpCount >= backupRecoverSuccesses {
				state.ActiveGroup = info.PrimaryGroup
				state.UpCount = 0
				state.DownCount = 0
				targetGroup = info.PrimaryGroup
			}
		}
		backupGroupState.mu.Unlock()

		if targetGroup == 0 {
			continue
		}
		if err := switchSiteCnameGroup(info, domain, targetGroup, lineMap); err != nil {
			log.Printf("[BackupGroup] switch site=%d target=%d failed: %v", info.SiteID, targetGroup, err)
		} else {
			log.Printf("[BackupGroup] switch site=%d target=%d success", info.SiteID, targetGroup)
		}
	}
}

func loadBackupSwitchSites() ([]siteCnameInfo, map[string]models.CnameDomain, []int64, error) {
	var sites []models.Site
	if err := db.DB.Where("enable = ?", true).Find(&sites).Error; err != nil {
		return nil, nil, nil, err
	}
	if len(sites) == 0 {
		return nil, nil, nil, nil
	}

	packIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		if site.UserPackageID != 0 {
			packIDs = append(packIDs, site.UserPackageID)
		}
	}
	packIDs = uniquePackageIDs(packIDs)
	packMap := map[int64]models.UserPackage{}
	if len(packIDs) > 0 {
		var packs []models.UserPackage
		if err := db.DB.Where("id IN ?", packIDs).Find(&packs).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, pack := range packs {
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

	infos := make([]siteCnameInfo, 0, len(sites))
	groupSet := map[int64]struct{}{}
	domainSet := map[string]struct{}{}
	for _, site := range sites {
		pkg, ok := packMap[site.UserPackageID]
		if !ok {
			continue
		}
		domainKey, host := resolveSiteCnameTarget(site, pkg)
		if domainKey == "" || host == "" {
			continue
		}
		primary, backup, enableBackup := resolveSiteGroups(site, pkg, planGroupMap[int64(pkg.PackageID)])
		if !enableBackup || primary == 0 || backup == 0 {
			continue
		}
		infos = append(infos, siteCnameInfo{
			SiteID:       site.ID,
			Hostname:     host,
			DomainKey:    domainKey,
			PrimaryGroup: primary,
			BackupGroup:  backup,
			EnableBackup: enableBackup,
		})
		groupSet[primary] = struct{}{}
		groupSet[backup] = struct{}{}
		domainSet[domainKey] = struct{}{}
	}

	if len(infos) == 0 {
		return nil, nil, nil, nil
	}

	groupIDs := make([]int64, 0, len(groupSet))
	for id := range groupSet {
		groupIDs = append(groupIDs, id)
	}
	if len(groupIDs) == 0 {
		return nil, nil, nil, nil
	}
	if len(domainSet) == 0 {
		return infos, map[string]models.CnameDomain{}, groupIDs, nil
	}

	domainList := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domainList = append(domainList, domain)
	}
	sort.Strings(domainList)
	var domainRows []models.CnameDomain
	if err := db.DB.Where("domain IN ?", domainList).Find(&domainRows).Error; err != nil {
		return nil, nil, nil, err
	}
	domainMap := make(map[string]models.CnameDomain, len(domainRows))
	for _, row := range domainRows {
		key := normalizePackageDomain(row.Domain)
		if key != "" {
			domainMap[key] = row
		}
	}
	return infos, domainMap, groupIDs, nil
}

func loadGroupOnlineStatus(groupIDs []int64) map[int64]bool {
	result := map[int64]bool{}
	if db.DB == nil || len(groupIDs) == 0 {
		return result
	}
	var lines []models.Line
	if err := db.DB.Select("node_group_id", "node_id", "node_ip_id", "enable").
		Where("node_group_id IN ? AND enable = ?", groupIDs, true).
		Find(&lines).Error; err != nil {
		return result
	}
	groupNodes := map[int64]map[int64]struct{}{}
	allNodeIDs := map[int64]struct{}{}
	for _, line := range lines {
		if !line.Enable {
			continue
		}
		nodeID := line.NodeID
		if nodeID == 0 {
			nodeID = line.NodeIPID
		}
		if nodeID == 0 {
			continue
		}
		if _, ok := groupNodes[line.NodeGroupID]; !ok {
			groupNodes[line.NodeGroupID] = map[int64]struct{}{}
		}
		groupNodes[line.NodeGroupID][nodeID] = struct{}{}
		allNodeIDs[nodeID] = struct{}{}
	}
	if len(allNodeIDs) == 0 {
		return result
	}
	nodeIDs := make([]int64, 0, len(allNodeIDs))
	for id := range allNodeIDs {
		nodeIDs = append(nodeIDs, id)
	}
	var nodes []models.Node
	if err := db.DB.Select("id", "enable").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return result
	}
	enabled := map[int64]bool{}
	for _, node := range nodes {
		if node.Enable {
			enabled[node.ID] = true
		}
	}
	for groupID, nodeSet := range groupNodes {
		for nodeID := range nodeSet {
			if !enabled[nodeID] {
				continue
			}
			if IsNodeOnline(nodeID, backupGroupOnlineWindow) {
				result[groupID] = true
				break
			}
		}
	}
	return result
}

func loadGroupLineMap(groupIDs []int64) map[int64]map[lineKey]struct{} {
	result := map[int64]map[lineKey]struct{}{}
	if db.DB == nil || len(groupIDs) == 0 {
		return result
	}
	var lines []models.Line
	if err := db.DB.Select("node_group_id", "line_id", "line_name", "enable").
		Where("node_group_id IN ? AND enable = ?", groupIDs, true).
		Find(&lines).Error; err != nil {
		return result
	}
	for _, line := range lines {
		if !line.Enable {
			continue
		}
		key := lineKey{ID: strings.TrimSpace(line.LineID), Name: strings.TrimSpace(line.LineName)}
		if key.ID == "" && key.Name == "" {
			continue
		}
		if _, ok := result[line.NodeGroupID]; !ok {
			result[line.NodeGroupID] = map[lineKey]struct{}{}
		}
		result[line.NodeGroupID][key] = struct{}{}
	}
	return result
}

func switchSiteCnameGroup(info siteCnameInfo, domain models.CnameDomain, targetGroup int64, lineMap map[int64]map[lineKey]struct{}) error {
	if targetGroup == 0 {
		return nil
	}
	lineSet := map[lineKey]struct{}{}
	for key := range lineMap[info.PrimaryGroup] {
		lineSet[key] = struct{}{}
	}
	for key := range lineMap[info.BackupGroup] {
		lineSet[key] = struct{}{}
	}
	if len(lineSet) == 0 {
		return nil
	}
	for key := range lineSet {
		if err := dns.SyncPackageLineRecords(domain, info.Hostname, targetGroup, key.ID, key.Name, "resync", nil); err != nil {
			return err
		}
	}
	return nil
}
