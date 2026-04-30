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
		releaseTime := formatBlockReleaseTime(row.BlockTime, blackTimeoutBySite[siteID])
		list = append(list, gin.H{
			"id":           offset + i + 1,
			"site_id":      siteID,
			"domain":       domain,
			"ip":           row.IP,
			"location":     formatBlockLocation(row.Country, row.Province),
			"filter":       blockFilterLabel(row.Status),
			"block_time":   formatBlockTime(row.BlockTime),
			"release_time": releaseTime,
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
		list = append(list, gin.H{
			"id":         offset + i + 1,
			"site_id":    siteID,
			"domain":     domain,
			"ip":         row.IP,
			"location":   formatBlockLocation(row.Country, row.Province),
			"filter":     blockFilterLabel(row.Status),
			"block_time": formatBlockTime(row.BlockTime),
			"is_manual":  false,
		})
	}

	writeBlockList(ctx, list, total)
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

func blockFilterLabel(status int) string {
	if status <= 0 {
		return "-"
	}
	return "HTTP_" + strconv.Itoa(status)
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
