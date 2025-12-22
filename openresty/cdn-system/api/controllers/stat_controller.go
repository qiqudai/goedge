package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type StatController struct{}

// ListRanking Retrieves data ranking
// GET /api/v1/admin/stats/ranking
func (c *StatController) ListRanking(ctx *gin.Context) {
	rankType := ctx.DefaultQuery("type", "domain")
	// timeRange := ctx.DefaultQuery("time_range", "10min")
	// keyword := ctx.Query("keyword")

	type RankItem struct {
		Rank          int    `json:"rank"`
		Item          string `json:"item"` // Domain, URL, IP, etc.
		RequestCount  int    `json:"request_count"`
		OutTraffic    string `json:"out_traffic"`
		OriginTraffic string `json:"origin_traffic"`
	}

	var list []RankItem

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
			{1, "中国", 80000, "5.0 GB", "1.2 GB"},
			{2, "美国", 5000, "200 MB", "50 MB"},
			{3, "日本", 2000, "100 MB", "20 MB"},
		}
	case "province":
		list = []RankItem{
			{1, "浙江省", 20000, "1.5 GB", "500 MB"},
			{2, "广东省", 15000, "1.2 GB", "400 MB"},
			{3, "北京市", 10000, "900 MB", "300 MB"},
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
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
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
