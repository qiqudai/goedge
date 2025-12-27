package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/services/dns"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SiteController struct{}

type siteListItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	UserName        string    `json:"user_name"`
	Domains         []string  `json:"domains"`
	DomainDisplay   string    `json:"domain_display"`
	ListenPorts     string    `json:"listen_ports"`
	OriginDisplay   string    `json:"origin_display"`
	CNAME           string    `json:"cname"`
	HTTPS           bool      `json:"https"`
	UserPackageID   int64     `json:"user_package_id"`
	UserPackageName string    `json:"user_package_name"`
	DNSProviderID   int64     `json:"dns_provider_id"`
	GroupID         int64     `json:"group_id"`
	GroupName       string    `json:"group_name"`
	NodeGroupID     int64     `json:"node_group_id"`
	NodeGroupName   string    `json:"node_group_name"`
	Status          bool      `json:"status"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
}

type siteQueryResult struct {
	Sites []models.Site
	Total int64
}

// List returns the list of sites for the current user
func (ctrl *SiteController) List(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := parseInt64(userID)
	result, err := querySites(c, &uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sites"})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items, "total": result.Total})
}

// AdminList returns the list of all sites for admin
func (ctrl *SiteController) AdminList(c *gin.Context) {
	result, err := querySites(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sites"})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": items, "total": result.Total})
}

// AdminCreate handles site creation for admin
func (ctrl *SiteController) AdminCreate(c *gin.Context) {
	site, groupID, err := parseSiteCreateRequest(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := createSiteWithGroup(site, groupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)

	c.JSON(http.StatusOK, gin.H{"message": "Site created successfully", "data": site})
}

// Create handles site creation for user
func (ctrl *SiteController) Create(c *gin.Context) {
	site, groupID, err := parseSiteCreateRequest(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := createSiteWithGroup(site, groupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)

	c.JSON(http.StatusOK, gin.H{"message": "Site created successfully", "data": site})
}

// AdminBatchCreate handles batch site creation
func (ctrl *SiteController) AdminBatchCreate(c *gin.Context) {
	var req struct {
		UserID        int64  `json:"user_id"`
		UserPackageID int64  `json:"user_package_id"`
		GroupID       int64  `json:"group_id"`
		DNSProviderID int64  `json:"dns_provider_id"`
		NodeGroupID   int64  `json:"node_group_id"`
		Data          string `json:"data"`
		IgnoreError   bool   `json:"ignore_error"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.UserID == 0 {
		req.UserID = parseInt64(mustGet(c, "userID"))
	}
	if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	if req.UserPackageID == 0 {
		defaultID, err := findDefaultUserPackageID(req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		req.UserPackageID = defaultID
	}
	if strings.TrimSpace(req.Data) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data is required"})
		return
	}

	nodeGroupID, err := resolveNodeGroupFromPackage(req.UserPackageID, req.NodeGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defaults, err := services.GetSiteDefaultMapWithGroup(req.UserID, req.GroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load defaults"})
		return
	}

	lines := splitLines(req.Data)
	created := 0
	createdIDs := make([]int64, 0)
	for _, line := range lines {
		item, err := parseBatchLine(line)
		if err != nil {
			if req.IgnoreError {
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, domain := range item.Domains {
			site := &models.Site{
				UserID:        req.UserID,
				UserPackageID: req.UserPackageID,
				DNSProviderID: req.DNSProviderID,
				NodeGroupID:   nodeGroupID,
				Domains:       []string{domain},
				Backends:      item.Backends,
				HttpListen:    []string{"80"},
				State:         "running",
				Enable:        true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			site.CnameHostname = domain + ".cdn.node.com"
			services.ApplySiteDefaults(site, defaults)
			if err := createSiteWithGroup(site, req.GroupID); err != nil {
				if req.IgnoreError {
					continue
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			created++
			createdIDs = append(createdIDs, site.ID)
		}
	}

	if created > 0 {
		services.BumpConfigVersion("site", createdIDs)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Batch create completed", "created": created})
}

// AdminBatchUpdate updates fields for selected sites
func (ctrl *SiteController) AdminBatchUpdate(c *gin.Context) {
	var req struct {
		IDs             []int64                `json:"ids"`
		UserPackageID   *int64                 `json:"user_package_id"`
		GroupID         *int64                 `json:"group_id"`
		DNSProviderID   *int64                 `json:"dns_provider_id"`
		HttpListen      *[]string              `json:"http_listen"`
		HttpsListen     *[]string              `json:"https_listen"`
		BalanceWay      *string                `json:"balance_way"`
		BackendProtocol *string                `json:"backend_protocol"`
		CcDefaultRule   *int64                 `json:"cc_default_rule"`
		BlackIP         *string                `json:"black_ip"`
		WhiteIP         *string                `json:"white_ip"`
		BlockRegion     *string                `json:"block_region"`
		Settings        map[string]interface{} `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.UserPackageID != nil {
			updates["user_package"] = *req.UserPackageID
		}
		if req.DNSProviderID != nil && tx.Migrator().HasColumn(&models.Site{}, "dns_provider_id") {
			updates["dns_provider_id"] = *req.DNSProviderID
		}
		if req.HttpListen != nil {
			updates["http_listen"] = encodeList(*req.HttpListen)
		}
		if req.HttpsListen != nil {
			updates["https_listen"] = encodeList(*req.HttpsListen)
		}
		if req.BalanceWay != nil {
			updates["balance_way"] = *req.BalanceWay
		}
		if req.BackendProtocol != nil {
			updates["backend_protocol"] = *req.BackendProtocol
		}
		if req.CcDefaultRule != nil {
			updates["cc_default_rule"] = *req.CcDefaultRule
		}
		if req.BlackIP != nil {
			updates["black_ip"] = *req.BlackIP
		}
		if req.WhiteIP != nil {
			updates["white_ip"] = *req.WhiteIP
		}
		if req.BlockRegion != nil {
			updates["block_region"] = *req.BlockRegion
		}
		if req.Settings != nil {
			b, _ := json.Marshal(req.Settings)
			updates["settings"] = string(b)
		}
		if len(updates) > 0 {
			if err := tx.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.GroupID != nil {
			if err := tx.Where("site_id IN ?", req.IDs).Delete(&models.SiteGroupRelation{}).Error; err != nil {
				return err
			}
			if *req.GroupID != 0 {
				relations := make([]models.SiteGroupRelation, 0, len(req.IDs))
				for _, id := range req.IDs {
					relations = append(relations, models.SiteGroupRelation{SiteID: id, GroupID: *req.GroupID})
				}
				if err := tx.Create(&relations).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Batch update failed"})
		return
	}

	services.BumpConfigVersion("site", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": "Batch update completed"})
}

// AdminBatchAction handles enable/disable/delete etc
func (ctrl *SiteController) AdminBatchAction(c *gin.Context) {
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
		if err := db.DB.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": true,
			"state":  "running",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	case "disable":
		if err := db.DB.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": false,
			"state":  "stop",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	case "delete":
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("site_id IN ?", req.IDs).Delete(&models.SiteGroupRelation{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.IDs).Delete(&models.Site{}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
	case "unlock", "clear_cache":
		// No-op for now; placeholder to keep UI consistent
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown action"})
		return
	}

	services.BumpConfigVersion("site", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": "Action completed"})
}

// AdminApplyCert sets HTTPS listen ports for selected sites
func (ctrl *SiteController) AdminApplyCert(c *gin.Context) {
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

	var sites []models.Site
	if err := db.DB.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	for i := range sites {
		if len(sites[i].HttpsListen) == 0 {
			sites[i].HttpsListen = []string{"443"}
		}
		sites[i].UpdatedAt = time.Now()
		if err := db.DB.Save(&sites[i]).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
	}

	services.BumpConfigVersion("site", req.IDs)

	c.JSON(http.StatusOK, gin.H{"message": "Certificate apply queued"})
}

// AdminExport exports list as CSV
func (ctrl *SiteController) AdminExport(c *gin.Context) {
	result, err := querySites(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export"})
		return
	}
	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=sites.csv")
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"ID", "User", "Domain", "Listen", "Origin", "CNAME", "HTTPS", "Package", "Group", "Region", "Status", "CreatedAt"})
	for _, item := range items {
		httpsVal := "no"
		if item.HTTPS {
			httpsVal = "yes"
		}
		statusVal := "disabled"
		if item.Status {
			statusVal = "enabled"
		}
		_ = writer.Write([]string{
			strconv.FormatInt(item.ID, 10),
			item.UserName,
			item.DomainDisplay,
			item.ListenPorts,
			item.OriginDisplay,
			item.CNAME,
			httpsVal,
			item.UserPackageName,
			item.GroupName,
			item.NodeGroupName,
			statusVal,
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

// AdminResolve checks DNS resolution for a domain
func (ctrl *SiteController) AdminResolve(c *gin.Context) {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain is required"})
		return
	}

	cname, _ := net.LookupCNAME(domain)
	hosts, _ := net.LookupHost(domain)

	c.JSON(http.StatusOK, gin.H{
		"domain": domain,
		"cname":  strings.TrimSuffix(cname, "."),
		"ips":    hosts,
	})
}

func findDefaultUserPackageID(userID int64) (int64, error) {
	var pkg models.UserPackage
	if userID != 0 {
		if err := db.DB.Where("uid = ?", userID).Order("id asc").First(&pkg).Error; err == nil {
			return pkg.ID, nil
		}
	}
	if err := db.DB.Order("id asc").First(&pkg).Error; err != nil {
		return 0, errors.New("user_package not found")
	}
	return pkg.ID, nil
}

func findDefaultDNSProviderID(userID int64) (int64, error) {
	if userID != 0 {
		if settings, err := loadCertDefaultSettings("user", "user", int(userID)); err != nil {
			return 0, err
		} else if settings != nil && settings.DNSAPI != 0 {
			return int64(settings.DNSAPI), nil
		}
	}
	settings, err := loadCertDefaultSettings("system", "global", 0)
	if err != nil || settings == nil || settings.DNSAPI == 0 {
		return 0, err
	}
	return int64(settings.DNSAPI), nil
}

func parseSiteCreateRequest(c *gin.Context, admin bool) (*models.Site, int64, error) {
	var req struct {
		UserID        int64    `json:"user_id"`
		UserPackageID int64    `json:"user_package_id"`
		DNSProviderID int64    `json:"dns_provider_id"`
		GroupID       int64    `json:"group_id"`
		NodeGroupID   int64    `json:"node_group_id"`
		Domains       []string `json:"domains"`
		DomainsInput  string   `json:"domains_input"`
		Backends      []string `json:"backends"`
		BackendsInput string   `json:"backends_input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, 0, errors.New("invalid request")
	}

	userID := req.UserID
	if !admin {
		userID = parseInt64(mustGet(c, "userID"))
	} else if userID == 0 {
		userID = parseInt64(mustGet(c, "userID"))
	}
	if userID == 0 {
		return nil, 0, errors.New("user_id is required")
	}
	if req.UserPackageID == 0 {
		defaultID, err := findDefaultUserPackageID(userID)
		if err != nil {
			return nil, 0, err
		}
		req.UserPackageID = defaultID
	}
	if req.DNSProviderID == 0 {
		defaultDNS, err := findDefaultDNSProviderID(userID)
		if err != nil {
			return nil, 0, err
		}
		if defaultDNS != 0 {
			req.DNSProviderID = defaultDNS
		}
	}

	domains := req.Domains
	if len(domains) == 0 && strings.TrimSpace(req.DomainsInput) != "" {
		domains = splitFields(req.DomainsInput)
	}
	if len(domains) == 0 {
		return nil, 0, errors.New("domain is required")
	}

	backends := req.Backends
	if len(backends) == 0 && strings.TrimSpace(req.BackendsInput) != "" {
		backends = splitFields(req.BackendsInput)
	}

	nodeGroupID, err := resolveNodeGroupFromPackage(req.UserPackageID, req.NodeGroupID)
	if err != nil {
		return nil, 0, err
	}

	site := &models.Site{
		UserID:        userID,
		UserPackageID: req.UserPackageID,
		DNSProviderID: req.DNSProviderID,
		NodeGroupID:   nodeGroupID,
		Domains:       domains,
		Backends:      backends,
		HttpListen:    []string{"80"},
		State:         "running",
		Enable:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if site.CnameHostname == "" && len(domains) > 0 {
		site.CnameHostname = domains[0] + ".cdn.node.com"
	}

	defaults, err := services.GetSiteDefaultMapWithGroup(userID, req.GroupID)
	if err != nil {
		return nil, 0, err
	}
	services.ApplySiteDefaults(site, defaults)

	return site, req.GroupID, nil
}

func resolveNodeGroupFromPackage(userPackageID int64, requestedID int64) (int64, error) {
	if requestedID != 0 {
		return requestedID, nil
	}
	if userPackageID == 0 {
		return 0, nil
	}
	var pkg models.UserPackage
	if err := db.DB.Select("node_group_id").Where("id = ?", userPackageID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return pkg.NodeGroupID, nil
}

func createSiteWithGroup(site *models.Site, groupID int64) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if len(site.Domains) == 0 {
			return errors.New("domain is required")
		}
		tx.Model(&models.Site{}).Where("domain LIKE ?", "%"+site.Domains[0]+"%").Count(&count)
		if count > 0 {
			return errors.New("domain already exists")
		}
		omitColumns := siteMissingColumns(tx)
		dbTx := tx
		if len(omitColumns) > 0 {
			dbTx = dbTx.Omit(omitColumns...)
		}
		if site.RegionID == 0 {
			dbTx = dbTx.Omit("region_id")
		}
		if site.NodeGroupID == 0 {
			dbTx = dbTx.Omit("node_group_id")
		}
		if !site.EnableBackupGroup || site.BackupNodeGroupID == 0 {
			dbTx = dbTx.Omit("backup_node_group")
		}
		if err := dbTx.Create(site).Error; err != nil {
			return err
		}
		if groupID != 0 {
			rel := models.SiteGroupRelation{SiteID: site.ID, GroupID: groupID}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func siteMissingColumns(tx *gorm.DB) []string {
	migrator := tx.Migrator()
	missing := make([]string, 0, 3)
	if !migrator.HasColumn(&models.Site{}, "dns_provider_id") {
		missing = append(missing, "dns_provider_id")
	}
	if !migrator.HasColumn(&models.Site{}, "settings") {
		missing = append(missing, "settings")
	}
	if !migrator.HasColumn(&models.Site{}, "cname_hostname2") {
		missing = append(missing, "cname_hostname2")
	}
	return missing
}

func ensureDNSRecords(site *models.Site) error {
	if site == nil || site.DNSProviderID == 0 || len(site.Domains) == 0 {
		return nil
	}
	var api models.DNSAPI
	if err := db.DB.Where("id = ?", site.DNSProviderID).First(&api).Error; err != nil {
		return err
	}
	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil || provider == nil {
		return err
	}
	for _, domain := range site.Domains {
		root, name := splitRootDomain(domain)
		if root == "" {
			continue
		}
		record := dns.DNSRecord{
			Type:  "CNAME",
			Name:  name,
			Value: site.CnameHostname,
			TTL:   600,
		}
		_ = provider.AddRecord(root, record)
	}
	return nil
}

func splitRootDomain(domain string) (string, string) {
	host := normalizeDomainHost(domain)
	if host == "" || net.ParseIP(host) != nil {
		return "", ""
	}
	if strings.HasPrefix(host, "*.") {
		host = strings.TrimPrefix(host, "*.")
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", ""
	}
	root := strings.Join(parts[len(parts)-2:], ".")
	name := "@"
	if len(parts) > 2 {
		name = strings.Join(parts[:len(parts)-2], ".")
	}
	return root, name
}

func normalizeDomainHost(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.TrimRight(host, ".")
}

func querySites(c *gin.Context, userID *int64) (*siteQueryResult, error) {
	query := db.DB.Model(&models.Site{})
	if userID != nil && *userID != 0 {
		query = query.Where("uid = ?", *userID)
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	searchField := strings.TrimSpace(c.DefaultQuery("search_field", "all"))
	like := "%" + keyword + "%"

	if keyword != "" {
		switch searchField {
		case "site_id":
			if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
				query = query.Where("id = ?", id)
			} else {
				return &siteQueryResult{Sites: []models.Site{}, Total: 0}, nil
			}
		case "domain", "multi_domain":
			query = query.Where("domain LIKE ?", like)
		case "origin":
			query = query.Where("backend LIKE ?", like)
		case "cname":
			query = query.Where("cname_hostname LIKE ? OR cname_domain LIKE ?", like, like)
		case "package":
			ids, err := findUserPackageIDsByName(keyword)
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return &siteQueryResult{Sites: []models.Site{}, Total: 0}, nil
			}
			query = query.Where("user_package IN ?", ids)
		case "group":
			siteIDs, err := findSiteIDsByGroupName(keyword)
			if err != nil {
				return nil, err
			}
			if len(siteIDs) == 0 {
				return &siteQueryResult{Sites: []models.Site{}, Total: 0}, nil
			}
			query = query.Where("id IN ?", siteIDs)
		case "user":
			userIDs, err := findUserIDsByKeyword(keyword)
			if err != nil {
				return nil, err
			}
			if len(userIDs) == 0 {
				return &siteQueryResult{Sites: []models.Site{}, Total: 0}, nil
			}
			query = query.Where("uid IN ?", userIDs)
		default: // all
			cond := db.DB.Where("domain LIKE ? OR backend LIKE ? OR cname_hostname LIKE ? OR cname_domain LIKE ?", like, like, like, like)
			if id, err := strconv.ParseInt(keyword, 10, 64); err == nil {
				cond = cond.Or("id = ?", id)
			}
			if userIDs, err := findUserIDsByKeyword(keyword); err == nil && len(userIDs) > 0 {
				cond = cond.Or("uid IN ?", userIDs)
			}
			if pkgIDs, err := findUserPackageIDsByName(keyword); err == nil && len(pkgIDs) > 0 {
				cond = cond.Or("user_package IN ?", pkgIDs)
			}
			if siteIDs, err := findSiteIDsByGroupName(keyword); err == nil && len(siteIDs) > 0 {
				cond = cond.Or("id IN ?", siteIDs)
			}
			query = query.Where(cond)
		case "http_port":
			query = query.Where("http_listen LIKE ?", like)
		case "https_port":
			query = query.Where("https_listen LIKE ?", like)
		}
	}

	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if pkgStr := c.Query("user_package_id"); pkgStr != "" {
		if id, err := strconv.Atoi(pkgStr); err == nil {
			query = query.Where("user_package = ?", id)
		}
	}
	if groupStr := c.Query("group_id"); groupStr != "" {
		groupIDStrs := strings.Split(groupStr, ",")
		var groupIDs []int64
		for _, idStr := range groupIDStrs {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				groupIDs = append(groupIDs, int64(id))
			}
		}
		if len(groupIDs) > 0 {
			siteIDs, err := findSiteIDsByGroupIDs(groupIDs)
			if err != nil {
				return nil, err
			}
			if len(siteIDs) == 0 {
				return &siteQueryResult{Sites: []models.Site{}, Total: 0}, nil
			}
			query = query.Where("id IN ?", siteIDs)
		}
	}
	if nodeGroupStr := c.Query("node_group_id"); nodeGroupStr != "" {
		if id, err := strconv.Atoi(nodeGroupStr); err == nil {
			query = query.Where("node_group_id = ?", id)
		}
	}
	if status := c.Query("status"); status != "" {
		if status == "enabled" {
			query = query.Where("enable = ?", true)
		} else if status == "disabled" {
			query = query.Where("enable = ?", false)
		}
	}
	if https := c.Query("https"); https != "" {
		if https == "1" || strings.ToLower(https) == "true" {
			query = query.Where("https_listen <> ''")
		} else if https == "0" || strings.ToLower(https) == "false" {
			query = query.Where("https_listen = '' OR https_listen IS NULL")
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

	query = query.Select(strings.Join([]string{
		"id",
		"uid",
		"user_package",
		"region_id",
		"node_group_id",
		"dns_provider_id",
		"cname_domain",
		"cname_hostname",
		"cname_hostname2",
		"cname_mode",
		"domain",
		"http_listen",
		"https_listen",
		"backend",
		"state",
		"enable",
		"create_at",
	}, ","))

	var sites []models.Site
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&sites).Error; err != nil {
		return nil, err
	}

	return &siteQueryResult{Sites: sites, Total: total}, nil
}

func buildSiteListItems(sites []models.Site) ([]siteListItem, error) {
	userMap, err := loadUsers(sites)
	if err != nil {
		return nil, err
	}
	pkgMap, err := loadUserPackages(sites)
	if err != nil {
		return nil, err
	}
	groupMap, relMap, err := loadSiteGroups(sites)
	if err != nil {
		return nil, err
	}
	nodeGroupMap, err := loadNodeGroups(sites)
	if err != nil {
		return nil, err
	}

	items := make([]siteListItem, 0, len(sites))
	for _, site := range sites {
		domains := site.Domains
		domainDisplay := strings.Join(domains, ",")
		if domainDisplay == "" && site.DomainRaw != "" {
			domainDisplay = site.DomainRaw
		}
		originDisplay := strings.Join(site.Backends, ",")
		if originDisplay == "" && site.BackendRaw != "" {
			originDisplay = strings.Trim(site.BackendRaw, "\"")
		}
		httpsOn := len(site.HttpsListen) > 0 || strings.TrimSpace(site.HttpsListenRaw) != ""
		httpPorts := parseListenPorts(site.HttpListen, site.HttpListenRaw, "80")
		httpsPorts := parseListenPorts(site.HttpsListen, site.HttpsListenRaw, "")
		listenPorts := buildListenDisplay(httpPorts, httpsPorts)

		cname := strings.TrimSpace(site.CnameHostname)
		if cname == "" && len(domains) > 0 && site.CnameDomain != "" {
			cname = domains[0] + "." + site.CnameDomain
		}
		if cname == "" {
			cname = "-"
		}

		item := siteListItem{
			ID:              site.ID,
			UserID:          site.UserID,
			UserName:        userMap[site.UserID],
			Domains:         domains,
			DomainDisplay:   domainDisplay,
			ListenPorts:     listenPorts,
			OriginDisplay:   originDisplay,
			CNAME:           cname,
			HTTPS:           httpsOn,
			UserPackageID:   site.UserPackageID,
			UserPackageName: pkgMap[site.UserPackageID],
			DNSProviderID:   site.DNSProviderID,
			GroupID:         relMap[site.ID],
			GroupName:       groupMap[relMap[site.ID]],
			NodeGroupID:     site.NodeGroupID,
			NodeGroupName:   nodeGroupMap[site.NodeGroupID],
			Status:          site.Enable,
			State:           site.State,
			CreatedAt:       site.CreatedAt,
		}
		items = append(items, item)
	}

	return items, nil
}

func parseListenPorts(ports []string, raw string, fallback string) []string {
	if len(ports) > 0 {
		return ports
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}
	raw = strings.Trim(raw, "[]{}\"")
	raw = strings.ReplaceAll(raw, "port", "")
	raw = strings.ReplaceAll(raw, ":", " ")
	fields := strings.Fields(raw)
	if len(fields) == 0 && fallback != "" {
		return []string{fallback}
	}
	return fields
}

func buildListenDisplay(httpPorts, httpsPorts []string) string {
	parts := make([]string, 0, len(httpPorts)+len(httpsPorts))
	for _, p := range httpPorts {
		if p != "" {
			parts = append(parts, p)
		}
	}
	for _, p := range httpsPorts {
		if p != "" {
			parts = append(parts, p+"s")
		}
	}
	return strings.Join(parts, " ")
}

func encodeList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func loadUsers(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.UserID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var users []models.User
	if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		result[u.ID] = u.Name
	}
	return result, nil
}

func loadUserPackages(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.UserPackageID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var pkgs []models.UserPackage
	if err := db.DB.Where("id IN ?", ids).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		result[p.ID] = p.Name
	}
	return result, nil
}

func loadSiteGroups(sites []models.Site) (map[int64]string, map[int64]int64, error) {
	siteIDs := uniqueIDs(sites, func(s models.Site) int64 { return s.ID })
	groupMap := map[int64]string{}
	relMap := map[int64]int64{}
	if len(siteIDs) == 0 {
		return groupMap, relMap, nil
	}

	var relations []models.SiteGroupRelation
	if err := db.DB.Where("site_id IN ?", siteIDs).Find(&relations).Error; err != nil {
		return nil, nil, err
	}
	groupIDs := make([]int64, 0, len(relations))
	for _, rel := range relations {
		relMap[rel.SiteID] = rel.GroupID
		groupIDs = append(groupIDs, rel.GroupID)
	}
	if len(groupIDs) == 0 {
		return groupMap, relMap, nil
	}
	var groups []models.SiteGroup
	if err := db.DB.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}
	return groupMap, relMap, nil
}

func loadNodeGroups(sites []models.Site) (map[int64]string, error) {
	ids := uniqueIDs(sites, func(s models.Site) int64 { return s.NodeGroupID })
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var groups []models.NodeGroup
	if err := db.DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, g := range groups {
		result[g.ID] = g.Name
	}
	return result, nil
}

func findUserIDsByKeyword(keyword string) ([]int64, error) {
	var users []models.User
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", like, like, like).Find(&users).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	return ids, nil
}

func findUserPackageIDsByName(keyword string) ([]int64, error) {
	var pkgs []models.UserPackage
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ?", like).Find(&pkgs).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(pkgs))
	for _, p := range pkgs {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

func findSiteIDsByGroupName(keyword string) ([]int64, error) {
	var groups []models.SiteGroup
	like := "%" + keyword + "%"
	if err := db.DB.Where("name LIKE ?", like).Find(&groups).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	return findSiteIDsByGroupIDs(ids)
}

func findSiteIDsByGroupID(groupID int64) ([]int64, error) {
	return findSiteIDsByGroupIDs([]int64{groupID})
}

func findSiteIDsByGroupIDs(groupIDs []int64) ([]int64, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var rels []models.SiteGroupRelation
	if err := db.DB.Where("group_id IN ?", groupIDs).Find(&rels).Error; err != nil {
		return nil, err
	}
	siteIDs := make([]int64, 0, len(rels))
	for _, rel := range rels {
		siteIDs = append(siteIDs, rel.SiteID)
	}
	return siteIDs, nil
}

func uniqueIDs(sites []models.Site, getter func(models.Site) int64) []int64 {
	seen := map[int64]struct{}{}
	for _, s := range sites {
		id := getter(s)
		if id == 0 {
			continue
		}
		seen[id] = struct{}{}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

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

func splitFields(input string) []string {
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")
	parts := strings.Fields(input)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitLines(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

type batchSiteItem struct {
	Domains  []string
	Backends []string
}

func parseBatchLine(line string) (*batchSiteItem, error) {
	item := &batchSiteItem{}
	segments := strings.Split(line, "|")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		kv := strings.SplitN(seg, "=", 2)
		if len(kv) != 2 {
			return nil, errors.New("invalid line format")
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "domain":
			item.Domains = splitByComma(val)
		case "ip":
			item.Backends = splitByComma(val)
		}
	}
	if len(item.Domains) == 0 {
		return nil, errors.New("domain is required")
	}
	return item, nil
}

func splitByComma(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
