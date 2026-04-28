package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"gorm.io/gorm"
)

func setSettingMapValue(settings map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 || settings == nil {
		return
	}
	current := settings
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		next, ok := current[key].(map[string]interface{})
		if !ok || next == nil {
			next = map[string]interface{}{}
			current[key] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

// List returns the list of sites for the current user
func (ctrl *SiteController) List(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := parseInt64(userID)
	result, err := querySites(c, &uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to fetch sites")})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to build sites")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items, "total": result.Total}})
}

// AdminList returns the list of all sites for admin
func (ctrl *SiteController) AdminList(c *gin.Context) {
	result, err := querySites(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to fetch sites")})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to build sites")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items, "total": result.Total}})
}

func (ctrl *SiteController) AdminGet(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	var site models.Site
	query := withSiteColumns(db.DB).Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseInt64(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&site).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": T("site not found")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to fetch site")})
		return
	}

	// Fetch Groups
	var relations []models.SiteGroupRelation
	db.DB.Where("site_id = ?", id).Find(&relations)
	for _, rel := range relations {
		site.GroupIDs = append(site.GroupIDs, rel.GroupID)
	}
	if len(site.GroupIDs) > 0 {
		site.GroupID = site.GroupIDs[0]
	}

	siteTypeMeta := ""
	if site.Settings == nil || site.Settings["site_type"] == nil {
		siteTypeMeta = services.LoadSiteTypeMeta(site.ID)
	}
	applyDefaultsIfSettingsMissing(&site, site.GroupID, siteTypeMeta)

	// Enrich Site Data
	items, err := buildSiteListItems([]models.Site{site})
	if err != nil || len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"site": site}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"site": items[0]}})
}

