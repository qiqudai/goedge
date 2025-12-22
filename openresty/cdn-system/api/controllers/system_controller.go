package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SystemController struct{}

const SystemSettingsKey = "system_settings"

// GetInfo
func (ctr *SystemController) GetInfo(c *gin.Context) {
	var sysConfig models.SysConfig
	result := db.DB.First(&sysConfig, "key = ?", SystemSettingsKey)

	var settings models.SystemSettings
	if result.Error != nil {
		// Return defaults
		settings = models.SystemSettings{
			SystemName:           "cdn 4.0",
			AdminTitle:           "cdn管理员控制台",
			UserTitle:            "cdn用户控制台",
			FooterText:           "Copyright © 2025 Cdn All Rights Reserved",
			CleanCacheDays:       30,
			CleanLoginLogDays:    30,
			CleanOpLogDays:       365,
			CleanSiteLogDays:     7,
			CleanNodeMonitorDays: 7,
			CleanTrafficDays:     90,
			CleanNodeTrafficDays: 45,
			BackupFrequency:      2,
			BackupRetention:      7,
			BackupDir:            "/data/backup/cdn/",
			SessionLife:          86400,
			LoginAdFile:          "none",
			RegisterMailTitle:    "cdn用户注册成功",
			RegisterMailContent:  "<p>尊敬的{{username}}:</p><p>您好！感谢您注册cdn。</p>",
		}
	} else {
		json.Unmarshal([]byte(sysConfig.Value), &settings)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": settings,
	})
}

// UpdateInfo
func (ctr *SystemController) UpdateInfo(c *gin.Context) {
	var req models.SystemSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	jsonBytes, _ := json.Marshal(req)

	// Upsert
	var sysConfig models.SysConfig
	db.DB.Where(models.SysConfig{Key: SystemSettingsKey}).Attrs(models.SysConfig{
		Value: string(jsonBytes),
	}).FirstOrCreate(&sysConfig)

	sysConfig.Value = string(jsonBytes)
	db.DB.Save(&sysConfig)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "System Settings Updated"})
}
