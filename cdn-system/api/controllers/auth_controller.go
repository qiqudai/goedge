package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/utils"
	"cdn-common/i18n"
	"encoding/json"
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
	Hash     string `json:"password_hash"`
	Captcha  string `json:"captcha"`
	Type     string `json:"captcha_type"`
}

type AuthController struct{}

func writeLoginLog(c *gin.Context, userID int64, success bool, postContent string) {
	data := map[string]interface{}{
		"uid":          nil,
		"ip":           resolveClientIP(c),
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
		writeLoginLog(c, 0, false, "invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	clientIP := resolveClientIP(c)
	if allowed, cooldown := services.AllowLoginAttempt(req.Username, clientIP); !allowed {
		writeLoginLog(c, 0, false, "rate limited")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":         T("Too many attempts"),
			"rate_limited":  true,
			"rate_cooldown": int(cooldown.Seconds()),
		})
		return
	}

	var user models.User
	// Support login by Name or Email
	if err := db.DB.Where("name = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeLoginLog(c, 0, false, "user not found: "+req.Username)
			c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T("auth.invalid_credentials")})
		} else {
			fmt.Printf("[Error] DB Query Failed: %v\n", err)
			writeLoginLog(c, 0, false, "db error: "+err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		}
		return
	}

	if !user.Enable {
		writeLoginLog(c, user.ID, false, "user disabled")
		c.JSON(http.StatusForbidden, gin.H{"error": T("User disabled")})
		return
	}

	// Map Type to Role (1=Admin, others=User)
	role := "user"
	if user.Type == 1 {
		role = "admin"
	}

	if !isLoginHostAllowed(c, role) {
		writeLoginLog(c, 0, false, "login host not allowed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T("auth.invalid_credentials")})
		return
	}

	providedHashed := strings.EqualFold(req.Hash, "sha256")
	if providedHashed && !passwordLooksHashed(req.Password) {
		writeLoginLog(c, 0, false, "invalid password hash")
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if !providedHashed {
		providedHashed = passwordLooksHashed(req.Password)
	}
	ok, upgrade := verifyPassword(user.Password, req.Password, providedHashed)
	if !ok {
		writeLoginLog(c, user.ID, false, "password mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T("auth.invalid_credentials")})
		return
	}
	if upgrade {
		if hashed, err := utils.HashPasswordForStorage(req.Password); err == nil {
			_ = db.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password", hashed).Error
		}
	}

	if needCaptcha, captchaType := resolveLoginCaptchaRequirement(user, req.Type); needCaptcha {
		if captchaType == "email" && strings.TrimSpace(user.Email) == "" {
			writeLoginLog(c, user.ID, false, "email missing for captcha")
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Email is required for login verification")})
			return
		}
		if captchaType == "sms" && strings.TrimSpace(user.Phone) == "" {
			writeLoginLog(c, user.ID, false, "phone missing for captcha")
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Phone is required for login verification")})
			return
		}
		var email string
		var phone string
		if captchaType == "email" {
			email = user.Email
		} else if captchaType == "sms" {
			phone = user.Phone
		}
		if !services.VerifyCaptcha(email, phone, req.Captcha) {
			writeLoginLog(c, user.ID, false, "captcha mismatch")
			c.JSON(http.StatusUnauthorized, gin.H{"error": T("Invalid captcha")})
			return
		}
	}

	tokenTTL := services.ResolveLoginSessionTTL()
	token, err := utils.GenerateTokenWithExpiry(user.ID, role, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to generate token")})
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

type LoginCaptchaRequest struct {
	Username string `json:"username" binding:"required"`
	Type     string `json:"type"`
}

func (ctr *AuthController) SendLoginCaptcha(c *gin.Context) {
	var req LoginCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	clientIP := resolveClientIP(c)
	if allowed, cooldown := services.AllowLoginCaptcha(req.Username, clientIP); !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":         T("Too many attempts"),
			"rate_limited":  true,
			"rate_cooldown": int(cooldown.Seconds()),
		})
		return
	}

	var user models.User
	if err := db.DB.Where("name = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T("auth.invalid_credentials")})
		return
	}
	if !user.Enable {
		c.JSON(http.StatusForbidden, gin.H{"error": T("User disabled")})
		return
	}

	needCaptcha, captchaType := resolveLoginCaptchaRequirement(user, req.Type)
	if !needCaptcha {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Captcha not enabled")})
		return
	}

	code := services.GenerateCaptchaCode(6)
	if captchaType == "email" {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Email is required for login verification")})
			return
		}
		title, body := resolveEmailCaptchaTemplate(code, user.Name)
		if err := services.SendEmail(email, title, body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to send email")})
			return
		}
		if err := services.StoreCaptcha(email, "", resolveClientIP(c), code); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to store captcha")})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("ok")})
		return
	}
	if captchaType == "sms" {
		phone := strings.TrimSpace(user.Phone)
		if phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Phone is required for login verification")})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": T("SMS verification is not configured")})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid captcha type")})
}

