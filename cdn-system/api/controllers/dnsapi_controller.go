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
	var uid int64
	if isUserRequest(c) {
		uid = parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		query = query.Where("uid = ?", uid)
	} else if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR type LIKE ?", like, like)
	}
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items}})
}

func (ctr *DNSAPIController) Create(c *gin.Context) {
	var req struct {
		UserID int64  `json:"user_id"`
		Name   string `json:"name"`
		Remark string `json:"remark"`
		Type   string `json:"type"`
		Auth   string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Name and type are required")})
		return
	}
	if req.UserID == 0 {
		req.UserID = parseUserID(mustGet(c, "userID"))
	}
	if isUserRequest(c) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (ctr *DNSAPIController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		var count int64
		if err := db.DB.Model(&models.DNSAPI{}).Where("id = ? AND uid = ?", id, uid).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
			return
		}
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
	}
	var req struct {
		Name   string `json:"name"`
		Remark string `json:"remark"`
		Type   string `json:"type"`
		Auth   string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	updates := map[string]interface{}{
		"name": req.Name,
		"des":  req.Remark,
		"type": req.Type,
		"auth": req.Auth,
	}
	if err := db.DB.Model(&models.DNSAPI{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

func (ctr *DNSAPIController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		var count int64
		if err := db.DB.Model(&models.DNSAPI{}).Where("id = ? AND uid = ?", id, uid).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete Failed")})
			return
		}
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
	}
	if err := db.DB.Delete(&models.DNSAPI{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Deleted")})
}

func (ctr *DNSAPIController) Types(c *gin.Context) {
	types := []gin.H{
		{"type": T("cloudflare"), "name": T("Cloudflare"), "fields": []string{"email", "api_key"}},
		{"type": T("aliyun"), "name": T("Aliyun"), "fields": []string{"access_key_id", "access_key_secret"}},
		{"type": T("dnspod"), "name": T("DNSPod.cn"), "fields": []string{"id", "token"}},
		{"type": T("dnspod_intl"), "name": T("DNSPod.com"), "fields": []string{"id", "token"}},
		{"type": T("godaddy"), "name": T("GoDaddy"), "fields": []string{"api_key", "api_secret"}},
		{"type": T("namecom"), "name": T("Name.com"), "fields": []string{"username", "api_token"}},
		{"type": T("namecheap"), "name": T("Namecheap"), "fields": []string{"user", "api_key", "ip"}},
		{"type": T("cloudns"), "name": T("ClouDNS"), "fields": []string{"auth_id", "auth_password"}},
		{"type": T("namesilo"), "name": T("Namesilo"), "fields": []string{"api_key"}},
		{"type": T("jdcloud"), "name": T("JDCloud"), "fields": []string{"access_key", "secret_key"}},
		{"type": T("dnsla"), "name": T("DNS.LA"), "fields": []string{"api_id", "api_pass"}},
		{"type": T("51dns"), "name": T("51DNS"), "fields": []string{"app_id", "app_secret"}},
		{"type": T("huawei"), "name": T("Huawei Cloud"), "fields": []string{"access_key_id", "secret_access_key"}},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"types": types}})
}
