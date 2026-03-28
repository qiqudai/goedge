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
	NotificationPeriod  string `json:"notification_period"`
	NotifyMethod        string `json:"notify_method"`
	NotifyMsgType       string `json:"notify_msg_type"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	BwExceedTimes       int    `json:"bw_exceed_times"`
	AutoSwitchEnable    bool   `json:"auto_switch_enable"`
	AutoSwitchThreshold int    `json:"auto_switch_threshold"`
	AutoSwitchDuration  int    `json:"auto_switch_duration"`
	AutoSwitchRecover   int    `json:"auto_switch_recover"`
	AutoSwitchMinWeight int    `json:"auto_switch_min_weight"`
	MonitorAPI          string `json:"monitor_api"`
	Interval            int    `json:"interval"`
	FailedTimes         int    `json:"failed_times"`
	FailedRate          string `json:"failed_rate"`
}

const nodeMonitorConfigKey = "node_monitor_config"

// GetMonitorConfig
func (ctr *MonitorController) GetConfig(c *gin.Context) {
	var sysConfig models.SysConfig
	result := db.DB.Where("name = ? AND type = ?", nodeMonitorConfigKey, "system").First(&sysConfig)

	var cfg NodeMonitorConfig
	if result.Error != nil {
		cfg = NodeMonitorConfig{
			NotificationPeriod:  "8-22",
			NotifyMethod:        "email sms",
			NotifyMsgType:       "node_ip_dns bandwidth monitor backup_ip backup_default_line backup_group",
			Email:               "",
			Phone:               "",
			BwExceedTimes:       2,
			AutoSwitchEnable:    false,
			AutoSwitchThreshold: 90,
			AutoSwitchDuration:  30,
			AutoSwitchRecover:   300,
			AutoSwitchMinWeight: 1,
			MonitorAPI:          "",
			Interval:            30,
			FailedTimes:         3,
			FailedRate:          "50",
		}
	} else {
		_ = json.Unmarshal([]byte(sysConfig.Value), &cfg)
		normalizeNodeMonitorConfig(&cfg)
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid JSON")})
		return
	}
	normalizeNodeMonitorConfig(&req)

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
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Create Error")})
			return
		}
	} else {
		// Update existing using Where clause because we don't have a simple ID primary key
		updates := map[string]interface{}{
			"value":     string(payload),
			"update_at": time.Now(),
		}
		if err := db.DB.Model(&models.SysConfig{}).Where("name = ? AND type = ?", nodeMonitorConfigKey, "system").Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Save Error")})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Monitor Config Updated")})
}

func normalizeNodeMonitorConfig(cfg *NodeMonitorConfig) {
	if cfg == nil {
		return
	}
	if cfg.AutoSwitchThreshold <= 0 || cfg.AutoSwitchThreshold > 100 {
		cfg.AutoSwitchThreshold = 90
	}
	if cfg.AutoSwitchDuration <= 0 {
		cfg.AutoSwitchDuration = 30
	}
	if cfg.AutoSwitchRecover < 300 {
		cfg.AutoSwitchRecover = 300
	}
	if cfg.AutoSwitchMinWeight <= 0 {
		cfg.AutoSwitchMinWeight = 1
	}
}
