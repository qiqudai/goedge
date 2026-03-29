package controllers

import (
	"cdn-api/services"
	"strings"

	"github.com/gin-gonic/gin"
)

func resolveAgentNodeValue(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return strings.TrimSpace(c.Query("node_id"))
}

func applyAgentNodeIdentity(c *gin.Context, nodeID *string, nodeIP *string) {
	if nodeID != nil && strings.TrimSpace(*nodeID) == "" {
		*nodeID = resolveAgentNodeValue(c)
	}
	if nodeIP != nil && strings.TrimSpace(*nodeIP) == "" {
		*nodeIP = resolveClientIP(c)
	}
}

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
