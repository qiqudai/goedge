package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
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

// List returns the list of sites for the current user
func (ctrl *SiteController) List(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := parseInt64(userID)
	result, err := querySites(c, &uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to fetch sites"})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to build sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items, "total": result.Total}})
}

// AdminList returns the list of all sites for admin
func (ctrl *SiteController) AdminList(c *gin.Context) {
	result, err := querySites(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to fetch sites"})
		return
	}

	items, err := buildSiteListItems(result.Sites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to build sites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": items, "total": result.Total}})
}

func (ctrl *SiteController) AdminGet(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var site models.Site
	if err := db.DB.Where("id = ?", id).First(&site).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "site not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch site"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"site": site}})
}

// AdminUpdate updates a single site config
func (ctrl *SiteController) AdminUpdate(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
		Settings        map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
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

		if req.Settings != nil {
			b, _ := json.Marshal(req.Settings)
			if tx.Migrator().HasColumn(&models.Site{}, "settings") {
				updates["settings"] = string(b)
			} else {
				updates["SettingsRaw"] = string(b)
			}
		}

		updates["updated_at"] = time.Now()

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed: " + err.Error()})
		return
	}

	services.BumpConfigVersion("site", []int64{id})
	c.JSON(http.StatusOK, gin.H{"message": "Site updated"})
}

// AdminCreate handles site creation for admin
func (ctrl *SiteController) AdminCreate(c *gin.Context) {
	site, groupIDs, err := parseSiteCreateRequest(c, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := createSiteWithGroup(site, groupIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.BumpConfigVersion("site", []int64{site.ID})
	_ = ensureDNSRecords(site)

	c.JSON(http.StatusOK, gin.H{"message": "Site created successfully", "data": site})
}

// Create handles site creation for user
func (ctrl *SiteController) Create(c *gin.Context) {
	site, groupIDs, err := parseSiteCreateRequest(c, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := createSiteWithGroup(site, groupIDs); err != nil {
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

			groupIDs := []int64{}
			if req.GroupID != 0 {
				groupIDs = append(groupIDs, req.GroupID)
			}

			if err := createSiteWithGroup(site, groupIDs); err != nil {
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
