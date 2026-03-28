package middleware

import (
	"cdn-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaintenanceRequired blocks user requests when maintenance is enabled.
func MaintenanceRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := services.LoadSystemConfig()
		if err != nil {
			c.Next()
			return
		}
		enabled, msg := services.ParseMaintenance(cfg["maintain"])
		if !enabled {
			c.Next()
			return
		}
		if msg == "" {
			msg = "System maintenance in progress"
		}
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":        503,
			"msg":         msg,
			"maintenance": true,
			"data": gin.H{
				"message": msg,
			},
		})
	}
}
