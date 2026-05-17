package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type BlockLogController struct{}

const (
	blockDefaultRange = "7d"
	blockTimeLayout   = "2006-01-02 15:04:05"
)

// ListCurrent Retrieves current blocked IPs
// GET /api/v1/admin/logs/block/current
func (c *BlockLogController) ListCurrent(ctx *gin.Context) {
	page, pageSize := parseBlockPage(ctx, 10)
	offset := (page - 1) * pageSize
	filterType := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("type", "ip")))
	keyword := strings.TrimSpace(ctx.Query("keyword"))

	index, hostFilter, ok := resolveBlockHostFilter(ctx)
	if !ok {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	ipFilter := ""
	if filterType == "ip" {
		ipFilter = keyword
	}
	if filterType == "site_id" {
		siteID := parseBlockSiteID(keyword)
		siteFilter, ok := resolveSiteHostFilter(index, siteID)
		if !ok {
			writeBlockList(ctx, []gin.H{}, 0)
			return
		}
		hostFilter = siteFilter
	}

	statsRange := resolveBlockRange(ctx, blockDefaultRange)
	rows, total, err := services.QueryBlockedCurrent(statsRange.Start, statsRange.End, hostFilter, ipFilter, pageSize, offset)
	if err != nil {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	list := make([]gin.H, 0, len(rows))
	siteIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		siteID, _ := resolveBlockSite(index, row.Host)
		if siteID > 0 {
			siteIDs = append(siteIDs, siteID)
		}
	}
	blackTimeoutBySite := loadSiteBlackTimeoutSeconds(siteIDs)
	for i, row := range rows {
		siteID, domain := resolveBlockSite(index, row.Host)
		blockMeta := parseBlockMeta(row.Status, row.BlockFrom)
		releaseTime := "-"
		if siteID > 0 {
			releaseTime = formatBlockReleaseTime(row.BlockTime, blackTimeoutBySite[siteID])
		}
		list = append(list, gin.H{
			"id":            offset + i + 1,
			"site_id":       siteID,
			"domain":        domain,
			"ip":            row.IP,
			"location":      formatBlockLocation(row.Country, row.Province),
			"filter":        blockMeta.Label,
			"block_module":  blockMeta.Module,
			"source_module": blockMeta.SourceModule,
			"block_rule":    blockMeta.Rule,
			"block_rule_id": blockMeta.RuleID,
			"block_config":  blockMeta.Config,
			"condition":     blockMeta.Condition,
			"block_source":  blockMeta.Source,
			"block_time":    formatBlockTime(row.BlockTime),
			"release_time":  releaseTime,
		})
	}

	writeBlockList(ctx, list, total)
}

// ListStats Retrieves block statistics
// GET /api/v1/admin/logs/block/stats
func (c *BlockLogController) ListStats(ctx *gin.Context) {
	page, pageSize := parseBlockPage(ctx, 10)
	offset := (page - 1) * pageSize

	index, hostFilter, ok := resolveBlockHostFilter(ctx)
	if !ok {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	statsRange := resolveBlockRange(ctx, blockDefaultRange)
	rows, total, err := services.QueryBlockedStats(statsRange.Start, statsRange.End, hostFilter, pageSize, offset)
	if err != nil {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	list := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		siteID, domain := resolveBlockSite(index, row.Host)
		list = append(list, gin.H{
			"site_id": siteID,
			"domain":  domain,
			"count":   row.Count,
		})
	}

	writeBlockList(ctx, list, total)
}

// ListHistory Retrieves history of blocked IPs
// GET /api/v1/admin/logs/block/history
func (c *BlockLogController) ListHistory(ctx *gin.Context) {
	page, pageSize := parseBlockPage(ctx, 10)
	offset := (page - 1) * pageSize
	filterType := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("type", "ip")))
	keyword := strings.TrimSpace(ctx.Query("keyword"))

	index, hostFilter, ok := resolveBlockHostFilter(ctx)
	if !ok {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	ipFilter := ""
	switch filterType {
	case "ip":
		ipFilter = keyword
	case "site_id":
		siteID := parseBlockSiteID(keyword)
		siteFilter, ok := resolveSiteHostFilter(index, siteID)
		if !ok {
			writeBlockList(ctx, []gin.H{}, 0)
			return
		}
		hostFilter = siteFilter
	case "time_range":
		// Handled by range parsing below.
	default:
		ipFilter = keyword
	}

	start, end := resolveBlockHistoryRange(ctx)
	rows, total, err := services.QueryBlockedHistory(start, end, hostFilter, ipFilter, pageSize, offset)
	if err != nil {
		writeBlockList(ctx, []gin.H{}, 0)
		return
	}

	list := make([]gin.H, 0, len(rows))
	for i, row := range rows {
		siteID, domain := resolveBlockSite(index, row.Host)
		blockMeta := parseBlockMeta(row.Status, row.BlockFrom)
		list = append(list, gin.H{
			"id":            offset + i + 1,
			"site_id":       siteID,
			"domain":        domain,
			"ip":            row.IP,
			"location":      formatBlockLocation(row.Country, row.Province),
			"filter":        blockMeta.Label,
			"block_module":  blockMeta.Module,
			"source_module": blockMeta.SourceModule,
			"block_rule":    blockMeta.Rule,
			"block_rule_id": blockMeta.RuleID,
			"block_config":  blockMeta.Config,
			"condition":     blockMeta.Condition,
			"block_source":  blockMeta.Source,
			"block_time":    formatBlockTime(row.BlockTime),
			"is_manual":     false,
		})
	}

	writeBlockList(ctx, list, total)
}

