package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	if err := db.DB.Where("id = ?", id).First(&site).Error; err != nil {
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

	var oldSite models.Site
	if err := db.DB.Where("id = ?", id).First(&oldSite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load site")})
		return
	}

	var req struct {
		IDs             []int64                `json:"ids"`
		UserPackageID   *int64                 `json:"user_package_id"`
		GroupID         *int64                 `json:"group_id"`
		GroupIDs        *[]int64               `json:"group_ids"`
		DNSProviderID   *int64                 `json:"dns_provider_id"`
		HttpListen      *[]string              `json:"http_listen"`
		HttpsListen     *[]string              `json:"https_listen"`
		BalanceWay      *string                `json:"balance_way"`
		BackendProtocol *string                `json:"backend_protocol"`
		CertID          *int64                 `json:"cert_id"`
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

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.UserPackageID != nil && *req.UserPackageID > 0 {
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
		if req.Backends != nil {
			updates["backend"] = encodeList(*req.Backends)
		}
		if req.BalanceWay != nil {
			updates["balance_way"] = *req.BalanceWay
		}
		if req.BackendProtocol != nil {
			updates["backend_protocol"] = *req.BackendProtocol
		}
		if req.CertID != nil {
			if *req.CertID < 0 {
				updates["cert_id"] = 0
			} else {
				updates["cert_id"] = *req.CertID
			}
		}
		if req.CertID != nil {
			if *req.CertID < 0 {
				updates["cert_id"] = 0
			} else {
				updates["cert_id"] = *req.CertID
			}
		}
		if req.Domains != nil && len(*req.Domains) > 0 {
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
		_ = services.SyncUserDNSRecords(&oldSite, &newSite)
		if oldSite.UserPackageID != newSite.UserPackageID {
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

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)

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

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)

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
	if req.UserID == 0 {
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
		HttpListen        *[]string              `json:"http_listen"`
		HttpsListen       *[]string              `json:"https_listen"`
		BalanceWay        *string                `json:"balance_way"`
		BackendProtocol   *string                `json:"backend_protocol"`
		Backends          *[]string              `json:"backends"`
		CertID            *int64                 `json:"cert_id"`
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
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("ids is required")})
		return
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
		if req.Backends != nil {
			updates["backend"] = encodeList(*req.Backends)
		}
		if req.BalanceWay != nil {
			updates["balance_way"] = *req.BalanceWay
		}
		if req.BackendProtocol != nil {
			updates["backend_protocol"] = *req.BackendProtocol
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
				var pkg models.UserPackage
				if err := tx.First(&pkg, site.UserPackageID).Error; err != nil {
					continue
				}

				// Recalculate CnameHostname.
				newCnameHostname := site.CnameHostname
				siteMode := ""
				if req.CnameMode != nil {
					siteMode = *req.CnameMode
				} else {
					siteMode = site.CnameMode
				}

				cnameDomain := site.CnameDomain
				if req.CnameDomain != nil {
					cnameDomain = *req.CnameDomain
				}

				// Recalculate based on mode.
				if siteMode == "package" || (siteMode == "" && pkg.CnameMode == "package") {
					// Package mode.
					if pkg.CnameHostname != "" {
						newCnameHostname = pkg.CnameHostname
						if pkg.CnameDomain != "" {
							newCnameHostname += "." + pkg.CnameDomain
						} else if cnameDomain != "" {
							newCnameHostname += "." + cnameDomain
						} else {
							newCnameHostname += ".cdn.node.com"
						}
					}
				} else {
					// Custom or default mode.
					if len(site.Domains) > 0 && cnameDomain != "" {
						newCnameHostname = buildSiteCname(site.Domains[0], cnameDomain)
					}
				}

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
	if req.UserPackageID != nil {
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
		payload := map[string]interface{}{
			"action":   "clear_cache",
			"site_ids": req.IDs,
		}
		raw, _ := json.Marshal(payload)
		task := models.Task{
			Type:     "clear_cache",
			Name:     i18n.T("cache.clear_task_name"),
			Data:     string(raw),
			State:    "waiting",
			Enable:   true,
			CreateAt: now,
			RetryAt:  &now,
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

	var sites []models.Site
	if err := db.DB.Where("id IN ?", req.IDs).Find(&sites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load sites")})
		return
	}
	if len(sites) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": T("site not found")})
		return
	}

	createdIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		if len(site.Domains) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("site domains are empty")})
			return
		}
		if err := ensureNoExistingCert(site.UserID, site.Domains); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": T(err.Error())})
			return
		}

		certType, dnsapi := resolveCertDefaults(site.UserID)
		cert := models.Cert{
			UserID:      int(site.UserID),
			Name:        defaultCertName(site.Domains[0]),
			Description: fmt.Sprintf("site_id:%d", site.ID),
			Type:        certType,
			Domain:      strings.Join(site.Domains, ","),
			DNSAPI:      normalizeDNSAPIValue(dnsapi),
			AutoRenew:   true,
			Enable:      true,
			State:       "waiting",
			CreateAt:    time.Now(),
			UpdateAt:    time.Now(),
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
			if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Update("update_at", now).Error; err != nil {
				return err
			}
			if len(site.HttpsListen) == 0 && strings.TrimSpace(site.HttpsListenRaw) == "" {
				if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).
					Update("https_listen", encodeList([]string{"443"})).Error; err != nil {
					return err
				}
			}
			if tx.Migrator().HasColumn(&models.Site{}, "cert_id") {
				if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Update("cert_id", cert.ID).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create cert")})
			return
		}

		createdIDs = append(createdIDs, int64(cert.ID))
	}

	services.BumpConfigVersion("site", req.IDs)
	services.BumpConfigVersion("cert", createdIDs)
	services.IssueCertsAsync(time.Now().Unix(), createdIDs)

	c.JSON(http.StatusOK, gin.H{"message": T("Certificate apply queued")})
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

	cname, _ := net.LookupCNAME(domain)
	hosts, _ := net.LookupHost(domain)

	c.JSON(http.StatusOK, gin.H{
		"domain": domain,
		"cname":  strings.TrimSuffix(cname, "."),
		"ips":    hosts,
	})
}

