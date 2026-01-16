package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserPackageController struct{}

type userPackageRow struct {
	models.UserPackage
	IPv6   bool   `json:"ipv6"`
	Status string `json:"status"`
	// Explicitly expose these fields to debug visibility issue
	CnameDomain   string `json:"cname_domain"`
	CnameHostname string `json:"cname_hostname"`
	CnameMode     string `json:"cname_mode"`
	RecordID      string `json:"record_id"`
}

// ListUserPackages - GET /api/v1/admin/user_packages?user_id=xx
func (ctr *UserPackageController) ListUserPackages(c *gin.Context) {
	var packs []models.UserPackage
	query := db.DB.Model(&models.UserPackage{})
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid != 0 {
			query = query.Where("uid = ?", uid)
		}
	} else if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if err := query.Order("id desc").Find(&packs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	ipv6Map, err := loadUserPackageBoolConfig(packs, "ipv6")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Config Error")})
		return
	}

	now := time.Now()
	list := make([]userPackageRow, 0, len(packs))
	for _, pack := range packs {
		status := "active"
		if !pack.EndAt.IsZero() && pack.EndAt.Before(now) {
			status = "expired"
		}
		// Force strings.TrimSpace to ensure no whitespace hiding
		list = append(list, userPackageRow{
			UserPackage:   pack,
			IPv6:          ipv6Map[pack.ID],
			Status:        status,
			CnameDomain:   strings.TrimSpace(pack.CnameDomain),
			CnameHostname: strings.TrimSpace(pack.CnameHostname),
			CnameMode:     pack.CnameMode,
			RecordID:      pack.RecordID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

// UpdateUserPackage - PUT /api/v1/user/user_packages/:id
func (ctr *UserPackageController) UpdateUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	isUserReq := isUserRequest(c)
	var req struct {
		Name           string  `json:"name"`
		IPv6           *bool   `json:"ipv6"`
		EndAt          string  `json:"end_at"`
		RegionID       int64   `json:"region_id"`
		NodeGroupID    int64   `json:"node_group_id"`
		BackupGroupID  int64   `json:"backup_group_id"`
		Traffic        *string `json:"traffic"`
		Bandwidth      *string `json:"bandwidth"` // Pointer to distinguish empty vs unchanged if needed, or just string
		Connection     *string `json:"connection"`
		Domain         *string `json:"domain"`
		MainDomain     *string `json:"main_domain_limit"`
		HttpPort       *string `json:"http_port"`
		StreamPort     *string `json:"stream_port"`
		CustomCCRule   *bool   `json:"custom_cc_rule"`
		Websocket      *bool   `json:"websocket"`
		PriceMonthly   float64 `json:"price_monthly"`
		PriceQuarterly float64 `json:"price_quarterly"`
		PriceYearly    float64 `json:"price_yearly"`
		CnameHostname  string  `json:"cname_hostname"`
		CnameDomain    string  `json:"cname_domain"`
		CnameMode      string  `json:"cname_mode"`
		Http3Enabled   *bool   `json:"http3_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}

	query := db.DB.Model(&models.UserPackage{}).Where("id = ?", id)
	if isUserReq {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		query = query.Where("uid = ?", uid)
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(req.Name); name != "" {
		updates["name"] = name
	}
	if req.EndAt != "" {
		if t, err := time.Parse(time.DateTime, req.EndAt); err == nil {
			updates["end_at"] = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", req.EndAt); err == nil {
			updates["end_at"] = t
		}
	}

	// Groups: avoid writing zero IDs that violate FKs.
	if req.RegionID > 0 {
		updates["region_id"] = req.RegionID
	} else if !isUserReq && req.RegionID != 0 {
		updates["region_id"] = req.RegionID
	}
	if req.NodeGroupID > 0 {
		updates["node_group_id"] = req.NodeGroupID
	} else if !isUserReq && req.NodeGroupID != 0 {
		updates["node_group_id"] = req.NodeGroupID
	}
	if req.BackupGroupID > 0 {
		updates["backup_node_group"] = req.BackupGroupID
	} else if !isUserReq && req.BackupGroupID != 0 {
		updates["backup_node_group"] = req.BackupGroupID
	}

	// Resources - Handle "limited" vs "unlimited" (empty string or -1 usually)
	// Frontend sends string or number.
	if req.Traffic != nil {
		updates["traffic"] = *req.Traffic
	}
	if req.Bandwidth != nil {
		updates["bandwidth"] = *req.Bandwidth
	}
	if req.Connection != nil {
		updates["connection"] = *req.Connection
	}
	if req.Domain != nil {
		updates["domain"] = *req.Domain
	}
	if req.MainDomain != nil {
		updates["main_domain_limit"] = parseIntValue(*req.MainDomain)
	}
	if req.HttpPort != nil {
		updates["http_port"] = *req.HttpPort
	}
	if req.StreamPort != nil {
		updates["stream_port"] = *req.StreamPort
	}
	if req.CustomCCRule != nil {
		updates["custom_cc_rule"] = *req.CustomCCRule
	}
	if req.Websocket != nil {
		updates["websocket"] = *req.Websocket
	}

	// Price (admin-only updates)
	if !isUserReq {
		updates["month_price"] = req.PriceMonthly
		updates["quarter_price"] = req.PriceQuarterly
		updates["year_price"] = req.PriceYearly
	}

	// CNAME
	cnameHostname := strings.TrimSpace(req.CnameHostname)
	cnameDomain := strings.TrimSpace(req.CnameDomain)
	cnameMode := strings.TrimSpace(req.CnameMode)
	if cnameHostname != "" || !isUserReq {
		updates["cname_hostname"] = cnameHostname
	}
	if cnameDomain != "" || !isUserReq {
		updates["cname_domain"] = cnameDomain
	}
	if cnameMode != "" || !isUserReq {
		updates["cname_mode"] = cnameMode
	}

	// DEBUG LOG
	fmt.Printf("[DEBUG] UpdateUserPackage ID=%d ReqDomain=%s UpdatesDomain=%v\n", id, req.CnameDomain, updates["cname_domain"])

	if len(updates) > 0 {
		if err := query.Updates(updates).Error; err != nil {
			log.Printf("[Error] UpdateUserPackage id=%d updates=%v err=%v", id, updates, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
			return
		}
	}

	if req.IPv6 != nil {
		if err := saveUserPackageBoolConfig(id, "ipv6", *req.IPv6); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Config Update Failed")})
			return
		}
	}
	if req.Http3Enabled != nil {
		if err := saveUserPackageBoolConfig(id, "http3_enabled", *req.Http3Enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Config Update Failed")})
			return
		}
	}

	// Trigger Sync
	if err := services.NewUserPackageService().SyncUserPackage(id, "update"); err != nil {
		// Log error but don't fail request? Or warning?
		fmt.Printf("[WARN] SyncUserPackage Failed: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

// RenewUserPackage - POST /api/v1/user/user_packages/:id/renew
func (ctr *UserPackageController) RenewUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	var req struct {
		Period string `json:"period"`
		Months int    `json:"months"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}

	months := req.Months
	if months <= 0 {
		switch strings.ToLower(strings.TrimSpace(req.Period)) {
		case "month":
			months = 1
		case "quarter":
			months = 3
		case "year":
			months = 12
		}
	}
	if months <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid period")})
		return
	}

	var pack models.UserPackage
	query := db.DB.Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": T("Not Found")})
		return
	}

	now := time.Now()
	base := pack.EndAt
	if base.IsZero() || base.Before(now) {
		base = now
	}
	newEnd := base.AddDate(0, months, 0)
	if err := db.DB.Model(&models.UserPackage{}).Where("id = ?", pack.ID).Update("end_at", newEnd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Renew Failed")})
		return
	}

	// Trigger Sync
	if err := services.NewUserPackageService().SyncUserPackage(pack.ID, "renew"); err != nil {
		fmt.Printf("[WARN] SyncUserPackage Failed: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"end_at": newEnd}})
}

// SwitchUserPackage - POST /api/v1/user/user_packages/:id/switch
func (ctr *UserPackageController) SwitchUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	if err := ensureUserPackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}

	var req struct {
		PackageID int64 `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid Params")})
		return
	}
	if req.PackageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("package_id is required")})
		return
	}

	var pack models.UserPackage
	query := db.DB.Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Forbidden")})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": T("Not Found")})
		return
	}

	var pkg models.Package
	if err := db.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": T("Package not found")})
		return
	}

	currentPkg := (*models.Package)(nil)
	if pack.PackageID > 0 {
		var existing models.Package
		if err := db.DB.Where("id = ?", pack.PackageID).First(&existing).Error; err == nil {
			currentPkg = &existing
		}
	}
	changeType := classifyPackageChange(pack, currentPkg, pkg)
	allowUpgrade := true
	allowDowngrade := true
	if cfg, err := services.LoadSystemConfig(); err == nil {
		if val, ok := cfg["package_allow_upgrade"]; ok {
			allowUpgrade = services.ParseBoolFlag(val)
		}
		if val, ok := cfg["package_allow_downgrade"]; ok {
			allowDowngrade = services.ParseBoolFlag(val)
		}
	}
	if changeType == "upgrade" && !allowUpgrade {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Upgrade is disabled")})
		return
	}
	if changeType == "downgrade" && !allowDowngrade {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": T("Downgrade is disabled")})
		return
	}

	updates := map[string]interface{}{
		"name":              pkg.Name,
		"package":           pkg.ID,
		"region_id":         pkg.RegionID,
		"node_group_id":     pkg.NodeGroupID,
		"backup_node_group": pkg.BackupNode,
		"traffic":           pkg.Traffic,
		"bandwidth":         pkg.Bandwidth,
		"connection":        pkg.Connection,
		"domain":            pkg.DomainLimit,
		"custom_cc_rule":    pkg.CustomCCRule,
		"websocket":         pkg.Websocket,
		"l2_origin":         pkg.L2Origin,
		"month_price":       pkg.MonthPrice,
		"quarter_price":     pkg.QuarterPrice,
		"year_price":        pkg.YearPrice,
	}

	if err := db.DB.Model(&models.UserPackage{}).Where("id = ?", pack.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
		return
	}

	// Trigger Sync
	if err := services.NewUserPackageService().SyncUserPackage(pack.ID, "upgrade"); err != nil {
		fmt.Printf("[WARN] SyncUserPackage Failed: %v\n", err)
	}
	resyncSitesForUserPackage(pack.ID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

func resyncSitesForUserPackage(userPackageID int64) {
	if userPackageID == 0 {
		return
	}
	var sites []models.Site
	if err := db.DB.Where("user_package = ?", userPackageID).Find(&sites).Error; err != nil {
		log.Printf("[WARN] resyncSitesForUserPackage load failed package=%d err=%v", userPackageID, err)
		return
	}
	updated := make([]int64, 0)
	for _, site := range sites {
		changed, err := refreshSiteCnameHostname(&site, nil, nil)
		if err != nil {
			log.Printf("[WARN] resyncSitesForUserPackage refresh failed site=%d err=%v", site.ID, err)
		}
		if changed {
			updated = append(updated, site.ID)
		}
		resyncSiteCnameForSite(site)
	}
	if len(updated) > 0 {
		services.BumpConfigVersion("site", updated)
	}
}

func loadUserPackageBoolConfig(packs []models.UserPackage, name string) (map[int64]bool, error) {
	result := make(map[int64]bool)
	if len(packs) == 0 {
		return result, nil
	}

	ids := make([]int64, 0, len(packs))
	for _, pack := range packs {
		ids = append(ids, pack.ID)
	}

	var cfgs []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND name = ? AND scope_id IN ?", "user_package_config", "user_package", name, ids).Find(&cfgs).Error; err != nil {
		return result, err
	}

	for _, cfg := range cfgs {
		result[cfg.ScopeID] = parseBoolString(cfg.Value)
	}
	return result, nil
}

func saveUserPackageBoolConfig(userPackageID int64, name string, value bool) error {
	if userPackageID == 0 {
		return nil
	}
	val := "0"
	if value {
		val = "1"
	}

	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", name, "user_package_config", "user_package", userPackageID)
	var cfg models.ConfigItem
	if err := query.First(&cfg).Error; err == nil {
		cfg.Value = val
		cfg.Enable = true
		cfg.UpdatedAt = time.Now()
		return db.DB.Omit("CreatedAt").Save(&cfg).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	cfg = models.ConfigItem{
		Name:      name,
		Value:     val,
		Type:      "user_package_config",
		ScopeID:   userPackageID,
		ScopeName: "user_package",
		Enable:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return db.DB.Create(&cfg).Error
}

func parseBoolString(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseIntValue(val string) int {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return 0
}

func classifyPackageChange(current models.UserPackage, currentPkg *models.Package, target models.Package) string {
	currentScore := 0.0
	if currentPkg != nil {
		currentScore = packageScore(*currentPkg)
	} else {
		currentScore = userPackageScore(current)
	}
	targetScore := packageScore(target)
	return comparePackageScore(currentScore, targetScore)
}

func comparePackageScore(current, target float64) string {
	const epsilon = 0.0001
	if target > current+epsilon {
		return "upgrade"
	}
	if target < current-epsilon {
		return "downgrade"
	}
	return "same"
}

func packageScore(pkg models.Package) float64 {
	price := normalizedPrice(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice)
	if price > 0 {
		return float64(price)
	}
	return resourceScore(pkg.Traffic, pkg.Bandwidth, pkg.Connection, pkg.DomainLimit)
}

func userPackageScore(pkg models.UserPackage) float64 {
	price := normalizedPrice(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice)
	if price > 0 {
		return float64(price)
	}
	return resourceScore(int64(pkg.Traffic), pkg.Bandwidth, int64(pkg.Connection), int64(pkg.DomainLimit))
}

func normalizedPrice(month, quarter, year int64) int64 {
	if month > 0 {
		return month
	}
	if quarter > 0 {
		return quarter / 3
	}
	if year > 0 {
		return year / 12
	}
	return 0
}

func resourceScore(traffic int64, bandwidth string, connection int64, domain int64) float64 {
	score := float64(traffic) + float64(connection) + float64(domain)
	score += parseBandwidthMbps(bandwidth)
	return score
}

func parseBandwidthMbps(raw string) float64 {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "0" || value == "unlimited" || value == "unlimit" {
		return 0
	}
	multiplier := 1.0
	switch {
	case strings.HasSuffix(value, "g"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "g")
	case strings.HasSuffix(value, "m"):
		value = strings.TrimSuffix(value, "m")
	case strings.HasSuffix(value, "k"):
		multiplier = 1.0 / 1024
		value = strings.TrimSuffix(value, "k")
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed * multiplier
}
