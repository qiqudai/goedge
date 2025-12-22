package controllers

import (
	"cdn-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type LogController struct{}

// ListLoginLogs
// GET /api/v1/admin/logs/login
func (ctr *LogController) ListLoginLogs(c *gin.Context) {
	// Mock Data
	logs := []models.UserLoginLog{
		{ID: 1, UserID: 1, Username: "admin", IP: "127.0.0.1", Status: 1, CreatedAt: time.Now()},
		{ID: 2, UserID: 2, Username: "test_user", IP: "192.168.1.5", Status: 0, CreatedAt: time.Now().Add(-1 * time.Hour)},
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": 2,
		},
	})
}

// ListOpLogs
// GET /api/v1/admin/logs/operation
func (ctr *LogController) ListOpLogs(c *gin.Context) {
	// Mock Data
	logs := []models.UserOperationLog{
		{ID: 1, UserID: 1, Username: "admin", Action: "Update Config", Description: "Modified Nginx Settings", IP: "127.0.0.1", CreatedAt: time.Now()},
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": 1,
		},
	})
}

// ListAccessLogs
// GET /api/v1/admin/logs/access
func (ctr *LogController) ListAccessLogs(c *gin.Context) {
	// Mock Data based on user request fields
	type AccessLog struct {
		Time        string  `json:"time"`
		Domain      string  `json:"domain"`
		Port        int     `json:"port"`
		Scheme      string  `json:"scheme"`
		Method      string  `json:"method"`
		URI         string  `json:"uri"`
		Status      int     `json:"status"`
		ClientIP    string  `json:"client_ip"`
		Location    string  `json:"location"`
		Origin      string  `json:"origin"`
		ContentType string  `json:"content_type"`
		Referer     string  `json:"referer"`
		UserAgent   string  `json:"user_agent"`
		OriginTime  float64 `json:"origin_time"`
		Bytes       int     `json:"bytes"`
		CacheHit    string  `json:"cache_hit"` // HIT, MISS
		L1Hit       string  `json:"l1_hit"`
		L2Hit       string  `json:"l2_hit"`
		L2IP        string  `json:"l2_ip"`
		NodeID      int     `json:"node_id"`
	}

	logs := []AccessLog{
		{
			Time:        time.Now().Format("2006-01-02 15:04:05"),
			Domain:      "for-test.cdnfly.cn",
			Port:        80,
			Scheme:      "HTTP/1.1",
			Method:      "GET",
			URI:         "/favicon.ico",
			Status:      403,
			ClientIP:    "121.237.36.26",
			Location:    "中国-江苏省-南京市",
			Origin:      "1.1.1.1:80",
			ContentType: "text/html; charset=UTF-8",
			Referer:     "",
			UserAgent:   "Dalvik/2.1.0 (Linux; U; Android 10; SM-G981B Build/QP1A.190711.020)",
			OriginTime:  0.023,
			Bytes:       3085,
			CacheHit:    "MISS",
			L1Hit:       "MISS",
			L2Hit:       "MISS",
			L2IP:        "",
			NodeID:      1,
		},
		{
			Time:        time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			Domain:      "for-test.cdnfly.cn",
			Port:        80,
			Scheme:      "HTTP/1.1",
			Method:      "GET",
			URI:         "/index.php?model=abc&action=list",
			Status:      200,
			ClientIP:    "114.231.58.200",
			Location:    "中国-江苏省-南通市",
			Origin:      "1.1.1.1:80",
			ContentType: "text/html; charset=UTF-8",
			Referer:     "https://www.google.com",
			UserAgent:   "Mozilla/5.0 (Linux; Android 10; SM-A205U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.101 Mobile Safari/537.36",
			OriginTime:  0.022,
			Bytes:       3067,
			CacheHit:    "MISS",
			L1Hit:       "MISS",
			L2Hit:       "MISS",
			L2IP:        "",
			NodeID:      1,
		},
		{
			Time:        time.Now().Add(-15 * time.Minute).Format("2006-01-02 15:04:05"),
			Domain:      "156.227.1.72-no-config",
			Port:        80,
			Scheme:      "HTTP/1.1",
			Method:      "GET",
			URI:         "/.env",
			Status:      404, // Mock 404
			ClientIP:    "78.153.140.179",
			Location:    "英国-英格兰-伦敦",
			Origin:      "-",
			ContentType: "text/html; charset=UTF-8",
			Referer:     "",
			UserAgent:   "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; Trident/4.0)",
			OriginTime:  0,
			Bytes:       3972,
			CacheHit:    "MISS",
			L1Hit:       "MISS",
			L2Hit:       "MISS",
			L2IP:        "",
			NodeID:      2,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": 3,
		},
	})
}