func resyncSiteCnameForSite(site models.Site) {
	groupID, err := resolveNodeGroupFromPackage(site.UserPackageID, site.NodeGroupID)
	if err == nil && groupID != 0 {
		resyncGroupLineCnames(groupID)
	}

	backupGroup := site.BackupNodeGroupID
	enableBackup := site.EnableBackupGroup
	if !enableBackup {
		var pkg models.UserPackage
		if err := db.DB.Select("backup_node_group", "enable_backup_group").
			Where("id = ?", site.UserPackageID).
			First(&pkg).Error; err == nil {
			if backupGroup == 0 {
				backupGroup = pkg.BackupNodeGroup
			}
			enableBackup = pkg.EnableBackup
		}
	}
	if enableBackup && backupGroup != 0 {
		resyncGroupLineCnames(backupGroup)
	}
}

func resyncGroupLineCnames(groupID int64) {
	if groupID == 0 {
		return
	}
	var lines []models.Line
	if err := db.DB.Select("line_id", "line_name").
		Where("node_group_id = ?", groupID).
		Find(&lines).Error; err != nil {
		return
	}
	lineMap := map[string]string{}
	for _, line := range lines {
		lineID := strings.TrimSpace(line.LineID)
		if lineID == "" {
			lineID = "default"
		}
		lineName := strings.TrimSpace(line.LineName)
		if lineName == "" {
			lineName = lineID
		}
		if _, ok := lineMap[lineID]; !ok {
			lineMap[lineID] = lineName
		}
	}
	for lineID, lineName := range lineMap {
		ids := loadLineNodeIDs(groupID, lineID)
		_ = services.SyncPackageCnameForLineChange(groupID, lineID, lineName, ids, "resync")
	}
}
