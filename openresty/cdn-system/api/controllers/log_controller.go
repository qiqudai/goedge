package controllers

import (
	"cdn-api/db"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LogController struct{}

type LoginLogRow struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id" gorm:"column:uid"`
	Username    string    `json:"username"`
	IP          string    `json:"ip"`
	Success     bool      `json:"success"`
	PostContent string    `json:"post_content"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:create_at"`
}

type OpLogRow struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id" gorm:"column:uid"`
	Type        string    `json:"type"`
	Action      string    `json:"action"`
	Content     string    `json:"content"`
	Diff        string    `json:"diff"`
	IP          string    `json:"ip"`
	Process     string    `json:"process"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:create_at"`
	Username    string    `json:"username"`
	Description string    `json:"description"`
}

type AccessLogRow struct {
	Timestamp            time.Time `json:"timestamp"`
	NodeID               string    `json:"node_id"`
	NodeIP               string    `json:"node_ip"`
	RemoteAddr           string    `json:"remote_addr"`
	Host                 string    `json:"host"`
	Method               string    `json:"method"`
	URI                  string    `json:"uri"`
	Status               int       `json:"status"`
	Bytes                uint64    `json:"bytes"`
	RequestTime          float64   `json:"request_time"`
	UpstreamAddr         string    `json:"upstream_addr"`
	UpstreamResponseTime float64   `json:"upstream_response_time"`
	UpstreamCacheStatus  string    `json:"upstream_cache_status"`
	HTTPReferer          string    `json:"http_referer"`
	HTTPUserAgent        string    `json:"http_user_agent"`
	Scheme               string    `json:"scheme"`
	SSLProtocol          string    `json:"ssl_protocol"`
	SSLCipher            string    `json:"ssl_cipher"`
}

// ListLoginLogs
// GET /api/v1/admin/logs/login
func (ctr *LogController) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := c.Query("keyword")
	query := db.DB.Table("login_log").Select("login_log.*, user.name as username").
		Joins("left join user on user.id = login_log.uid")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("user.name LIKE ? OR login_log.ip LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var logs []LoginLogRow
	if err := query.Order("login_log.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
	})
}

// ListOpLogs
// GET /api/v1/admin/logs/operation
func (ctr *LogController) ListOpLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := c.Query("keyword")
	query := db.DB.Table("op_log").Select("op_log.*, user.name as username, op_log.content as description").
		Joins("left join user on user.id = op_log.uid")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("user.name LIKE ? OR op_log.action LIKE ? OR op_log.content LIKE ? OR op_log.ip LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var logs []OpLogRow
	if err := query.Order("op_log.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
	})
}

