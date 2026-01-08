package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthController struct{}

func writeLoginLog(c *gin.Context, userID int64, success bool, postContent string) {
	data := map[string]interface{}{
		"uid":          nil,
		"ip":           c.ClientIP(),
		"success":      success,
		"post_content": postContent,
		"create_at":    time.Now(),
	}
	if userID > 0 {
		data["uid"] = userID
	}
	_ = db.DB.Table("login_log").Create(&data).Error
}

func (ctr *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[DEBUG] Login Bind Error: %v\n", err)
		writeLoginLog(c, 0, false, "invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	fmt.Printf("[DEBUG] Login Request: Username=%s, PasswordLen=%d\n", req.Username, len(req.Password))

	var user models.User
	// Support login by Name or Email
	if err := db.DB.Where("name = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeLoginLog(c, 0, false, "user not found: "+req.Username)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (user not found)"})
		} else {
			fmt.Printf("[Error] DB Query Failed: %v\n", err)
			writeLoginLog(c, 0, false, "db error: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	if !user.Enable {
		writeLoginLog(c, user.ID, false, "user disabled")
		c.JSON(http.StatusForbidden, gin.H{"error": "User disabled"})
		return
	}

	// Map Type to Role (1=Admin, others=User)
	role := "user"
	if user.Type == 1 {
		role = "admin"
	}

	if !isLoginHostAllowed(c, role) {
		writeLoginLog(c, 0, false, "login host not allowed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (user not found)"})
		return
	}

	if !verifyPassword(user.Password, req.Password) {
		writeLoginLog(c, user.ID, false, "password mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (password mismatch)"})
		return
	}

	tokenTTL := services.ResolveLoginSessionTTL()
	token, err := utils.GenerateTokenWithExpiry(user.ID, role, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	writeLoginLog(c, user.ID, true, "ok")
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"role":  role,
		"uid":   user.ID,
		"name":  user.Name,
	})
}

func verifyPassword(stored, provided string) bool {
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil
	}
	return stored == provided
}

func isLoginHostAllowed(c *gin.Context, role string) bool {
	cfg, err := services.LoadSystemConfig()
	if err != nil {
		return true
	}
	host := resolveRequestHost(c)
	if host == "" {
		return true
	}

	bindHosts := services.SplitHostList(cfg["bind-master-host"])
	var limitValue string
	if role == "admin" {
		limitValue = strings.TrimSpace(cfg["limit_admin_login_domain"])
	} else {
		limitValue = strings.TrimSpace(cfg["limit_user_login_domain"])
	}
	return hostAllowedByLimit(host, limitValue, bindHosts)
}

func resolveRequestHost(c *gin.Context) string {
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if strings.Contains(host, ",") {
		host = strings.TrimSpace(strings.Split(host, ",")[0])
	}
	return services.NormalizeHost(host)
}

func hostAllowedByLimit(host string, limitValue string, bindHosts []string) bool {
	if limitValue == "" {
		return true
	}
	host = services.NormalizeHost(host)
	if host == "" {
		return true
	}
	limits := services.SplitHostList(limitValue)
	if len(limits) == 0 {
		return true
	}
	hasDot := false
	for _, item := range limits {
		if strings.Contains(item, ".") {
			hasDot = true
			break
		}
	}
	if hasDot {
		for _, item := range limits {
			if host == item {
				return true
			}
		}
		return false
	}
	if len(bindHosts) == 0 {
		return true
	}
	for _, prefix := range limits {
		prefix = strings.TrimSuffix(prefix, ".")
		if prefix == "" {
			continue
		}
		for _, base := range bindHosts {
			if base == "" {
				continue
			}
			if host == prefix+"."+base {
				return true
			}
		}
	}
	return false
}
