package controllers

import (
	"cdn-api/services"
	"strings"

	"github.com/gin-gonic/gin"
)

func resolveClientIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	header := services.ResolveMasterClientIPHeader()
	if header != "" {
		raw := strings.TrimSpace(c.GetHeader(header))
		if raw != "" {
			if idx := strings.Index(raw, ","); idx != -1 {
				raw = strings.TrimSpace(raw[:idx])
			}
			if raw != "" {
				return raw
			}
		}
	}
	return c.ClientIP()
}
