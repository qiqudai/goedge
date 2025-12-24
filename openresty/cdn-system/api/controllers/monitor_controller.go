package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type MonitorController struct{}

type NodeMonitorConfig struct {
	NotificationPeriod string `json:"notification_period"`
	NotifyMethod       string `json:"notify_method"`
	NotifyMsgType      string `json:"notify_msg_type"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	BwExceedTimes      int    `json:"bw_exceed_times"`
	MonitorAPI         string `json:"monitor_api"`
	Interval           int    `json:"interval"`
	FailedTimes        int    `json:"failed_times"`
	FailedRate         string `json:"failed_rate"`
}

const nodeMonitorConfigKey = "node_monitor_config"

// GetMonitorConfig
func (ctr *MonitorController) GetConfig(c *gin.Context) {
	var sysConfig models.SysConfig
	result := db.DB.Where("name = ? AND type = ?", nodeMonitorConfigKey, "system").First(&sysConfig)

	var cfg NodeMonitorConfig
	if result.Error != nil {
		cfg = NodeMonitorConfig{
			NotificationPeriod: "8-22",
			NotifyMethod:       "email sms",
			NotifyMsgType:      "node_ip_dns bandwidth monitor backup_ip backup_default_line backup_group",
			Email:              "",
			Phone:              "",
			BwExceedTimes:      2,
			MonitorAPI:         "",
			Interval:           30,
			FailedTimes:        3,
			FailedRate:         "50",
		}
	} else {
		_ = json.Unmarshal([]byte(sysConfig.Value), &cfg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": cfg,
	})
}

// UpdateMonitorConfig
// UpdateMonitorConfig
func (ctr *MonitorController) UpdateConfig(c *gin.Context) {
	var req NodeMonitorConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	payload, _ := json.Marshal(req)

	var sysConfig models.SysConfig
	// Check if exists
	err := db.DB.Where("name = ? AND type = ?", nodeMonitorConfigKey, "system").First(&sysConfig).Error
	if err != nil {
		// Create new
		sysConfig = models.SysConfig{
			Name:      nodeMonitorConfigKey,
			Type:      "system",
			Value:     string(payload),
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			TaskID:    nil, // Allow NULL
		}
		if err := db.DB.Create(&sysConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Create Error: " + err.Error()})
			return
		}
	} else {
		// Update existing using Where clause because we don't have a simple ID primary key
		updates := map[string]interface{}{
			"value":     string(payload),
			"update_at": time.Now(),
		}
		if err := db.DB.Model(&models.SysConfig{}).Where("name = ? AND type = ?", nodeMonitorConfigKey, "system").Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Save Error: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Monitor Config Updated"})
}
