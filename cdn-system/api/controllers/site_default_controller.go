package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"fmt"
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
	// Allow querying by scope if parameters are present, regardless of user context (Admin accesses global configs)
	scopeName := strings.TrimSpace(c.Query("scope_name"))
	scopeIDStr := strings.TrimSpace(c.Query("scope_id"))
	if scopeName != "" || scopeIDStr != "" {
		scopeID, _ := strconv.ParseInt(scopeIDStr, 10, 64)
		if scopeName == "" {
			scopeName = "global"
		}
		var items []models.ConfigItem
		
		if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ?", "site_default_config", scopeName, scopeID).
			Order("name asc").Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
			return
		}
		list := make([]gin.H, 0, len(items))
		for _, item := range items {
			list = append(list, gin.H{
				"name":       item.Name,
				"value":      item.Value,
				"type":       item.Type,
				"scope_id":   item.ScopeID,
				"scope_name": item.ScopeName,
				"enable":     item.Enable,
				"user_id":    int64(0), // Simplified for list response
				"user_name":  "",
				"group_name": "",
			})
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
		return
	}
	if userID == 0 && !isUserRequest(c) {
		var items []models.ConfigItem
		if err := db.DB.Where("type = ? AND scope_name IN ? AND scope_id <> ?", "site_default_config", []string{"global", "group", "user"}, 0).
			Order("scope_name asc, scope_id asc, name asc").Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
			return
		}
		groupMap := loadSiteGroupMap(items)
		userMap := loadUserNameMapForDefaults(items, groupMap)
		list := make([]gin.H, 0, len(items))
		for _, item := range items {
			scopeName := item.ScopeName
			if scopeName == "user" {
				scopeName = "global"
			}
			userID := int64(0)
			groupName := ""
			if item.ScopeName == "global" || item.ScopeName == "user" {
				userID = item.ScopeID
			} else if item.ScopeName == "group" {
				if group, ok := groupMap[item.ScopeID]; ok {
					userID = group.UserID
					groupName = group.Name
				}
			}
			list = append(list, gin.H{
				"name":       item.Name,
				"value":      item.Value,
				"type":       item.Type,
				"scope_id":   item.ScopeID,
				"scope_name": scopeName,
				"enable":     item.Enable,
				"user_id":    userID,
				"user_name":  userMap[userID],
				"group_name": groupName,
			})
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
		return
	}

	if userID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []models.ConfigItem{}}})
		return
	}

	var groups []models.SiteGroup
	if err := db.DB.Where("uid = ?", userID).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	groupIDs := make([]int64, 0, len(groups))
	groupMap := map[int64]models.SiteGroup{}
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
		groupMap[g.ID] = g
	}

	var items []models.ConfigItem
	query := db.DB.Where("type = ? AND scope_name IN ? AND scope_id = ?", "site_default_config", []string{"global", "user"}, userID)
	if len(groupIDs) > 0 {
		query = query.Or("type = ? AND scope_name = ? AND scope_id IN ?", "site_default_config", "group", groupIDs)
	}
	if err := query.Order("scope_name asc, scope_id asc, name asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}

	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		groupName := ""
		if item.ScopeName == "group" {
			if group, ok := groupMap[item.ScopeID]; ok {
				groupName = group.Name
			}
		}
		scopeName := item.ScopeName
		if scopeName == "user" {
			scopeName = "global"
		}
		list = append(list, gin.H{
			"name":       item.Name,
			"value":      item.Value,
			"type":       item.Type,
			"scope_id":   item.ScopeID,
			"scope_name": scopeName,
			"enable":     item.Enable,
			"user_id":    userID,
			"user_name":  "",
			"group_name": groupName,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

func (ctr *SiteDefaultController) Create(c *gin.Context) {
	var req struct {
		UserID    int64                  `json:"user_id"`
		Name      string                 `json:"name"`
		Value     string                 `json:"value"`
		ScopeName string                 `json:"scope_name"`
		ScopeID   int64                  `json:"scope_id"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}

	scopeName := strings.TrimSpace(req.ScopeName)
	if scopeName == "" {
		scopeName = "global"
	}
	scopeID := req.ScopeID
	// if scopeName == "global" && scopeID == 0 && isUserRequest(c) {
	// 	scopeID = userID
	// }

	userID := req.UserID
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	}
	if userID == 0 && isUserRequest(c) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
		return
	}

	if scopeName == "group" {
		if scopeID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("scope_id is required")})
			return
		}
		if userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
			return
		}
		if err := ensureSiteGroupOwner(scopeID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid group")})
			return
		}
	}

	// Batch Update Logic
	if len(req.Data) > 0 {
		for k, v := range req.Data {
			name := strings.TrimSpace(k)
			if name == "" {
				continue
			}
			val := ""
			switch v.(type) {
			case string:
				val = v.(string)
			case bool:
				if v.(bool) {
					val = "1"
				} else {
					val = "0"
				}
			case float64:
				val = strconv.FormatFloat(v.(float64), 'f', -1, 64)
			default:
				// Fallback generic
				// Or use fmt.Sprintf("%v", v)
				// For map/slice, maybe JSON stringify? 
				// The frontend usually sends stringified JSON for proxy_cache.
				// But let's be safe.
				val = fmt.Sprintf("%v", v)
			}
			
			// Perform Upsert for this item
			var item models.ConfigItem
			query := db.DB.Where("name = ? AND type = ? AND scope_id = ?", name, "site_default_config", scopeID)
			if scopeName == "global" {
				query = query.Where("scope_name IN ?", []string{"global", "user"})
			} else {
				query = query.Where("scope_name = ?", scopeName)
			}

			if err := query.First(&item).Error; err == nil {
				updates := map[string]interface{}{
					"value":      val,
					"enable":     true,
					"scope_name": scopeName,
					"scope_id":   scopeID,
					"update_at":  time.Now(),
				}
				query.Updates(updates)
			} else {
				now := time.Now()
				newItem := models.ConfigItem{
					Name:      name,
					Value:     val,
					Type:      "site_default_config",
					ScopeID:   scopeID,
					ScopeName: scopeName,
					Enable:    true,
					CreatedAt: now,
					UpdatedAt: now,
				}
				db.DB.Create(&newItem)
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Saved")})
		return
	}

	// Single Item Logic (Existing)
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("name is required")})
		return
	}

	var item models.ConfigItem
	query := db.DB.Where("name = ? AND type = ? AND scope_id = ?", req.Name, "site_default_config", scopeID)
	if scopeName == "global" {
		query = query.Where("scope_name IN ?", []string{"global", "user"})
	} else {
		query = query.Where("scope_name = ?", scopeName)
	}
	if err := query.First(&item).Error; err == nil {
		updates := map[string]interface{}{
			"value":      req.Value,
			"enable":     true,
			"scope_name": scopeName,
			"scope_id":   scopeID,
			"update_at":  time.Now(),
		}
		if err := query.Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}

	now := time.Now()
	item = models.ConfigItem{
		Name:      req.Name,
		Value:     req.Value,
		Type:      "site_default_config",
		ScopeID:   scopeID,
		ScopeName: scopeName,
		Enable:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Created")})
}

func (ctr *SiteDefaultController) Update(c *gin.Context) {
	name, _ := url.PathUnescape(c.Param("name"))
	name = strings.TrimSpace(name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("name is required")})
		return
	}

	var req struct {
		UserID       int64  `json:"user_id"`
		Value        string `json:"value"`
		ScopeName    string `json:"scope_name"`
		ScopeID      int64  `json:"scope_id"`
		OldScopeName string `json:"old_scope_name"`
		OldScopeID   int64  `json:"old_scope_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}

	userID := req.UserID
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
	}
	if userID == 0 && isUserRequest(c) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
		return
	}

	scopeName := strings.TrimSpace(req.ScopeName)
	if scopeName == "" {
		scopeName = "global"
	}
	scopeID := req.ScopeID
	if scopeName == "global" && scopeID == 0 && isUserRequest(c) {
		scopeID = userID
	}
	if scopeName == "group" {
		if scopeID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("scope_id is required")})
			return
		}
		if userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
			return
		}
		if err := ensureSiteGroupOwner(scopeID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid group")})
			return
		}
	}

	lookupScopeName := strings.TrimSpace(req.OldScopeName)
	lookupScopeID := req.OldScopeID
	if lookupScopeName == "" {
		lookupScopeName = scopeName
		lookupScopeID = scopeID
	}

	updates := map[string]interface{}{
		"value":     req.Value,
		"enable":    true,
		"update_at": time.Now(),
		"scope_name": scopeName,
		"scope_id": scopeID,
	}
	query := db.DB.Model(&models.ConfigItem{}).
		Where("name = ? AND type = ? AND scope_id = ?", name, "site_default_config", lookupScopeID)
	if lookupScopeName == "global" {
		query = query.Where("scope_name IN ?", []string{"global", "user"})
	} else {
		query = query.Where("scope_name = ?", lookupScopeName)
	}
	if err := query.Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

