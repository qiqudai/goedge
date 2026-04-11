package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	trafficWorkerInterval = 10 * time.Minute
)

// StartUserPackageTrafficWorker checks traffic usage and applies traffic_limit.
func StartUserPackageTrafficWorker() {
	log.Printf("[Traffic] Worker started")
	go func() {
		for {
			checkUserPackageTraffic()
			time.Sleep(trafficWorkerInterval)
		}
	}()
}

func checkUserPackageTraffic() {
	httpCfg := buildHTTPConfig()
	if !db.ClickHouseEnabled() && httpCfg == nil {
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
	if err := db.DB.Where("traffic > 0 AND (is_expired = ? OR is_expired IS NULL)", false).Find(&packages).Error; err != nil {
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

	filterMap := make(map[int64]HostFilter, len(packageIDs))
	for _, site := range sites {
		if len(site.Domains) == 0 {
			continue
		}
		for _, domain := range site.Domains {
			exact, wildcard := splitHostPattern(domain)
			if exact == "" && wildcard == "" {
				continue
			}
			filter := filterMap[site.UserPackageID]
			if exact != "" && !containsString(filter.Exact, exact) {
				filter.Exact = append(filter.Exact, exact)
			}
			if wildcard != "" && !containsString(filter.Wildcards, wildcard) {
				filter.Wildcards = append(filter.Wildcards, wildcard)
			}
			filterMap[site.UserPackageID] = filter
		}
	}

	for _, pkg := range packages {
		if !pkg.EndAt.IsZero() && pkg.EndAt.Before(now) {
			continue
		}
		hostFilter := filterMap[pkg.ID]
		if hostFilter.Empty() {
			continue
		}

		startAt := pkg.StartAt
		if startAt.IsZero() || startAt.After(now) {
			startAt = now.Add(-24 * time.Hour)
		}
		usedBytes, err := sumTrafficBytesByFilter(hostFilter, startAt, now, httpCfg)
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
			siteIDs, err := applyTrafficLimit(pkg.ID)
			if err != nil {
				log.Printf("[Traffic] Package %d limit failed: %v", pkg.ID, err)
				continue
			}
			if len(siteIDs) == 0 {
				log.Printf("[Traffic] Package %d exceeded but no sites to limit", pkg.ID)
			}
			if len(siteIDs) > 0 {
				notifyTrafficExceed(pkg, usedGB, limitGB, len(siteIDs))
			}
		} else {
			clearTrafficLimit(pkg.ID)
		}
	}
}

func sumTrafficBytesByFilter(hostFilter HostFilter, start, end time.Time, httpCfg *httpCKConfig) (uint64, error) {
	if hostFilter.Empty() {
		return 0, nil
	}
	if httpCfg != nil {
		return sumTrafficBytesByFilterHTTP(httpCfg, hostFilter, start, end)
	}
	condition, condArgs := hostFilter.SQLConditionForExpr(AccessLogSiteExpr())
	if condition == "" {
		return 0, nil
	}
	args := make([]interface{}, 0, 2+len(condArgs))
	args = append(args, start, end)
	args = append(args, condArgs...)
	query := fmt.Sprintf(
		"SELECT sum(bytes) FROM node_access_logs WHERE ts >= ? AND ts <= ? AND %s AND %s",
		AccessLogRealSiteTrafficCondition(),
		condition,
	)
	var sum uint64
	if err := db.CK.QueryRow(query, args...).Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

func sumTrafficBytesByFilterHTTP(cfg *httpCKConfig, hostFilter HostFilter, start, end time.Time) (uint64, error) {
	if cfg == nil || hostFilter.Empty() {
		return 0, nil
	}
	startStr := formatTime(start)
	endStr := formatTime(end)
	condition := hostFilter.HTTPConditionForExpr(AccessLogSiteExpr())
	if condition == "" {
		return 0, nil
	}
	query := fmt.Sprintf(
		"SELECT sum(bytes) FROM node_access_logs WHERE ts >= toDateTime('%s') AND ts <= toDateTime('%s') AND %s AND %s",
		startStr,
		endStr,
		AccessLogRealSiteTrafficCondition(),
		condition,
	)
	params := url.Values{}
	params.Set("query", query)
	if cfg.database != "" {
		params.Set("database", cfg.database)
	}
	endpoint := cfg.baseURL + "/?" + params.Encode()

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return 0, err
	}
	if cfg.user != "" {
		req.SetBasicAuth(cfg.user, cfg.pass)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("http status %s", resp.Status)
	}
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseUint(raw, 10, 64)
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func applyTrafficLimit(packageID int64) ([]int64, error) {
	var siteIDs []int64
	if err := db.DB.Model(&models.Site{}).
		Where("user_package = ? AND enable = ? AND (state = ? OR state = ? OR state IS NULL)", packageID, true, "", "running").
		Pluck("id", &siteIDs).Error; err != nil {
		log.Printf("[Traffic] Package %d load sites failed: %v", packageID, err)
		return nil, err
	}
	if len(siteIDs) == 0 {
		return siteIDs, nil
	}
	if err := db.DB.Model(&models.Site{}).Where("id IN ?", siteIDs).
		Update("state", "traffic_limit").Error; err != nil {
		log.Printf("[Traffic] Package %d update sites failed: %v", packageID, err)
		return nil, err
	}
	BumpConfigVersion("site", siteIDs)
	log.Printf("[Traffic] Package %d exceeded, %d sites limited", packageID, len(siteIDs))
	return siteIDs, nil
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

func notifyTrafficExceed(pkg models.UserPackage, usedGB, limitGB float64, siteCount int) {
	userID := int64(pkg.UserID)
	if userID == 0 {
		return
	}
	title := "Traffic limit exceeded"
	content := fmt.Sprintf("Package %s exceeded traffic (%.2fGB/%.2fGB). %d site(s) have been limited.", pkg.Name, usedGB, limitGB, siteCount)
	_ = CreateUserMessage(userID, "traffic-exceed", title, content, pkg.ID, 0)
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
