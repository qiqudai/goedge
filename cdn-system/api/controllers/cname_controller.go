package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/response"
	"cdn-api/services"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CnameController struct{}

func (c *CnameController) ListDomains(ctx *gin.Context) {
	if err := ensureCnameTable(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to init cname table")})
		return
	}
	var list []models.CnameDomain
	result := db.DB.Find(&list)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to fetch List")})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list}})
}

func (c *CnameController) CreateDomain(ctx *gin.Context) {
	var input struct {
		Domain        string `json:"domain"`
		Note          string `json:"note"`
		DNSProviderID int64  `json:"dns_provider_id"`
	}
	if err := ctx.BindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("Invalid params")})
		return
	}

	if err := ensureCnameTable(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to init cname table")})
		return
	}

	domain := normalizeDomainInput(input.Domain)
	if domain == "" || !isValidDomain(domain) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("Invalid domain")})
		return
	}
	if input.DNSProviderID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("DNS provider is required")})
		return
	}
	if err := db.DB.First(&models.DNSAPI{}, input.DNSProviderID).Error; err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("DNS provider not found")})
		return
	}

	model := models.CnameDomain{
		Domain:        domain,
		DNSProviderID: input.DNSProviderID,
		Note:          input.Note,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.DB.Create(&model).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to create")})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "msg": T("Success")})
}

func (c *CnameController) UpdateDomain(ctx *gin.Context) {
	id := ctx.Param("id")
	var input struct {
		Domain        string `json:"domain"`
		Note          string `json:"note"`
		DNSProviderID int64  `json:"dns_provider_id"`
	}
	if err := ctx.BindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("Invalid params")})
		return
	}

	if err := ensureCnameTable(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to init cname table")})
		return
	}

	var existing models.CnameDomain
	_ = db.DB.Where("id = ?", id).First(&existing).Error

	domain := normalizeDomainInput(input.Domain)
	if domain == "" || !isValidDomain(domain) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("Invalid domain")})
		return
	}
	if input.DNSProviderID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("DNS provider is required")})
		return
	}
	if err := db.DB.First(&models.DNSAPI{}, input.DNSProviderID).Error; err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": response.CodeBadRequest, "msg": T("DNS provider not found")})
		return
	}

	updates := map[string]interface{}{
		"domain":          domain,
		"dns_provider_id": input.DNSProviderID,
		"note":            input.Note,
		"update_at":       time.Now(),
	}

	if err := db.DB.Model(&models.CnameDomain{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to update")})
		return
	}

	domainsToSync := []string{domain}
	oldDomain := strings.TrimSpace(existing.Domain)
	if oldDomain != "" && !strings.EqualFold(oldDomain, domain) {
		domainsToSync = append(domainsToSync, oldDomain)
	}
	if errs := services.ResyncDNSForCnameDomains(domainsToSync); len(errs) > 0 {
		log.Printf("[DNS] cname domain resync failed id=%s domains=%v err=%s", id, domainsToSync, strings.Join(errs, "; "))
	}
	ctx.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "msg": T("Success")})
}

func (c *CnameController) DeleteDomain(ctx *gin.Context) {
	if err := ensureCnameTable(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to init cname table")})
		return
	}
	id := ctx.Param("id")
	var model models.CnameDomain
	if err := db.DB.First(&model, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": response.CodeNotFound, "msg": T("Not Found")})
		return
	}
	inUse, err := isCnameDomainInUse(model.Domain)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to validate cname_domain")})
		return
	}
	if inUse {
		ctx.JSON(http.StatusConflict, gin.H{"code": response.CodeConflict, "msg": T("CNAME domain is in use")})
		return
	}
	if err := db.DB.Delete(&models.CnameDomain{}, id).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "msg": T("Failed to delete")})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "msg": T("Success")})
}

func ensureCnameTable() error {
	if db.DB == nil {
		return nil
	}
	if err := db.DB.Exec(`
CREATE TABLE IF NOT EXISTS cname_domains (
  id INT(11) NOT NULL AUTO_INCREMENT,
  domain VARCHAR(255) NOT NULL,
  dns_provider_id BIGINT NOT NULL DEFAULT 0,
  note VARCHAR(255) DEFAULT '',
  create_at DATETIME DEFAULT NULL,
  update_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_cname_domains_domain (domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`).Error; err != nil {
		return err
	}
	if db.DB.Migrator().HasColumn(&models.CnameDomain{}, "dns_provider_id") {
		return nil
	}
	return db.DB.Migrator().AddColumn(&models.CnameDomain{}, "DNSProviderID")
}

func normalizeDomainInput(input string) string {
	domain := strings.TrimSpace(strings.ToLower(input))
	if strings.HasPrefix(domain, "http://") {
		domain = strings.TrimPrefix(domain, "http://")
	} else if strings.HasPrefix(domain, "https://") {
		domain = strings.TrimPrefix(domain, "https://")
	}
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "#"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "?"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	domain = strings.TrimRight(domain, ".")
	return domain
}

func isValidDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" || len(part) > 63 {
			return false
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return false
		}
		for i := 0; i < len(part); i++ {
			ch := part[i]
			isLower := ch >= 'a' && ch <= 'z'
			isDigit := ch >= '0' && ch <= '9'
			if !(isLower || isDigit || ch == '-') {
				return false
			}
		}
	}
	return true
}

func isCnameDomainInUse(domain string) (bool, error) {
	if db.DB == nil {
		return false, nil
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false, nil
	}
	// Site CNAME rows use (cname_domain=prefix, cname_hostname=root). During
	// the one-time migration also retain the legacy-column check so neither the
	// source nor the target root can be removed while referenced.
	if db.DB.Migrator().HasColumn(&models.Site{}, "cname_hostname") && db.DB.Migrator().HasColumn(&models.Site{}, "cname_domain") {
		var count int64
		if err := db.DB.Model(&models.Site{}).Where("cname_hostname = ? OR cname_domain = ?", domain, domain).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	type refCheck struct {
		model  interface{}
		column string
	}
	checks := []refCheck{
		{&models.Forward{}, "cname_domain"},
		{&models.NodeGroup{}, "cname_domain"},
		{&models.Package{}, "cname_domain"},
		{&models.UserPackage{}, "cname_domain"},
		{&models.Plan{}, "cname_domain"},
	}
	for _, item := range checks {
		if !db.DB.Migrator().HasColumn(item.model, item.column) {
			continue
		}
		var count int64
		if err := db.DB.Model(item.model).Where(item.column+" = ?", domain).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}