func resolveLoginCaptchaRequirement(user models.User, requestedType string) (bool, string) {
	emailEnabled, smsEnabled := services.ResolveLoginCaptchaConfig()
	if !emailEnabled && !smsEnabled {
		return false, ""
	}
	requestedType = strings.ToLower(strings.TrimSpace(requestedType))
	if requestedType == "email" && emailEnabled {
		return true, "email"
	}
	if requestedType == "sms" && smsEnabled {
		return true, "sms"
	}
	pref := strings.ToLower(strings.TrimSpace(user.LoginCaptcha))
	if pref == "email" && emailEnabled {
		return true, "email"
	}
	if pref == "sms" && smsEnabled {
		return true, "sms"
	}
	if emailEnabled {
		return true, "email"
	}
	if smsEnabled {
		return true, "sms"
	}
	return false, ""
}

func resolveEmailCaptchaTemplate(code string, username string) (string, string) {
	title := "验证码"
	body := "您的验证码是 " + code
	cfg, err := services.LoadSystemConfig()
	if err != nil {
		return title, body
	}
	raw := strings.TrimSpace(cfg["email_captcha_templ"])
	if raw == "" {
		return title, body
	}
	var payload struct {
		Title string `json:"title"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return title, body
	}
	if strings.TrimSpace(payload.Title) != "" {
		title = payload.Title
	}
	if strings.TrimSpace(payload.Data) != "" {
		body = payload.Data
	}
	body = strings.ReplaceAll(body, "{{captcha}}", code)
	body = strings.ReplaceAll(body, "{{code}}", code)
	body = strings.ReplaceAll(body, "{{username}}", username)
	return title, body
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hash     string `json:"password_hash"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func (ctr *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	cfg, err := services.LoadSystemConfig()
	if err != nil || !services.ParseBoolFlag(cfg["allow_register"]) {
		c.JSON(http.StatusForbidden, gin.H{"error": T("Registration disabled")})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	if strings.EqualFold(req.Hash, "sha256") && !passwordLooksHashed(password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}

	var exists int64
	if err := db.DB.Model(&models.User{}).Where("name = ? OR email = ?", username, strings.TrimSpace(req.Email)).Count(&exists).Error; err == nil && exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("User already exists")})
		return
	}

	hashed, err := utils.HashPasswordForStorage(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
		return
	}

	now := time.Now()
	userID, err := services.GenerateOrdinaryUserID(db.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
		return
	}
	user := models.User{
		ID:        userID,
		Name:      username,
		Email:     strings.TrimSpace(req.Email),
		Phone:     strings.TrimSpace(req.Phone),
		Password:  string(hashed),
		Enable:    true,
		Type:      2,
		GroupID:   0,
		CreatedAt: now,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("ok")})
}

func verifyPassword(stored, provided string, providedHashed bool) (bool, bool) {
	stored = strings.TrimSpace(stored)
	provided = strings.TrimSpace(provided)
	if stored == "" || provided == "" {
		return false, false
	}
	if isBcryptHash(stored) {
		if providedHashed {
			return bcrypt.CompareHashAndPassword([]byte(stored), []byte(strings.ToLower(provided))) == nil, false
		}
		normalized := utils.NormalizePasswordInput(provided)
		if bcrypt.CompareHashAndPassword([]byte(stored), []byte(normalized)) == nil {
			return true, false
		}
		if bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil {
			return true, true
		}
		return false, false
	}
	if providedHashed {
		return utils.NormalizePasswordInput(stored) == strings.ToLower(provided), true
	}
	return stored == provided, stored == provided
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$")
}

func passwordLooksHashed(value string) bool {
	return utils.IsSHA256Hex(strings.TrimSpace(value))
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
