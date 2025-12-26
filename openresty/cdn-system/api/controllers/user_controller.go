package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/utils"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct{}

// ListUsers returns paginated user list
// GET /api/v1/admin/users?page=1&size=20
func (ctr *UserController) ListUsers(c *gin.Context) {
	if err := db.Ensure(); err != nil {
		db.Init()
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var users []models.User
	query := db.DB.Model(&models.User{})
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ? OR email LIKE ? OR phone LIKE ? OR qq LIKE ? OR des LIKE ?",
			keywordLike, keywordLike, keywordLike, keywordLike, keywordLike)
		if id, err := strconv.ParseInt(strings.TrimSpace(keyword), 10, 64); err == nil {
			query = query.Or("id = ?", id)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"list":  users,
			"total": total,
		},
	})
}

// ToggleStatus enables or disables a user
// PUT /api/v1/admin/users/:id/status
func (ctr *UserController) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid params"})
		return
	}

	enabled := req.Status == 1
	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("enable", enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "User " + id + " status updated to " + strconv.Itoa(req.Status)})
}

// DeleteUser removes a user
// DELETE /api/v1/admin/users/:id
func (ctr *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Delete Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "User " + id + " deleted"})
}

func getContextUserID(c *gin.Context) int64 {
	if val, ok := c.Get("userID"); ok {
		switch t := val.(type) {
		case float64:
			return int64(t)
		case int:
			return int64(t)
		case int64:
			return t
		case string:
			if i, err := strconv.ParseInt(t, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// ResetPurgeUsage resets purge/preheat usage for a user
// POST /api/v1/admin/users/:id/purge/reset
func (ctr *UserController) ResetPurgeUsage(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id"})
		return
	}
	payload := map[string]interface{}{
		"date":        time.Now().Format("2006-01-02"),
		"refresh_url": 0,
		"refresh_dir": 0,
		"preheat":     0,
	}
	raw, _ := json.Marshal(payload)
	var cfg models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "purge_usage", "user", "user", userID)
	if err := query.First(&cfg).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Reset failed"})
			return
		}
		cfg = models.SysConfig{
			Name:      "purge_usage",
			Value:     string(raw),
			Type:      "user",
			ScopeID:   int(userID),
			ScopeName: "user",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.DB.Create(&cfg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Reset failed"})
			return
		}
	} else {
		cfg.Value = string(raw)
		cfg.UpdatedAt = time.Now()
		if err := db.DB.Save(&cfg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Reset failed"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Reset success"})
}

// Impersonate generates a token for the target user (admin only)
// POST /api/v1/admin/users/:id/impersonate
func (ctr *UserController) Impersonate(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid user id"})
		return
	}
	var user models.User
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "User not found"})
		return
	}
	if !user.Enable {
		c.JSON(http.StatusForbidden, gin.H{"msg": "User disabled"})
		return
	}

	role := "user"
	if user.Type == 1 {
		role = "admin"
	}
	token, err := utils.GenerateToken(user.ID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"role":  role,
		"uid":   user.ID,
		"name":  user.Name,
	})
}
