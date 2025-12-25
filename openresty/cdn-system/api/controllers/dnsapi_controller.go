package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DNSAPIController struct{}

func (ctr *DNSAPIController) List(c *gin.Context) {
	var items []models.DNSAPI
	query := db.DB.Model(&models.DNSAPI{})
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR type LIKE ?", like, like)
	}
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items}})
}

func (ctr *DNSAPIController) Create(c *gin.Context) {
	var req struct {
		UserID    int64  `json:"user_id"`
		Name      string `json:"name"`
		Remark    string `json:"remark"`
		Type      string `json:"type"`
		Auth      string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Name and type are required"})
		return
	}
	if req.UserID == 0 {
		req.UserID = parseUserID(mustGet(c, "userID"))
	}
	var item models.DNSAPI
	item = models.DNSAPI{
		UserID: req.UserID,
		Name:   req.Name,
		Remark: req.Remark,
		Type:   req.Type,
		Auth:   req.Auth,
	}
	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (ctr *DNSAPIController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}
	var req struct {
		Name      string `json:"name"`
		Remark    string `json:"remark"`
		Type      string `json:"type"`
		Auth      string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}
	updates := map[string]interface{}{
		"name": req.Name,
		"des":  req.Remark,
		"type": req.Type,
		"auth": req.Auth,
	}
	if err := db.DB.Model(&models.DNSAPI{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

func (ctr *DNSAPIController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}
	if err := db.DB.Delete(&models.DNSAPI{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Deleted"})
}

func (ctr *DNSAPIController) Types(c *gin.Context) {
	types := []gin.H{
		{"type": "cloudflare", "name": "Cloudflare", "fields": []string{"email", "key"}},
		{"type": "aliyun", "name": "Aliyun", "fields": []string{"id", "secret"}},
		{"type": "dnspod", "name": "DNSPod", "fields": []string{"id", "token"}},
		{"type": "godaddy", "name": "GoDaddy", "fields": []string{"key", "secret"}},
		{"type": "huawei", "name": "Huawei", "fields": []string{"id", "secret"}},
		{"type": "dnsla", "name": "DNSLA", "fields": []string{"id", "secret"}},
		{"type": "51dns", "name": "51DNS", "fields": []string{"id", "secret"}},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"types": types}})
}