func (ctr *SiteDefaultController) Delete(c *gin.Context) {
	name, _ := url.PathUnescape(c.Param("name"))
	name = strings.TrimSpace(name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("name is required")})
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
	if userID == 0 && isUserRequest(c) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
		return
	}

	scopeName := strings.TrimSpace(c.Query("scope_name"))
	scopeID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("scope_id")), 10, 64)
	if scopeName == "" {
		scopeName = "global"
	}
	if scopeName == "global" && scopeID == 0 {
		if isUserRequest(c) {
			scopeID = userID
		}
	}
	if scopeName == "group" && scopeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("scope_id is required")})
		return
	}
	if scopeName == "group" {
		if userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("user_id is required")})
			return
		}
		if err := ensureSiteGroupOwner(scopeID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid group")})
			return
		}
	}

	query := db.DB.Where("name = ? AND type = ? AND scope_id = ?", name, "site_default_config", scopeID)
	if scopeName == "global" {
		query = query.Where("scope_name IN ?", []string{"global", "user"})
	} else {
		query = query.Where("scope_name = ?", scopeName)
	}
	if err := query.Delete(&models.ConfigItem{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Deleted")})
}

type siteGroupMeta struct {
	ID     int64
	UserID int64
	Name   string
}

func loadSiteGroupMap(items []models.ConfigItem) map[int64]siteGroupMeta {
	groupIDs := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.ScopeName != "group" || item.ScopeID == 0 {
			continue
		}
		if _, ok := seen[item.ScopeID]; ok {
			continue
		}
		seen[item.ScopeID] = struct{}{}
		groupIDs = append(groupIDs, item.ScopeID)
	}
	result := map[int64]siteGroupMeta{}
	if len(groupIDs) == 0 {
		return result
	}
	var groups []models.SiteGroup
	if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return result
	}
	for _, group := range groups {
		result[group.ID] = siteGroupMeta{ID: group.ID, UserID: group.UserID, Name: group.Name}
	}
	return result
}

func loadUserNameMapForDefaults(items []models.ConfigItem, groupMap map[int64]siteGroupMeta) map[int64]string {
	ids := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, item := range items {
		var userID int64
		if item.ScopeName == "global" {
			userID = item.ScopeID
		} else if item.ScopeName == "group" {
			if group, ok := groupMap[item.ScopeID]; ok {
				userID = group.UserID
			}
		}
		if userID == 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		ids = append(ids, userID)
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

func ensureSiteGroupOwner(groupID, userID int64) error {
	var group models.SiteGroup
	if err := db.DB.Where("id = ? AND uid = ?", groupID, userID).First(&group).Error; err != nil {
		return err
	}
	return nil
}
