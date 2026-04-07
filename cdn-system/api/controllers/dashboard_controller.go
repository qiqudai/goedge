package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardController struct{}

const (
	dashboardTimeLayout       = "2006-01-02 15:04:05"
	dashboardAnnouncementType = "announcement"
)

// Index Retrieves aggregated dashboard data
// GET /api/v1/admin/dashboard
func (c *DashboardController) Index(ctx *gin.Context) {
	now := time.Now()
	role := resolveDashboardRole(ctx)
	userID := parseUserID(mustGet(ctx, "userID"))
	isUser := isUserRequest(ctx)

	hostFilter := resolveHostFilter(ctx)

	userInfo := loadDashboardUserInfo(userID, role)
	overviewRange := resolveDashboardRange(ctx, "overview_range", "today", now)
	chartRange := resolveDashboardRange(ctx, "chart_range", "today", now)
	opsRange := resolveDashboardRange(ctx, "ops_range", "7d", now)

	overview := emptyOverviewStats()
	charts := emptyChartStats()
	topDomains := []gin.H{}
	topURLs := []gin.H{}
	topIPs := []gin.H{}
	topCountries := []gin.H{}

	if !isUser || !hostFilter.Empty() {
		topRange := services.ResolveStatsRange("30min", "", "", now)
		overview = buildOverviewStats(overviewRange, hostFilter)
		charts = buildChartStats(chartRange, hostFilter)
		topDomains = buildTopList(queryRanking("domain", topRange, hostFilter, 10))
		topURLs = buildTopList(queryRanking("url", topRange, hostFilter, 10))
		topIPs = buildTopList(queryRanking("ip", topRange, hostFilter, 10))
		topCountries = buildTopList(queryRegionRanking("country", topRange, hostFilter, 10))
	}

	announcements := loadAnnouncements(5)
	packageInfo := loadPackageInfo(userID)
	resources := loadDashboardResources(userID, isUser)
	ops := gin.H{}
	if !isUser {
		ops = loadOpsSummary(opsRange)
	}

	systemStatus, license := loadSystemStatus()

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user":          userInfo,
			"stats":         overview,
			"charts":        charts,
			"top_domains":   topDomains,
			"top_urls":      topURLs,
			"top_ips":       topIPs,
			"top_countries": topCountries,
			"announcements": announcements,
			"package":       packageInfo,
			"resources":     resources,
			"ops":           ops,
			"system_status": systemStatus,
			"license":       license,
		},
	})
}

func resolveDashboardRole(ctx *gin.Context) string {
	if val, ok := ctx.Get("role"); ok {
		if role, ok := val.(string); ok && role != "" {
			return role
		}
	}
	if isUserRequest(ctx) {
		return "user"
	}
	return "admin"
}

func resolveDashboardRange(ctx *gin.Context, key, fallback string, now time.Time) services.StatsRange {
	rangeKey := strings.TrimSpace(ctx.Query(key))
	if rangeKey == "" {
		rangeKey = strings.TrimSpace(ctx.Query("range"))
	}
	if rangeKey == "" {
		rangeKey = fallback
	}
	return services.ResolveStatsRange(rangeKey, "", "", now)
}

func loadDashboardUserInfo(userID int64, role string) gin.H {
	userInfo := gin.H{
		"role": role,
	}
	if userID == 0 {
		return userInfo
	}
	var user models.User
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return userInfo
	}
	lastLoginAt, lastLoginIP := loadLastLogin(userID)
	authState := T("dashboard.auth_unverified")
	if user.CertVerified {
		authState = T("dashboard.auth_verified")
	}
	userInfo["username"] = user.Name
	userInfo["id"] = user.ID
	userInfo["level"] = "V0"
	userInfo["auth_state"] = authState
	userInfo["last_login"] = lastLoginAt
	userInfo["login_ip"] = lastLoginIP
	userInfo["avatar"] = ""
	return userInfo
}

func loadLastLogin(userID int64) (string, string) {
	if userID == 0 {
		return "-", "-"
	}
	var row struct {
		IP        string
		CreatedAt time.Time `gorm:"column:create_at"`
	}
	err := db.DB.Table("login_log").
		Select("ip, create_at").
		Where("uid = ? AND success = ?", userID, true).
		Order("id desc").
		Limit(1).
		Scan(&row).Error
	if err != nil || row.CreatedAt.IsZero() {
		return "-", "-"
	}
	return row.CreatedAt.Format(dashboardTimeLayout), strings.TrimSpace(row.IP)
}

