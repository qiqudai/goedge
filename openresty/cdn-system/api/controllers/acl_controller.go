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

func parseInt64(v interface{}) int64 {
	switch t := v.(type) {
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
	return 0
}

func mustGet(c *gin.Context, key string) interface{} {
	if val, ok := c.Get(key); ok {
		return val
	}
	return nil
}

type ACLController struct{}

type ACLCondition struct {
	Item     string `json:"item"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type ACLRule struct {
	Conditions  []ACLCondition `json:"conditions"`
	Action      string         `json:"action"`      // allow, deny
	DenyStatus  int            `json:"deny_status"` // 403
	RedirectURL string         `json:"redirect_url"`
}

type ACLData struct {
	Rules              []ACLRule `json:"rules"`
	DefaultDenyStatus  int       `json:"default_deny_status"`
	DefaultRedirectURL string    `json:"default_redirect_url"`
}

type aclPayload struct {
	Name               string    `json:"name"`
	Description        string    `json:"des"`
	DefaultAction      string    `json:"default_action"`
	Enable             bool      `json:"enable"`
	Rules              []ACLRule `json:"rules"`
	UserID             int64     `json:"user_id"`
	DefaultDenyStatus  int       `json:"default_deny_status"`
	DefaultRedirectURL string    `json:"default_redirect_url"`
}

func (ctr *ACLController) List(c *gin.Context) {
	query := db.DB.Model(&models.ACL{})
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid != 0 {
			query = query.Where("uid = ?", uid)
		}
	}
	if name := strings.TrimSpace(c.Query("name")); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	status := strings.TrimSpace(c.Query("status"))
	if status == "on" {
		query = query.Where("enable = ?", true)
	} else if status == "off" {
		query = query.Where("enable = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load ACL"})
		return
	}

	var items []models.ACL
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load ACL"})
		return
	}

	userMap, _ := loadUsersByIDs(uniqueACLUserIDs(items))
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":             item.ID,
			"user_id":        item.UserID,
			"uid":            item.UserID,
			"user":           gin.H{"username": userMap[item.UserID], "id": item.UserID},
			"name":           item.Name,
			"des":            item.Description,
			"default_action": item.DefaultAction,
			"enable":         item.Enable,
			"create_time":    item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": total}})
}

func (ctr *ACLController) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item models.ACL
	if err := db.DB.Where("id = ?", id).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "acl not found"})
		return
	}
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 || item.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	rules, denyStatus, redirectURL := parseACLData(item.Data)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":                   item.ID,
			"user_id":              item.UserID,
			"name":                 item.Name,
			"des":                  item.Description,
			"default_action":       item.DefaultAction,
			"enable":               item.Enable,
			"rules":                rules,
			"default_deny_status":  denyStatus,
			"default_redirect_url": redirectURL,
		},
	})
}

func (ctr *ACLController) Create(c *gin.Context) {
	uid := int64(0)
	if isUserRequest(c) {
		uid = parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
	}
	
	var req aclPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	
	if !isUserRequest(c) && req.UserID > 0 {
		uid = req.UserID
	}

	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.DefaultAction == "" {
		req.DefaultAction = "allow"
	}
	
	dataObj := ACLData{
		Rules:              req.Rules,
		DefaultDenyStatus:  req.DefaultDenyStatus,
		DefaultRedirectURL: req.DefaultRedirectURL,
	}
	b, _ := json.Marshal(dataObj)
	
	item := models.ACL{
		UserID:        uid,
		Name:          req.Name,
		Description:   req.Description,
		DefaultAction: req.DefaultAction,
		Enable:        req.Enable,
		Data:          string(b),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create"})
		return
	}

	services.BumpConfigVersion("acl", []int64{item.ID})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (ctr *ACLController) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	
	var item models.ACL
	if err := db.DB.Where("id = ?", id).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "acl not found"})
		return
	}

	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		if item.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}
	
	var req aclPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if !isUserRequest(c) && req.UserID > 0 {
		item.UserID = req.UserID
	}
	
	if req.DefaultAction == "" {
		req.DefaultAction = "allow"
	}
	
	dataObj := ACLData{
		Rules:              req.Rules,
		DefaultDenyStatus:  req.DefaultDenyStatus,
		DefaultRedirectURL: req.DefaultRedirectURL,
	}
	b, _ := json.Marshal(dataObj)
	
	item.Name = req.Name
	item.Description = req.Description
	item.DefaultAction = req.DefaultAction
	item.Enable = req.Enable
	item.Data = string(b)
	item.UpdatedAt = time.Now()

	if err := db.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}

	services.BumpConfigVersion("acl", []int64{id})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "updated"})
}

func (ctr *ACLController) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		var item models.ACL
		if err := db.DB.Where("id = ?", id).First(&item).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "acl not found"})
			return
		}
		if item.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}
	if err := db.DB.Delete(&models.ACL{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}

	services.BumpConfigVersion("acl", []int64{id})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

func parseACLData(raw string) ([]ACLRule, int, string) {
	if strings.TrimSpace(raw) == "" {
		return []ACLRule{}, 0, ""
	}
	// Try parsing as ACLData struct
	var data ACLData
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		return data.Rules, data.DefaultDenyStatus, data.DefaultRedirectURL
	}
	
	// Legacy or simple list fallback
	var items []ACLRule
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items, 0, ""
	}
	return []ACLRule{}, 0, ""
}

func uniqueACLUserIDs(items []models.ACL) []int64 {
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.UserID == 0 {
			continue
		}
		seen[item.UserID] = struct{}{}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}
