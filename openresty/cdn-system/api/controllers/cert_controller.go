package controllers

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CertController struct{}

// List returns the list of certificates for the current user
func (ctrl *CertController) List(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := parseUserID(userID)
	result, err := queryCerts(c, &uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch certificates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result.Certs, "total": result.Total})
}

// Upload handles certificate upload
func (ctrl *CertController) Upload(c *gin.Context) {
	certModel, err := buildCertFromRequest(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DB.Create(certModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save certificate"})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(certModel.ID)})

	c.JSON(http.StatusOK, gin.H{"message": "Certificate uploaded successfully", "data": certModel})
}

// AdminList returns all certificates for admin
func (ctrl *CertController) AdminList(c *gin.Context) {
	result, err := queryCerts(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch certificates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": result.Certs, "total": result.Total})
}

func (ctrl *CertController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	certModel, err := buildCertFromRequest(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	certModel.ID = id
	certModel.UpdateAt = time.Now()
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        certModel.Name,
		"des":         certModel.Description,
		"type":        certModel.Type,
		"domain":      certModel.Domain,
		"dnsapi":      certModel.DNSAPI,
		"cert":        certModel.Cert,
		"key":         certModel.Key,
		"start_time":  certModel.StartTime,
		"expire_time": certModel.ExpireTime,
		"auto_renew":  certModel.AutoRenew,
		"update_at":   certModel.UpdateAt,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update certificate"})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(id)})

	c.JSON(http.StatusOK, gin.H{"message": "Certificate updated successfully"})
}

func (ctrl *CertController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := db.DB.Delete(&models.Cert{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete certificate"})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(id)})

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (ctrl *CertController) BatchAction(c *gin.Context) {
	var req struct {
		Action string  `json:"action"`
		IDs    []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}

	switch strings.ToLower(req.Action) {
	case "enable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Update("enable", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	case "disable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Update("enable", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	case "force_disable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable":     false,
			"auto_renew": false,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	case "delete":
		if err := db.DB.Where("id IN ?", req.IDs).Delete(&models.Cert{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown action"})
		return
	}

	services.BumpConfigVersion("cert", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": "Action completed"})
}

func (ctrl *CertController) BatchCreate(c *gin.Context) {
	var req struct {
		UserID    int64  `json:"user_id"`
		Type      string `json:"type"`
		DNSAPI    int    `json:"dnsapi"`
		AutoRenew bool   `json:"auto_renew"`
		Domains   string `json:"domains"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if strings.TrimSpace(req.Domains) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domains is required"})
		return
	}

	userID := req.UserID
	if userID == 0 {
		userID = int64(parseUserID(mustGet(c, "userID")))
	}

	typeName := normalizeCertType(req.Type)
	if typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	domains := splitLines(req.Domains)
	if len(domains) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domains is required"})
		return
	}

	now := time.Now()
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, domain := range domains {
			cert := models.Cert{
				UserID:     int(userID),
				Name:       defaultCertName(domain),
				Type:       typeName,
				Domain:     domain,
				DNSAPI:     req.DNSAPI,
				AutoRenew:  req.AutoRenew,
				Enable:     true,
				CreateAt:   now,
				UpdateAt:   now,
				StartTime:  nil,
				ExpireTime: nil,
			}
			if err := tx.Create(&cert).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create certificates"})
		return
	}

	services.BumpConfigVersion("cert", []int64{})

	c.JSON(http.StatusOK, gin.H{"message": "Batch created", "created": len(domains)})
}

func (ctrl *CertController) Reissue(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}
	if strings.TrimSpace(config.App.AcmeEmail) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "acme_email is required"})
		return
	}
	services.IssueCertsAsync(req.IDs)
	c.JSON(http.StatusOK, gin.H{"message": "Reissue submitted"})
}

func (ctrl *CertController) Download(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var cert models.Cert
	if err := db.DB.Where("id = ?", id).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}
	content := cert.Cert + "\n" + cert.Key + "\n"
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=cert_"+strconv.Itoa(cert.ID)+".pem")
	c.Writer.Write([]byte(content))
}

type certListResult struct {
	Certs []models.Cert
	Total int64
}

type certDefaultSettings struct {
	Type   string `json:"type"`
	DNSAPI int    `json:"dnsapi"`
}

func isUserRequest(c *gin.Context) bool {
	path := c.FullPath()
	if strings.HasPrefix(path, "/api/v1/user/") {
		return true
	}
	return strings.HasPrefix(c.Request.URL.Path, "/api/v1/user/")
}

func loadCertDefaultSettings(scopeType, scopeName string, scopeID int) (*certDefaultSettings, error) {
	var sys models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "cert_default_settings", scopeType, scopeName, scopeID)
	if err := query.First(&sys).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var settings certDefaultSettings
	if err := json.Unmarshal([]byte(sys.Value), &settings); err != nil {
		return nil, err
	}
	if settings.Type == "" {
		settings.Type = "system"
	}
	return &settings, nil
}

func queryCerts(c *gin.Context, userID *int64) (*certListResult, error) {
	query := db.DB.Model(&models.Cert{})
	if userID != nil && *userID != 0 {
		query = query.Where("uid = ?", *userID)
	}
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	searchField := strings.TrimSpace(c.DefaultQuery("search_field", "all"))
	if keyword != "" {
		like := "%" + keyword + "%"
		switch searchField {
		case "name":
			query = query.Where("name LIKE ?", like)
		case "domain":
			query = query.Where("domain LIKE ?", like)
		case "type":
			query = query.Where("type LIKE ?", like)
		default:
			query = query.Where("name LIKE ? OR domain LIKE ? OR type LIKE ?", like, like, like)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var certs []models.Cert
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&certs).Error; err != nil {
		return nil, err
	}
	return &certListResult{Certs: certs, Total: total}, nil
}

func (ctrl *CertController) GetDefaultSettings(c *gin.Context) {
	var targetUserID int64
	if uidStr := strings.TrimSpace(c.Query("user_id")); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			targetUserID = uid
		}
	}
	if targetUserID == 0 && isUserRequest(c) {
		targetUserID = parseUserID(mustGet(c, "userID"))
	}

	if targetUserID != 0 {
		if settings, err := loadCertDefaultSettings("user", "user", int(targetUserID)); err == nil && settings != nil {
			c.JSON(http.StatusOK, gin.H{"data": settings})
			return
		}
	}
	if settings, err := loadCertDefaultSettings("system", "global", 0); err == nil && settings != nil {
		c.JSON(http.StatusOK, gin.H{"data": settings})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": certDefaultSettings{Type: "system", DNSAPI: 0}})
}

func (ctrl *CertController) UpdateDefaultSettings(c *gin.Context) {
	var req struct {
		UserID int64 `json:"user_id"`
		certDefaultSettings
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.Type == "" {
		req.Type = "system"
	}
	targetUserID := req.UserID
	if targetUserID == 0 && isUserRequest(c) {
		targetUserID = parseUserID(mustGet(c, "userID"))
	}

	scopeType := "system"
	scopeName := "global"
	scopeID := 0
	if targetUserID != 0 {
		scopeType = "user"
		scopeName = "user"
		scopeID = int(targetUserID)
	}

	payload := certDefaultSettings{Type: req.Type, DNSAPI: req.DNSAPI}
	b, _ := json.Marshal(payload)
	var sys models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "cert_default_settings", scopeType, scopeName, scopeID)
	if err := query.First(&sys).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
			return
		}
		sys = models.SysConfig{
			Name:      "cert_default_settings",
			Value:     string(b),
			Type:      scopeType,
			ScopeID:   scopeID,
			ScopeName: scopeName,
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.DB.Create(&sys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
			return
		}
	} else {
		sys.Value = string(b)
		sys.UpdatedAt = time.Now()
		if err := db.DB.Save(&sys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Updated"})
}

func buildCertFromRequest(c *gin.Context, allowUserID bool) (*models.Cert, error) {
	var input struct {
		UserID    int64  `json:"user_id"`
		Name      string `json:"name"`
		Des       string `json:"des"`
		Type      string `json:"type"`
		Domain    string `json:"domain"`
		DNSAPI    int    `json:"dnsapi"`
		Cert      string `json:"cert"`
		Key       string `json:"key"`
		AutoRenew bool   `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return nil, errors.New("invalid request")
	}

	typeName := normalizeCertType(input.Type)
	if typeName == "" {
		typeName = "upload"
	}

	userID := input.UserID
	if !allowUserID || userID == 0 {
		userID = int64(parseUserID(mustGet(c, "userID")))
	}
	if userID == 0 {
		return nil, errors.New("user_id is required")
	}

	certModel := &models.Cert{
		UserID:      int(userID),
		Name:        strings.TrimSpace(input.Name),
		Description: input.Des,
		Type:        typeName,
		Domain:      strings.TrimSpace(input.Domain),
		DNSAPI:      input.DNSAPI,
		Cert:        input.Cert,
		Key:         input.Key,
		AutoRenew:   input.AutoRenew,
		Enable:      true,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
	}

	if typeName == "upload" {
		if strings.TrimSpace(input.Cert) == "" || strings.TrimSpace(input.Key) == "" {
			return nil, errors.New("cert and key are required for upload")
		}
		domains, notBefore, notAfter, err := parseCert(input.Cert)
		if err != nil {
			return nil, err
		}
		if certModel.Domain == "" {
			certModel.Domain = strings.Join(domains, ",")
		}
		certModel.StartTime = &notBefore
		certModel.ExpireTime = &notAfter
		if certModel.Name == "" {
			certModel.Name = defaultCertName(domains[0])
		}
	} else {
		if certModel.Domain == "" {
			return nil, errors.New("domain is required")
		}
		if certModel.Name == "" {
			certModel.Name = defaultCertName(strings.Split(certModel.Domain, ",")[0])
		}
	}

	return certModel, nil
}

func parseCert(certPEM string) ([]string, time.Time, time.Time, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, time.Time{}, time.Time{}, errors.New("invalid PEM certificate")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, time.Time{}, time.Time{}, errors.New("failed to parse certificate: " + err.Error())
	}
	domains := []string{}
	if x509Cert.Subject.CommonName != "" {
		domains = append(domains, x509Cert.Subject.CommonName)
	}
	domains = append(domains, x509Cert.DNSNames...)
	domains = uniqueStrings(domains)
	return domains, x509Cert.NotBefore, x509Cert.NotAfter, nil
}

func uniqueStrings(in []string) []string {
	set := map[string]struct{}{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			set[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func defaultCertName(domain string) string {
	if domain == "" {
		return "免费证书"
	}
	return domain + "免费证书"
}

func normalizeCertType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "upload", "self":
		return "upload"
	case "zerossl":
		return "zerossl"
	case "letsencrypt", "let's encrypt", "lets encrypt":
		return "letsencrypt"
	case "buypass":
		return "buypass"
	case "google":
		return "google"
	}
	return value
}

func parseUserID(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			return id
		}
	}
	return 0
}