func buildOverviewStats(rng services.StatsRange, hostFilter services.HostFilter) gin.H {
	totals, err := services.QueryAccessTotalsRealTraffic(rng.Start, rng.End, hostFilter)
	if err != nil {
		return emptyOverviewStats()
	}

	peakRange := rng
	peakRange.Bucket = resolveOverviewPeakBucket(rng)
	peakBuckets, err := services.QueryAccessBucketsRealTraffic(peakRange.Start, peakRange.End, peakRange.Bucket, hostFilter)
	if err != nil {
		peakBuckets = []services.AccessBucket{}
	}
	peakSeries := services.BuildBucketSeries(peakRange, peakBuckets)
	peakMbps := 0.0
	if peakRange.Bucket > 0 {
		for _, bytes := range peakSeries.Bytes {
			val := services.BytesToMbps(bytes, peakRange.Bucket)
			if val > peakMbps {
				peakMbps = val
			}
		}
	}
	nodePeakText := "-"
	if nodePeakMbps, err := services.QueryNodeBandwidthPeakMbps(rng.Start, rng.End); err == nil {
		nodePeakText = services.FormatBandwidth(nodePeakMbps)
	}
	return gin.H{
		"bandwidth_peak":      services.FormatBandwidth(services.RoundFloat(peakMbps, 2)),
		"node_bandwidth_peak": nodePeakText,
		"requests":            services.FormatCount(totals.Requests),
		"traffic":             services.FormatBytes(totals.Bytes),
		"blocked_ips":         services.FormatCount(totals.BlockedIPs),
	}
}

func resolveOverviewPeakBucket(rng services.StatsRange) time.Duration {
	total := rng.End.Sub(rng.Start)
	if total <= 0 {
		return rng.Bucket
	}
	if total <= 48*time.Hour {
		return time.Minute
	}
	if total <= 14*24*time.Hour {
		return 5 * time.Minute
	}
	if total <= 45*24*time.Hour {
		return time.Hour
	}
	return rng.Bucket
}

func buildChartStats(rng services.StatsRange, hostFilter services.HostFilter) gin.H {
	buckets, err := services.QueryAccessBucketsRealTraffic(rng.Start, rng.End, rng.Bucket, hostFilter)
	if err != nil {
		return emptyChartStats()
	}
	series := services.BuildBucketSeries(rng, buckets)
	bandwidth := make([]float64, 0, len(series.Bytes))
	traffic := make([]float64, 0, len(series.Bytes))
	requests := make([]float64, 0, len(series.Requests))
	blocked := make([]float64, 0, len(series.BlockedIPs))
	for i := range series.Bytes {
		bandwidth = append(bandwidth, services.RoundFloat(services.BytesToMbps(series.Bytes[i], rng.Bucket), 2))
		traffic = append(traffic, services.RoundFloat(services.BytesToMB(series.Bytes[i]), 2))
		requests = append(requests, float64(series.Requests[i]))
		blocked = append(blocked, float64(series.BlockedIPs[i]))
	}
	return gin.H{
		"x_axis":    series.XAxis,
		"bandwidth": bandwidth,
		"requests":  requests,
		"traffic":   traffic,
		"blocked":   blocked,
	}
}

func emptyOverviewStats() gin.H {
	return gin.H{
		"bandwidth_peak":      "-",
		"node_bandwidth_peak": "-",
		"requests":            "0",
		"traffic":             "0 B",
		"blocked_ips":         "0",
	}
}

func emptyChartStats() gin.H {
	return gin.H{
		"x_axis":    []string{},
		"bandwidth": []float64{},
		"requests":  []float64{},
		"traffic":   []float64{},
		"blocked":   []float64{},
	}
}

func queryRanking(rankType string, rng services.StatsRange, hostFilter services.HostFilter, limit int) []services.RankItem {
	list, err := services.QueryAccessRanking(rankType, rng.Start, rng.End, hostFilter, "", limit)
	if err != nil {
		return []services.RankItem{}
	}
	return list
}