// AdminUpdate updates a single site config
func (ctrl *SiteController) AdminUpdate(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	isUserReq := isUserRequest(c)
	userID := int64(0)
	if isUserReq {
		userID = parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	var oldSite models.Site
	if err := withSiteColumns(db.DB).Where("id = ?", id).First(&oldSite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load site")})
		return
	}
	if isUserReq && oldSite.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
		return
	}

	var req struct {
		IDs             []int64                `json:"ids"`
		UserPackageID   *int64                 `json:"user_package_id"`
		GroupID         *int64                 `json:"group_id"`
		GroupIDs        *[]int64               `json:"group_ids"`
		DNSProviderID   *int64                 `json:"dns_provider_id"`
		CertID          *int64                 `json:"cert_id"`
		HttpListen      *[]string              `json:"http_listen"`
		HttpsListen     *[]string              `json:"https_listen"`
		BalanceWay      *string                `json:"balance_way"`
		BackendProtocol *string                `json:"backend_protocol"`
		Domains         *[]string              `json:"domains"`
		Enable          *bool                  `json:"enable"`
		State           *string                `json:"state"`
		Backends        *[]string              `json:"backends"`
		Settings        map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if err := validateListenPortsAgainstPolicy(derefStringSlice(req.HttpListen), derefStringSlice(req.HttpsListen)); err != nil {
		if portErr, ok := err.(*listenPortPolicyError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": portErr.Error(), "data": portErr.ResponseData()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Domains != nil {
		normalizedDomains := normalizeSiteDomains(*req.Domains)
		if len(normalizedDomains) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("domain is required")})
			return
		}
		req.Domains = &normalizedDomains
	}

	if isUserReq && req.UserPackageID != nil && *req.UserPackageID > 0 {
		if err := ensureUserPackageOwnership(userID, *req.UserPackageID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	if isUserReq && (req.GroupIDs != nil || req.GroupID != nil) {
		groupIDs := []int64{}
		if req.GroupIDs != nil {
			groupIDs = *req.GroupIDs
		} else if req.GroupID != nil && *req.GroupID != 0 {
			groupIDs = []int64{*req.GroupID}
		}
		if len(groupIDs) > 0 {
			allowed, err := filterSiteGroupIDsForUser(groupIDs, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load groups")})
				return
			}
			if len(allowed) != len(groupIDs) {
				c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
				return
			}
			if req.GroupIDs != nil {
				req.GroupIDs = &allowed
			} else if req.GroupID != nil {
				if len(allowed) == 0 {
					zero := int64(0)
					req.GroupID = &zero
				} else {
					req.GroupID = &allowed[0]
				}
			}
		}
	}

	if req.Domains != nil || req.UserPackageID != nil {
		domainsForCheck := oldSite.Domains
		if req.Domains != nil {
			domainsForCheck = *req.Domains
		}

		packageID := oldSite.UserPackageID
		if req.UserPackageID != nil && *req.UserPackageID > 0 {
			packageID = *req.UserPackageID
		}

		if err := services.CheckDomainLimitForUpdate(oldSite.UserID, packageID, oldSite.ID, domainsForCheck); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}
	}

	ccDefaultFromSettings, hasCCDefault := extractCCDefaultRuleFromSettings(req.Settings)
	blacklistFromSettings, hasBlacklist := extractSecurityIPList(req.Settings, "blacklist")
	whitelistFromSettings, hasWhitelist := extractSecurityIPList(req.Settings, "whitelist")
	if hasBlacklist {
		setSecurityIPList(req.Settings, "blacklist", blacklistFromSettings)
	}
	if hasWhitelist {
		setSecurityIPList(req.Settings, "whitelist", whitelistFromSettings)
	}
	if req.Settings != nil {
		delete(req.Settings, "backend_protocol")
		services.NormalizeSiteSettings(req.Settings)
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.UserPackageID != nil && *req.UserPackageID > 0 {
			updates["user_package"] = *req.UserPackageID
		}
		if req.DNSProviderID != nil && tx.Migrator().HasColumn(&models.Site{}, "dns_provider_id") {
			updates["dns_provider_id"] = *req.DNSProviderID
		}
		if req.CertID != nil && tx.Migrator().HasColumn(&models.Site{}, "cert_id") {
			updates["cert_id"] = *req.CertID
		}
		if req.HttpListen != nil {
			updates["http_listen"] = encodeList(*req.HttpListen)
		}
		if req.HttpsListen != nil {
			updates["https_listen"] = encodeList(*req.HttpsListen)
		}
		if req.Backends != nil {
			updates["backend"] = encodeList(*req.Backends)
		}
		if req.BalanceWay != nil {
			updates["balance_way"] = *req.BalanceWay
		}
		if req.BackendProtocol != nil {
			updates["backend_protocol"] = normalizeBackendProtocolValue(*req.BackendProtocol)
		}
		if req.Domains != nil {
			updates["domain"] = encodeList(*req.Domains)
		}
		if req.Enable != nil {
			updates["enable"] = *req.Enable
			if *req.Enable {
				updates["state"] = "running"
			} else {
				updates["state"] = "stop"
			}
		}
		if req.State != nil {
			state := strings.ToLower(strings.TrimSpace(*req.State))
			switch state {
			case "running", "stop", "locked", "site_locked", "traffic_limit", "conn_limit", "expired", "timeout":
				updates["state"] = state
			}
		}

		if req.Settings != nil {
			b, _ := json.Marshal(req.Settings)
			if tx.Migrator().HasColumn(&models.Site{}, "settings") {
				updates["settings"] = string(b)
			} else {
				updates["SettingsRaw"] = string(b)
			}
		}
		if hasCCDefault {
			updates["cc_default_rule"] = ccDefaultFromSettings
		}
		if hasBlacklist {
			updates["black_ip"] = encodeList(blacklistFromSettings)
		}
		if hasWhitelist {
			updates["white_ip"] = encodeList(whitelistFromSettings)
		}

		updates["update_at"] = time.Now()

		if err := tx.Model(&models.Site{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		// Update Groups
		if req.GroupIDs != nil || req.GroupID != nil {
			// Determine final group IDs
			var finalGroupIDs []int64
			if req.GroupIDs != nil {
				finalGroupIDs = *req.GroupIDs
			} else if req.GroupID != nil && *req.GroupID != 0 {
				finalGroupIDs = []int64{*req.GroupID}
			}

			// Delete old relations
			if err := tx.Where("site_id = ?", id).Delete(&models.SiteGroupRelation{}).Error; err != nil {
				return err
			}

			// Create new relations
			if len(finalGroupIDs) > 0 {
				for _, gid := range finalGroupIDs {
					if gid != 0 {
						rel := models.SiteGroupRelation{SiteID: id, GroupID: gid}
						if err := tx.Create(&rel).Error; err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Update failed")})
		return
	}

	services.BumpConfigVersion("site", []int64{id})
	var newSite models.Site
	if err := db.DB.Where("id = ?", id).First(&newSite).Error; err == nil {
		_, _ = refreshSiteCnameHostname(&newSite, nil, nil)
		_ = services.SyncUserDNSRecords(&oldSite, &newSite)
		if shouldResyncSiteCname(oldSite, newSite) {
			resyncSiteCnameForSite(newSite)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": T("Site updated")})
}

// AdminCreate handles site creation for admin
func (ctrl *SiteController) AdminCreate(c *gin.Context) {
	site, groupIDs, err := parseSiteCreateRequest(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if err := createSiteWithGroup(site, groupIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if !db.DB.Migrator().HasColumn(&models.Site{}, "settings") {
		siteType := ""
		if site.Settings != nil {
			siteType = strings.TrimSpace(fmt.Sprint(site.Settings["site_type"]))
		}
		if siteType != "" {
			_ = services.UpsertSiteTypeMeta(site.ID, siteType)
		}
	}

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)
	resyncSiteCnameForSite(*site)

	c.JSON(http.StatusOK, gin.H{"message": T("Site created successfully"), "data": site})
}

// Create handles site creation for user
func (ctrl *SiteController) Create(c *gin.Context) {
	site, groupIDs, err := parseSiteCreateRequest(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if err := createSiteWithGroup(site, groupIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	if !db.DB.Migrator().HasColumn(&models.Site{}, "settings") {
		siteType := ""
		if site.Settings != nil {
			siteType = strings.TrimSpace(fmt.Sprint(site.Settings["site_type"]))
		}
		if siteType != "" {
			_ = services.UpsertSiteTypeMeta(site.ID, siteType)
		}
	}

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)
	resyncSiteCnameForSite(*site)

	c.JSON(http.StatusOK, gin.H{"message": T("Site created successfully"), "data": site})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if isUserRequest(c) {
		req.UserID = parseInt64(mustGet(c, "userID"))
	} else if req.UserID == 0 {
		req.UserID = parseInt64(mustGet(c, "userID"))
	}
	if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
		return
	}
	if req.UserPackageID == 0 {
		defaultID, err := findDefaultUserPackageID(req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}
		req.UserPackageID = defaultID
	}
	if isUserRequest(c) && req.UserPackageID > 0 {
		if err := ensureUserPackageOwnership(req.UserID, req.UserPackageID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}
	if strings.TrimSpace(req.Data) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("data is required")})
		return
	}

	nodeGroupID, err := resolveNodeGroupFromPackage(req.UserPackageID, req.NodeGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}

	// defaults, err := services.GetSiteDefaultMapWithGroup(req.UserID, req.GroupID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load defaults")})
	// 	return
	// }

	// Generate BatchID
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	batchID := hex.EncodeToString(b)

	lines := splitLines(req.Data)
	created := 0
	var createdTasks []*models.Task
	allDomains := make([]string, 0, len(lines))
	batchItems := make([]*batchSiteItem, 0, len(lines))

	for _, line := range lines {
		item, err := parseBatchLine(line)
		if err != nil {
			if req.IgnoreError {
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}
		batchItems = append(batchItems, item)
		allDomains = append(allDomains, item.Domains...)
	}

	if err := services.CheckDomainLimit(req.UserID, req.UserPackageID, allDomains); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
		return
	}
	if conflictDomain, err := findSiteDomainConflict(db.DB, allDomains, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to validate domain uniqueness")})
		return
	} else if conflictDomain != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", T(i18n.T("site.domain_exists")), conflictDomain)})
		return
	}

	for _, item := range batchItems {
		for _, domain := range item.Domains {
			payload := services.SiteCreatePayload{
				UserID:        req.UserID,
				UserPackageID: req.UserPackageID,
				DNSProviderID: req.DNSProviderID,
				NodeGroupID:   nodeGroupID,
				GroupID:       req.GroupID,
				Domain:        domain,
				Backends:      item.Backends,
			}

			if task, err := services.CreateSiteCreateTask(payload, batchID); err != nil {
				fmt.Printf("Failed to create site task: %v\n", err)
			} else {
				created++
				createdTasks = append(createdTasks, task)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": T("Batch create submitted"), "batch_id": batchID, "created": created, "tasks": createdTasks})
}

// AdminBatchProgress returns the progress of a batch task
func (ctrl *SiteController) AdminBatchProgress(c *gin.Context) {
	batchID := c.Param("id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("batch_id is required")})
		return
	}

	var tasks []models.Task
	if err := db.DB.Where("batch_id = ?", batchID).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to query tasks")})
		return
	}

	total := len(tasks)
	success := 0
	fail := 0
	running := 0
	pending := 0

	type FailItem struct {
		Domain string `json:"domain"`
		Reason string `json:"reason"`
	}
	var failItems []FailItem

	for _, t := range tasks {
		switch t.State {
		case "success":
			success++
		case "fail":
			fail++

			var payload services.SiteCreatePayload
			_ = json.Unmarshal([]byte(t.Data), &payload)
			domain := payload.Domain
			if domain == "" {
				domain = "Unknown"
			}
			failItems = append(failItems, FailItem{
				Domain: domain,
				Reason: t.Ret, // Use Ret column
			})
		case "running", "retrying":
			running++
		default:
			pending++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":      total,
		"success":    success,
		"fail":       fail,
		"running":    running,
		"pending":    pending,
		"done":       success + fail,
		"percent":    0,
		"fail_items": failItems,
	})
}

// AdminBatchUpdate updates fields for selected sites
func (ctrl *SiteController) AdminBatchUpdate(c *gin.Context) {
	var req struct {
		IDs               []int64                `json:"ids"`
		UserPackageID     *int64                 `json:"user_package_id"`
		GroupID           *int64                 `json:"group_id"`
		GroupIDs          *[]int64               `json:"group_ids"`
		DNSProviderID     *int64                 `json:"dns_provider_id"`
		CertID            *int64                 `json:"cert_id"`
		HttpListen        *[]string              `json:"http_listen"`
		HttpsListen       *[]string              `json:"https_listen"`
		BalanceWay        *string                `json:"balance_way"`
		BackendProtocol   *string                `json:"backend_protocol"`
		Backends          *[]string              `json:"backends"`
		CcDefaultRule     *int64                 `json:"cc_default_rule"`
		BlackIP           *string                `json:"black_ip"`
		WhiteIP           *string                `json:"white_ip"`
		BlockRegion       *string                `json:"block_region"`
		Settings          map[string]interface{} `json:"settings"`
		CnameDomain       *string                `json:"cname_domain"`
		CnameMode         *string                `json:"cname_mode"`
		RegionID          *int64                 `json:"region_id"`
		NodeGroupID       *int64                 `json:"node_group_id"`
		BackupNodeGroupID *int64                 `json:"backup_node_group_id"`
		EnableBackupGroup *bool                  `json:"enable_backup_group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if err := validateListenPortsAgainstPolicy(derefStringSlice(req.HttpListen), derefStringSlice(req.HttpsListen)); err != nil {
		if portErr, ok := err.(*listenPortPolicyError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": portErr.Error(), "data": portErr.ResponseData()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
	}
	if isUserRequest(c) {
		userID := parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		allowed, err := filterSiteIDsForUser(req.IDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
			return
		}
		if len(allowed) == 0 || len(allowed) != len(req.IDs) {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		req.IDs = allowed
		if req.UserPackageID != nil && *req.UserPackageID > 0 {
			if err := ensureUserPackageOwnership(userID, *req.UserPackageID); err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
				return
			}
		}
		if req.GroupIDs != nil || req.GroupID != nil {
			groupIDs := []int64{}
			if req.GroupIDs != nil {
				groupIDs = *req.GroupIDs
			} else if req.GroupID != nil && *req.GroupID != 0 {
				groupIDs = []int64{*req.GroupID}
			}
			if len(groupIDs) > 0 {
				allowedGroups, err := filterSiteGroupIDsForUser(groupIDs, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load groups")})
					return
				}
				if len(allowedGroups) != len(groupIDs) {
					c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
					return
				}
				if req.GroupIDs != nil {
					req.GroupIDs = &allowedGroups
				} else if req.GroupID != nil {
					req.GroupID = &allowedGroups[0]
				}
			}
		}
	}

	if req.CnameDomain != nil {
		if err := ensureCnameTable(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to init cname table")})
			return
		}
		normalized := normalizeDomainInput(*req.CnameDomain)
		if normalized == "" || !isValidDomain(normalized) {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid cname_domain")})
			return
		}
		var cd models.CnameDomain
		if err := db.DB.Where("domain = ?", normalized).First(&cd).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": T("cname_domain not found")})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to validate cname_domain")})
			return
		}
		req.CnameDomain = &normalized
	}

	ccDefaultFromSettings, hasCCDefault := extractCCDefaultRuleFromSettings(req.Settings)
	blacklistFromSettings, hasBlacklist := extractSecurityIPList(req.Settings, "blacklist")
	whitelistFromSettings, hasWhitelist := extractSecurityIPList(req.Settings, "whitelist")
	blacklistFromInput := []string(nil)
	whitelistFromInput := []string(nil)
	if req.BlackIP != nil {
		blacklistFromInput = parseStringListValue(*req.BlackIP)
	}
	if req.WhiteIP != nil {
		whitelistFromInput = parseStringListValue(*req.WhiteIP)
	}
	if req.BlackIP != nil {
		setSecurityIPList(req.Settings, "blacklist", blacklistFromInput)
	} else if hasBlacklist {
		setSecurityIPList(req.Settings, "blacklist", blacklistFromSettings)
	}
	if req.WhiteIP != nil {
		setSecurityIPList(req.Settings, "whitelist", whitelistFromInput)
	} else if hasWhitelist {
		setSecurityIPList(req.Settings, "whitelist", whitelistFromSettings)
	}
	if req.Settings != nil {
		delete(req.Settings, "backend_protocol")
		services.NormalizeSiteSettings(req.Settings)
	}
	if req.CertID != nil {
		if req.Settings == nil {
			req.Settings = map[string]interface{}{}
		}
		setSettingMapValue(req.Settings, []string{"https", "certificate_id"}, *req.CertID)
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.UserPackageID != nil {
			updates["user_package"] = *req.UserPackageID
		}
		if req.DNSProviderID != nil && tx.Migrator().HasColumn(&models.Site{}, "dns_provider_id") {
			updates["dns_provider_id"] = *req.DNSProviderID
		}
		if req.CertID != nil && tx.Migrator().HasColumn(&models.Site{}, "cert_id") {
			updates["cert_id"] = *req.CertID
		}
		if req.HttpListen != nil {
			updates["http_listen"] = encodeList(*req.HttpListen)
		}
		if req.HttpsListen != nil {
			updates["https_listen"] = encodeList(*req.HttpsListen)
		}
		if req.Backends != nil {
			updates["backend"] = encodeList(*req.Backends)
		}
		if req.BalanceWay != nil {
			updates["balance_way"] = *req.BalanceWay
		}
		if req.BackendProtocol != nil {
			updates["backend_protocol"] = normalizeBackendProtocolValue(*req.BackendProtocol)
		}
		if req.CcDefaultRule != nil {
			updates["cc_default_rule"] = *req.CcDefaultRule
			if req.Settings != nil {
				setCCDefaultRuleInSettings(req.Settings, *req.CcDefaultRule)
			}
		} else if hasCCDefault {
			updates["cc_default_rule"] = ccDefaultFromSettings
		}
		if req.BlackIP != nil {
			updates["black_ip"] = encodeList(blacklistFromInput)
		} else if hasBlacklist {
			updates["black_ip"] = encodeList(blacklistFromSettings)
		}
		if req.WhiteIP != nil {
			updates["white_ip"] = encodeList(whitelistFromInput)
		} else if hasWhitelist {
			updates["white_ip"] = encodeList(whitelistFromSettings)
		}
		if req.BlockRegion != nil {
			updates["block_region"] = *req.BlockRegion
		}
		if req.CnameDomain != nil {
			updates["cname_domain"] = *req.CnameDomain
		}
		if req.CnameMode != nil {
			updates["cname_mode"] = *req.CnameMode
		}

		// Recalculate CnameHostname when CNAME fields change.
		if req.CnameDomain != nil || req.CnameMode != nil {
			// Load sites to recalculate CNAME.
			var sites []models.Site
			if err := tx.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
				return err
			}

			for _, site := range sites {
				pkgID := site.UserPackageID
				if req.UserPackageID != nil && *req.UserPackageID > 0 {
					pkgID = *req.UserPackageID
				}

				var pkg models.UserPackage
				if err := tx.First(&pkg, pkgID).Error; err != nil {
					continue
				}

				newCnameHostname := computeSiteCnameHostname(site, pkg, req.CnameMode, req.CnameDomain)

				if newCnameHostname != site.CnameHostname {
					if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Update("cname_hostname", newCnameHostname).Error; err != nil {
						return err
					}
				}
			}
		}
		if req.RegionID != nil {
			updates["region_id"] = *req.RegionID
		}
		if req.NodeGroupID != nil {
			updates["node_group_id"] = *req.NodeGroupID
		}
		if req.BackupNodeGroupID != nil {
			updates["backup_node_group"] = *req.BackupNodeGroupID
		}
		if req.EnableBackupGroup != nil {
			updates["enable_backup_group"] = *req.EnableBackupGroup
		}
		if req.Settings != nil {
			var sites []models.Site
			if err := tx.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
				return err
			}
			for _, site := range sites {
				merged := mergeSettingsMaps(site.Settings, req.Settings)
				b, _ := json.Marshal(merged)
				siteUpdates := copyUpdateMap(updates)
				siteUpdates["settings"] = string(b)
				if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Updates(siteUpdates).Error; err != nil {
					return err
				}
			}
		} else if len(updates) > 0 {
			if err := tx.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.GroupID != nil || req.GroupIDs != nil {
			if err := tx.Where("site_id IN ?", req.IDs).Delete(&models.SiteGroupRelation{}).Error; err != nil {
				return err
			}
			finalGroupIDs := []int64{}
			if req.GroupIDs != nil {
				finalGroupIDs = *req.GroupIDs
			} else if req.GroupID != nil && *req.GroupID != 0 {
				finalGroupIDs = []int64{*req.GroupID}
			}
			if len(finalGroupIDs) > 0 {
				relations := make([]models.SiteGroupRelation, 0, len(req.IDs)*len(finalGroupIDs))
				for _, id := range req.IDs {
					for _, gid := range finalGroupIDs {
						if gid == 0 {
							continue
						}
						relations = append(relations, models.SiteGroupRelation{SiteID: id, GroupID: gid})
					}
				}
				if len(relations) > 0 {
					if err := tx.Create(&relations).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Batch update failed")})
		return
	}

	services.BumpConfigVersion("site", req.IDs)
	needResync := req.UserPackageID != nil || req.CnameDomain != nil || req.CnameMode != nil || req.NodeGroupID != nil || req.BackupNodeGroupID != nil || req.EnableBackupGroup != nil
	if needResync {
		var sites []models.Site
		if err := db.DB.Where("id IN ?", req.IDs).Find(&sites).Error; err == nil {
			for _, site := range sites {
				resyncSiteCnameForSite(site)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": T("Batch update completed")})
}

// AdminBatchAction handles enable/disable/delete etc
func (ctrl *SiteController) AdminBatchAction(c *gin.Context) {
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
	userID := int64(0)
	if isUserRequest(c) {
		userID = parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		allowed, err := filterSiteIDsForUser(req.IDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
			return
		}
		if len(allowed) == 0 || len(allowed) != len(req.IDs) {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		req.IDs = allowed
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "enable":
		if err := db.DB.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": true,
			"state":  "running",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Update failed")})
			return
		}
	case "disable":
		if err := db.DB.Model(&models.Site{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"enable": false,
			"state":  "stop",
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Update failed")})
			return
		}
	case "delete":
		var sites []models.Site
		_ = db.DB.Where("id IN ?", req.IDs).Find(&sites).Error
		for _, site := range sites {
			if site.Enable {
				c.JSON(http.StatusBadRequest, gin.H{"error": T("Please disable site before delete")})
				return
			}
		}
		for _, site := range sites {
			_ = services.SyncUserDNSRecords(&site, nil)
		}
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("site_id IN ?", req.IDs).Delete(&models.SiteGroupRelation{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.IDs).Delete(&models.Site{}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Delete failed")})
			return
		}
	case "unlock":
		// No-op for now; placeholder to keep UI consistent
	case "clear_cache":
		// Create a cache clear task that will be pulled by all agents.
		now := time.Now()
		var sites []models.Site
		if err := db.DB.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
			return
		}
		domainSet := make(map[string]struct{})
		domains := make([]string, 0, len(sites))
		for _, site := range sites {
			for _, domain := range site.Domains {
				trimmed := strings.TrimSpace(domain)
				if trimmed == "" {
					continue
				}
				if _, exists := domainSet[trimmed]; exists {
					continue
				}
				domainSet[trimmed] = struct{}{}
				domains = append(domains, trimmed)
			}
		}
		payload := map[string]interface{}{
			"action":   "clear_cache",
			"site_ids": req.IDs,
			"domains":  domains,
		}
		raw, _ := json.Marshal(payload)
		metaRaw := ""
		if userID != 0 {
			meta, _ := json.Marshal(map[string]int64{"user_id": userID})
			metaRaw = string(meta)
		}
		task := models.Task{
			Type:     "clear_cache",
			Name:     i18n.T("cache.clear_task_name"),
			Data:     string(raw),
			Res:      metaRaw,
			State:    "waiting",
			Enable:   true,
			CreateAt: now,
			RetryAt:  &now,
		}
		if targets := buildConnectedTargetsJSON(); targets != "" {
			task.TargetsJSON = targets
		}
		if err := db.DB.Create(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Create task failed")})
			return
		}
		services.TriggerDispatchPending()
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"task_id": task.ID}})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Unknown action")})
		return
	}

	if action != "unlock" {
		services.BumpConfigVersion("site", req.IDs)
	}

	c.JSON(http.StatusOK, gin.H{"message": T("Action completed")})
}

// AdminApplyCert sets HTTPS listen ports for selected sites
func (ctrl *SiteController) AdminApplyCert(c *gin.Context) {
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
	if isUserRequest(c) {
		userID := parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		allowed, err := filterSiteIDsForUser(req.IDs, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
			return
		}
		if len(allowed) == 0 || len(allowed) != len(req.IDs) {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		req.IDs = allowed
	}

	var sites []models.Site
	if err := db.DB.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
		return
	}
	if len(sites) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": T("site not found")})
		return
	}

	defaultCertPEM := ""
	defaultKeyPEM := ""
	defaultKeyEncrypted := ""
	hasDefaultCert := false
	if systemCfg, err := services.LoadSystemConfig(); err == nil {
		defaultCertPEM = strings.TrimSpace(systemCfg["https_cert"])
		defaultKeyPEM = strings.TrimSpace(systemCfg["https_key"])
		if defaultKeyPEM != "" {
			if encryptedKey, err := services.Crypto.Encrypt(defaultKeyPEM); err == nil {
				defaultKeyEncrypted = encryptedKey
			} else {
				defaultKeyEncrypted = defaultKeyPEM
			}
		}
		hasDefaultCert = defaultCertPEM != "" && defaultKeyEncrypted != ""
	}

	type applyCertSkipItem struct {
		SiteID int64  `json:"site_id"`
		Domain string `json:"domain"`
		Reason string `json:"reason"`
	}

	skipped := make([]applyCertSkipItem, 0, len(sites))
	createdIDs := make([]int64, 0, len(sites))
	appliedSiteIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		if len(site.Domains) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("site domains are empty")})
			return
		}
		if isSiteHTTPSOn(site) {
			skipped = append(skipped, applyCertSkipItem{
				SiteID: site.ID,
				Domain: site.Domains[0],
				Reason: "该网站已开启https，已忽略，继续下一个申请",
			})
			continue
		}

		certType, dnsapi := resolveCertDefaults(site.UserID)
		if !hasWildcardDomain(site.Domains) {
			dnsapi = 0
		}

		if err := ensureNoExistingCert(site.UserID, site.Domains); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}
		cert := models.Cert{
			UserID:      int(site.UserID),
			Name:        defaultCertName(site.Domains[0]),
			Description: fmt.Sprintf("site_id:%d", site.ID),
			Type:        certType,
			Domain:      strings.Join(site.Domains, ","),
			DNSAPI:      normalizeDNSAPIValue(dnsapi),
			AutoRenew:   false,
			Enable:      true,
			State:       "waiting",
			CreateAt:    time.Now(),
			UpdateAt:    time.Now(),
		}
		if hasDefaultCert {
			cert.Cert = defaultCertPEM
			cert.Key = defaultKeyEncrypted
		}

		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Select(
				"UserID",
				"Name",
				"Description",
				"Type",
				"Domain",
				"DNSAPI",
				"Cert",
				"Key",
				"StartTime",
				"ExpireTime",
				"AutoRenew",
				"CreateAt",
				"UpdateAt",
				"Enable",
				"State",
				"Version",
			).Create(&cert).Error; err != nil {
				return err
			}
			now := time.Now()
			siteUpdates := map[string]interface{}{
				"update_at": now,
			}
			settings := site.Settings
			if settings == nil {
				settings = map[string]interface{}{}
			}
			httpsCfg, ok := settings["https"].(map[string]interface{})
			if !ok || httpsCfg == nil {
				httpsCfg = map[string]interface{}{}
				settings["https"] = httpsCfg
			}
			httpsCfg["enable"] = true
			httpsCfg["certificate_id"] = cert.ID
			if settingsRaw, err := json.Marshal(settings); err == nil {
				if tx.Migrator().HasColumn(&models.Site{}, "settings") {
					siteUpdates["settings"] = string(settingsRaw)
				} else {
					siteUpdates["SettingsRaw"] = string(settingsRaw)
				}
			}
			if len(site.HttpsListen) == 0 && strings.TrimSpace(site.HttpsListenRaw) == "" {
				siteUpdates["https_listen"] = encodeList([]string{"443"})
			}
			if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Updates(siteUpdates).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create cert")})
			return
		}

		createdIDs = append(createdIDs, int64(cert.ID))
		appliedSiteIDs = append(appliedSiteIDs, site.ID)
	}

	if len(appliedSiteIDs) > 0 {
		services.BumpConfigVersion("site", appliedSiteIDs)
	}
	if len(createdIDs) > 0 {
		services.BumpConfigVersion("cert", createdIDs)
		services.IssueCertsAsync(time.Now().Unix(), createdIDs)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": T("Certificate apply queued"),
		"data": gin.H{
			"created_ids": createdIDs,
			"skipped":     skipped,
		},
	})
}

func isSiteHTTPSOn(site models.Site) bool {
	httpsOn := len(site.HttpsListen) > 0 || strings.TrimSpace(site.HttpsListenRaw) != ""
	if site.Settings != nil {
		if httpsCfg, ok := site.Settings["https"].(map[string]interface{}); ok {
			if enable, ok := httpsCfg["enable"]; ok {
				httpsOn = parseBoolValue(enable, httpsOn)
			}
		}
	}
	return httpsOn
}

func ensureNoExistingCert(userID int64, domains []string) error {
	if len(domains) == 0 {
		return errors.New("domain is required")
	}
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if exists, err := certExistsForDomain(userID, domain); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("certificate already exists for domain %s", domain)
		}
	}
	return nil
}

func certExistsForDomain(userID int64, domain string) (bool, error) {
	var cert models.Cert
	patterns := []string{
		domain,
		domain + ",%",
		"%," + domain,
		"%," + domain + ",%",
	}
	query := db.DB.Where("uid = ?", userID).
		Where("(domain = ? OR domain LIKE ? OR domain LIKE ? OR domain LIKE ?)", patterns[0], patterns[1], patterns[2], patterns[3])
	if err := query.First(&cert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func resolveCertDefaults(userID int64) (string, int) {
	if userID != 0 {
		if settings, err := loadCertDefaultSettings("system", "user", int(userID)); err == nil && settings != nil {
			return normalizeCertType(settings.Type), settings.DNSAPI
		}
	}
	if settings, err := loadCertDefaultSettings("system", "global", 0); err == nil && settings != nil {
		return normalizeCertType(settings.Type), settings.DNSAPI
	}
	return "letsencrypt", 0
}

func extractCCDefaultRuleFromSettings(settings map[string]interface{}) (int64, bool) {
	if settings == nil {
		return 0, false
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		return 0, false
	}
	raw, ok := security["default_rule"]
	if !ok {
		return 0, false
	}
	return parseInt64(raw), true
}

func setCCDefaultRuleInSettings(settings map[string]interface{}, ruleID int64) {
	if settings == nil {
		return
	}
	security, ok := settings["security"].(map[string]interface{})
	if !ok || security == nil {
		security = map[string]interface{}{}
		settings["security"] = security
	}
	security["default_rule"] = ruleID
}

// AdminExport exports list as CSV
func (ctrl *SiteController) AdminExport(c *gin.Context) {
	if isUserRequest(c) {
		userID := parseInt64(mustGet(c, "userID"))
		if userID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		result, err := querySites(c, &userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to export")})
			return
		}
		items, err := buildSiteListItems(result.Sites)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to export")})
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
		return
	}

	result, err := querySites(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to export")})
		return
	}
	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to export")})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": T("domain is required")})
		return
	}
	expectedCNAME := strings.TrimSpace(c.Query("expected_cname"))

	testedDomain, wildcardTested := buildResolveTestDomain(domain)
	cnameChain, ips, dnsErr, httpOK, httpURL, httpStatus, httpErr := resolveDomainReport(testedDomain)
	finalCNAME := ""
	if len(cnameChain) > 0 {
		finalCNAME = cnameChain[len(cnameChain)-1]
	}
	matchedExpected := cnameMatchesExpected(expectedCNAME, testedDomain, cnameChain)

	c.JSON(http.StatusOK, gin.H{
		"domain":                 domain,
		"tested_domain":          testedDomain,
		"wildcard_tested":        wildcardTested,
		"cname":                  finalCNAME,
		"cname_chain":            cnameChain,
		"ips":                    ips,
		"dns_ok":                 len(cnameChain) > 0 || len(ips) > 0,
		"dns_error":              resolveErrText(dnsErr),
		"http_ok":                httpOK,
		"http_url":               httpURL,
		"http_status":            httpStatus,
		"http_error":             resolveErrText(httpErr),
		"expected_cname":         expectedCNAME,
		"cname_matches_expected": matchedExpected,
	})
}

func resolveDomainReport(host string) ([]string, []string, error, bool, string, int, error) {
	type dnsResult struct {
		cnameChain []string
		ips        []string
		err        error
	}
	type httpResult struct {
		ok     bool
		url    string
		status int
		err    error
	}

	dnsCh := make(chan dnsResult, 1)
	httpCh := make(chan httpResult, 1)

	go func() {
		cnameChain, ips, err := lookupDNSReport(host)
		dnsCh <- dnsResult{cnameChain: cnameChain, ips: ips, err: err}
	}()
	go func() {
		ok, url, status, err := probeHTTPReachability(host)
		httpCh <- httpResult{ok: ok, url: url, status: status, err: err}
	}()

	dnsRes := <-dnsCh
	httpRes := <-httpCh
	return dnsRes.cnameChain, dnsRes.ips, dnsRes.err, httpRes.ok, httpRes.url, httpRes.status, httpRes.err
}

func buildResolveTestDomain(domain string) (string, bool) {
	domain = normalizeResolveHost(domain)
	if !strings.HasPrefix(domain, "*.") {
		return domain, false
	}
	root := strings.TrimSpace(strings.TrimPrefix(domain, "*."))
	if root == "" {
		return domain, false
	}
	return randomResolveLabel(8) + "." + root, true
}

func randomResolveLabel(size int) string {
	if size <= 0 {
		size = 8
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "resolvecheck"
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

func normalizeResolveHost(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	if idx := strings.Index(value, "/"); idx >= 0 {
		value = value[:idx]
	}
	if idx := strings.Index(value, "?"); idx >= 0 {
		value = value[:idx]
	}
	if idx := strings.Index(value, "#"); idx >= 0 {
		value = value[:idx]
	}
	value = strings.TrimSuffix(value, ".")
	return value
}

func appendUniqueResolveValue(values []string, value string) []string {
	value = normalizeResolveHost(value)
	if value == "" {
		return values
	}
	for _, item := range values {
		if normalizeResolveHost(item) == value {
			return values
		}
	}
	return append(values, value)
}

func systemResolverAddrs() []string {
	cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || cfg == nil || len(cfg.Servers) == 0 {
		return []string{"127.0.0.1:53"}
	}
	addrs := make([]string, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		addrs = append(addrs, net.JoinHostPort(server, cfg.Port))
	}
	if len(addrs) == 0 {
		return []string{"127.0.0.1:53"}
	}
	return addrs
}

func queryDNS(host string, qType uint16) ([]dns.RR, error) {
	fqdn := dns.Fqdn(host)
	client := &dns.Client{Timeout: 3 * time.Second}
	msg := &dns.Msg{}
	msg.SetQuestion(fqdn, qType)
	msg.RecursionDesired = true

	var lastErr error
	for _, resolver := range systemResolverAddrs() {
		resp, _, err := client.Exchange(msg, resolver)
		if err != nil {
			lastErr = err
			continue
		}
		if resp == nil {
			lastErr = errors.New("empty dns response")
			continue
		}
		if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
			lastErr = fmt.Errorf("dns rcode %s", dns.RcodeToString[resp.Rcode])
			continue
		}
		answers := make([]dns.RR, 0, len(resp.Answer)+len(resp.Extra))
		answers = append(answers, resp.Answer...)
		answers = append(answers, resp.Extra...)
		return answers, nil
	}
	if lastErr == nil {
		lastErr = errors.New("dns lookup failed")
	}
	return nil, lastErr
}

func lookupDNSReport(host string) ([]string, []string, error) {
	host = normalizeResolveHost(host)
	if host == "" {
		return nil, nil, errors.New("empty host")
	}

	cnameChain := make([]string, 0, 4)
	visited := map[string]struct{}{}
	current := host
	for i := 0; i < 8; i++ {
		normalizedCurrent := normalizeResolveHost(current)
		if normalizedCurrent == "" {
			break
		}
		if _, ok := visited[normalizedCurrent]; ok {
			break
		}
		visited[normalizedCurrent] = struct{}{}

		answers, err := queryDNS(normalizedCurrent, dns.TypeCNAME)
		if err != nil {
			if len(cnameChain) == 0 {
				break
			}
			break
		}

		next := ""
		for _, answer := range answers {
			if record, ok := answer.(*dns.CNAME); ok {
				next = normalizeResolveHost(record.Target)
				break
			}
		}
		if next == "" {
			break
		}
		cnameChain = appendUniqueResolveValue(cnameChain, next)
		current = next
	}

	type ipLookupResult struct {
		ips []string
		err error
	}
	hostLookupCh := make(chan ipLookupResult, 1)
	aLookupCh := make(chan ipLookupResult, 1)
	aaaaLookupCh := make(chan ipLookupResult, 1)

	go func() {
		ips := make([]string, 0, 4)
		hosts, err := net.LookupHost(host)
		if err == nil {
			for _, item := range hosts {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				ips = appendUniqueResolveValue(ips, item)
			}
		}
		hostLookupCh <- ipLookupResult{ips: ips, err: err}
	}()
	go func() {
		ips := make([]string, 0, 4)
		answers, err := queryDNS(host, dns.TypeA)
		for _, answer := range answers {
			if record, ok := answer.(*dns.A); ok {
				ips = appendUniqueResolveValue(ips, record.A.String())
			}
		}
		aLookupCh <- ipLookupResult{ips: ips, err: err}
	}()
	go func() {
		ips := make([]string, 0, 4)
		answers, err := queryDNS(host, dns.TypeAAAA)
		for _, answer := range answers {
			if record, ok := answer.(*dns.AAAA); ok {
				ips = appendUniqueResolveValue(ips, record.AAAA.String())
			}
		}
		aaaaLookupCh <- ipLookupResult{ips: ips, err: err}
	}()

	ips := make([]string, 0, 8)
	var lookupErr error
	for _, result := range []ipLookupResult{<-hostLookupCh, <-aLookupCh, <-aaaaLookupCh} {
		for _, item := range result.ips {
			ips = appendUniqueResolveValue(ips, item)
		}
		if result.err != nil && lookupErr == nil {
			lookupErr = result.err
		}
	}

	if len(cnameChain) == 0 && len(ips) == 0 && lookupErr != nil {
		return nil, nil, lookupErr
	}
	return cnameChain, ips, nil
}

func cnameMatchesExpected(expected, testedDomain string, cnameChain []string) bool {
	expected = normalizeResolveHost(expected)
	if expected == "" || expected == "-" {
		return false
	}
	testedDomain = normalizeResolveHost(testedDomain)
	if testedDomain == expected {
		return true
	}
	for _, item := range cnameChain {
		item = normalizeResolveHost(item)
		if item == expected {
			return true
		}
	}
	return false
}

func probeHTTPReachability(host string) (bool, string, int, error) {
	host = normalizeResolveHost(host)
	if host == "" {
		return false, "", 0, errors.New("empty host")
	}

	type probeResult struct {
		ok     bool
		url    string
		status int
		err    error
	}
	schemes := []string{"https", "http"}
	resultCh := make(chan probeResult, len(schemes))
	for _, scheme := range schemes {
		go func(scheme string) {
			resultCh <- probeHTTPScheme(host, scheme)
		}(scheme)
	}

	results := make([]probeResult, 0, len(schemes))
	for range schemes {
		results = append(results, <-resultCh)
	}
	for _, preferred := range schemes {
		for _, result := range results {
			if result.ok && strings.HasPrefix(result.url, preferred+"://") {
				return true, result.url, result.status, nil
			}
		}
	}
	for _, result := range results {
		if result.err != nil {
			return false, "", 0, result.err
		}
	}
	return false, "", 0, errors.New("http probe failed")
}

func probeHTTPScheme(host, scheme string) (result struct {
	ok     bool
	url    string
	status int
	err    error
}) {
	client := &http.Client{
		Timeout: 4 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	url := scheme + "://" + host
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		result.err = err
		return
	}
	req.Header.Set("User-Agent", "cdn-system-resolve-check/1.0")

	resp, err := client.Do(req)
	if err != nil {
		result.err = err
		return
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)
	if resp.StatusCode <= 0 {
		result.err = fmt.Errorf("unexpected status %d", resp.StatusCode)
		return
	}
	result.ok = true
	result.url = url
	result.status = resp.StatusCode
	return
}

func resolveErrText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func computeSiteCnameHostname(site models.Site, pkg models.UserPackage, overrideMode, overrideDomain *string) string {
	newCnameHostname := strings.TrimSpace(site.CnameHostname)

	siteMode := strings.TrimSpace(site.CnameMode)
	if overrideMode != nil {
		siteMode = strings.TrimSpace(*overrideMode)
	}

	cnameDomain := strings.TrimSpace(site.CnameDomain)
	if overrideDomain != nil {
		cnameDomain = strings.TrimSpace(*overrideDomain)
	}
	pkgDomain := strings.TrimSpace(pkg.CnameDomain)

	pkgMode := strings.TrimSpace(pkg.CnameMode)
	if siteMode == "package" || (siteMode == "" && pkgMode == "package") {
		pkgHost := strings.TrimSpace(pkg.CnameHostname)
		if pkgHost == "" {
			pkgHost = strings.TrimSpace(pkg.RecordID)
		}
		if pkgHost != "" {
			newCnameHostname = pkgHost
			if pkgDomain != "" {
				newCnameHostname += "." + pkgDomain
			} else if cnameDomain != "" {
				newCnameHostname += "." + cnameDomain
			} else {
				newCnameHostname += ".cdn.node.com"
			}
		}
		return newCnameHostname
	}

	effectiveDomain := cnameDomain
	if effectiveDomain == "" {
		effectiveDomain = pkgDomain
	}
	if effectiveDomain == "" {
		effectiveDomain = "cdn.node.com"
	}
	if len(site.Domains) > 0 && effectiveDomain != "" {
		return buildSiteCname(site.Domains[0], effectiveDomain)
	}
	return newCnameHostname
}

func refreshSiteCnameHostname(site *models.Site, overrideMode, overrideDomain *string) (bool, error) {
	if site == nil || site.UserPackageID == 0 {
		return false, nil
	}
	if overrideMode == nil && overrideDomain == nil && strings.TrimSpace(site.CnameHostname) != "" {
		return false, nil
	}
	var pkg models.UserPackage
	if err := db.DB.Select("cname_mode", "cname_hostname", "cname_domain", "record_id").
		Where("id = ?", site.UserPackageID).
		First(&pkg).Error; err != nil {
		return false, err
	}

	updateDomain := false
	if overrideDomain == nil && strings.TrimSpace(site.CnameDomain) == "" && strings.TrimSpace(pkg.CnameDomain) != "" {
		site.CnameDomain = strings.TrimSpace(pkg.CnameDomain)
		updateDomain = true
	}
	newCnameHostname := computeSiteCnameHostname(*site, pkg, overrideMode, overrideDomain)
	if newCnameHostname == "" || (newCnameHostname == site.CnameHostname && !updateDomain) {
		return false, nil
	}
	updates := map[string]interface{}{
		"cname_hostname": newCnameHostname,
	}
	if updateDomain {
		updates["cname_domain"] = site.CnameDomain
	}
	if err := db.DB.Model(&models.Site{}).
		Where("id = ?", site.ID).
		Updates(updates).Error; err != nil {
		return false, err
	}
	site.CnameHostname = newCnameHostname
	return true, nil
}

func shouldResyncSiteCname(oldSite, newSite models.Site) bool {
	if oldSite.UserPackageID != newSite.UserPackageID {
		return true
	}
	if strings.TrimSpace(oldSite.CnameDomain) != strings.TrimSpace(newSite.CnameDomain) {
		return true
	}
	if strings.TrimSpace(oldSite.CnameMode) != strings.TrimSpace(newSite.CnameMode) {
		return true
	}
	if strings.TrimSpace(oldSite.CnameHostname) != strings.TrimSpace(newSite.CnameHostname) {
		return true
	}
	if oldSite.NodeGroupID != newSite.NodeGroupID {
		return true
	}
	if oldSite.BackupNodeGroupID != newSite.BackupNodeGroupID {
		return true
	}
	if oldSite.EnableBackupGroup != newSite.EnableBackupGroup {
		return true
	}
	return false
}

func resyncSiteCnameForSite(site models.Site) {
	services.ResyncSiteCnameForSite(site)
}

func resyncGroupLineCnames(groupID int64) {
	services.ResyncGroupLineCnames(groupID)
}
