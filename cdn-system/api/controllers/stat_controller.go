package controllers

import (
	"cdn-api/services"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)
type StatController struct{}

// ListRanking Retrieves data ranking
// GET /api/v1/admin/stats/ranking
func (c *StatController) ListRanking(ctx *gin.Context) {
	rankType := ctx.DefaultQuery("type", "domain")
	keyword := strings.TrimSpace(ctx.Query("keyword"))
	limit := services.ResolveResRankSize()

	type RankItem struct {
		Rank          int    `json:"rank"`
		Item          string `json:"item"` // Domain, URL, IP, etc.
		RequestCount  int    `json:"request_count"`
		OutTraffic    string `json:"out_traffic"`
		OriginTraffic string `json:"origin_traffic"`
	}

	hostFilter := resolveHostFilter(ctx)
	statsRange := resolveStatsRangeFromRequest(ctx)

	if rankType == "latency" {
		latencyList := services.QueryLatencyRanking(statsRange.Start, statsRange.End, hostFilter, keyword, limit)
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"list": latencyList,
			},
		})
		return
	}

	var source []services.RankItem
	switch rankType {
	case "country", "province":
		source, _ = services.QueryRegionRanking(rankType, statsRange.Start, statsRange.End, hostFilter, keyword, limit)
	default:
		source, _ = services.QueryAccessRanking(rankType, statsRange.Start, statsRange.End, hostFilter, keyword, limit)
	}
	list := make([]RankItem, 0, len(source))
	for i, item := range source {
		list = append(list, RankItem{
			Rank:          i + 1,
			Item:          item.Item,
			RequestCount:  int(item.RequestCount),
			OutTraffic:    services.FormatBytes(item.OutBytes),
			OriginTraffic: services.FormatBytes(item.OriginBytes),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

type usagePoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// ListUsage Retrieves usage series for plans
// GET /api/v1/user/usage?range=today|yesterday|7days|30days
func (c *StatController) ListUsage(ctx *gin.Context) {
	rangeKey := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("range", "today")))
	statsRange := services.ResolveStatsRange(rangeKey, "", "", time.Now())
	hostFilter := resolveHostFilter(ctx)
	if isUserRequest(ctx) && hostFilter.Empty() {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"x_axis": []string{},
				"values": []float64{},
				"list":   []usagePoint{},
				"total":  0,
				"avg":    0,
				"peak":   0,
				"unit":   T("MB"),
			},
		})
		return
	}

	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "values": []float64{}, "list": []usagePoint{}, "total": 0, "avg": 0, "peak": 0, "unit": T("MB")}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	totals, _ := services.QueryAccessTotals(statsRange.Start, statsRange.End, hostFilter)

	unit := T("MB")
	divider := float64(1024 * 1024)
	if totals.Bytes >= 1024*1024*1024 {
		unit = T("GB")
		divider = float64(1024 * 1024 * 1024)
	}

	values := make([]float64, 0, len(series.Bytes))
	list := make([]usagePoint, 0, len(series.Bytes))
	var total float64
	var peak float64
	for i, b := range series.Bytes {
		val := float64(b) / divider
		val = services.RoundFloat(val, 2)
		values = append(values, val)
		list = append(list, usagePoint{Time: series.XAxis[i], Value: val})
		total += val
		if val > peak {
			peak = val
		}
	}
	avg := 0.0
	if len(values) > 0 {
		avg = services.RoundFloat(total/float64(len(values)), 2)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis": series.XAxis,
			"values": values,
			"list":   list,
			"total":  services.RoundFloat(total, 2),
			"avg":    avg,
			"peak":   services.RoundFloat(peak, 2),
			"unit":   unit,
		},
	})
}

// ListBasic Retrieves basic statistics (Bandwidth, Traffic, QPS)
// GET /api/v1/admin/stats/basic
func (c *StatController) ListBasic(ctx *gin.Context) {
	hostFilter := resolveHostFilter(ctx)
	statsRange := resolveStatsRangeFromRequest(ctx)
	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		log.Printf("[stats] basic query failed: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}, "qps": []float64{}}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	bandwidth := make([]float64, 0, len(series.Bytes))
	traffic := make([]float64, 0, len(series.Bytes))
	qps := make([]float64, 0, len(series.Requests))
	seconds := statsRange.Bucket.Seconds()
	for i := range series.Bytes {
		bandwidth = append(bandwidth, services.RoundFloat(services.BytesToMbps(series.Bytes[i], statsRange.Bucket), 2))
		traffic = append(traffic, services.RoundFloat(services.BytesToMB(series.Bytes[i]), 2))
		value := 0.0
		if seconds > 0 {
			value = float64(series.Requests[i]) / seconds
		}
		qps = append(qps, services.RoundFloat(value, 2))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":    series.XAxis,
			"bandwidth": bandwidth,
			"traffic":   traffic,
			"qps":       qps,
		},
	})
}

// ListQuality Retrieves quality statistics (Hit Rate, 4xx, 5xx)
// GET /api/v1/admin/stats/quality
func (c *StatController) ListQuality(ctx *gin.Context) {
	hostFilter := resolveHostFilter(ctx)
	statsRange := resolveStatsRangeFromRequest(ctx)
	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		log.Printf("[stats] quality query failed: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "hit_rate": []float64{}, "status_4xx": []float64{}, "status_5xx": []float64{}}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	hitRate := make([]float64, 0, len(series.Requests))
	status4xx := make([]float64, 0, len(series.Status4xx))
	status5xx := make([]float64, 0, len(series.Status5xx))
	for i := range series.Requests {
		value := 0.0
		if series.Requests[i] > 0 {
			value = float64(series.HitCount[i]) / float64(series.Requests[i]) * 100
		}
		hitRate = append(hitRate, services.RoundFloat(value, 2))
		status4xx = append(status4xx, float64(series.Status4xx[i]))
		status5xx = append(status5xx, float64(series.Status5xx[i]))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":     series.XAxis,
			"hit_rate":   hitRate,
			"status_4xx": status4xx,
			"status_5xx": status5xx,
		},
	})
}

