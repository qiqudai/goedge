package controllers

import (
	"cdn-common/i18n"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{}

// Index Retrieves aggregated dashboard data
// GET /api/v1/admin/dashboard
func (c *DashboardController) Index(ctx *gin.Context) {
	user := gin.H{
		"username":   "fctyang666",
		"id":         17,
		"level":      "V0",
		"auth_state": i18n.T("dashboard.auth_unverified"),
		"last_login": "2025-12-20 14:05:03",
		"login_ip":   i18n.T("dashboard.login_ip_sample"),
		"avatar":     "F",
	}

	stats := gin.H{
		"bandwidth_peak": "127.43 Mbps",
		"requests":       i18n.T("dashboard.requests_sample"),
		"traffic":        "563.98 GB",
		"blocked_ips":    i18n.T("dashboard.blocked_ips_zero"),
	}

	times, bandwidth := generateTimeSeries(12, 50, 20)
	_, requests := generateTimeSeries(12, 10000, 5000)
	_, traffic := generateTimeSeries(12, 500, 200)
	_, blocked := generateTimeSeries(12, 50, 30)

	charts := gin.H{
		"x_axis":    times,
		"bandwidth": bandwidth,
		"requests":  requests,
		"traffic":   traffic,
		"blocked":   blocked,
	}

	topDomains := []gin.H{
		{"name": "api.ilumx.cn:443", "count": 21162, "traffic": "5.57 MB"},
		{"name": "api1.acfwcj.cn:443", "count": 17069, "traffic": "13.70 MB"},
		{"name": "api.b1hauw.cn:443", "count": 10980, "traffic": "21.10 MB"},
		{"name": "api2.sdzxhk.cn:443", "count": 9027, "traffic": "6.14 MB"},
		{"name": "api4.sdzxhk.cn:443", "count": 8271, "traffic": "5.45 MB"},
		{"name": "api3.sdzxhk.cn:443", "count": 8057, "traffic": "5.39 MB"},
		{"name": "api.mv2yas.cn:443", "count": 7925, "traffic": "15.24 MB"},
		{"name": "api5.sdzxhk.cn:443", "count": 7808, "traffic": "5.15 MB"},
		{"name": "cl.odqgw.cn:443", "count": 6803, "traffic": "174.26 MB"},
		{"name": "api.js15ak.cn:443", "count": 6583, "traffic": "12.65 MB"},
	}

	announcements := []gin.H{
		{"id": 1, "title": i18n.T("dashboard.notice_maintenance"), "time": "2025-12-21"},
		{"id": 2, "title": i18n.T("dashboard.notice_new_feature"), "time": "2025-12-20"},
	}

	packageInfo := gin.H{
		"name":    i18n.T("dashboard.plan_business"),
		"desc":    i18n.T("dashboard.plan_desc"),
		"percent": 0,
	}

	resources := gin.H{
		"domains":  211,
		"forward":  1,
		"certs":    264,
		"packages": 1,
	}

	ops := gin.H{
		"summary": gin.H{
			"users":    i18n.T("dashboard.no_data"),
			"packages": i18n.T("dashboard.no_data"),
			"recharge": i18n.T("dashboard.no_data"),
		},
	}

	systemStatus := gin.H{
		"master":     true,
		"elastic":    true,
		"agent":      true,
		"checked_at": "2025-12-22 21:24:56",
	}

	license := gin.H{
		"total_nodes":   30,
		"current_nodes": 1,
		"expire_at":     "2224-11-04 16:14:36",
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user":          user,
			"stats":         stats,
			"charts":        charts,
			"top_domains":   topDomains,
			"top_urls":      []gin.H{},
			"top_ips":       []gin.H{},
			"top_countries": []gin.H{},
			"announcements": announcements,
			"package":       packageInfo,
			"resources":     resources,
			"ops":           ops,
			"system_status": systemStatus,
			"license":       license,
		},
	})
}
