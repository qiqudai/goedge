package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct{}

type userSaveRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Des      string `json:"des"`
	Phone    string `json:"phone"`
	QQ       string `json:"qq"`
	Password string `json:"password"`
	GroupID  int    `json:"group_id"`
	Enable   bool   `json:"enable"`
	Type     int    `json:"type"`

	// Security
	LoginCaptcha string `json:"login_captcha"`
	WhiteIP      string `json:"white_ip"`
}

func normalizeUserGroupID(groupID int) (int, error) {
	if groupID <= 0 {
		return 0, nil
	}

	var count int64
	if err := db.DB.Model(&models.UserGroup{}).Where("id = ?", groupID).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errors.New("user group not found")
	}

	return groupID, nil
}

// ListUsers returns paginated user list
// GET /api/v1/admin/users?page=1&size=20
func (ctr *UserController) ListUsers(c *gin.Context) {
	if err := db.Ensure(); err != nil {
		db.Init()
	}
	page, pageSize := parsePageParams(c, 20)
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var users []models.User
	query := db.DB.Model(&models.User{})
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ? OR email LIKE ? OR phone LIKE ? OR qq LIKE ? OR des LIKE ?",
			keywordLike, keywordLike, keywordLike, keywordLike, keywordLike)
		if id, err := strconv.ParseInt(strings.TrimSpace(keyword), 10, 64); err == nil {
			query = query.Or("id = ?", id)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  T("success"),
		"data": gin.H{
			"list":  users,
			"total": total,
		},
	})
}

// ToggleStatus enables or disables a user
// PUT /api/v1/admin/users/:id/status
func (ctr *UserController) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid params")})
		return
	}

	enabled := req.Status == 1
	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("enable", enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": fmt.Sprintf(T("User %s status updated to %d"), id, req.Status)})
}

// DeleteUser removes a user
// DELETE /api/v1/admin/users/:id
func (ctr *UserController) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid user id")})
		return
	}
	if id == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Built-in admin (ID=1) cannot be deleted"})
		return
	}

	type refCheck struct {
		table string
		col   string
		count int64
	}
	checks := []refCheck{
		{table: "cert", col: "uid"},
		{table: "user_package", col: "uid"},
		{table: "site", col: "uid"},
		{table: "stream", col: "uid"},
		{table: "dnsapi", col: "uid"},
		{table: "acl", col: "uid"},
		{table: "cc_rule", col: "uid"},
		{table: "cc_match", col: "uid"},
		{table: "cc_filter", col: "uid"},
		{table: "api_key", col: "uid"},
	}
	blockers := make([]string, 0)
	for i := range checks {
		sql := fmt.Sprintf("%s = ?", checks[i].col)
		if err := db.DB.Table(checks[i].table).Where(sql, id).Count(&checks[i].count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
			return
		}
		if checks[i].count > 0 {
			blockers = append(blockers, fmt.Sprintf("%s:%d", checks[i].table, checks[i].count))
		}
	}
	if len(blockers) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "User has related resources, delete blocked: " + strings.Join(blockers, ", "),
		})
		return
	}

	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": fmt.Sprintf(T("User %d deleted"), id)})
}

// ListUserGroups returns all user groups.
// GET /api/v1/admin/user_groups
func (ctr *UserController) ListUserGroups(c *gin.Context) {
	if err := db.Ensure(); err != nil {
		db.Init()
	}
	db.DB.AutoMigrate(&models.UserGroup{})
	var groups []models.UserGroup
	if err := db.DB.Order("id asc").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  T("success"),
		"data": gin.H{"list": groups},
	})
}

