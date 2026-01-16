package controllers

import (
	"archive/zip"
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to fetch certificates")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result.Certs, "total": result.Total})
}

// Upload handles certificate upload
func (ctrl *CertController) Upload(c *gin.Context) {
	certModel, err := buildCertFromRequest(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if err := db.DB.Omit("task_id", "issue_task_id").Create(certModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to save certificate")})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(certModel.ID)})

	c.JSON(http.StatusOK, gin.H{"message": T("Certificate uploaded successfully"), "data": certModel})
}

// AdminList returns all certificates for admin
func (ctrl *CertController) AdminList(c *gin.Context) {
	result, err := queryCerts(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to fetch certificates")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": result.Certs, "total": result.Total})
}

func (ctrl *CertController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	role := c.GetString("role")
	isAdmin := role == "admin"

	// Allow admin to specify UserID
	certModel, err := buildCertFromRequest(c, isAdmin)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	certModel.ID = id
	certModel.UpdateAt = time.Now()

	updates := map[string]interface{}{
		"name":       certModel.Name,
		"des":        certModel.Description,
		"type":       certModel.Type,
		"domain":     certModel.Domain,
		"dnsapi":     certModel.DNSAPI,
		"auto_renew": certModel.AutoRenew,
		"update_at":  certModel.UpdateAt,
	}

	certProvided := strings.TrimSpace(certModel.Cert) != "" || strings.TrimSpace(certModel.Key) != ""
	if certModel.Type == "upload" && certProvided {
		updates["cert"] = certModel.Cert
		updates["key"] = certModel.Key
		if certModel.StartTime != nil || certModel.ExpireTime != nil {
			updates["start_time"] = certModel.StartTime
			updates["expire_time"] = certModel.ExpireTime
		} else if strings.TrimSpace(certModel.Cert) != "" {
			if domains, notBefore, notAfter, err := parseCert(certModel.Cert); err == nil {
				updates["start_time"] = notBefore
				updates["expire_time"] = notAfter
				if certModel.Domain == "" && len(domains) > 0 {
					updates["domain"] = strings.Join(domains, ",")
				}
			}
		}
	}

	if isAdmin && certModel.UserID > 0 {
		updates["uid"] = certModel.UserID
	}

	query := db.DB.Model(&models.Cert{}).Where("id = ?", id)
	if !isAdmin {
		// Enforce ownership for non-admins
		uid := parseUserID(mustGet(c, "userID"))
		query = query.Where("uid = ?", uid)
	}

	result := query.Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update certificate")})
		return
	}
	if result.RowsAffected == 0 {
		// Possibly not found or permission denied (uid mismatch)
		// We can't distinguish easily without prior query, but generic error is fine or 404
		c.JSON(http.StatusNotFound, gin.H{"error": T("Certificate not found or permission denied")})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(id)})

	c.JSON(http.StatusOK, gin.H{"message": T("Certificate updated successfully")})
}

func (ctrl *CertController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": i18n.T("cert.invalid_id")})
		return
	}

	// 1. Check if disabled
	var cert models.Cert
	if err := db.DB.First(&cert, id).Error; err == nil {
		if cert.Enable {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": i18n.T("cert.in_use_disable_first")})
			return
		}
	}

	if err := db.DB.Delete(&models.Cert{}, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": i18n.T("cert.delete_failed")})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(id)})

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": i18n.T("cert.delete_success")})
}

func (ctrl *CertController) BatchAction(c *gin.Context) {
	var req struct {
		Action string  `json:"action"`
		IDs    []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
	}

	switch strings.ToLower(req.Action) {
	case "enable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Update("enable", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": i18n.T("cert.enable_failed")})
			return
		}
	case "auto_renew_enable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Update("auto_renew", true).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": i18n.T("cert.action_failed")})
			return
		}
	case "auto_renew_disable":
		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Update("auto_renew", false).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": i18n.T("cert.action_failed")})
			return
		}
	case "disable", "force_disable":
		// 1. Check if used by any site
		var siteCount int64
		// Site table has cert_id column in database
		db.DB.Table("site").Where("cert_id IN ?", req.IDs).Count(&siteCount)
		if siteCount > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": i18n.T("cert.site_ref_disable_first")})
			return
		}

		updates := map[string]interface{}{"enable": false}
		if strings.ToLower(req.Action) == "force_disable" {
			updates["auto_renew"] = false
		}

		if err := db.DB.Model(&models.Cert{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": i18n.T("cert.action_failed")})
			return
		}
	case "delete":
		// 1. Check if all are disabled
		var enabledCount int64
		db.DB.Model(&models.Cert{}).Where("id IN ? AND enable = ?", req.IDs, true).Count(&enabledCount)
		if enabledCount > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": i18n.T("cert.selected_enabled_disable_first")})
			return
		}

		if err := db.DB.Where("id IN ?", req.IDs).Delete(&models.Cert{}).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": i18n.T("cert.delete_failed")})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": i18n.T("cert.unknown_action")})
		return
	}

	services.BumpConfigVersion("cert", req.IDs)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": i18n.T("cert.action_success")})
}

