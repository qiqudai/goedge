package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	trafficWorkerInterval = 10 * time.Minute
	trafficHostChunkSize  = 200
)

// StartUserPackageTrafficWorker checks traffic usage and applies traffic_limit.
func StartUserPackageTrafficWorker() {
	go func() {
		for {
			checkUserPackageTraffic()
			time.Sleep(trafficWorkerInterval)
		}
	}()
}

func checkUserPackageTraffic() {
	if !db.ClickHouseEnabled() {
		return
	}

	systemCfg, err := LoadConfigMap("system", "global", 0)
	if err != nil {
		log.Printf("[Traffic] Failed to load system config: %v", err)
		return
	}
	if !parseBoolConfig(systemCfg["traffic_excceed_close_site"]) {
		return
	}
	trafficFactor := parseFloatConfig(systemCfg["tcp_traffic_factor"], 1.0)
	if trafficFactor <= 0 {
		trafficFactor = 1.0
	}

	var packages []models.UserPackage
	if err := db.DB.Where("traffic > 0 AND is_expired = ?", false).Find(&packages).Error; err != nil {
		log.Printf("[Traffic] Failed to load user packages: %v", err)
		return
	}
	if len(packages) == 0 {
		return
	}

	now := time.Now()
	packageIDs := make([]int64, 0, len(packages))
	for _, p := range packages {
		if !p.EndAt.IsZero() && p.EndAt.Before(now) {
			continue
		}
		packageIDs = append(packageIDs, p.ID)
	}
	if len(packageIDs) == 0 {
		return
	}

	var sites []models.Site
	if err := db.DB.Where("user_package IN ?", packageIDs).Find(&sites).Error; err != nil {
		log.Printf("[Traffic] Failed to load sites: %v", err)
		return
	}

	domainMap := make(map[int64]map[string]struct{}, len(packageIDs))
	for _, site := range sites {
		if len(site.Domains) == 0 {
			continue
		}
		set := domainMap[site.UserPackageID]
		if set == nil {
			set = make(map[string]struct{})
			domainMap[site.UserPackageID] = set
		}
		for _, domain := range site.Domains {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain == "" {
				continue
			}
			set[domain] = struct{}{}
		}
	}

	for _, pkg := range packages {
		if !pkg.EndAt.IsZero() && pkg.EndAt.Before(now) {
			continue
		}
		hostSet := domainMap[pkg.ID]
		if len(hostSet) == 0 {
			continue
		}
		hosts := make([]string, 0, len(hostSet))
		for host := range hostSet {
			hosts = append(hosts, host)
		}

		startAt := pkg.StartAt
		if startAt.IsZero() || startAt.After(now) {
			startAt = now.Add(-24 * time.Hour)
		}
		usedBytes, err := sumTrafficBytesByHosts(hosts, startAt, now)
		if err != nil {
			log.Printf("[Traffic] Package %d query failed: %v", pkg.ID, err)
			continue
		}
		if trafficFactor != 1 {
			usedBytes = uint64(float64(usedBytes) * trafficFactor)
		}

		usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
		limitGB := float64(pkg.Traffic)
		if limitGB <= 0 {
			continue
		}

		if usedGB >= limitGB {
			applyTrafficLimit(pkg.ID)
		} else {
			clearTrafficLimit(pkg.ID)
		}
	}
}

func sumTrafficBytesByHosts(hosts []string, start, end time.Time) (uint64, error) {
	if len(hosts) == 0 {
		return 0, nil
	}
	var total uint64
	for i := 0; i < len(hosts); i += trafficHostChunkSize {
		endIdx := i + trafficHostChunkSize
		if endIdx > len(hosts) {
			endIdx = len(hosts)
		}
		chunk := hosts[i:endIdx]
		placeholders := make([]string, 0, len(chunk))
		args := make([]interface{}, 0, 2+len(chunk))
		args = append(args, start, end)
		for _, host := range chunk {
			placeholders = append(placeholders, "?")
			args = append(args, host)
		}
		query := fmt.Sprintf("SELECT sum(bytes) FROM node_access_logs WHERE ts >= ? AND ts <= ? AND host IN (%s)", strings.Join(placeholders, ","))
		var sum uint64
		if err := db.CK.QueryRow(query, args...).Scan(&sum); err != nil {
			return total, err
		}
		total += sum
	}
	return total, nil
}

func applyTrafficLimit(packageID int64) {
	var siteIDs []int64
	if err := db.DB.Model(&models.Site{}).
		Where("user_package = ? AND enable = ? AND (state = ? OR state = ? OR state IS NULL)", packageID, true, "", "running").
		Pluck("id", &siteIDs).Error; err != nil {
		log.Printf("[Traffic] Package %d load sites failed: %v", packageID, err)
		return
	}
	if len(siteIDs) == 0 {
		return
	}
	if err := db.DB.Model(&models.Site{}).Where("id IN ?", siteIDs).
		Update("state", "traffic_limit").Error; err != nil {
		log.Printf("[Traffic] Package %d update sites failed: %v", packageID, err)
		return
	}
	BumpConfigVersion("site", siteIDs)
	log.Printf("[Traffic] Package %d exceeded, %d sites limited", packageID, len(siteIDs))
}

func clearTrafficLimit(packageID int64) {
	var siteIDs []int64
	if err := db.DB.Model(&models.Site{}).
		Where("user_package = ? AND enable = ? AND state = ?", packageID, true, "traffic_limit").
		Pluck("id", &siteIDs).Error; err != nil {
		log.Printf("[Traffic] Package %d load sites failed: %v", packageID, err)
		return
	}
	if len(siteIDs) == 0 {
		return
	}
	if err := db.DB.Model(&models.Site{}).Where("id IN ?", siteIDs).
		Update("state", "running").Error; err != nil {
		log.Printf("[Traffic] Package %d restore sites failed: %v", packageID, err)
		return
	}
	BumpConfigVersion("site", siteIDs)
	log.Printf("[Traffic] Package %d recovered, %d sites resumed", packageID, len(siteIDs))
}

func parseBoolConfig(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseFloatConfig(value string, fallback float64) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return fallback
}