// CreateUserGroup creates a user group.
// POST /api/v1/admin/user_groups
func (ctr *UserController) CreateUserGroup(c *gin.Context) {
	db.DB.AutoMigrate(&models.UserGroup{})
	var req struct {
		Name string `json:"name"`
		Des  string `json:"des"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request body")})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Group name is required")})
		return
	}
	var exists int64
	if err := db.DB.Model(&models.UserGroup{}).Where("name = ?", name).Count(&exists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Group already exists")})
		return
	}
	group := models.UserGroup{Name: name, Des: strings.TrimSpace(req.Des)}
	if err := db.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Create Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Create success"), "data": group})
}

// DeleteUserGroup deletes a user group.
// DELETE /api/v1/admin/user_groups/:id
func (ctr *UserController) DeleteUserGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid group id")})
		return
	}
	var usingCount int64
	if err := db.DB.Model(&models.User{}).Where("group_id = ?", id).Count(&usingCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		return
	}
	if usingCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Group is in use")})
		return
	}
	if err := db.DB.Delete(&models.UserGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Delete Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Delete success")})
}

// CreateUser creates a user from the admin user list.
// POST /api/v1/admin/users
func (ctr *UserController) CreateUser(c *gin.Context) {
	db.DB.AutoMigrate(&models.User{}) // Ensure columns exist
	var req userSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request body")})
		return
	}

	name := strings.TrimSpace(req.Name)
	password := strings.TrimSpace(req.Password)
	email := strings.TrimSpace(req.Email)
	phone := strings.TrimSpace(req.Phone)
	if name == "" || password == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Username, password and email are required")})
		return
	}

	query := db.DB.Model(&models.User{}).Where("name = ?", name)
	if email != "" {
		query = query.Or("email = ?", email)
	}
	if phone != "" {
		query = query.Or("phone = ?", phone)
	}
	var exists int64
	if err := query.Count(&exists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("User already exists")})
		return
	}

	hash, err := utils.HashPasswordForStorage(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
		return
	}

	userType := req.Type
	if userType != 1 {
		userType = 2
	}
	var userID int64
	if userType == 2 {
		userID, err = services.GenerateOrdinaryUserID(db.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
			return
		}
	}
	groupID, err := normalizeUserGroupID(req.GroupID)
	if err != nil {
		if err.Error() == "user group not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid user group")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		return
	}
	user := models.User{
		ID:           userID,
		Email:        email,
		Name:         name,
		Description:  strings.TrimSpace(req.Des),
		Phone:        phone,
		QQ:           strings.TrimSpace(req.QQ),
		Password:     hash,
		GroupID:      groupID,
		Enable:       req.Enable,
		Type:         userType,
		LoginCaptcha: strings.TrimSpace(req.LoginCaptcha),
		WhiteIP:      strings.TrimSpace(req.WhiteIP),
		CreatedAt:    time.Now(),
	}
	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create user")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("User created successfully"), "data": user})
}

func getContextUserID(c *gin.Context) int64 {
	if val, ok := c.Get("userID"); ok {
		switch t := val.(type) {
		case float64:
			return int64(t)
		case int:
			return int64(t)
		case int64:
			return t
		case string:
			if i, err := strconv.ParseInt(t, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// ResetPurgeUsage resets purge/preheat usage for a user
// POST /api/v1/admin/users/:id/purge/reset
func (ctr *UserController) ResetPurgeUsage(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid user id")})
		return
	}
	payload := map[string]interface{}{
		"date":        time.Now().Format("2006-01-02"),
		"refresh_url": 0,
		"refresh_dir": 0,
		"preheat":     0,
	}
	raw, _ := json.Marshal(payload)
	var cfg models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "purge_usage", "user", "user", userID)
	if err := query.First(&cfg).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Reset failed")})
			return
		}
		cfg = models.SysConfig{
			Name:      "purge_usage",
			Value:     string(raw),
			Type:      "user",
			ScopeID:   int(userID),
			ScopeName: "user",
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.DB.Create(&cfg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Reset failed")})
			return
		}
	} else {
		cfg.Value = string(raw)
		cfg.UpdatedAt = time.Now()
		if err := db.DB.Save(&cfg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Reset failed")})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Reset success")})
}

// Impersonate generates a token for the target user (admin only)
// POST /api/v1/admin/users/:id/impersonate
func (ctr *UserController) Impersonate(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid user id")})
		return
	}
	var user models.User
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("User not found")})
		return
	}
	if !user.Enable {
		c.JSON(http.StatusForbidden, gin.H{"msg": T("User disabled")})
		return
	}

	role := "user"
	if user.Type == 1 {
		role = "admin"
	}
	tokenTTL := services.ResolveLoginSessionTTL()
	token, err := utils.GenerateTokenWithExpiry(user.ID, role, tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to generate token")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"role":  role,
		"uid":   user.ID,
		"name":  user.Name,
	})
}

// UpdateUser updates user information
// PUT /api/v1/admin/users/:id
func (ctr *UserController) UpdateUser(c *gin.Context) {
	db.DB.AutoMigrate(&models.User{}) // Ensure columns exist
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid user id")})
		return
	}

	var req userSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request body")})
		return
	}

	// Fetch existing
	var user models.User
	if err := db.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": T("User not found")})
		return
	}

	updates := map[string]interface{}{}
	updates["email"] = req.Email
	updates["name"] = req.Name
	updates["des"] = req.Des
	updates["phone"] = req.Phone
	updates["qq"] = req.QQ
	groupID, err := normalizeUserGroupID(req.GroupID)
	if err != nil {
		if err.Error() == "user group not found" {
			c.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid user group")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Database Error")})
		return
	}
	updates["group_id"] = groupID
	updates["enable"] = req.Enable
	if req.Type == 1 || req.Type == 2 {
		updates["type"] = req.Type
	}

	updates["login_captcha"] = req.LoginCaptcha
	updates["white_ip"] = req.WhiteIP

	if req.Password != "" {
		hash, err := utils.HashPasswordForStorage(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update user")})
			return
		}
		updates["password"] = hash
	}

	if err := db.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update user")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("User updated successfully")})
}