func queryRegionRanking(regionType string, rng services.StatsRange, hostFilter services.HostFilter, limit int) []services.RankItem {
	list, err := services.QueryRegionRanking(regionType, rng.Start, rng.End, hostFilter, "", limit)
	if err != nil {
		return []services.RankItem{}
	}
	return list
}

func buildTopList(items []services.RankItem) []gin.H {
	if len(items) == 0 {
		return []gin.H{}
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Item)
		if name == "" {
			name = "-"
		}
		list = append(list, gin.H{
			"name":    name,
			"count":   item.RequestCount,
			"traffic": services.FormatBytes(item.OutBytes),
		})
	}
	return list
}

func loadAnnouncements(limit int) []gin.H {
	if limit <= 0 {
		limit = 5
	}
	var items []models.Message
	err := db.DB.Model(&models.Message{}).
		Where("type = ? AND is_show = ?", dashboardAnnouncementType, true).
		Order("id desc").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return []gin.H{}
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":    item.ID,
			"title": item.Title,
			"time":  item.CreatedAt.Format("2006-01-02"),
		})
	}
	return list
}

func loadPackageInfo(userID int64) gin.H {
	if userID == 0 {
		return gin.H{}
	}
	var pkg models.UserPackage
	err := db.DB.Where("uid = ? AND (is_expired = ? OR is_expired IS NULL)", userID, false).
		Order("id desc").
		First(&pkg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return gin.H{}
		}
		return gin.H{}
	}

	hostFilter, err := buildPackageHostFilter(userID, pkg.ID)
	if err != nil || hostFilter.Empty() {
		return gin.H{
			"name":    pkg.Name,
			"desc":    "",
			"percent": 0,
		}
	}

	start := pkg.StartAt
	if start.IsZero() || start.After(time.Now()) {
		start = time.Now().Add(-24 * time.Hour)
	}
	totals, err := services.QueryAccessTotals(start, time.Now(), hostFilter)
	if err != nil {
		return gin.H{
			"name":    pkg.Name,
			"desc":    "",
			"percent": 0,
		}
	}

	usedGB := float64(totals.Bytes) / (1024 * 1024 * 1024)
	limitGB := float64(pkg.Traffic)
	percent := 0
	desc := fmt.Sprintf("%.2f GB used", usedGB)
	if limitGB > 0 {
		percent = int(services.RoundFloat(usedGB/limitGB*100, 0))
		if percent > 100 {
			percent = 100
		}
		desc = fmt.Sprintf("%.2f GB / %.2f GB", usedGB, limitGB)
	}

	return gin.H{
		"name":    pkg.Name,
		"desc":    desc,
		"percent": percent,
	}
}

func buildPackageHostFilter(userID, packageID int64) (services.HostFilter, error) {
	if userID == 0 || packageID == 0 {
		return services.HostFilter{}, nil
	}
	var siteIDs []int64
	if err := db.DB.Model(&models.Site{}).
		Where("uid = ? AND user_package = ?", userID, packageID).
		Pluck("id", &siteIDs).Error; err != nil {
		return services.HostFilter{}, err
	}
	if len(siteIDs) == 0 {
		return services.HostFilter{}, nil
	}
	idx, err := services.LoadSiteHostIndex(userID)
	if err != nil {
		return services.HostFilter{}, err
	}
	filter := services.HostFilter{}
	seenExact := map[string]struct{}{}
	seenWildcard := map[string]struct{}{}
	for _, siteID := range siteIDs {
		siteFilter, ok := idx.SiteFilters[siteID]
		if !ok {
			continue
		}
		for _, host := range siteFilter.Exact {
			if _, ok := seenExact[host]; ok {
				continue
			}
			seenExact[host] = struct{}{}
			filter.Exact = append(filter.Exact, host)
		}
		for _, host := range siteFilter.Wildcards {
			if _, ok := seenWildcard[host]; ok {
				continue
			}
			seenWildcard[host] = struct{}{}
			filter.Wildcards = append(filter.Wildcards, host)
		}
	}
	return filter, nil
}

