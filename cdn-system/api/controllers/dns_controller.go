package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"log"
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
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid != 0 {
			query = query.Where("uid = ?", uid)
		}
	} else if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to fetch providers")})
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
		{"type": T("aliyun"), "name": T("Aliyun"), "fields": []string{"access_key_id", "access_key_secret"}},
		{"type": T("huawei"), "name": T("Huawei"), "fields": []string{"id", "secret"}},
		{"type": T("dnsla"), "name": T("DNSLA"), "fields": []string{"id", "secret"}},
		{"type": T("dnspod"), "name": T("DNSPod"), "fields": []string{"id", "token"}},
		{"type": T("dnspod_intl"), "name": T("DNSPod Intl"), "fields": []string{"secret_id", "secret_key"}},
		{"type": T("51dns"), "name": T("51DNS"), "fields": []string{"id", "secret"}},
		{"type": T("cloudflare"), "name": T("Cloudflare"), "fields": []string{"email", "api_key"}},
		{"type": T("godaddy"), "name": T("GoDaddy"), "fields": []string{"key", "secret"}},
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Request")})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Name and type are required")})
		return
	}

	if req.Type == "dnspod_intl" {
		var auth struct {
			SecretID  string `json:"secret_id"`
			SecretKey string `json:"secret_key"`
		}
		if err := json.Unmarshal([]byte(req.Credentials), &auth); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials: invalid auth format")})
			return
		}
		if strings.TrimSpace(auth.SecretID) == "" || strings.TrimSpace(auth.SecretKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials: secret_id/secret_key required")})
			return
		}
	}

	// Validate Credentials with Factory
	if _, err := dns.GetProvider(req.Type, req.Credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials")})
		return
	}

	if req.UserID == 0 {
		req.UserID = parseUserID(mustGet(c, "userID"))
	}

	item := models.DNSAPI{
		UserID: req.UserID,
		Name:   req.Name,
		Remark: "",
		Type:   req.Type,
		Auth:   req.Credentials,
	}

	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// UpdateProvider
func (ctr *DnsController) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	var req struct {
		UserID      int64  `json:"user_id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Credentials string `json:"credentials"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Request")})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Name and type are required")})
		return
	}

	if req.Type == "dnspod_intl" {
		var auth struct {
			SecretID  string `json:"secret_id"`
			SecretKey string `json:"secret_key"`
		}
		if err := json.Unmarshal([]byte(req.Credentials), &auth); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials: invalid auth format")})
			return
		}
		if strings.TrimSpace(auth.SecretID) == "" || strings.TrimSpace(auth.SecretKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials: secret_id/secret_key required")})
			return
		}
	}

	// Validate Credentials with Factory
	if _, err := dns.GetProvider(req.Type, req.Credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Credentials")})
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
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update failed")})
			return
		}
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
	}

	updates := map[string]interface{}{
		"name": req.Name,
		"type": req.Type,
		"auth": req.Credentials,
	}
	if err := db.DB.Model(&models.DNSAPI{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update failed")})
		return
	}

	if errs := services.ResyncDNSForProvider(id); len(errs) > 0 {
		log.Printf("[DNS] provider resync failed id=%d err=%s", id, strings.Join(errs, "; "))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// DeleteProvider
func (ctr *DnsController) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}
	var used int64
	if err := db.DB.Model(&models.CnameDomain{}).Where("dns_provider_id = ?", id).Count(&used).Error; err == nil && used > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("DNS provider is in use")})
		return
	}
	if err := db.DB.Delete(&models.DNSAPI{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// TestDNS checks whether the configured DNS provider can fetch records.
func (ctr *DnsController) TestDNS(c *gin.Context) {
	var domain models.CnameDomain
	if err := db.DB.Where("dns_provider_id <> 0").Order("id desc").First(&domain).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("cname domains not configured")})
		return
	}
	if strings.TrimSpace(domain.Domain) == "" {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("cname domains not configured")})
		return
	}
	if domain.DNSProviderID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns provider not configured")})
		return
	}

	var api models.DNSAPI
	if err := db.DB.Where("id = ?", domain.DNSProviderID).First(&api).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns provider not configured")})
		return
	}

	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil || provider == nil {
		if err == nil {
			err = errors.New("dns provider not available")
		}
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if _, err := provider.GetRecords(domain.Domain); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// FixRecords rebuilds line A records and site CNAME records.
func (ctr *DnsController) FixRecords(c *gin.Context) {
	errs := services.RepairDNSRecords()
	if len(errs) > 0 {
		translated := translateDNSErrorList(errs)
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": strings.Join(translated, "; "), "error": strings.Join(translated, "; "), "detail": strings.Join(errs, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// ClearInvalid removes non-line DNS records under line domains.
func (ctr *DnsController) ClearInvalid(c *gin.Context) {
	errs := services.CleanupInvalidDNSRecords()
	if len(errs) > 0 {
		translated := translateDNSErrorList(errs)
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": strings.Join(translated, "; "), "error": strings.Join(translated, "; "), "detail": strings.Join(errs, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

func translateDNSErrorList(errs []string) []string {
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, 0, len(errs))
	seen := map[string]struct{}{}
	for _, item := range errs {
		msg, _ := resolveDNSSyncErrorMessageFromString(item)
		if msg == "" {
			continue
		}
		if _, ok := seen[msg]; ok {
			continue
		}
		seen[msg] = struct{}{}
		out = append(out, msg)
	}
	if len(out) == 0 {
		return errs
	}
	return out
}
