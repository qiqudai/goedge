package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func parseSiteCreateRequest(c *gin.Context, admin bool) (*models.Site, []int64, error) {
	var req struct {
		UserID        int64    `json:"user_id"`
		UserPackageID int64    `json:"user_package_id"`
		DNSProviderID int64    `json:"dns_provider_id"`
		GroupID       int64    `json:"group_id"`
		GroupIDs      []int64  `json:"group_ids"`
		NodeGroupID   int64    `json:"node_group_id"`
		SiteType      string   `json:"site_type"`
		Domains       []string `json:"domains"`
		DomainsInput  string   `json:"domains_input"`
		Backends      []string `json:"backends"`
		BackendsInput string   `json:"backends_input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, nil, errors.New("Invalid request")
	}

	userID := req.UserID
	if !admin {
		userID = parseInt64(mustGet(c, "userID"))
	} else if userID == 0 {
		userID = parseInt64(mustGet(c, "userID"))
	}
	if userID == 0 {
		return nil, nil, errors.New("user_id is required")
	}
	if req.UserPackageID == 0 {
		defaultID, err := findDefaultUserPackageID(userID)
		if err != nil {
			return nil, nil, err
		}
		req.UserPackageID = defaultID
	}
	if req.DNSProviderID == 0 {
		defaultDNS, err := findDefaultDNSProviderID(userID)
		if err != nil {
			return nil, nil, err
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
		return nil, nil, errors.New("domain is required")
	}

	if err := services.CheckDomainLimit(userID, req.UserPackageID, domains); err != nil {
		return nil, nil, err
	}

	backends := req.Backends
	if len(backends) == 0 && strings.TrimSpace(req.BackendsInput) != "" {
		backends = splitFields(req.BackendsInput)
	}

	nodeGroupID, err := resolveNodeGroupFromPackage(req.UserPackageID, req.NodeGroupID)
	if err != nil {
		return nil, nil, err
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
	siteType := strings.ToLower(strings.TrimSpace(req.SiteType))
	if siteType == "" {
		siteType = "website"
	}
	site.Settings = map[string]interface{}{
		"site_type": siteType,
	}
	var userPkg models.UserPackage
	if err := db.DB.First(&userPkg, req.UserPackageID).Error; err == nil {
		if site.NodeGroupID == 0 && userPkg.NodeGroupID != 0 {
			site.NodeGroupID = userPkg.NodeGroupID
		}
		if site.RegionID == 0 && userPkg.RegionID != 0 {
			site.RegionID = userPkg.RegionID
		}
		if !site.EnableBackupGroup && userPkg.EnableBackup && userPkg.BackupNodeGroup != 0 {
			site.EnableBackupGroup = true
			site.BackupNodeGroupID = userPkg.BackupNodeGroup
		}
		fmt.Printf("[DEBUG] CreateSite PkgID=%d Mode='%s' Host='%s' Dom='%s'\n", req.UserPackageID, userPkg.CnameMode, userPkg.CnameHostname, userPkg.CnameDomain)
		if strings.TrimSpace(userPkg.CnameMode) == "package" && userPkg.CnameHostname != "" {
			fmt.Println("[DEBUG] Using Package CNAME Mode")
			site.CnameHostname = userPkg.CnameHostname
			if userPkg.CnameDomain != "" {
				site.CnameHostname += "." + userPkg.CnameDomain
			}
		} else {
			fmt.Println("[DEBUG] Using Default CNAME Mode")
			if userPkg.CnameDomain != "" {
				site.CnameDomain = userPkg.CnameDomain
			} else {
				// Fallback to default
				site.CnameDomain = "cdn.node.com"
			}
			if len(domains) > 0 {
				site.CnameHostname = buildSiteCname(domains[0], site.CnameDomain)
			}
		}
	} else {
		fmt.Printf("[DEBUG] CreateSite Failed to load pkg: %v\n", err)
	}
	if site.RegionID == 0 {
		site.RegionID = resolveRegionFromPackage(req.UserPackageID, site.NodeGroupID)
	}

	defaults, err := services.GetSiteDefaultMapWithGroup(userID, req.GroupID)
	if err != nil {
		return nil, nil, err
	}
	if globalDefaults := services.GetGlobalDefaultConfig(); globalDefaults != nil {
		services.ApplySiteTemplateDefaultsByType(site, globalDefaults)
	}
	services.ApplySiteDefaults(site, defaults)
	normalizeBackendProtocolForSchema(site)

	// Force HTTPS OFF by default - REMOVED to allow ApplySiteDefaults to work
	// site.HttpsListen = []string{}

	// Handle GroupIDs
	groupIDs := req.GroupIDs
	if len(groupIDs) == 0 && req.GroupID != 0 {
		groupIDs = []int64{req.GroupID}
	}

	return site, groupIDs, nil
}

func createSiteWithGroup(site *models.Site, groupIDs []int64) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if len(site.Domains) == 0 {
			return errors.New("domain is required")
		}
		tx.Model(&models.Site{}).Where("domain LIKE ?", "%"+site.Domains[0]+"%").Count(&count)
		if count > 0 {
			return errors.New(i18n.T("site.domain_exists"))
		}
		omitColumns := siteMissingColumns(tx)
		dbTx := tx
		if len(omitColumns) > 0 {
			dbTx = dbTx.Omit(omitColumns...)
		}
		// Always omit cert_id on create to keep compatibility with schemas lacking this column.
		dbTx = dbTx.Omit("CertID", "cert_id")
		if site.RegionID == 0 {
			dbTx = dbTx.Omit("RegionID")
		}
		if site.NodeGroupID == 0 {
			dbTx = dbTx.Omit("NodeGroupID")
		}
		if !site.EnableBackupGroup || site.BackupNodeGroupID == 0 {
			dbTx = dbTx.Omit("BackupNodeGroupID")
		}
		selectCols := selectSiteCreateColumns(tx, site)
		if len(selectCols) > 0 {
			dbTx = dbTx.Select(selectCols)
		}
		if err := dbTx.Create(site).Error; err != nil {
			if isUnknownColumnError(err, "cert_id") {
				if retryErr := dbTx.Omit("CertID", "cert_id").Create(site).Error; retryErr == nil {
					goto created
				} else {
					return retryErr
				}
			}
			return err
		}
	created:
		if len(groupIDs) > 0 {
			for _, gid := range groupIDs {
				if gid != 0 {
					rel := models.SiteGroupRelation{SiteID: site.ID, GroupID: gid}
					if err := tx.Create(&rel).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func applyDefaultsIfSettingsMissing(site *models.Site, groupID int64, siteType string) {
	if site == nil {
		return
	}
	settingsEmpty := site.Settings == nil || len(site.Settings) == 0
	if !settingsEmpty && len(site.Settings) == 1 {
		if _, ok := site.Settings["site_type"]; ok {
			settingsEmpty = true
		}
	}
	if site.Settings == nil {
		site.Settings = map[string]interface{}{}
	}
	if _, ok := site.Settings["site_type"]; !ok {
		siteType = strings.TrimSpace(siteType)
		if siteType != "" {
			site.Settings["site_type"] = siteType
		}
	}
	if _, ok := site.Settings["site_type"]; !ok {
		site.Settings["site_type"] = "website"
	}
	if globalDefaults := services.GetGlobalDefaultConfig(); globalDefaults != nil {
		services.ApplySiteTemplateDefaultsByType(site, globalDefaults)
	}
	if defaults, err := services.GetSiteDefaultMapWithGroup(site.UserID, groupID); err == nil {
		services.ApplySiteDefaults(site, defaults)
	}
	if settingsEmpty {
		if scopedDefaults := services.GetSiteScopedDefaultMap(site.UserID, groupID); scopedDefaults != nil {
			services.ApplySiteDefaultsScopedOverrides(site, scopedDefaults)
		}
	}
	if site.Settings != nil {
		services.NormalizeSiteSettings(site.Settings)
	}
}

var backendProtocolMaxLen int64 = -1
var backendProtocolLenOnce sync.Once

func resolveBackendProtocolMaxLen() int64 {
	backendProtocolLenOnce.Do(func() {
		backendProtocolMaxLen = -1
		if db.DB == nil {
			return
		}
		if cols, err := db.DB.Migrator().ColumnTypes(&models.Site{}); err == nil {
			for _, col := range cols {
				if strings.EqualFold(col.Name(), "backend_protocol") {
					if length, ok := col.Length(); ok {
						backendProtocolMaxLen = length
					}
					break
				}
			}
		}
	})
	return backendProtocolMaxLen
}

func normalizeBackendProtocolValue(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return raw
	}
	maxLen := resolveBackendProtocolMaxLen()
	if maxLen <= 0 || int64(len(raw)) <= maxLen {
		return raw
	}
	if strings.EqualFold(raw, "follow_port") && maxLen >= int64(len("follow")) {
		return "follow"
	}
	if maxLen < 1 {
		return ""
	}
	if int64(len(raw)) > maxLen {
		return raw[:maxLen]
	}
	return raw
}

func normalizeBackendProtocolForSchema(site *models.Site) {
	if site == nil {
		return
	}
	site.BackendProtocol = normalizeBackendProtocolValue(site.BackendProtocol)
}

func isUnknownColumnError(err error, column string) bool {
	if err == nil || column == "" {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") && strings.Contains(msg, strings.ToLower(column))
}

func selectSiteCreateColumns(tx *gorm.DB, site *models.Site) []string {
	if tx == nil || site == nil {
		return nil
	}
	existing := map[string]struct{}{}
	if types, err := tx.Migrator().ColumnTypes(&models.Site{}); err == nil {
		for _, ct := range types {
			name := strings.TrimSpace(ct.Name())
			if name != "" {
				existing[name] = struct{}{}
			}
		}
	}

	stmt := &gorm.Statement{DB: tx}
	if err := stmt.Parse(site); err != nil || stmt.Schema == nil {
		return nil
	}
	cols := make([]string, 0, len(stmt.Schema.Fields))
	for _, field := range stmt.Schema.Fields {
		if !field.Creatable || field.DBName == "" {
			continue
		}
		if len(existing) > 0 {
			if _, ok := existing[field.DBName]; !ok {
				continue
			}
		} else if !tx.Migrator().HasColumn(&models.Site{}, field.DBName) {
			continue
		}

		switch field.DBName {
		case "cert_id":
			continue
		case "region_id":
			if site.RegionID == 0 {
				continue
			}
		case "node_group_id":
			if site.NodeGroupID == 0 {
				continue
			}
		case "backup_node_group":
			if !site.EnableBackupGroup || site.BackupNodeGroupID == 0 {
				continue
			}
		}
		cols = append(cols, field.DBName)
	}
	return cols
}

func filterSiteIDsForUser(ids []int64, userID int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	var allowed []int64
	if err := db.DB.Model(&models.Site{}).Where("uid = ? AND id IN ?", userID, ids).Pluck("id", &allowed).Error; err != nil {
		return nil, err
	}
	return allowed, nil
}

func filterSiteGroupIDsForUser(groupIDs []int64, userID int64) ([]int64, error) {
	if len(groupIDs) == 0 {
		return []int64{}, nil
	}
	var allowed []int64
	if err := db.DB.Model(&models.SiteGroup{}).Where("uid = ? AND id IN ?", userID, groupIDs).Pluck("id", &allowed).Error; err != nil {
		return nil, err
	}
	return allowed, nil
}

func ensureUserPackageOwnership(userID, packageID int64) error {
	if userID == 0 || packageID == 0 {
		return errors.New("invalid user/package")
	}
	var count int64
	if err := db.DB.Model(&models.UserPackage{}).Where("uid = ? AND id = ?", userID, packageID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user_package not found")
	}
	return nil
}

func siteMissingColumns(tx *gorm.DB) []string {
	migrator := tx.Migrator()
	missing := make([]string, 0, 3)
	if !migrator.HasColumn(&models.Site{}, "dns_provider_id") {
		missing = append(missing, "DNSProviderID")
	}
	if !migrator.HasColumn(&models.Site{}, "settings") {
		missing = append(missing, "SettingsRaw")
	}
	if !migrator.HasColumn(&models.Site{}, "cname_hostname2") {
		missing = append(missing, "CnameHostname2")
	}
	if !migrator.HasColumn(&models.Site{}, "cert_id") {
		missing = append(missing, "CertID")
	}
	return missing
}

func ensureDNSRecords(site *models.Site) error {
	if site == nil || len(site.Domains) == 0 {
		return nil
	}
	_, _ = refreshSiteCnameHostname(site, nil, nil)
	if site.DNSProviderID == 0 {
		return nil
	}
	return services.SyncUserDNSRecords(nil, site)
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

	selectCols := []string{
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
	}
	if db.DB.Migrator().HasColumn(&models.Site{}, "cert_id") {
		selectCols = append(selectCols, "cert_id")
	}
	query = query.Select(strings.Join(selectCols, ","))

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
	regionMap, err := loadRegions(sites)
	if err != nil {
		return nil, err
	}

	siteIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		if site.ID != 0 {
			siteIDs = append(siteIDs, site.ID)
		}
	}
	siteTypeMap := services.LoadSiteTypeMetaMap(siteIDs)

	items := make([]siteListItem, 0, len(sites))
	for i := range sites {
		site := sites[i]
		groupIDs := relMap[site.ID]
		groupID := int64(0)
		if len(groupIDs) > 0 {
			groupID = groupIDs[0]
		} else if site.GroupID != 0 {
			groupID = site.GroupID
		}
		applyDefaultsIfSettingsMissing(&site, groupID, siteTypeMap[site.ID])

		domains := site.Domains
		domainDisplay := strings.Join(domains, ",")
		if domainDisplay == "" && site.DomainRaw != "" {
			domainDisplay = site.DomainRaw
		}
		originDisplay := strings.Join(site.Backends, ",")
		if originDisplay == "" && site.BackendRaw != "" {
			originDisplay = strings.Trim(site.BackendRaw, "\"")
		}
		httpOn := len(site.HttpListen) > 0 || strings.TrimSpace(site.HttpListenRaw) != ""
		httpsOn := len(site.HttpsListen) > 0 || strings.TrimSpace(site.HttpsListenRaw) != ""
		if site.Settings != nil {
			if httpsCfg, ok := site.Settings["https"].(map[string]interface{}); ok {
				if enable, ok := httpsCfg["enable"]; ok {
					httpsOn = parseBoolValue(enable, httpsOn)
				}
			}
		}
		httpPorts := parseListenPorts(site.HttpListen, site.HttpListenRaw, "")
		httpsPorts := parseListenPorts(site.HttpsListen, site.HttpsListenRaw, "")
		if httpOn && len(httpPorts) == 0 {
			httpPorts = []string{"80"}
		}
		if httpsOn && len(httpsPorts) == 0 {
			httpsPorts = []string{"443"}
		}

		var listenParts []string
		if httpOn && len(httpPorts) > 0 {
			listenParts = append(listenParts, "HTTP:"+strings.Join(httpPorts, ","))
		}
		if httpsOn && len(httpsPorts) > 0 {
			listenParts = append(listenParts, "HTTPS:"+strings.Join(httpsPorts, ","))
		}
		listenPorts := strings.Join(listenParts, " ")

		pkg := pkgMap[site.UserPackageID]
		cname := strings.TrimSpace(site.CnameHostname)
		if cname == "" {
			cname = "-"
		}

		settings := site.Settings
		if settings == nil {
			settings = map[string]interface{}{}
		}
		mergeSecurityIPList(settings, "blacklist", site.BlackIPRaw)
		mergeSecurityIPList(settings, "whitelist", site.WhiteIPRaw)

		item := siteListItem{
			ID:              site.ID,
			UserID:          site.UserID,
			UserName:        userMap[site.UserID],
			Domains:         domains,
			DomainDisplay:   domainDisplay,
			ListenPorts:     listenPorts,
			HttpListen:      site.HttpListen,
			HttpsListen:     site.HttpsListen,
			OriginDisplay:   originDisplay,
			CNAME:           cname,
			Backends:        site.Backends,
			HTTPS:           httpsOn,
			CertID:          site.CertID,
			UserPackageID:   site.UserPackageID,
			UserPackageName: pkg.Name,
			DNSProviderID:   site.DNSProviderID,
			GroupID:         0,
			GroupIDs:        groupIDs,
			GroupName:       "",
			NodeGroupID:     site.NodeGroupID,
			NodeGroupName:   nodeGroupMap[site.NodeGroupID],
			RegionID:        site.RegionID,
			RegionName:      regionMap[site.RegionID],
			Status:          site.Enable,
			State:           site.State,
			Settings:        settings,
			ExpireTime:      pkg.EndAt.Format("2006-01-02"),
			CreatedAt:       site.CreatedAt,
			UpdatedAt:       site.UpdatedAt,
		}
		if len(item.GroupIDs) > 0 {
			item.GroupID = item.GroupIDs[0]
			item.GroupName = groupMap[item.GroupIDs[0]]
			names := make([]string, 0, len(item.GroupIDs))
			for _, gid := range item.GroupIDs {
				if name, ok := groupMap[gid]; ok {
					names = append(names, name)
				}
			}
			if len(names) > 0 {
				item.GroupName = strings.Join(names, ", ")
			}
		}
		items = append(items, item)
	}

	return items, nil
}
