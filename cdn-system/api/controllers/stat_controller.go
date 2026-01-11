package controllers

import (
	"cdn-api/db"
	"cdn-common/i18n"
	"fmt"
	"math"
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
	timeRange := ctx.DefaultQuery("time_range", "10min")
	keyword := strings.TrimSpace(ctx.Query("keyword"))

	type RankItem struct {
		Rank          int    `json:"rank"`
		Item          string `json:"item"` // Domain, URL, IP, etc.
		RequestCount  int    `json:"request_count"`
		OutTraffic    string `json:"out_traffic"`
		OriginTraffic string `json:"origin_traffic"`
	}

	var list []RankItem

	if rankType == "latency" {
		latencyList := listLatencyRanking(ctx, timeRange, keyword)
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"list": latencyList,
			},
		})
		return
	}

	switch rankType {
	case "domain":
		list = []RankItem{
			{1, "api.ilumx.cn:443", 10335, "2.53 MB", "2.53 MB"},
			{2, "api1.acfwcj.cn:443", 8623, "6.79 MB", "6.79 MB"},
			{3, "api.mv2yas.cn:443", 4916, "9.56 MB", "9.56 MB"},
			{4, "api.fxapi2.com:443", 4908, "315.64 MB", "422.80 MB"},
			{5, "api3.sdzxhk.cn:443", 4043, "2.71 MB", "2.71 MB"},
		}
	case "url":
		list = []RankItem{
			{1, "https://api.ilumx.cn:443/ws", 9947, "2.25 MB", "2.25 MB"},
			{2, "https://api.mv2yas.cn:443//user/mine", 1347, "2.13 MB", "2.13 MB"},
			{3, "https://api1.acfwcj.cn:443/api/user/my", 1086, "1.06 MB", "1.06 MB"},
		}
	case "ip":
		list = []RankItem{
			{1, "211.90.251.15", 5002, "120 MB", "10 MB"},
			{2, "36.49.228.76", 3200, "50 MB", "5 MB"},
			{3, "112.229.182.52", 1500, "20 MB", "1 MB"},
		}
	case "country":
		list = []RankItem{
			{1, i18n.T("stat.country_china"), 80000, "5.0 GB", "1.2 GB"},
			{2, i18n.T("stat.country_usa"), 5000, "200 MB", "50 MB"},
			{3, i18n.T("stat.country_japan"), 2000, "100 MB", "20 MB"},
		}
	case "province":
		list = []RankItem{
			{1, i18n.T("stat.province_zhejiang"), 20000, "1.5 GB", "500 MB"},
			{2, i18n.T("stat.province_guangdong"), 15000, "1.2 GB", "400 MB"},
			{3, i18n.T("stat.province_beijing"), 10000, "900 MB", "300 MB"},
		}
	case "referer":
		list = []RankItem{
			{1, "-", 50000, "2.0 GB", "800 MB"},
			{2, "https://www.google.com", 1500, "100 MB", "20 MB"},
			{3, "https://www.baidu.com", 800, "50 MB", "10 MB"},
		}
	default:
		// Generate random mock
		for i := 1; i <= 10; i++ {
			list = append(list, RankItem{
				Rank:          i,
				Item:          fmt.Sprintf("Mock Item %s - %d", rankType, i),
				RequestCount:  rand.Intn(10000),
				OutTraffic:    strconv.Itoa(rand.Intn(100)) + " MB",
				OriginTraffic: strconv.Itoa(rand.Intn(100)) + " MB",
			})
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

type LatencyRankItem struct {
	Rank         int     `json:"rank"`
	Item         string  `json:"item"`
	RequestCount int     `json:"request_count"`
	AvgTime      float64 `json:"avg_time"`
	MaxTime      float64 `json:"max_time"`
	MinTime      float64 `json:"min_time"`
	P95Time      float64 `json:"p95_time"`
}

func listLatencyRanking(ctx *gin.Context, timeRange, keyword string) []LatencyRankItem {
	if !db.ClickHouseEnabled() {
		return []LatencyRankItem{}
	}

	start, end := parseStatsTimeRange(ctx, timeRange)
	conditions := []string{"ts >= ? AND ts <= ? AND request_time > 0"}
	args := []interface{}{start, end}
	if keyword != "" {
		conditions = append(conditions, "(host LIKE ? OR uri LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}

	whereSQL := strings.Join(conditions, " AND ")
	querySQL := fmt.Sprintf(`SELECT host, uri, count() AS request_count,
		avg(request_time) AS avg_time,
		max(request_time) AS max_time,
		min(request_time) AS min_time,
		quantile(0.95)(request_time) AS p95_time
		FROM node_access_logs WHERE %s
		GROUP BY host, uri
		ORDER BY avg_time DESC
		LIMIT 50`, whereSQL)

	rows, err := db.CK.Query(querySQL, args...)
	if err != nil {
		return []LatencyRankItem{}
	}
	defer rows.Close()

	list := make([]LatencyRankItem, 0)
	rank := 1
	for rows.Next() {
		var host, uri string
		var reqCount uint64
		var avgTime, maxTime, minTime, p95Time float64
		if err := rows.Scan(&host, &uri, &reqCount, &avgTime, &maxTime, &minTime, &p95Time); err != nil {
			continue
		}
		item := host
		if uri != "" {
			item = host + uri
		}
		list = append(list, LatencyRankItem{
			Rank:         rank,
			Item:         item,
			RequestCount: int(reqCount),
			AvgTime:      roundFloat(avgTime, 3),
			MaxTime:      roundFloat(maxTime, 3),
			MinTime:      roundFloat(minTime, 3),
			P95Time:      roundFloat(p95Time, 3),
		})
		rank++
	}
	return list
}

func parseStatsTimeRange(ctx *gin.Context, rangeKey string) (time.Time, time.Time) {
	now := time.Now()
	switch strings.ToLower(strings.TrimSpace(rangeKey)) {
	case "30min":
		return now.Add(-30 * time.Minute), now
	case "1h":
		return now.Add(-1 * time.Hour), now
	case "custom":
		layout := "2006-01-02 15:04:05"
		startRaw := ctx.Query("start_time")
		endRaw := ctx.Query("end_time")
		if startRaw != "" && endRaw != "" {
			start, err1 := time.Parse(layout, startRaw)
			end, err2 := time.Parse(layout, endRaw)
			if err1 == nil && err2 == nil {
				return start, end
			}
		}
		if values := ctx.QueryArray("timeRange[]"); len(values) >= 2 {
			start, err1 := time.Parse(layout, values[0])
			end, err2 := time.Parse(layout, values[1])
			if err1 == nil && err2 == nil {
				return start, end
			}
		}
	}
	return now.Add(-10 * time.Minute), now
}

func roundFloat(val float64, precision int) float64 {
	if precision < 0 {
		return val
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(val*factor) / factor
}

// Helper to generate time series data
func generateTimeSeries(count int, base float64, variance float64) ([]string, []float64) {
	times := make([]string, count)
	values := make([]float64, count)

	for i := 0; i < count; i++ {
		times[i] = fmt.Sprintf("%d:%02d", 10+i/6, (i%6)*10) // Mock time 10:00, 10:10...
		val := base + (rand.Float64()-0.5)*variance
		if val < 0 {
			val = 0
		}
		values[i] = float64(int(val*100)) / 100 // Round to 2 decimals
	}
	return times, values
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
	now := time.Now()

	var start time.Time
	var count int
	var step time.Duration
	var labelFormat string
	var base float64
	var variance float64

	switch rangeKey {
	case "yesterday":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
		count = 24
		step = time.Hour
		labelFormat = "15:00"
		base = 32
		variance = 18
	case "7days":
		start = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).Add(-6 * 24 * time.Hour)
		count = 7 * 24
		step = time.Hour
		labelFormat = "01-02 15:00"
		base = 28
		variance = 20
	case "30days":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -29)
		count = 30
		step = 24 * time.Hour
		labelFormat = "2006-01-02"
		base = 220
		variance = 120
	default:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		count = 24
		step = time.Hour
		labelFormat = "15:00"
		base = 35
		variance = 20
	}

	xAxis, values, list := generateUsageSeries(start, count, step, labelFormat, base, variance)

	var total float64
	var peak float64
	for _, v := range values {
		total += v
		if v > peak {
			peak = v
		}
	}
	avg := 0.0
	if len(values) > 0 {
		avg = float64(int((total/float64(len(values)))*100)) / 100
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis": xAxis,
			"values": values,
			"list":   list,
			"total":  float64(int(total*100)) / 100,
			"avg":    avg,
			"peak":   float64(int(peak*100)) / 100,
			"unit":   "MB",
		},
	})
}

// ListBasic Retrieves basic statistics (Bandwidth, Traffic, QPS)
// GET /api/v1/admin/stats/basic
func (c *StatController) ListBasic(ctx *gin.Context) {
	// Mock 12 points (e.g., last 2 hours, 10 min interval)
	times, bandwidth := generateTimeSeries(12, 100, 50) // Mbps
	_, traffic := generateTimeSeries(12, 500, 200)      // MB
	_, qps := generateTimeSeries(12, 5000, 1000)        // QPS

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":    times,
			"bandwidth": bandwidth,
			"traffic":   traffic,
			"qps":       qps,
		},
	})
}

// ListQuality Retrieves quality statistics (Hit Rate, 4xx, 5xx)
// GET /api/v1/admin/stats/quality
func (c *StatController) ListQuality(ctx *gin.Context) {
	times, hitRate := generateTimeSeries(12, 95, 5) // %
	_, status4xx := generateTimeSeries(12, 10, 5)   // count
	_, status5xx := generateTimeSeries(12, 2, 2)    // count

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":     times,
			"hit_rate":   hitRate,
			"status_4xx": status4xx,
			"status_5xx": status5xx,
		},
	})
}

// ListOrigin Retrieves origin statistics (Origin Bandwidth, Origin Traffic)
// GET /api/v1/admin/stats/origin
func (c *StatController) ListOrigin(ctx *gin.Context) {
	times, bandwidth := generateTimeSeries(12, 20, 10) // Mbps
	_, traffic := generateTimeSeries(12, 100, 50)      // MB

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":           times,
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
             t = now.Add(time.Duration(i-count)*time.Hour)
        } else {
             t = now.AddDate(0, 0, i-count)
        }
		times[i] = t.Format(labelFormat)
        
		inTraffic[i] = float64(rand.Intn(1000)) / 10.0 // 0-100 MB
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
