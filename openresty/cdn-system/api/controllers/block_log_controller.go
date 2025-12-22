package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type BlockLogController struct{}

// ListCurrent Retrieves current blocked IPs
// GET /api/v1/admin/logs/block/current
func (c *BlockLogController) ListCurrent(ctx *gin.Context) {
	type BlockedItem struct {
		ID          int    `json:"id"`
		SiteID      int    `json:"site_id"`
		Domain      string `json:"domain"`
		IP          string `json:"ip"`
		Location    string `json:"location"`
		Filter      string `json:"filter"`
		BlockTime   string `json:"block_time"`
		ReleaseTime string `json:"release_time"`
	}

	// Mock Data
	list := []BlockedItem{
		{1, 30073, "example.com", "211.90.251.15", "中国_浙江省", "区域屏蔽", time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05"), time.Now().Add(23 * time.Hour).Format("2006-01-02 15:04:05")},
		{2, 30073, "test.com", "36.49.228.76", "中国_吉林省", "WAF规则", time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04:05"), "永久"},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": 2,
		},
	})
}

// ListStats Retrieves block statistics
// GET /api/v1/admin/logs/block/stats
func (c *BlockLogController) ListStats(ctx *gin.Context) {
	type StatItem struct {
		SiteID int `json:"site_id"`
		Count  int `json:"count"`
	}

	// Mock Data
	list := []StatItem{
		{30073, 150},
		{30074, 89},
		{30075, 12},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": 3,
		},
	})
}

// ListHistory Retrieves history of blocked IPs
// GET /api/v1/admin/logs/block/history
func (c *BlockLogController) ListHistory(ctx *gin.Context) {
	type HistoryItem struct {
		ID        int    `json:"id"`
		SiteID    int    `json:"site_id"`
		Domain    string `json:"domain"`
		IP        string `json:"ip"`
		Location  string `json:"location"`
		Filter    string `json:"filter"`
		BlockTime string `json:"block_time"`
		IsManual  bool   `json:"is_manual"`
	}

	// Mock Data
	list := []HistoryItem{
		{1, 30073, "example.com", "211.90.251.15", "中国_浙江省", "区域屏蔽", "2025-12-22 08:15:33", false},
		{2, 30073, "example.com", "36.49.228.76", "中国_吉林省", "区域屏蔽", "2025-12-22 08:15:33", false},
		{3, 30073, "example.com", "112.229.182.52", "中国_山东省", "CC防御", "2025-12-22 08:15:33", true},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": 3,
		},
	})
}
