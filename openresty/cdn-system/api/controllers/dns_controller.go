package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"net/http"
	"strconv"
	"strings"

	_ "cdn-api/services/dns/providers"

	"github.com/gin-gonic/gin"
)

type DnsController struct{}

// ListProviders
func (ctr *DnsController) ListProviders(c *gin.Context) {
	var list []models.DNSAPI
	query := db.DB.Model(&models.DNSAPI{})
	if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to fetch providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": list,
		},
	})
}

// GetProviderTypes
func (ctr *DnsController) GetProviderTypes(c *gin.Context) {
	types := []gin.H{
		{"type": "aliyun", "name": "Aliyun", "fields": []string{"access_key_id", "access_key_secret"}},
		{"type": "huawei", "name": "Huawei", "fields": []string{"id", "secret"}},
		{"type": "dnsla", "name": "DNSLA", "fields": []string{"id", "secret"}},
		{"type": "dnspod", "name": "DNSPod", "fields": []string{"id", "token"}},
		{"type": "dnspod_intl", "name": "DNSPod Intl", "fields": []string{"id", "token"}},
		{"type": "51dns", "name": "51DNS", "fields": []string{"id", "secret"}},
		{"type": "cloudflare", "name": "Cloudflare", "fields": []string{"email", "api_key"}},
		{"type": "godaddy", "name": "GoDaddy", "fields": []string{"key", "secret"}},
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"types": types,
		},
	})
}

// CreateProvider
func (ctr *DnsController) CreateProvider(c *gin.Context) {
	var req struct {
		UserID      int64  `json:"user_id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Credentials string `json:"credentials"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Request"})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Name and type are required"})
		return
	}

	// Validate Credentials with Factory
	if _, err := dns.GetProvider(req.Type, req.Credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Credentials: " + err.Error()})
		return
	}

	if req.UserID == 0 {
		req.UserID = parseUserID(mustGet(c, "userID"))
	}

	item := models.DNSAPI{
		UserID:    req.UserID,
		Name:      req.Name,
		Remark:    "",
		Type:      req.Type,
		Auth:      req.Credentials,
	}

	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

// DeleteProvider
func (ctr *DnsController) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}
	if err := db.DB.Delete(&models.DNSAPI{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}
