package controllers

import (
	"cdn-api/config"
	"cdn-api/db"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListAccessLogsUsesHTTPClickHouse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if strings.Contains(query, "SELECT count()") {
			_, _ = w.Write([]byte(`{"count()":1}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"ts":"2026-07-16 01:00:00","node_id":"79","node_ip":"82.41.66.65","remote_addr":"203.0.113.8","host":"aws.cl.tomhcq.cn","method":"GET","uri":"/asset.png","status":200,"bytes":42,"request_time":0.01,"upstream_addr":"10.0.0.1","upstream_connect_time":0.001,"upstream_header_time":0.002,"upstream_response_time":0.003,"upstream_cache_status":"HIT","slow_reason":"正常","slow_advice":"","http_referer":"-","http_user_agent":"test","scheme":"https","ssl_protocol":"TLSv1.3","ssl_cipher":"cipher"}` + "\n"))
	}))
	defer server.Close()

	oldEnabled, oldDSN, oldCK := config.App.ClickHouseEnabled, config.App.ClickHouseDSN, db.CK
	config.App.ClickHouseEnabled = true
	config.App.ClickHouseDSN = server.URL + "/cdn_logs"
	db.CK = nil
	t.Cleanup(func() {
		config.App.ClickHouseEnabled = oldEnabled
		config.App.ClickHouseDSN = oldDSN
		db.CK = oldCK
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs/access?domain=aws.cl.tomhcq.cn", nil)

	(&LogController{}).ListAccessLogs(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"total":1`) || !strings.Contains(w.Body.String(), `"host":"aws.cl.tomhcq.cn"`) {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"upstream_addr":"10.0.0.1"`) {
		t.Fatalf("admin response must redact upstream address for non-spider traffic: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"timestamp":"2026-07-16T01:00:00Z"`) {
		t.Fatalf("ClickHouse timestamp was not converted to the response field: %s", w.Body.String())
	}
}
