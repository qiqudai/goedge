package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthController struct{}

func (ctr *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	// Support login by Name or Email
	if err := db.DB.Where("name = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (user not found)"})
		return
	}

	if !user.Enable {
		c.JSON(http.StatusForbidden, gin.H{"error": "User disabled"})
		return
	}

	if !verifyPassword(user.Password, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (password mismatch)"})
		return
	}

	// Map Type to Role (1=Admin, others=User)
	role := "user"
	if user.Type == 1 {
		role = "admin"
	}

	token, err := utils.GenerateToken(user.ID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

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
