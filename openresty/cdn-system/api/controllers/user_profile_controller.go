package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserProfileController struct{}

type profileResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	QQ           string `json:"qq"`
	Balance      int64  `json:"balance"`
	CertName     string `json:"cert_name"`
	CertNo       string `json:"cert_no"`
	CertVerified bool   `json:"cert_verified"`
	WhiteIP      string `json:"white_ip"`
	LoginCaptcha string `json:"login_captcha"`
	CreatedAt    string `json:"create_at"`
}

type updateProfileRequest struct {
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	QQ           string `json:"qq"`
	CertName     string `json:"cert_name"`
	CertNo       string `json:"cert_no"`
	WhiteIP      string `json:"white_ip"`
	LoginCaptcha string `json:"login_captcha"`
}

type updatePasswordRequest struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

func verifyPassword(stored, provided string) bool {
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil
	}
	return stored == provided
}

// GetProfile
// GET /api/v1/user/profile
func (ctr *UserProfileController) GetProfile(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var user models.User
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": profileResponse{
			ID:           user.ID,
			Name:         user.Name,
			Email:        user.Email,
			Phone:        user.Phone,
			QQ:           user.QQ,
			Balance:      user.Balance,
			CertName:     user.CertName,
			CertNo:       user.CertNo,
			CertVerified: user.CertVerified,
			WhiteIP:      user.WhiteIP,
			LoginCaptcha: user.LoginCaptcha,
			CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// UpdateProfile
// PUT /api/v1/user/profile
func (ctr *UserProfileController) UpdateProfile(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	updates := map[string]interface{}{
		"email":         req.Email,
		"phone":         req.Phone,
		"qq":            req.QQ,
		"cert_name":     req.CertName,
		"cert_no":       req.CertNo,
		"white_ip":      req.WhiteIP,
		"login_captcha": req.LoginCaptcha,
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

// UpdatePassword
// PUT /api/v1/user/password
func (ctr *UserProfileController) UpdatePassword(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var req updatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}
	if req.Current == "" || req.Next == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid password"})
		return
	}

	var user models.User
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	if !verifyPassword(user.Password, req.Current) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Current password invalid"})
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Next), bcrypt.DefaultCost)
	if err := db.DB.Model(&models.User{}).Where("id = ?", userID).Update("password", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Password updated"})
}