func loadDashboardResources(userID int64, isUser bool) gin.H {
	siteQuery := db.DB.Model(&models.Site{}).Select("id, domain")
	if isUser && userID != 0 {
		siteQuery = siteQuery.Where("uid = ?", userID)
	}
	var sites []models.Site
	_ = siteQuery.Find(&sites).Error
	domainCount := countUniqueDomains(sites)

	forwardCount := int64(0)
	forwardQuery := db.DB.Model(&models.Forward{})
	if isUser && userID != 0 {
		forwardQuery = forwardQuery.Where("uid = ?", userID)
	}
	_ = forwardQuery.Count(&forwardCount).Error

	certCount := int64(0)
	certQuery := db.DB.Model(&models.Cert{})
	if isUser && userID != 0 {
		certQuery = certQuery.Where("uid = ?", userID)
	}
	_ = certQuery.Count(&certCount).Error

	packageCount := int64(0)
	packageQuery := db.DB.Model(&models.UserPackage{})
	if isUser && userID != 0 {
		packageQuery = packageQuery.Where("uid = ?", userID)
	}
	_ = packageQuery.Count(&packageCount).Error

	return gin.H{
		"domains":  domainCount,
		"forward":  forwardCount,
		"certs":    certCount,
		"packages": packageCount,
	}
}

func countUniqueDomains(sites []models.Site) int64 {
	if len(sites) == 0 {
		return 0
	}
	seen := map[string]struct{}{}
	for _, site := range sites {
		for _, domain := range site.Domains {
			host := normalizeDashboardDomain(domain)
			if host == "" {
				continue
			}
			seen[host] = struct{}{}
		}
	}
	return int64(len(seen))
}

func normalizeDashboardDomain(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	if host == "" {
		return ""
	}
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
	host = strings.TrimPrefix(host, "*.")
	host = strings.TrimRight(host, ".")
	return host
}

func loadOpsSummary(rng services.StatsRange) gin.H {
	userCount := int64(0)
	_ = db.DB.Model(&models.User{}).
		Where("type <> ? AND create_at BETWEEN ? AND ?", 1, rng.Start, rng.End).
		Count(&userCount).Error

	packageCount := int64(0)
	_ = db.DB.Model(&models.Order{}).
		Where("LOWER(type) IN ? AND LOWER(state) IN ? AND create_at BETWEEN ? AND ?",
			[]string{"purchase", "renew"},
			[]string{"paid", "success", "done"},
			rng.Start, rng.End).
		Count(&packageCount).Error

	rechargeSum := int64(0)
	_ = db.DB.Model(&models.Order{}).
		Select("COALESCE(SUM(amount),0)").
		Where("LOWER(type) IN ? AND LOWER(state) IN ? AND create_at BETWEEN ? AND ?",
			[]string{"recharge"},
			[]string{"paid", "success", "done"},
			rng.Start, rng.End).
		Scan(&rechargeSum).Error

	rechargeText := fmt.Sprintf("%.2f", float64(rechargeSum)/100.0)
	return gin.H{
		"summary": gin.H{
			"users":    userCount,
			"packages": packageCount,
			"recharge": rechargeText,
		},
	}
}

func loadSystemStatus() (gin.H, gin.H) {
	totalNodes := int64(0)
	_ = db.DB.Model(&models.Node{}).Where("pid = 0").Count(&totalNodes).Error

	onlineNodes := int64(0)
	var nodeIDs []int64
	_ = db.DB.Model(&models.Node{}).Where("pid = 0 AND enable = ?", true).Pluck("id", &nodeIDs).Error
	for _, id := range nodeIDs {
		if services.IsNodeOnline(id, 90*time.Second) {
			onlineNodes++
		}
	}

	ckHealth := services.CheckClickHouseHealth()
	systemStatus := gin.H{
		"master":       true,
		"ck":           ckHealth.OK,
		"ck_tips":      ckHealth.Errors,
		"ck_db":        ckHealth.Database,
		"ck_missing":   ckHealth.MissingTable,
		"agent":        totalNodes > 0 && onlineNodes == totalNodes,
		"agent_total":  totalNodes,
		"agent_online": onlineNodes,
		"checked_at":   time.Now().Format(dashboardTimeLayout),
	}

	license := gin.H{
		"total_nodes":   totalNodes,
		"current_nodes": onlineNodes,
		"expire_at":     "-",
	}

	return systemStatus, license
}