func (ctrl *CertController) BatchCreate(c *gin.Context) {
	var req struct {
		UserID    int64           `json:"user_id"`
		Type      string          `json:"type"`
		DNSAPI    int             `json:"dnsapi"`
		AutoRenew bool            `json:"auto_renew"`
		Domains   json.RawMessage `json:"domains"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if len(req.Domains) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("domains is required")})
		return
	}

	userID := req.UserID
	if userID == 0 {
		userID = int64(parseUserID(mustGet(c, "userID")))
	}

	typeName := normalizeCertType(req.Type)
	if typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("type is required")})
		return
	}

	domains, err := parseBatchDomains(req.Domains)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(domains) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("domains is required")})
		return
	}
	if hasWildcardDomain(domains) && req.DNSAPI <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("wildcard requires dnsapi")})
		return
	}

	// Generate BatchID (fits in int(11) pid)
	batchID := time.Now().Unix()

	now := time.Now()
	var createdIDs []int64

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		for _, domain := range domains {
			dnsapiPtr := normalizeDNSAPIValue(req.DNSAPI)
			cert := models.Cert{
				UserID:     int(userID),
				Name:       defaultCertName(domain),
				Type:       typeName,
				Domain:     domain,
				DNSAPI:     dnsapiPtr,
				AutoRenew:  req.AutoRenew,
				Enable:     true,
				CreateAt:   now,
				UpdateAt:   now,
				State:      "waiting", // Initial state
				StartTime:  nil,
				ExpireTime: nil,
			}
			if err := tx.Omit("task_id", "issue_task_id").Create(&cert).Error; err != nil {
				return err
			}
			createdIDs = append(createdIDs, int64(cert.ID))
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create certificates")})
		return
	}

	services.BumpConfigVersion("cert", []int64{})

	// Trigger async issuance
	services.IssueCertsAsync(batchID, createdIDs)

	c.JSON(http.StatusOK, gin.H{"batch_id": strconv.FormatInt(batchID, 10), "count": len(createdIDs), "ids": createdIDs})
}

func (ctrl *CertController) WildcardCreate(c *gin.Context) {
	var req struct {
		UserID    int64  `json:"user_id"`
		Type      string `json:"type"`
		DNSAPI    int    `json:"dnsapi"`
		AutoRenew bool   `json:"auto_renew"`
		Domain    string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}

	domain := normalizeDomainHost(req.Domain)
	if !strings.HasPrefix(domain, "*.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("wildcard domain is required")})
		return
	}
	if isIPDomain(domain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("domain must not be IP")})
		return
	}

	typeName := normalizeCertType(req.Type)
	if typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("type is required")})
		return
	}

	userID := req.UserID
	if userID == 0 {
		userID = int64(parseUserID(mustGet(c, "userID")))
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
		return
	}

	now := time.Now()
	cert := models.Cert{
		UserID:     int(userID),
		Name:       defaultCertName(domain),
		Type:       typeName,
		Domain:     domain,
		DNSAPI:     normalizeDNSAPIValue(req.DNSAPI),
		AutoRenew:  req.AutoRenew,
		Enable:     true,
		CreateAt:   now,
		UpdateAt:   now,
		State:      "waiting",
		StartTime:  nil,
		ExpireTime: nil,
	}
	if err := db.DB.Omit("task_id", "issue_task_id").Create(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create certificates")})
		return
	}

	services.BumpConfigVersion("cert", []int64{int64(cert.ID)})
	services.IssueCertsAsync(time.Now().Unix(), []int64{int64(cert.ID)})

	c.JSON(http.StatusOK, gin.H{"id": cert.ID})
}

func (ctrl *CertController) GetDNSChallenge(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	query := db.DB.Model(&models.Cert{}).Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		query = query.Where("uid = ?", uid)
	}

	var cert models.Cert
	if err := query.First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": T("Certificate not found")})
		return
	}

	info, err := services.ParseDNSChallengeInfo(cert.Ret)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

func (ctrl *CertController) VerifyDNSChallenge(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	query := db.DB.Model(&models.Cert{}).Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		query = query.Where("uid = ?", uid)
	}

	var cert models.Cert
	if err := query.First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": T("Certificate not found")})
		return
	}

	info, err := services.ParseDNSChallengeInfo(cert.Ret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ok, err := services.CheckDNSChallengeTXT(info.FQDN, info.RecordValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("DNS TXT record not found")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (ctrl *CertController) Reissue(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
	}
	if strings.TrimSpace(config.App.AcmeEmail) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("acme_email is required")})
		return
	}
	services.IssueCertsAsync(time.Now().Unix(), req.IDs)
	c.JSON(http.StatusOK, gin.H{"message": T("Reissue submitted")})
}

func (ctrl *CertController) Download(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	query := db.DB.Model(&models.Cert{}).Where("id = ?", id)
	userID := int64(0)
	if isUserRequest(c) {
		userID = parseUserID(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		query = query.Where("uid = ?", userID)
	}
	var cert models.Cert
	if err := query.First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": T("Certificate not found")})
		return
	}

	rawDomain := strings.TrimSpace(c.Query("domain"))
	domainKey := normalizeCertDomainKey(rawDomain)
	if domainKey == "" {
		domainKey = primaryCertDomain(cert.Domain)
	}
	if domainKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("domain is required")})
		return
	}

	certs, err := loadCertsByDomain(domainKey, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load certificates")})
		return
	}
	if len(certs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": T("Certificate not found")})
		return
	}

	data, filename, err := buildCertZip(domainKey, certs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to build download")})
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write(data)
}

type certDefaultSettings struct {
	Type   string `json:"type"`
	DNSAPI int    `json:"dnsapi"`
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

type CertDetail struct {
	ID          int        `json:"id"`
	Uid         int        `json:"uid"`
	Name        string     `json:"name"`
	Description string     `json:"des"`
	Type        string     `json:"type"`
	Domain      string     `json:"domain"`
	DNSAPI      int        `json:"dnsapi"`
	Cert        string     `json:"cert"`
	Key         string     `json:"key"`
	StartTime   *time.Time `json:"start_time"`
	ExpireTime  *time.Time `json:"expire_time"`
	AutoRenew   bool       `json:"auto_renew"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
	Enable      bool       `json:"enable"`
	TaskID      int64      `json:"task_id"`
	State       string     `json:"state"`
	Ret         string     `json:"ret"`
	Version     int        `json:"version"`

	UserName     string `json:"user_name,omitempty"`
	IssueTaskRet string `json:"issue_task_ret,omitempty"`
}

type certListResult struct {
	Certs []CertDetail
	Total int64
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

	// Manual Preload
	var userIDs []int64
	var taskIDs []int64
	for _, cert := range certs {
		if cert.UserID > 0 {
			userIDs = append(userIDs, int64(cert.UserID))
		}
		if cert.IssueTaskID > 0 {
			taskIDs = append(taskIDs, cert.IssueTaskID)
		}
	}

	usersMap := make(map[int64]string)
	if len(userIDs) > 0 {
		var users []struct {
			ID   int64  `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := db.DB.Model(&models.User{}).Select("id, name").Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				usersMap[u.ID] = u.Name
			}
		}
		// Debug Log
		fmt.Printf("DEBUG: Found %d users for IDs: %v\n", len(users), userIDs)
	} else {
		fmt.Println("DEBUG: No userIDs collected from certs.")
	}

	tasksMap := make(map[int64]string)
	if len(taskIDs) > 0 {
		var tasks []struct {
			ID  int64  `gorm:"column:id"`
			Ret string `gorm:"column:ret"`
		}
		if err := db.DB.Model(&models.Task{}).Select("id, ret").Where("id IN ?", taskIDs).Find(&tasks).Error; err == nil {
			for _, t := range tasks {
				tasksMap[t.ID] = t.Ret
			}
		}
	}

	var details []CertDetail
	for _, cert := range certs {
		exposeCert := shouldExposeCertData(cert)
		certValue := ""
		keyValue := ""
		if exposeCert {
			certValue = cert.Cert
			if strings.TrimSpace(cert.Key) != "" {
				if dec, err := services.Crypto.Decrypt(cert.Key); err == nil {
					keyValue = dec
				} else {
					keyValue = cert.Key
				}
			}
		}
		detail := CertDetail{
			ID:          cert.ID,
			Uid:         cert.UserID,
			Name:        cert.Name,
			Description: cert.Description,
			Type:        cert.Type,
			Domain:      cert.Domain,
			DNSAPI:      toDNSAPIValue(cert.DNSAPI),
			Cert:        certValue,
			Key:         keyValue,
			StartTime:   cert.StartTime,
			ExpireTime:  cert.ExpireTime,
			AutoRenew:   cert.AutoRenew,
			CreateAt:    cert.CreateAt,
			UpdateAt:    cert.UpdateAt,
			Enable:      cert.Enable,
			TaskID:      cert.TaskID,
			State:       cert.State,
			Ret:         cert.Ret,
			Version:     cert.Version,
		}
		if name, ok := usersMap[int64(cert.UserID)]; ok {
			detail.UserName = name
		}
		if ret, ok := tasksMap[cert.IssueTaskID]; ok {
			detail.IssueTaskRet = ret
		}
		details = append(details, detail)
	}

	if len(details) > 0 {
		b, _ := json.Marshal(details[0])
		fmt.Printf("DEBUG: First Cert Detail JSON: %s\n", string(b))
	}

	return &certListResult{Certs: details, Total: total}, nil
}

func shouldExposeCertData(cert models.Cert) bool {
	if strings.EqualFold(strings.TrimSpace(cert.Type), "upload") {
		return true
	}
	state := strings.ToLower(strings.TrimSpace(cert.State))
	return state == "ready" || state == "success"
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
		if settings, err := loadCertDefaultSettings("system", "user", int(targetUserID)); err == nil && settings != nil {
			c.JSON(http.StatusOK, gin.H{"data": settings})
			return
		}
	}
	if settings, err := loadCertDefaultSettings("system", "global", 0); err == nil && settings != nil {
		c.JSON(http.StatusOK, gin.H{"data": settings})
		return
	}

	// Create default if not exists
	defaultSettings := certDefaultSettings{Type: "system", DNSAPI: 0}
	b, _ := json.Marshal(defaultSettings)
	sys := models.SysConfig{
		Name:      "cert_default_settings",
		Value:     string(b),
		Type:      "system",
		ScopeID:   0,
		ScopeName: "global",
		Enable:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.DB.Create(&sys)

	c.JSON(http.StatusOK, gin.H{"data": defaultSettings})
}

func (ctrl *CertController) UpdateDefaultSettings(c *gin.Context) {
	var req struct {
		UserID int64 `json:"user_id"`
		certDefaultSettings
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if req.Type == "" {
		req.Type = "system"
	}
	targetUserID := req.UserID
	if targetUserID == 0 && isUserRequest(c) {
		targetUserID = parseUserID(mustGet(c, "userID"))
	}

	configType := "system"
	scopeName := "global"
	scopeID := 0
	if targetUserID != 0 {
		scopeName = "user"
		scopeID = int(targetUserID)
	}

	payload := certDefaultSettings{Type: req.Type, DNSAPI: req.DNSAPI}
	b, _ := json.Marshal(payload)
	var sys models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "cert_default_settings", configType, scopeName, scopeID)
	if err := query.First(&sys).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Save failed")})
			return
		}
		sys = models.SysConfig{
			Name:      "cert_default_settings",
			Value:     string(b),
			Type:      configType,
			ScopeID:   scopeID,
			ScopeName: scopeName,
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.DB.Create(&sys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Save failed")})
			return
		}
	} else {
		// Fix: Use explicit Update because 'config' table might not have primary key 'id' for GORM Save()
		updates := map[string]interface{}{
			"value":     string(b),
			"update_at": time.Now(),
		}
		if err := query.Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Save failed")})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("saved")})
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

	certValue := input.Cert
	keyValue := input.Key
	if typeName != "upload" {
		certValue = ""
		keyValue = ""
	}

	certModel := &models.Cert{
		UserID:      int(userID),
		Name:        strings.TrimSpace(input.Name),
		Description: input.Des,
		Type:        typeName,
		Domain:      strings.TrimSpace(input.Domain),
		DNSAPI:      normalizeDNSAPIValue(input.DNSAPI),
		Cert:        certValue,
		Key:         keyValue,
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
		return nil, time.Time{}, time.Time{}, errors.New("failed to parse certificate")
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

func parseBatchDomains(raw json.RawMessage) ([]string, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil, errors.New("domains is required")
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return normalizeDomains(splitLines(asString))
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return normalizeDomains(list)
	}
	return nil, errors.New("invalid domains")
}

func normalizeDomains(domains []string) ([]string, error) {
	clean := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = normalizeDomainHost(domain)
		if strings.HasPrefix(domain, "*.") {
			base := strings.TrimPrefix(domain, "*.")
			base = normalizeDomainHost(base)
			if base != "" {
				domain = "*." + base
			}
		}
		if domain == "" {
			continue
		}
		if isIPDomain(domain) {
			return nil, fmt.Errorf("invalid domain: %s", domain)
		}
		clean = append(clean, domain)
	}
	return uniqueStrings(clean), nil
}

func isIPDomain(domain string) bool {
	trimmed := strings.TrimPrefix(domain, "*.")
	return net.ParseIP(trimmed) != nil
}

func hasWildcardDomain(domains []string) bool {
	for _, domain := range domains {
		if strings.HasPrefix(strings.TrimSpace(domain), "*.") {
			return true
		}
	}
	return false
}

func defaultCertName(domain string) string {
	if domain == "" {
		return i18n.T("cert.free")
	}
	return domain + i18n.T("cert.free")
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

func normalizeDNSAPIValue(val int) *int {
	if val <= 0 {
		return nil
	}
	return &val
}

func toDNSAPIValue(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}

func loadCertsByDomain(domainKey string, userID int64) ([]models.Cert, error) {
	if domainKey == "" || db.DB == nil {
		return []models.Cert{}, nil
	}
	like := "%" + domainKey + "%"
	query := db.DB.Model(&models.Cert{}).Where("domain LIKE ?", like)
	if userID != 0 {
		query = query.Where("uid = ?", userID)
	}
	var rows []models.Cert
	if err := query.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Cert, 0, len(rows))
	for _, row := range rows {
		if certDomainMatches(domainKey, row.Domain) {
			out = append(out, row)
		}
	}
	return out, nil
}

func buildCertZip(domainKey string, certs []models.Cert) ([]byte, string, error) {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)

	safeDomain := sanitizeCertFilename(domainKey)
	if safeDomain == "" {
		safeDomain = "certs"
	}
	for _, cert := range certs {
		base := fmt.Sprintf("%s_%d_%s", safeDomain, cert.ID, normalizeCertType(cert.Type))
		base = sanitizeCertFilename(base)
		if base == "" {
			base = fmt.Sprintf("cert_%d", cert.ID)
		}

		certPem := strings.TrimSpace(cert.Cert)
		keyPem := strings.TrimSpace(cert.Key)
		if keyPem != "" {
			if dec, err := services.Crypto.Decrypt(keyPem); err == nil {
				keyPem = dec
			}
		}

		if err := writeZipFile(writer, base+".pem", certPem+"\n"); err != nil {
			return nil, "", err
		}
		if err := writeZipFile(writer, base+".key", keyPem+"\n"); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	filename := filepath.Base(safeDomain + ".zip")
	return buf.Bytes(), filename, nil
}

func writeZipFile(writer *zip.Writer, name string, content string) error {
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write([]byte(content))
	return err
}

func normalizeCertDomainKey(raw string) string {
	domains := splitCertDomains(raw)
	if len(domains) == 0 {
		return ""
	}
	return domains[0]
}

func primaryCertDomain(raw string) string {
	domains := splitCertDomains(raw)
	if len(domains) == 0 {
		return ""
	}
	return domains[0]
}

func splitCertDomains(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	fields := splitFields(raw)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		host := normalizeDomainHost(field)
		if strings.HasPrefix(host, "*.") {
			base := strings.TrimPrefix(host, "*.")
			base = normalizeDomainHost(base)
			if base != "" {
				host = "*." + base
			}
		}
		if host == "" {
			continue
		}
		out = append(out, host)
	}
	return uniqueStrings(out)
}

func certDomainMatches(domainKey string, raw string) bool {
	if domainKey == "" || raw == "" {
		return false
	}
	domainKey = normalizeDomainHost(domainKey)
	for _, domain := range splitCertDomains(raw) {
		if normalizeDomainHost(domain) == domainKey {
			return true
		}
	}
	return false
}

func sanitizeCertFilename(input string) string {
	if input == "" {
		return ""
	}
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, input)
	return strings.Trim(safe, "._-")
}