// UnblockIP removes blocked records by IP.
// POST /api/v1/admin/logs/block/unblock_ip
func (c *BlockLogController) UnblockIP(ctx *gin.Context) {
	var req struct {
		IP     string `json:"ip"`
		Domain string `json:"domain"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	ip := strings.TrimSpace(req.IP)
	if ip == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ip is required"})
		return
	}
	host := strings.TrimSpace(req.Domain)
	var err error
	if host != "" {
		err = services.DeleteBlockedLogsByHostIPs([]services.BlockedLogKey{{Host: host, IP: ip}})
	} else {
		err = services.DeleteBlockedLogsByIPs([]string{ip})
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unblock failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// UnblockBatch removes blocked records by selected rows.
// POST /api/v1/admin/logs/block/unblock_batch
func (c *BlockLogController) UnblockBatch(ctx *gin.Context) {
	var req struct {
		Items []struct {
			IP     string `json:"ip"`
			Domain string `json:"domain"`
		} `json:"items"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	ips := make([]string, 0, len(req.Items))
	keys := make([]services.BlockedLogKey, 0, len(req.Items))
	seen := make(map[string]struct{})
	for _, item := range req.Items {
		ip := strings.TrimSpace(item.IP)
		if ip == "" {
			continue
		}
		host := strings.TrimSpace(item.Domain)
		if host != "" {
			key := host + "\x00" + ip
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, services.BlockedLogKey{Host: host, IP: ip})
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}
	if len(keys) == 0 && len(ips) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "items is required"})
		return
	}
	if len(keys) > 0 {
		if err := services.DeleteBlockedLogsByHostIPs(keys); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unblock failed"})
			return
		}
	}
	if len(ips) > 0 {
		if err := services.DeleteBlockedLogsByIPs(ips); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unblock failed"})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// UnblockSite removes all blocked records for selected sites.
// POST /api/v1/admin/logs/block/unblock_site
func (c *BlockLogController) UnblockSite(ctx *gin.Context) {
	var req struct {
		SiteIDs []int64 `json:"site_ids"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if len(req.SiteIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "site_ids is required"})
		return
	}
	var sites []models.Site
	if err := db.DB.Select("id, domain").Where("id IN ?", req.SiteIDs).Find(&sites).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "load site failed"})
		return
	}
	hosts := make([]string, 0, len(sites)*2)
	seen := make(map[string]struct{})
	for _, site := range sites {
		for _, domain := range parseStringListValue(site.DomainRaw) {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			if _, ok := seen[domain]; ok {
				continue
			}
			seen[domain] = struct{}{}
			hosts = append(hosts, domain)
		}
	}
	if len(hosts) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
		return
	}
	if err := services.DeleteBlockedLogsByHosts(hosts); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unblock failed"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

func parseBlockPage(ctx *gin.Context, defaultSize int) (int, int) {
	page, pageSize := parsePageParams(ctx, defaultSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func resolveBlockRange(ctx *gin.Context, fallback string) services.StatsRange {
	rangeKey := strings.TrimSpace(ctx.Query("time_range"))
	if rangeKey == "" {
		rangeKey = strings.TrimSpace(ctx.Query("range"))
	}
	if rangeKey == "" {
		rangeKey = fallback
	}
	return services.ResolveStatsRange(rangeKey, "", "", time.Now())
}

func resolveBlockHistoryRange(ctx *gin.Context) (time.Time, time.Time) {
	startRaw := strings.TrimSpace(ctx.Query("start_time"))
	endRaw := strings.TrimSpace(ctx.Query("end_time"))
	if startRaw != "" && endRaw != "" {
		custom := services.ResolveStatsRange("custom", startRaw, endRaw, time.Now())
		if !custom.Start.IsZero() && !custom.End.IsZero() {
			return custom.Start, custom.End
		}
	}
	rng := services.ResolveStatsRange(blockDefaultRange, "", "", time.Now())
	return rng.Start, rng.End
}

func resolveBlockHostFilter(ctx *gin.Context) (*services.SiteHostIndex, services.HostFilter, bool) {
	isUser := isUserRequest(ctx)
	userID := parseUserID(mustGet(ctx, "userID"))
	if isUser && userID == 0 {
		return nil, services.HostFilter{}, false
	}
	var idx *services.SiteHostIndex
	if isUser {
		loaded, err := services.LoadSiteHostIndex(userID)
		if err == nil {
			idx = loaded
		}
	} else {
		loaded, err := services.LoadSiteHostIndex(0)
		if err == nil {
			idx = loaded
		}
	}

	if isUser {
		if idx == nil || idx.Filter.Empty() {
			return idx, services.HostFilter{}, false
		}
		return idx, idx.Filter, true
	}

	return idx, services.HostFilter{}, true
}

func resolveSiteHostFilter(index *services.SiteHostIndex, siteID int64) (services.HostFilter, bool) {
	if siteID == 0 || index == nil {
		return services.HostFilter{}, false
	}
	filter, ok := index.SiteFilters[siteID]
	if !ok || filter.Empty() {
		return services.HostFilter{}, false
	}
	return filter, true
}

func resolveBlockSite(index *services.SiteHostIndex, host string) (int64, string) {
	host = strings.TrimSpace(host)
	if index == nil {
		return 0, host
	}
	if match, ok := index.Matcher.Match(host); ok {
		return match.SiteID, host
	}
	return 0, host
}

func parseBlockSiteID(keyword string) int64 {
	siteID, _ := strconv.ParseInt(strings.TrimSpace(keyword), 10, 64)
	return siteID
}

func formatBlockLocation(country string, province string) string {
	country = strings.TrimSpace(country)
	province = strings.TrimSpace(province)
	if country == "-" {
		country = ""
	}
	if province == "-" {
		province = ""
	}
	if country == "" && province == "" {
		return "-"
	}
	if province == "" {
		return country
	}
	if country == "" {
		return province
	}
	return country + "-" + province
}

func blockFilterLabel(status int, blockSource string) string {
	return parseBlockMeta(status, blockSource).Label
}

type blockMeta struct {
	Label        string
	Module       string
	SourceModule string
	Rule         string
	RuleID       int64
	Config       string
	Condition    string
	Source       string
}

func parseBlockMeta(status int, sourceRaw string) blockMeta {
	sourceRaw = strings.TrimSpace(sourceRaw)
	pairs := parseBlockSourcePairs(sourceRaw)
	moduleKey := strings.ToLower(strings.TrimSpace(firstBlockPair(pairs, "type")))
	if moduleKey == "" {
		moduleKey = strings.ToLower(sourceRaw)
	}
	rule := firstBlockPair(pairs, "rule")
	sourceModule := firstBlockPair(pairs, "module")
	condition := firstBlockPair(pairs, "condition")
	config := firstBlockPair(pairs, "config")
	if config == "" {
		config = firstBlockPair(pairs, "config_id")
	}
	if config == "" {
		config = firstBlockPair(pairs, "mode")
	}
	if config == "" {
		config = firstBlockPair(pairs, "action")
	}
	if config == "" {
		config = firstBlockPair(pairs, "filter")
	}
	ruleID, _ := strconv.ParseInt(strings.TrimSpace(firstBlockPair(pairs, "rule_id")), 10, 64)

	module := moduleName(moduleKey, status)
	parts := []string{module}
	if sourceModule != "" {
		parts = append(parts, "模块:"+sourceModule)
	}
	if rule != "" {
		parts = append(parts, "规则:"+rule)
	}
	if ruleID > 0 {
		parts = append(parts, "规则ID:"+strconv.FormatInt(ruleID, 10))
	}
	if config != "" {
		parts = append(parts, "配置:"+config)
	}
	if condition != "" {
		parts = append(parts, "条件:"+condition)
	}
	if status > 0 {
		parts = append(parts, "HTTP_"+strconv.Itoa(status))
	}

	return blockMeta{
		Label:        strings.Join(parts, " | "),
		Module:       module,
		SourceModule: sourceModule,
		Rule:         rule,
		RuleID:       ruleID,
		Config:       config,
		Condition:    condition,
		Source:       sourceRaw,
	}
}

func parseBlockSourcePairs(source string) map[string]string {
	result := make(map[string]string)
	if source == "" {
		return result
	}
	parts := strings.Split(source, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 || idx >= len(part)-1 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		if key == "" {
			continue
		}
		result[key] = val
	}
	return result
}

func firstBlockPair(pairs map[string]string, key string) string {
	if value, ok := pairs[key]; ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func moduleName(moduleKey string, status int) string {
	switch moduleKey {
	case "ip_block":
		return "IP黑名单"
	case "anti_cc":
		return "Anti-CC"
	case "cc", "cc_guard":
		return "CC防护"
	case "cc_rate_limit":
		return "CC频控"
	case "waf":
		return "WAF"
	case "local_protection":
		return "本地防护"
	case "origin":
		return "源站返回"
	default:
		if status <= 0 {
			return "-"
		}
		return localProtectionLabel(status)
	}
}

func localProtectionLabel(status int) string {
	if status <= 0 {
		return "-"
	}
	switch status {
	case 418:
		return "CC防护"
	case 429:
		return "频控拦截"
	case 451:
		return "地区限制"
	case 410:
		return "策略拒绝"
	case 403:
		return "访问控制"
	default:
		return "HTTP_" + strconv.Itoa(status)
	}
}

func formatBlockTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(blockTimeLayout)
}

func formatBlockReleaseTime(blockTime time.Time, timeoutSeconds int64) string {
	if timeoutSeconds <= 0 || blockTime.IsZero() {
		return "PERMANENT"
	}
	return blockTime.Add(time.Duration(timeoutSeconds) * time.Second).Format(blockTimeLayout)
}

func loadSiteBlackTimeoutSeconds(siteIDs []int64) map[int64]int64 {
	result := make(map[int64]int64)
	uniq := make(map[int64]struct{})
	filtered := make([]int64, 0, len(siteIDs))
	for _, id := range siteIDs {
		if id <= 0 {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		filtered = append(filtered, id)
	}
	if len(filtered) == 0 {
		return result
	}
	var sites []models.Site
	if err := db.DB.Select("id, settings").Where("id IN ?", filtered).Find(&sites).Error; err != nil {
		return result
	}
	for _, site := range sites {
		result[site.ID] = parseSiteBlackTimeoutSeconds(site.SettingsRaw)
	}
	return result
}

func parseSiteBlackTimeoutSeconds(settingsRaw string) int64 {
	settingsRaw = strings.TrimSpace(settingsRaw)
	if settingsRaw == "" {
		return 0
	}
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(settingsRaw), &settings); err != nil {
		return 0
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok {
		return 0
	}
	raw, ok := security["ip_black_timeout"]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

func writeBlockList(ctx *gin.Context, list []gin.H, total interface{}) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}
