package middleware

import (
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
// It supports a global token via APP_AGENT_TOKEN and per-node tokens in DB.
func AuthAgent() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}
		token := parts[1]

		if envToken := os.Getenv("APP_AGENT_TOKEN"); envToken != "" && token == envToken {
			c.Set("agentToken", token)
			c.Next()
			return
		}

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
	}
}
