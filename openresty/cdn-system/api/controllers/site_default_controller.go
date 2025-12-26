package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SiteDefaultController struct{}

func (ctr *SiteDefaultController) List(c *gin.Context) {
	userID := int64(0)
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	} else if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			userID = uid
		}
	}
	if userID == 0 && !isUserRequest(c) {
		var items []models.ConfigItem
		if err := db.DB.Where("type = ? AND scope_name = ?", "site_default_config", "user").
			Order("scope_id asc, name asc").Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
			return
		}
		userMap := loadUserNameMap(items)
		list := make([]gin.H, 0, len(items))
		for _, item := range items {
			list = append(list, gin.H{
				"name":       item.Name,
				"value":      item.Value,
				"type":       item.Type,
				"scope_id":   item.ScopeID,
				"scope_name": item.ScopeName,
				"enable":     item.Enable,
				"user_id":    item.ScopeID,
				"user_name":  userMap[item.ScopeID],
			})
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
		return
	}

	if userID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []models.ConfigItem{}}})
		return
	}

	var items []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ?", "site_default_config", "user", userID).
		Order("name asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items}})
}

func (ctr *SiteDefaultController) Create(c *gin.Context) {
	var req struct {
		UserID int64  `json:"user_id"`
		Name   string `json:"name"`
		Value  string `json:"value"`
		Enable bool   `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "name is required"})
		return
	}

	userID := req.UserID
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "user_id is required"})
		return
	}

	var item models.ConfigItem
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", req.Name, "site_default_config", "user", userID)
	if err := query.First(&item).Error; err == nil {
		item.Value = req.Value
		item.Enable = req.Enable
		item.UpdatedAt = time.Now()
		if err := db.DB.Save(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}

	now := time.Now()
	item = models.ConfigItem{
		Name:      req.Name,
		Value:     req.Value,
		Type:      "site_default_config",
		ScopeID:   userID,
		ScopeName: "user",
		Enable:    req.Enable,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Created"})
}

func (ctr *SiteDefaultController) Update(c *gin.Context) {
	name, _ := url.PathUnescape(c.Param("name"))
	name = strings.TrimSpace(name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "name is required"})
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Value  string `json:"value"`
		Enable bool   `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}

	userID := req.UserID
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "user_id is required"})
		return
	}

	updates := map[string]interface{}{
		"value":     req.Value,
		"enable":    req.Enable,
		"update_at": time.Now(),
	}
	if err := db.DB.Model(&models.ConfigItem{}).
		Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", name, "site_default_config", "user", userID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

func (ctr *SiteDefaultController) Delete(c *gin.Context) {
	name, _ := url.PathUnescape(c.Param("name"))
	name = strings.TrimSpace(name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "name is required"})
		return
	}

	userID := int64(0)
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	} else if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			userID = uid
		}
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "user_id is required"})
		return
	}

	if err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", name, "site_default_config", "user", userID).
		Delete(&models.ConfigItem{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Deleted"})
}

func loadUserNameMap(items []models.ConfigItem) map[int64]string {
	ids := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.ScopeID == 0 {
			continue
		}
		if _, ok := seen[item.ScopeID]; ok {
			continue
		}
		seen[item.ScopeID] = struct{}{}
		ids = append(ids, item.ScopeID)
	}
	result := map[int64]string{}
	if len(ids) == 0 {
		return result
	}
	var users []models.User
	if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return result
	}
	for _, user := range users {
		result[user.ID] = user.Name
	}
	return result
}
