package controllers

import (
	"cdn-api/services"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
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

func generateUsageSeries(start time.Time, count int, step time.Duration, labelFormat string, base float64, variance float64) ([]string, []float64, []usagePoint) {
	times := make([]string, 0, count)
	values := make([]float64, 0, count)
	points := make([]usagePoint, 0, count)

	for i := 0; i < count; i++ {
		timestamp := start.Add(time.Duration(i) * step)
		label := timestamp.Format(labelFormat)
		val := base + (rand.Float64()-0.5)*variance
		if val < 0 {
			val = 0
		}
		val = float64(int(val*100)) / 100
		times = append(times, label)
		values = append(values, val)
		points = append(points, usagePoint{Time: label, Value: val})
	}
	return times, values, points
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
	window := ctx.DefaultQuery("window", "30d")
	// nodeID := ctx.Query("node_id")
	// excludeNIC := ctx.Query("exclude_nic")
	// startTime := ctx.Query("start_time")
	// endTime := ctx.Query("end_time")

	// Determine points based on window
	count := 30
	labelFormat := "2006-01-02"

	switch window {
	case "1d":
		count = 24
		labelFormat = "15:00"
	case "7d":
		count = 7
		labelFormat = "2006-01-02"
	case "30d":
		count = 30
		labelFormat = "2006-01-02"
	case "custom":
		count = 12 // Arbitrary for custom range
		labelFormat = "2006-01-02"
	}

	times := make([]string, count)
	inTraffic := make([]float64, count)
	outTraffic := make([]float64, count)

	now := time.Now()

	for i := 0; i < count; i++ {
		var t time.Time
		if window == "1d" {
			t = now.Add(time.Duration(i-count) * time.Hour)
		} else {
			t = now.AddDate(0, 0, i-count)
		}
		times[i] = t.Format(labelFormat)

		inTraffic[i] = float64(rand.Intn(1000)) / 10.0  // 0-100 MB
		outTraffic[i] = float64(rand.Intn(2000)) / 10.0 // 0-200 MB
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":      times,
			"in_traffic":  inTraffic,
			"out_traffic": outTraffic,
		},
	})
}

// ListNodeRanking retrieves a node ranking list by metric/time window.
// GET /api/v1/admin/stats/node_ranking?metric=bandwidth|connection|load|disk&window=1m|5m|30m|1h
func (c *StatController) ListNodeRanking(ctx *gin.Context) {
	metric := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("metric", "bandwidth")))
	window := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("window", "1m")))

	type nodeRankItem struct {
		Rank int    `json:"rank"`
		Node string `json:"node"`
		NIC  string `json:"nic"`
		Out  string `json:"out"`
		In   string `json:"in"`
	}

	baseOut := 120.0
	baseIn := 30.0
	switch metric {
	case "connection":
		baseOut = 8000
		baseIn = 3000
	case "load":
		baseOut = 2.5
		baseIn = 1.2
	case "disk":
		baseOut = 65
		baseIn = 45
	}

	var unit string
	switch metric {
	case "connection":
		unit = " conn"
	case "load":
		unit = ""
	case "disk":
		unit = "%"
	default:
		unit = " Mbps"
	}

	_ = window // reserved for future: weighting based on window

	nics := []string{"eth0", "ens3", "enp1s0", "bond0"}
	list := make([]nodeRankItem, 0, 10)
	for i := 1; i <= 10; i++ {
		out := baseOut + (rand.Float64()-0.5)*baseOut*0.4
		in := baseIn + (rand.Float64()-0.5)*baseIn*0.4
		if out < 0 {
			out = 0
		}
		if in < 0 {
			in = 0
		}
		node := fmt.Sprintf("node-%d", i)
		nic := nics[i%len(nics)]
		format := func(v float64) string {
			switch metric {
			case "connection":
				return strconv.Itoa(int(v)) + unit
			case "disk":
				return strconv.Itoa(int(v)) + unit
			case "load":
				return fmt.Sprintf("%.2f", v)
			default:
				return fmt.Sprintf("%.1f", v) + unit
			}
		}
		list = append(list, nodeRankItem{
			Rank: i,
			Node: node,
			NIC:  nic,
			Out:  format(out),
			In:   format(in),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

// ListNodeMetrics retrieves time-series points for a metric.
// GET /api/v1/admin/stats/node_metrics?metric=bandwidth|connection|load|disk&window=1h|6h|12h|custom&start_time=YYYY-MM-DD HH:mm:ss&end_time=YYYY-MM-DD HH:mm:ss
func (c *StatController) ListNodeMetrics(ctx *gin.Context) {
	metric := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("metric", "bandwidth")))
	window := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("window", "1h")))
	startRaw := strings.TrimSpace(ctx.Query("start_time"))
	endRaw := strings.TrimSpace(ctx.Query("end_time"))

	type metricPoint struct {
		Time  string  `json:"time"`
		Value float64 `json:"value"`
	}

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now
	count := 12
	step := 5 * time.Minute
	labelFormat := "15:04"

	switch window {
	case "6h":
		start = now.Add(-6 * time.Hour)
		count = 36
		step = 10 * time.Minute
		labelFormat = "15:04"
	case "12h":
		start = now.Add(-12 * time.Hour)
		count = 72
		step = 10 * time.Minute
		labelFormat = "01-02 15:04"
	case "custom":
		layout := "2006-01-02 15:04:05"
		loc := now.Location()
		startParsed, err1 := time.ParseInLocation(layout, startRaw, loc)
		endParsed, err2 := time.ParseInLocation(layout, endRaw, loc)
		if err1 != nil || err2 != nil || endParsed.Before(startParsed) {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []metricPoint{}}})
			return
		}
		start = startParsed
		end = endParsed
		total := end.Sub(start)
		if total <= 0 {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []metricPoint{}}})
			return
		}
		// Aim for ~60 points, cap to keep payload reasonable.
		count = 60
		if total < 60*time.Minute {
			count = int(total / time.Minute)
			if count < 10 {
				count = 10
			}
		}
		if count > 200 {
			count = 200
		}
		step = total / time.Duration(count)
		labelFormat = "2006-01-02 15:04"
	default:
		// 1h
	}

	base := 100.0
	variance := 30.0
	switch metric {
	case "connection":
		base = 8000
		variance = 3000
	case "load":
		base = 2.0
		variance = 1.2
	case "disk":
		base = 60
		variance = 20
	}

	points := make([]metricPoint, 0, count)
	cur := start
	for i := 0; i < count && !cur.After(end); i++ {
		val := base + (rand.Float64()-0.5)*variance
		if val < 0 {
			val = 0
		}
		if metric == "disk" && val > 100 {
			val = 100
		}
		val = float64(int(val*100)) / 100
		points = append(points, metricPoint{
			Time:  cur.Format(labelFormat),
			Value: val,
		})
		cur = cur.Add(step)
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
