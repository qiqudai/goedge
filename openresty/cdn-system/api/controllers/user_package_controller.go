package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}
	ipv6Map, err := loadUserPackageBoolConfig(packs, "ipv6")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Config Error"})
		return
	}

	now := time.Now()
	list := make([]userPackageRow, 0, len(packs))
	for _, pack := range packs {
		status := "active"
		if !pack.EndAt.IsZero() && pack.EndAt.Before(now) {
			status = "expired"
		}
		list = append(list, userPackageRow{
			UserPackage: pack,
			IPv6:        ipv6Map[pack.ID],
			Status:      status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

// UpdateUserPackage - PUT /api/v1/user/user_packages/:id
func (ctr *UserPackageController) UpdateUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	var req struct {
		Name string `json:"name"`
		IPv6 *bool  `json:"ipv6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}

	query := db.DB.Model(&models.UserPackage{}).Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
			return
		}
		query = query.Where("uid = ?", uid)
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(req.Name); name != "" {
		updates["name"] = name
	}
	if len(updates) > 0 {
		if err := query.Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
			return
		}
	}

	if req.IPv6 != nil {
		if err := saveUserPackageBoolConfig(id, "ipv6", *req.IPv6); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Config Update Failed"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

// RenewUserPackage - POST /api/v1/user/user_packages/:id/renew
func (ctr *UserPackageController) RenewUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	var req struct {
		Period string `json:"period"`
		Months int    `json:"months"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid period"})
		return
	}

	var pack models.UserPackage
	query := db.DB.Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Not Found"})
		return
	}

	now := time.Now()
	base := pack.EndAt
	if base.IsZero() || base.Before(now) {
		base = now
	}
	newEnd := base.AddDate(0, months, 0)
	if err := db.DB.Model(&models.UserPackage{}).Where("id = ?", pack.ID).Update("end_at", newEnd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Renew Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"end_at": newEnd}})
}

// SwitchUserPackage - POST /api/v1/user/user_packages/:id/switch
func (ctr *UserPackageController) SwitchUserPackage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	var req struct {
		PackageID int64 `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}
	if req.PackageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "package_id is required"})
		return
	}

	var pack models.UserPackage
	query := db.DB.Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Not Found"})
		return
	}

	var pkg models.Package
	if err := db.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Package not found"})
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
		"month_price":       pkg.MonthPrice,
		"quarter_price":     pkg.QuarterPrice,
		"year_price":        pkg.YearPrice,
	}

	if err := db.DB.Model(&models.UserPackage{}).Where("id = ?", pack.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
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
		return db.DB.Save(&cfg).Error
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
