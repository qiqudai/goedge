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
	var userPkg models.UserPackage
	if err := db.DB.First(&userPkg, req.UserPackageID).Error; err == nil {
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

	defaults, err := services.GetSiteDefaultMapWithGroup(userID, req.GroupID)
	if err != nil {
		return nil, nil, err
	}
	services.ApplySiteDefaults(site, defaults)
	if globalDefaults := services.GetGlobalDefaultConfig(); globalDefaults != nil {
		services.ApplySiteTemplateDefaults(site, globalDefaults.Website)
	}

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
	regionMap, err := loadRegions(sites)
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

		cname := strings.TrimSpace(site.CnameHostname)
		pkg := pkgMap[site.UserPackageID]

		// Priority: Site Mode > Package Mode > Default
		siteMode := strings.TrimSpace(site.CnameMode)
		pkgMode := strings.TrimSpace(pkg.CnameMode)

		isPkgMode := siteMode == "package" || (siteMode == "" && pkgMode == "package")

		if isPkgMode && pkg.CnameHostname != "" {
			cname = pkg.CnameHostname
			if pkg.CnameDomain != "" {
				cname += "." + pkg.CnameDomain
			} else if site.CnameDomain != "" {
				cname += "." + site.CnameDomain
			} else {
				cname += ".cdn.node.com"
			}
		} else {
			// Custom or Default mode
			// Reconstruct CNAME to ensure it reflects current CnameDomain (important for batch updates)
			if len(domains) > 0 && site.CnameDomain != "" {
				cname = buildSiteCname(domains[0], site.CnameDomain)
			}
		}
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
			UserPackageID:   site.UserPackageID,
			UserPackageName: pkg.Name,
			DNSProviderID:   site.DNSProviderID,
			GroupID:         0,
			GroupIDs:        relMap[site.ID],
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
