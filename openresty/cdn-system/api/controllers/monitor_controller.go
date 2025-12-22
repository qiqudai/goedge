package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type MonitorController struct{}

// GetMonitorConfig
func (ctr *MonitorController) GetConfig(c *gin.Context) {
    // Mock Config
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
		    "notify_email": "admin@example.com",
		    "notify_telegram": "-100123456789",
		    "template_ip_up": "Node {{node_id}} IP {{ip}} Recovered",
		    "template_ip_down": "Node {{node_id}} IP {{ip}} Down",
		},
	})
}

// UpdateMonitorConfig
func (ctr *MonitorController) UpdateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"msg": "Monitor Config Updated"})
}
