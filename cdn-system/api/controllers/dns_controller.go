package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
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

// DeleteProvider
func (ctr *DnsController) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
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
	var groups []models.NodeGroup
	if err := db.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Database Error")})
		return
	}
	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
		return
	}

	errs := make([]string, 0)
	for _, group := range groups {
		resolvedGroup, err := dns.EnsureGroupDNSConfig(group.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		group = resolvedGroup
		var lines []models.Line
		if err := db.DB.Select("line_id", "line_name", "node_id", "node_ip_id", "enable").
			Where("node_group_id = ?", group.ID).
			Find(&lines).Error; err != nil {
			errs = append(errs, err.Error())
			continue
		}
		lineMap := map[string]*struct {
			Name    string
			NodeIDs []int64
		}{}
		for _, line := range lines {
			if !line.Enable {
				continue
			}
			lineKey := strings.TrimSpace(line.LineID)
			if lineKey == "" {
				lineKey = "default"
			}
			item := lineMap[lineKey]
			if item == nil {
				lineName := strings.TrimSpace(line.LineName)
				if lineName == "" {
					lineName = lineKey
				}
				item = &struct {
					Name    string
					NodeIDs []int64
				}{Name: lineName}
				lineMap[lineKey] = item
			}
			nodeID := line.NodeIPID
			if nodeID == 0 {
				nodeID = line.NodeID
			}
			if nodeID != 0 {
				item.NodeIDs = append(item.NodeIDs, nodeID)
			}
		}

		for lineKey, item := range lineMap {
			ids := uniqueInt64List(item.NodeIDs)
			if err := dns.SyncLineRecords(group.ID, lineKey, item.Name, "resync", ids); err != nil {
				errs = append(errs, err.Error())
			}
			if err := services.SyncPackageCnameForLineChange(group.ID, lineKey, item.Name, ids, "resync"); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": strings.Join(errs, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}

// ClearInvalid removes non-line DNS records under line domains.
func (ctr *DnsController) ClearInvalid(c *gin.Context) {
	var groups []models.NodeGroup
	if err := db.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Database Error")})
		return
	}
	if len(groups) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
		return
	}

	errs := make([]string, 0)
	allowed := map[string]map[string]struct{}{}
	for _, group := range groups {
		resolvedGroup, err := dns.EnsureGroupDNSConfig(group.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		group = resolvedGroup
		domainKey := normalizeDomainInput(group.CnameDomain)
		if domainKey == "" {
			continue
		}
		host := normalizeGroupHostname(group.CnameHostname, domainKey)
		if host == "" {
			continue
		}
		if _, ok := allowed[domainKey]; !ok {
			allowed[domainKey] = map[string]struct{}{}
		}
		allowed[domainKey][host] = struct{}{}
	}
	if len(allowed) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
		return
	}

	domainList := make([]string, 0, len(allowed))
	for domain := range allowed {
		domainList = append(domainList, domain)
	}
	var domainRows []models.CnameDomain
	if err := db.DB.Where("domain IN ?", domainList).Find(&domainRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": T("Database Error")})
		return
	}
	if len(domainRows) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
		return
	}

	apis := map[int64]models.DNSAPI{}
	for _, domain := range domainRows {
		if domain.DNSProviderID == 0 {
			continue
		}
		api, ok := apis[domain.DNSProviderID]
		if !ok {
			if err := db.DB.Where("id = ?", domain.DNSProviderID).First(&api).Error; err != nil {
				errs = append(errs, err.Error())
				continue
			}
			apis[domain.DNSProviderID] = api
		}
		provider, err := dns.GetProvider(api.Type, api.Auth)
		if err != nil || provider == nil {
			if err == nil {
				err = errors.New("dns provider not available")
			}
			errs = append(errs, err.Error())
			continue
		}
		records, err := provider.GetRecords(domain.Domain)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		allowedHosts := allowed[normalizeDomainInput(domain.Domain)]
		for _, record := range records {
			if strings.EqualFold(record.Type, "NS") {
				continue
			}
			if strings.EqualFold(record.Type, "A") {
				if _, ok := allowedHosts[record.Name]; ok {
					continue
				}
			}
			if err := provider.DeleteRecord(domain.Domain, record); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": strings.Join(errs, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success")})
}
