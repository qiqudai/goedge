package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ForwardDefaultController struct{}

type forwardDefaultItem struct {
	ID        int64       `json:"id"`
	IDStr     string      `json:"id_str,omitempty"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Scope     string      `json:"scope"`
	GroupID   int64       `json:"group_id"`
	GroupName string      `json:"group_name"`
}

const forwardDefaultKey = "forward_default_settings"

func (ctrl *ForwardDefaultController) List(c *gin.Context) {
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		items, err := loadForwardDefaultItemsForUser(uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load settings")})
			return
		}
		c.JSON(http.StatusOK, gin.H{"list": items})
		return
	}
	items, err := loadForwardDefaultItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load settings")})
		return
	}

	if len(items) > 0 {
		groupMap, _ := loadForwardGroupMap(items)
		for i := range items {
			items[i].IDStr = strconv.FormatInt(items[i].ID, 10)
			if items[i].GroupID != 0 {
				items[i].GroupName = groupMap[items[i].GroupID]
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"list": items})
}

func (ctrl *ForwardDefaultController) Create(c *gin.Context) {
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		var req struct {
			Key     string      `json:"key"`
			Value   interface{} `json:"value"`
			Scope   string      `json:"scope"`
			GroupID int64       `json:"group_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
			return
		}
		value := encodeForwardDefaultValue(req.Key, req.Value)
		enable := true
		upsertReq := configItemUpsertRequest{
			Type:      "stream_default_config",
			ScopeName: "user",
			ScopeID:   uid,
			Items: []configItemPayload{
				{Name: req.Key, Value: value, Enable: &enable},
			},
		}
		if err := upsertConfigItems(upsertReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("save failed")})
			return
		}
		services.BumpConfigVersion("config_item", []int64{uid})
		c.JSON(http.StatusOK, gin.H{"message": T("created")})
		return
	}
	var req struct {
		Key     string      `json:"key"`
		Value   interface{} `json:"value"`
		Scope   string      `json:"scope"`
		GroupID int64       `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
		return
	}
	items, _ := loadForwardDefaultItems()
	items = append(items, forwardDefaultItem{
		ID:      time.Now().UnixMilli(),
		Key:     req.Key,
		Value:   req.Value,
		Scope:   req.Scope,
		GroupID: req.GroupID,
	})
	if err := saveForwardDefaultItems(items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("save failed")})
		return
	}
	services.BumpConfigVersion("forward_default", []int64{})
	c.JSON(http.StatusOK, gin.H{"message": T("created")})
}

func (ctrl *ForwardDefaultController) Delete(c *gin.Context) {
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		var req struct {
			ID    int64  `json:"id"`
			IDStr string `json:"id_str"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("id is required")})
			return
		}
		key := strings.TrimSpace(req.IDStr)
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("id is required")})
			return
		}
		if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", "stream_default_config", "user", uid, key).Delete(&models.ConfigItem{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("delete failed")})
			return
		}
		services.BumpConfigVersion("config_item", []int64{uid})
		c.JSON(http.StatusOK, gin.H{"message": T("deleted")})
		return
	}
	var req struct {
		ID    int64  `json:"id"`
		IDStr string `json:"id_str"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("id is required")})
		return
	}
	id := req.ID
	if id == 0 && req.IDStr != "" {
		parsed, err := strconv.ParseInt(req.IDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("id is required")})
			return
		}
		id = parsed
	}
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("id is required")})
		return
	}
	items, _ := loadForwardDefaultItems()
	nextItems := make([]forwardDefaultItem, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			nextItems = append(nextItems, item)
		}
	}
	if err := saveForwardDefaultItems(nextItems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("delete failed")})
		return
	}
	services.BumpConfigVersion("forward_default", []int64{})
	c.JSON(http.StatusOK, gin.H{"message": T("deleted")})
}

func loadForwardDefaultItems() ([]forwardDefaultItem, error) {
	var cfg models.SysConfig
	if err := db.DB.Where("name = ? AND type = ?", forwardDefaultKey, "system").First(&cfg).Error; err != nil {
		return []forwardDefaultItem{}, nil
	}
	var items []forwardDefaultItem
	if cfg.Value != "" {
		_ = json.Unmarshal([]byte(cfg.Value), &items)
	}
	return items, nil
}

func saveForwardDefaultItems(items []forwardDefaultItem) error {
	data, _ := json.Marshal(items)
	var cfg models.SysConfig
	query := db.DB.Where("name = ? AND type = ?", forwardDefaultKey, "system")
	if err := query.First(&cfg).Error; err != nil {
		// create
		cfg = models.SysConfig{
			Name:      forwardDefaultKey,
			Value:     string(data),
			Type:      "system",
			ScopeID:   0,
			ScopeName: "global",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return db.DB.Create(&cfg).Error
	}
	// update
	updates := map[string]interface{}{
		"value":     string(data),
		"update_at": time.Now(),
		"enable":    true,
	}
	return db.DB.Model(&models.SysConfig{}).
		Where("name = ? AND type = ?", forwardDefaultKey, "system").
		Updates(updates).Error
}

func loadForwardGroupMap(items []forwardDefaultItem) (map[int64]string, error) {
	groupIDs := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.GroupID != 0 {
			if _, ok := seen[item.GroupID]; !ok {
				seen[item.GroupID] = struct{}{}
				groupIDs = append(groupIDs, item.GroupID)
			}
		}
	}
	result := map[int64]string{}
	if len(groupIDs) == 0 {
		return result, nil
	}
	var groups []models.ForwardGroup
	if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, g := range groups {
		result[g.ID] = g.Name
	}
	return result, nil
}

func loadForwardDefaultItemsForUser(userID int64) ([]forwardDefaultItem, error) {
	var items []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ?", "stream_default_config", "user", userID).Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]forwardDefaultItem, 0, len(items))
	for _, item := range items {
		value := parseForwardDefaultValue(item.Name, item.Value)
		result = append(result, forwardDefaultItem{
			ID:      0,
			IDStr:   item.Name,
			Key:     item.Name,
			Value:   value,
			Scope:   "global",
			GroupID: 0,
		})
	}
	return result, nil
}

func parseForwardDefaultValue(key string, raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	switch strings.TrimSpace(key) {
	case "proxy_protocol":
		return parseBoolValue(trimmed, false)
	case "listen_protocol", "balance_way":
		return trimmed
	default:
		if trimmed == "" {
			return ""
		}
		if val := strings.ToLower(trimmed); val == "true" || val == "false" {
			return parseBoolValue(trimmed, false)
		}
		return trimmed
	}
}

func encodeForwardDefaultValue(key string, value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		raw, _ := json.Marshal(value)
		return strings.TrimSpace(string(raw))
	}
}
