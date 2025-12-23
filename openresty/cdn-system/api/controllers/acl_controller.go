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

type ACLController struct{}

type aclRuleItem struct {
	IP     string `json:"ip"`
	Action string `json:"action"`
}

type aclPayload struct {
	Name          string        `json:"name"`
	Description   string        `json:"des"`
	DefaultAction string        `json:"default_action"`
	Enable        bool          `json:"enable"`
	Rules         []aclRuleItem `json:"rules"`
}

func (ctr *ACLController) List(c *gin.Context) {
	query := db.DB.Model(&models.ACL{})
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
			"user":           userMap[item.UserID],
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

	rules := parseACLRuleItems(item.Data)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":             item.ID,
			"name":           item.Name,
			"des":            item.Description,
			"default_action": item.DefaultAction,
			"enable":         item.Enable,
			"rules":          rules,
		},
	})
}

func (ctr *ACLController) Create(c *gin.Context) {
	uid := parseInt64(mustGet(c, "userID"))
	var req aclPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.DefaultAction == "" {
		req.DefaultAction = "allow"
	}
	b, _ := json.Marshal(req.Rules)
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
	var req aclPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.DefaultAction == "" {
		req.DefaultAction = "allow"
	}
	b, _ := json.Marshal(req.Rules)
	updates := map[string]interface{}{
		"name":           req.Name,
		"des":            req.Description,
		"default_action": req.DefaultAction,
		"enable":         req.Enable,
		"data":           string(b),
		"update_at":      time.Now(),
	}
	if err := db.DB.Model(&models.ACL{}).Where("id = ?", id).Updates(updates).Error; err != nil {
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
	if err := db.DB.Delete(&models.ACL{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}

	services.BumpConfigVersion("acl", []int64{id})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

func parseACLRuleItems(raw string) []aclRuleItem {
	if strings.TrimSpace(raw) == "" {
		return []aclRuleItem{}
	}
	var items []aclRuleItem
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	return []aclRuleItem{}
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