// ListOrigin Retrieves origin statistics (Origin Bandwidth, Origin Traffic)
// GET /api/v1/admin/stats/origin
func (c *StatController) ListOrigin(ctx *gin.Context) {
	hostFilter := resolveHostFilter(ctx)
	statsRange := resolveStatsRangeFromRequest(ctx)
	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		log.Printf("[stats] origin query failed: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "origin_bandwidth": []float64{}, "origin_traffic": []float64{}}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	bandwidth := make([]float64, 0, len(series.OriginBytes))
	traffic := make([]float64, 0, len(series.OriginBytes))
	for i := range series.OriginBytes {
		bandwidth = append(bandwidth, services.RoundFloat(services.BytesToMbps(series.OriginBytes[i], statsRange.Bucket), 2))
		traffic = append(traffic, services.RoundFloat(services.BytesToMB(series.OriginBytes[i]), 2))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":           series.XAxis,
			"origin_bandwidth": bandwidth,
			"origin_traffic":   traffic,
		},
	})
}

// ListNodeTraffic Retrieves node traffic statistics
// GET /api/v1/admin/stats/node_traffic
func (c *StatController) ListNodeTraffic(ctx *gin.Context) {
	statsRange := resolveStatsRangeFromRequest(ctx)
	hostFilter := resolveHostFilter(ctx)
	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		log.Printf("[stats] node_traffic query failed: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "in_traffic": []float64{}, "out_traffic": []float64{}}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	outTraffic := make([]float64, 0, len(series.Bytes))
	originTraffic := make([]float64, 0, len(series.OriginBytes))
	for i := range series.Bytes {
		outTraffic = append(outTraffic, services.RoundFloat(services.BytesToMB(series.Bytes[i]), 2))
		originTraffic = append(originTraffic, services.RoundFloat(services.BytesToMB(series.OriginBytes[i]), 2))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":      series.XAxis,
			"in_traffic":  originTraffic,
			"out_traffic": outTraffic,
		},
	})
}

// ListNodeRanking retrieves a node ranking list by traffic.
// GET /api/v1/admin/stats/node_ranking
func (c *StatController) ListNodeRanking(ctx *gin.Context) {
	statsRange := resolveStatsRangeFromRequest(ctx)
	hostFilter := resolveHostFilter(ctx)
	limit := 20

	type nodeRankItem struct {
		Rank       int    `json:"rank"`
		Node       string `json:"node"`
		OutTraffic string `json:"out"`
		Requests   int    `json:"request_count"`
	}

	source, _ := services.QueryNodeTrafficRanking(statsRange.Start, statsRange.End, hostFilter, limit)
	list := make([]nodeRankItem, 0, len(source))
	for i, item := range source {
		list = append(list, nodeRankItem{
			Rank:       i + 1,
			Node:       item.Item,
			OutTraffic: services.FormatBytes(item.OutBytes),
			Requests:   int(item.RequestCount),
		})
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

// ListNodeMetrics retrieves time-series bandwidth points from access logs.
// GET /api/v1/admin/stats/node_metrics
func (c *StatController) ListNodeMetrics(ctx *gin.Context) {
	statsRange := resolveStatsRangeFromRequest(ctx)
	hostFilter := resolveHostFilter(ctx)

	type metricPoint struct {
		Time  string  `json:"time"`
		Value float64 `json:"value"`
	}

	buckets, err := services.QueryAccessBuckets(statsRange.Start, statsRange.End, statsRange.Bucket, hostFilter)
	if err != nil {
		log.Printf("[stats] node_metrics query failed: %v", err)
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []metricPoint{}}})
		return
	}
	series := services.BuildBucketSeries(statsRange, buckets)
	points := make([]metricPoint, 0, len(series.Bytes))
	for i, b := range series.Bytes {
		points = append(points, metricPoint{
			Time:  series.XAxis[i],
			Value: services.RoundFloat(services.BytesToMbps(b, statsRange.Bucket), 2),
		})
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": points,
		},
	})
}

func resolveStatsRangeFromRequest(ctx *gin.Context) services.StatsRange {
	rangeKey := strings.TrimSpace(ctx.Query("time_range"))
	if rangeKey == "" {
		rangeKey = strings.TrimSpace(ctx.Query("range"))
	}
	startRaw, endRaw := resolveCustomRangeParams(ctx)
	return services.ResolveStatsRange(rangeKey, startRaw, endRaw, time.Now())
}

func resolveCustomRangeParams(ctx *gin.Context) (string, string) {
	startRaw := strings.TrimSpace(ctx.Query("start_time"))
	endRaw := strings.TrimSpace(ctx.Query("end_time"))
	if startRaw == "" || endRaw == "" {
		if values := ctx.QueryArray("timeRange[]"); len(values) >= 2 {
			startRaw = values[0]
			endRaw = values[1]
		}
	}
	if startRaw == "" || endRaw == "" {
		if values := ctx.QueryArray("timeRange"); len(values) >= 2 {
			startRaw = values[0]
			endRaw = values[1]
		}
	}
	return startRaw, endRaw
}

func resolveHostFilter(ctx *gin.Context) services.HostFilter {
	if !isUserRequest(ctx) {
		return services.HostFilter{}
	}
	userID := parseUserID(mustGet(ctx, "userID"))
	if userID == 0 {
		return services.HostFilter{}
	}
	filter, err := services.LoadHostFilter(userID)
	if err != nil {
		return services.HostFilter{}
	}
	return filter
}
