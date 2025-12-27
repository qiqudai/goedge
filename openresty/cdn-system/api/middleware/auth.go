package middleware

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/utils"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired validates JWT and checks role
func AuthRequired(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		// Bearer <token>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Role Check
		if requiredRole != "" && claims.Role != requiredRole {
			// Super admin can access everything? maybe.
			// For now, strict check.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AuthAgent validates agent tokens (edge nodes).
// It supports a global token via APP_AGENT_TOKEN or per-node tokens in DB.
// It creates a session with "nodeID" if authenticated.
func AuthAgent() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization token missing"})
			return
		}

		// 1) Global token (preferred)
		if config.App.AgentToken != "" {
			if token != config.App.AgentToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid agent token"})
				return
			}
			c.Set("agentToken", token)
			c.Next()
			return
		}
		if envToken := os.Getenv("APP_AGENT_TOKEN"); envToken != "" {
			if token != envToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid agent token"})
				return
			}
			c.Set("agentToken", token)
			c.Next()
			return
		}

		// 2) Per-node token from DB (requires a token column)
		if db.DB != nil {
			var node models.Node
			if err := db.DB.Where("token = ?", token).First(&node).Error; err == nil {
				c.Set("nodeID", strconv.FormatInt(node.ID, 10))
				c.Set("agentToken", token)
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid agent token"})
		return

	}
}
