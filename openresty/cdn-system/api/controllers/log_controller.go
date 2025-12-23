package controllers

import (
	"cdn-api/db"
	"net/http"
	"strconv"
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
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  []interface{}{},
			"total": 0,
		},
	})
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
