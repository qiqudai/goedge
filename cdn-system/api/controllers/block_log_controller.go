package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
	for i, row := range rows {
		siteID, domain := resolveBlockSite(index, row.Host)
		list = append(list, gin.H{
			"id":           offset + i + 1,
			"site_id":      siteID,
			"domain":       domain,
			"ip":           row.IP,
			"location":     formatBlockLocation(row.IP),
			"filter":       blockFilterLabel(row.Status),
			"block_time":   formatBlockTime(row.BlockTime),
			"release_time": "PERMANENT",
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
			"location":   formatBlockLocation(row.IP),
			"filter":     blockFilterLabel(row.Status),
			"block_time": formatBlockTime(row.BlockTime),
			"is_manual":  false,
		})
	}

	writeBlockList(ctx, list, total)
}

func parseBlockPage(ctx *gin.Context, defaultSize int) (int, int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", strconv.Itoa(defaultSize)))
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

func formatBlockLocation(ip string) string {
	country, province := services.LookupIPRegion(ip)
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

func writeBlockList(ctx *gin.Context, list []gin.H, total interface{}) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}