// ListAccessLogs
// GET /api/v1/admin/logs/access
func (ctr *LogController) ListAccessLogs(c *gin.Context) {
	if !db.ClickHouseEnabled() {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)

	if domain := strings.TrimSpace(c.Query("domain")); domain != "" {
		if strings.TrimSpace(c.Query("domain_mode")) == "fuzzy" {
			conditions = append(conditions, "host LIKE ?")
			args = append(args, "%"+domain+"%")
		} else {
			conditions = append(conditions, "host = ?")
			args = append(args, domain)
		}
	}
	if clientIP := strings.TrimSpace(c.Query("client_ip")); clientIP != "" {
		conditions = append(conditions, "remote_addr = ?")
		args = append(args, clientIP)
	}
	if uri := strings.TrimSpace(c.Query("uri")); uri != "" {
		if c.Query("uri_mode") == "exact" {
			conditions = append(conditions, "uri = ?")
			args = append(args, uri)
		} else {
			conditions = append(conditions, "uri LIKE ?")
			args = append(args, "%"+uri+"%")
		}
	}
	if method := strings.TrimSpace(c.Query("method")); method != "" {
		conditions = append(conditions, "method = ?")
		args = append(args, strings.ToUpper(method))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if code, err := strconv.Atoi(status); err == nil {
			conditions = append(conditions, "status = ?")
			args = append(args, code)
		}
	}
	if statusMin := strings.TrimSpace(c.Query("status_min")); statusMin != "" {
		if code, err := strconv.Atoi(statusMin); err == nil {
			conditions = append(conditions, "status >= ?")
			args = append(args, code)
		}
	}
	if statusMax := strings.TrimSpace(c.Query("status_max")); statusMax != "" {
		if code, err := strconv.Atoi(statusMax); err == nil {
			conditions = append(conditions, "status <= ?")
			args = append(args, code)
		}
	}
	if nodeID := strings.TrimSpace(c.Query("node_id")); nodeID != "" {
		conditions = append(conditions, "node_id = ?")
		args = append(args, nodeID)
	}
	if nodeIP := strings.TrimSpace(c.Query("node_ip")); nodeIP != "" {
		conditions = append(conditions, "node_ip = ?")
		args = append(args, nodeIP)
	}
	if port := strings.TrimSpace(c.Query("port")); port != "" {
		conditions = append(conditions, "host LIKE ?")
		args = append(args, "%:"+port)
	}
	if scheme := strings.TrimSpace(c.Query("scheme")); scheme != "" {
		conditions = append(conditions, "scheme = ?")
		args = append(args, scheme)
	}
	if cacheStatus := strings.TrimSpace(c.Query("cache_status")); cacheStatus != "" {
		conditions = append(conditions, "upstream_cache_status = ?")
		args = append(args, cacheStatus)
	}
	if referer := strings.TrimSpace(c.Query("referer")); referer != "" {
		conditions = append(conditions, "http_referer LIKE ?")
		args = append(args, "%"+referer+"%")
	}
	if userAgent := strings.TrimSpace(c.Query("user_agent")); userAgent != "" {
		conditions = append(conditions, "http_user_agent LIKE ?")
		args = append(args, "%"+userAgent+"%")
	}
	if upstreamAddr := strings.TrimSpace(c.Query("upstream_addr")); upstreamAddr != "" {
		conditions = append(conditions, "upstream_addr LIKE ?")
		args = append(args, "%"+upstreamAddr+"%")
	}
	if sslProtocol := strings.TrimSpace(c.Query("ssl_protocol")); sslProtocol != "" {
		conditions = append(conditions, "ssl_protocol = ?")
		args = append(args, sslProtocol)
	}
	if sslCipher := strings.TrimSpace(c.Query("ssl_cipher")); sslCipher != "" {
		conditions = append(conditions, "ssl_cipher = ?")
		args = append(args, sslCipher)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		conditions = append(conditions, "(host LIKE ? OR uri LIKE ? OR remote_addr LIKE ?)")
		args = append(args, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	startTime, endTime := parseTimeRange(c)
	if !startTime.IsZero() && !endTime.IsZero() {
		conditions = append(conditions, "ts BETWEEN ? AND ?")
		args = append(args, startTime, endTime)
	}

	whereSQL := strings.Join(conditions, " AND ")
	countSQL := fmt.Sprintf("SELECT count() FROM node_access_logs WHERE %s", whereSQL)
	var total int64
	if err := db.CK.QueryRow(countSQL, args...).Scan(&total); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	querySQL := fmt.Sprintf(`SELECT ts, node_id, node_ip, remote_addr, host, method, uri, status, bytes,
		request_time, upstream_addr, upstream_response_time, upstream_cache_status, http_referer, http_user_agent,
		scheme, ssl_protocol, ssl_cipher
		FROM node_access_logs WHERE %s ORDER BY ts DESC LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.CK.Query(querySQL, args...)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}
	defer rows.Close()

	list := make([]AccessLogRow, 0)
	for rows.Next() {
		var row AccessLogRow
		if err := rows.Scan(
			&row.Timestamp,
			&row.NodeID,
			&row.NodeIP,
			&row.RemoteAddr,
			&row.Host,
			&row.Method,
			&row.URI,
			&row.Status,
			&row.Bytes,
			&row.RequestTime,
			&row.UpstreamAddr,
			&row.UpstreamResponseTime,
			&row.UpstreamCacheStatus,
			&row.HTTPReferer,
			&row.HTTPUserAgent,
			&row.Scheme,
			&row.SSLProtocol,
			&row.SSLCipher,
		); err == nil {
			list = append(list, row)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

func parseTimeRange(c *gin.Context) (time.Time, time.Time) {
	layout := "2006-01-02 15:04:05"
	values := c.QueryArray("timeRange[]")
	if len(values) == 0 {
		values = c.QueryArray("timeRange")
	}
	if len(values) >= 2 {
		start, _ := time.Parse(layout, values[0])
		end, _ := time.Parse(layout, values[1])
		return start, end
	}
	startRaw := c.Query("start_time")
	endRaw := c.Query("end_time")
	if startRaw != "" && endRaw != "" {
		start, _ := time.Parse(layout, startRaw)
		end, _ := time.Parse(layout, endRaw)
		return start, end
	}
	return time.Time{}, time.Time{}
}

// ListOpLogsUser
// GET /api/v1/user/logs/operation
func (ctr *LogController) ListOpLogsUser(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := c.Query("keyword")
	query := db.DB.Table("op_log").Where("uid = ?", userID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("action LIKE ? OR content LIKE ? OR ip LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var logs []OpLogRow
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
	})
}
