package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RuleController struct{}

func ensureCustomCCRuleAllowed(ctx *gin.Context) bool {
	if !isUserRequest(ctx) {
		return true
	}
	userID := parseUserID(mustGet(ctx, "userID"))
	if userID == 0 {
		ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
		return false
	}
	ok, err := services.NewUserPackageService().UserHasCustomCCRule(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to check permission")})
		return false
	}
	if !ok {
		ctx.JSON(http.StatusForbidden, gin.H{"error": T("custom cc rule not enabled")})
		return false
	}
	return true
}

// ================= CC Rules =================

// ListCCRuleGroups Lists CC rule groups
// GET /api/v1/admin/rules/cc/groups
func (c *RuleController) ListCCRuleGroups(ctx *gin.Context) {
	query := db.DB.Model(&models.CCRule{})
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID != 0 {
			query = query.Where("uid = ? OR uid = 0", userID)
		}
	}
	if name := strings.TrimSpace(ctx.Query("name")); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	status := strings.TrimSpace(ctx.Query("status"))
	if status == "on" {
		query = query.Where("enable = ?", true)
	} else if status == "off" {
		query = query.Where("enable = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load rules")})
		return
	}
	var items []models.CCRule
	if err := query.Order("sort asc, id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load rules")})
		return
	}

	userMap, _ := loadUserNameMapFromRules(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		inUse, _ := services.IsCCRuleGroupInUse(item.ID)
		isSystem := item.Internal || item.UserID == 0
		list = append(list, gin.H{
			"id":          item.ID,
			"user_id":     item.UserID,
			"uid":         item.UserID,
			"user":        gin.H{"username": userMap[item.UserID], "id": item.UserID},
			"name":        item.Name,
			"is_system":   isSystem,
			"type":        mapRuleType(item.UserID, item.Internal),
			"type_label":  ruleTypeLabel(isSystem),
			"in_use":      inUse,
			"is_on":       item.Enable,
			"is_show":     item.IsShow,
			"status":      T("status.normal"),
			"sort_order":  item.Sort,
			"create_time": item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// CreateCCRuleGroup Creates a new CC rule group
// POST /api/v1/admin/rules/cc/groups
func (c *RuleController) CreateCCRuleGroup(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	var req struct {
		Type         string                   `json:"type"`
		Name         string                   `json:"name"`
		Remark       string                   `json:"remark"`
		Rules        []map[string]interface{} `json:"rules"`
		IsVisible    bool                     `json:"is_visible"`
		VisibleUsers []int64                  `json:"visible_users"`
		SortOrder    int                      `json:"sort_order"`
		UserID       int64                    `json:"user_id"` // Admin allows selection
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	userID := int64(0)
	internal := false

	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		req.Type = "user"
	} else {
		// Admin request
		if req.Type == "system" {
			internal = true
		} else {
			// Admin creating user rule
			internal = false
			if req.UserID <= 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
				return
			}
			userID = req.UserID
		}
	}

	// Prepare JSON data for 'data' column
	dataMap := map[string]interface{}{
		"rules":         req.Rules,
		"visible_users": req.VisibleUsers,
	}
	dataBytes, _ := json.Marshal(dataMap)

	ccRule := models.CCRule{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Remark,
		Data:        string(dataBytes),
		Enable:      true, // Default enable or add is_on field if needed
		IsShow:      req.IsVisible,
		Sort:        req.SortOrder,
		Internal:    internal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createQuery := db.DB.Omit("TaskID")
	if internal {
		createQuery = db.DB.Omit("UserID", "TaskID")
	}
	if err := createQuery.Create(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create rule group")})
		return
	}
	services.BumpConfigVersion("cc_rule", []int64{ccRule.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateCCRuleGroup Updates an existing CC rule group
// PUT /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) UpdateCCRuleGroup(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	var req struct {
		Type         string                   `json:"type"`
		Name         string                   `json:"name"`
		Remark       string                   `json:"remark"`
		Rules        []map[string]interface{} `json:"rules"`
		IsVisible    bool                     `json:"is_visible"`
		VisibleUsers []int64                  `json:"visible_users"`
		SortOrder    int                      `json:"sort_order"`
		IsOn         bool                     `json:"is_on"`
		UserID       int64                    `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}

	var ccRule models.CCRule
	if err := db.DB.Where("id = ?", id).First(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("rule group not found")})
		return
	}

	if msgKey := services.GuardCCRuleGroupModify(ccRule); msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	// Permission check
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || ccRule.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		// User can't change type or assign to other users
		req.Type = "user"
	} else {
		// Admin can potentially change/reassign, but sticking to logic:
		if req.Type == "system" {
			ccRule.Internal = true
			ccRule.UserID = 0
		} else {
			ccRule.Internal = false
			if req.UserID > 0 {
				ccRule.UserID = req.UserID
			}
		}
	}

	if msgKey, err := services.GuardCCRuleGroupDisable(ccRule.ID, ccRule.Enable, req.IsOn); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update rule group")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	dataMap := map[string]interface{}{
		"rules":         req.Rules,
		"visible_users": req.VisibleUsers,
	}
	dataBytes, _ := json.Marshal(dataMap)

	ccRule.Name = req.Name
	ccRule.Description = req.Remark
	ccRule.Data = string(dataBytes)
	ccRule.IsShow = req.IsVisible
	ccRule.Sort = req.SortOrder
	ccRule.Enable = req.IsOn
	ccRule.UpdatedAt = time.Now()

	if !isUserRequest(ctx) && req.Type == "system" {
		if err := db.DB.Model(&models.CCRule{}).Where("id = ?", ccRule.ID).Updates(map[string]interface{}{
			"uid":       nil,
			"name":      ccRule.Name,
			"des":       ccRule.Description,
			"data":      ccRule.Data,
			"is_show":   ccRule.IsShow,
			"sort":      ccRule.Sort,
			"internal":  true,
			"enable":    ccRule.Enable,
			"update_at": ccRule.UpdatedAt,
		}).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update rule group")})
			return
		}
	} else if err := db.DB.Omit("CreatedAt", "TaskID").Save(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update rule group")})
		return
	}
	services.BumpConfigVersion("cc_rule", []int64{ccRule.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// GetRuleGroup Retrieves details of a rule group
// GET /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) GetRuleGroup(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	var rule models.CCRule
	if err := db.DB.Where("id = ?", id).First(&rule).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("rule not found")})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (rule.UserID != 0 && rule.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	rules := []gin.H{}
	visibleUsers := []int64{}
	if rule.Data != "" {
		var parsed struct {
			Rules        []map[string]interface{} `json:"rules"`
			VisibleUsers []int64                  `json:"visible_users"`
		}
		if err := json.Unmarshal([]byte(rule.Data), &parsed); err == nil {
			visibleUsers = parsed.VisibleUsers
			for _, r := range parsed.Rules {
				// Keep full map logic or simplify by just returning r
				rules = append(rules, r)
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":            rule.ID,
			"name":          rule.Name,
			"remark":        rule.Description,
			"user_id":       rule.UserID,
			"is_system":     rule.Internal || rule.UserID == 0,
			"type":          mapRuleType(rule.UserID, rule.Internal),
			"type_label":    ruleTypeLabel(rule.Internal || rule.UserID == 0),
			"in_use":        ccRuleInUse(rule.ID),
			"is_on":         rule.Enable,
			"is_visible":    rule.IsShow,
			"sort_order":    rule.Sort,
			"rules":         rules,
			"visible_users": visibleUsers,
		},
	})
}

// DeleteCCRuleGroup Deletes a rule group
// DELETE /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) DeleteCCRuleGroup(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}

	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	var rule models.CCRule
	if err := db.DB.Where("id = ?", id).First(&rule).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("rule group not found")})
		return
	}

	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || rule.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	if msgKey, err := services.GuardCCRuleGroupDelete(rule); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete rule group")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	if err := db.DB.Delete(&rule).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete rule group")})
		return
	}

	services.BumpConfigVersion("cc_rule", []int64{id})
	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// ListMatchers Lists available matchers
// GET /api/v1/admin/rules/cc/matchers
func (c *RuleController) ListMatchers(ctx *gin.Context) {
	query := db.DB.Model(&models.CCMatch{})
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID != 0 {
			query = query.Where("uid = ? OR uid = 0", userID)
		}
	}
	if name := strings.TrimSpace(ctx.Query("name")); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	status := strings.TrimSpace(ctx.Query("status"))
	if status == "on" {
		query = query.Where("enable = ?", true)
	} else if status == "off" {
		query = query.Where("enable = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load matchers")})
		return
	}
	var items []models.CCMatch
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load matchers")})
		return
	}

	// Correctly call loadUserNameMapFromMatchers
	userMap, _ := loadUserNameMapFromMatchers(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		inUse, _ := services.IsCCMatcherInUse(item.ID)
		isSystem := item.Internal || item.UserID == 0
		list = append(list, gin.H{
			"id":          item.ID,
			"user_id":     item.UserID,
			"uid":         item.UserID,
			"user":        gin.H{"username": userMap[item.UserID], "id": item.UserID},
			"name":        item.Name,
			"is_system":   isSystem,
			"type":        mapRuleType(item.UserID, item.Internal),
			"type_label":  ruleTypeLabel(isSystem),
			"in_use":      inUse,
			"status":      T("status.normal"),
			"is_on":       item.Enable,
			"create_time": item.CreatedAt.Format("2006-01-02 15:04:05"),
		},
		)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0, "data": gin.H{"list": list, "total": total},
	})
}

// CreateMatcher Creates a new matcher
// POST /api/v1/admin/rules/cc/matchers
func (c *RuleController) CreateMatcher(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		IsOn   bool                     `json:"is_on"`
		Rules  []map[string]interface{} `json:"rules"`
		UserID int64                    `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	userID := int64(0)
	internal := false

	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		req.Type = "user"
	} else {
		if req.Type == "system" {
			internal = true
			userID = 0
		} else {
			internal = false
			if req.UserID <= 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
				return
			}
			userID = req.UserID
		}
	}

	dataMap := map[string]interface{}{
		"rules": req.Rules,
	}
	dataBytes, _ := json.Marshal(dataMap)

	matcher := models.CCMatch{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Remark,
		Data:        string(dataBytes),
		Enable:      req.IsOn,
		Internal:    internal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createQuery := db.DB.Omit("TaskID")
	if internal {
		createQuery = db.DB.Omit("UserID", "TaskID")
	}
	if err := createQuery.Create(&matcher).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create matcher")})
		return
	}
	services.BumpConfigVersion("cc_match", []int64{matcher.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateMatcher Updates an existing matcher
// PUT /api/v1/admin/rules/cc/matchers/:id
func (c *RuleController) UpdateMatcher(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		IsOn   bool                     `json:"is_on"`
		Rules  []map[string]interface{} `json:"rules"`
		UserID int64                    `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}

	var matcher models.CCMatch
	if err := db.DB.Where("id = ?", id).First(&matcher).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("matcher not found")})
		return
	}

	if msgKey := services.GuardCCMatcherModify(matcher); msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || matcher.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		req.Type = "user"
	} else {
		// Admin logic
		if req.Type == "system" {
			matcher.Internal = true
			matcher.UserID = 0
		} else {
			if req.UserID <= 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
				return
			}
			matcher.Internal = false
			matcher.UserID = req.UserID
		}
	}

	if msgKey, err := services.GuardCCMatcherDisable(matcher.ID, matcher.Enable, req.IsOn); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update matcher")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	dataMap := map[string]interface{}{
		"rules": req.Rules,
	}
	dataBytes, _ := json.Marshal(dataMap)

	matcher.Name = req.Name
	matcher.Description = req.Remark
	matcher.Data = string(dataBytes)
	matcher.Enable = req.IsOn
	matcher.UpdatedAt = time.Now()

	if !isUserRequest(ctx) && req.Type == "system" {
		if err := db.DB.Model(&models.CCMatch{}).Where("id = ?", matcher.ID).Updates(map[string]interface{}{
			"uid":       nil,
			"name":      matcher.Name,
			"des":       matcher.Description,
			"data":      matcher.Data,
			"enable":    matcher.Enable,
			"internal":  true,
			"update_at": matcher.UpdatedAt,
		}).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update matcher")})
			return
		}
	} else if err := db.DB.Omit("CreatedAt", "TaskID").Save(&matcher).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update matcher")})
		return
	}
	services.BumpConfigVersion("cc_match", []int64{matcher.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// GetMatcher Retrieves details of a matcher
// GET /api/v1/admin/rules/cc/matchers/:id
func (c *RuleController) GetMatcher(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	var matcher models.CCMatch
	if err := db.DB.Where("id = ?", id).First(&matcher).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("matcher not found")})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (matcher.UserID != 0 && matcher.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	rules := []gin.H{}
	if matcher.Data != "" {
		var parsed struct {
			Rules []map[string]interface{} `json:"rules"`
		}
		if err := json.Unmarshal([]byte(matcher.Data), &parsed); err == nil {
			for _, r := range parsed.Rules {
				// Simply return the rule object as map
				rules = append(rules, r)
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":         matcher.ID,
			"user_id":    matcher.UserID,
			"uid":        matcher.UserID,
			"name":       matcher.Name,
			"remark":     matcher.Description,
			"is_system":  matcher.Internal || matcher.UserID == 0,
			"is_on":      matcher.Enable,
			"type":       mapRuleType(matcher.UserID, matcher.Internal),
			"type_label": ruleTypeLabel(matcher.Internal || matcher.UserID == 0),
			"in_use":     ccMatcherInUse(matcher.ID),
			"rules":      rules,
		},
	})
}

// DeleteMatcher Deletes a matcher
// DELETE /api/v1/admin/rules/cc/matchers/:id
func (c *RuleController) DeleteMatcher(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}

	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}

	var matcher models.CCMatch
	if err := db.DB.Where("id = ?", id).First(&matcher).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("matcher not found")})
		return
	}

	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || matcher.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	if services.IsCCMatcherInternal(matcher) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("cc_match.system_protected")})
		return
	}
	if msgKey, err := services.GuardCCMatcherDelete(matcher.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete matcher")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	if err := db.DB.Delete(&matcher).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete matcher")})
		return
	}

	services.BumpConfigVersion("cc_match", []int64{id})
	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// ListFilters Lists available filters
// GET /api/v1/admin/rules/cc/filters
func (c *RuleController) ListFilters(ctx *gin.Context) {
	query := db.DB.Model(&models.CCFilter{})
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID != 0 {
			query = query.Where("uid = ? OR uid = 0", userID)
		}
	}
	if name := strings.TrimSpace(ctx.Query("name")); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	status := strings.TrimSpace(ctx.Query("status"))
	if status == "on" {
		query = query.Where("enable = ?", true)
	} else if status == "off" {
		query = query.Where("enable = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load filters")})
		return
	}
	var items []models.CCFilter
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to load filters")})
		return
	}

	// Correctly call loadUserNameMapFromFilters
	userMap, _ := loadUserNameMapFromFilters(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		inUse, _ := services.IsCCFilterInUse(item.ID)
		isSystem := item.Internal || item.UserID == 0
		list = append(list, gin.H{
			"id":          item.ID,
			"user_id":     item.UserID,
			"uid":         item.UserID,
			"user":        gin.H{"username": userMap[item.UserID], "id": item.UserID},
			"name":        item.Name,
			"is_system":   isSystem,
			"type":        mapRuleType(item.UserID, item.Internal),
			"type_label":  ruleTypeLabel(isSystem),
			"in_use":      inUse,
			"action":      item.Type,
			"status":      T("status.normal"),
			"is_on":       item.Enable,
			"create_time": item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0, "data": gin.H{"list": list, "total": total},
	})
}

// CreateFilter Creates a new filter
// POST /api/v1/admin/rules/cc/filters
func (c *RuleController) CreateFilter(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	var req struct {
		Type         string                 `json:"type"`
		Name         string                 `json:"name"`
		Remark       string                 `json:"remark"`
		Enable       bool                   `json:"enable"`
		Action       string                 `json:"action"`
		MatchMode    string                 `json:"match_mode"`
		Blacklist    bool                   `json:"blacklist"`
		WithinSecond int                    `json:"within_second"`
		MaxReq       int                    `json:"max_req"`
		MaxReqPerURI int                    `json:"max_req_per_uri"`
		Auth         map[string]interface{} `json:"auth"`
		UserID       int64                  `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}
	userID := int64(0)
	internal := false

	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		req.Type = "user"
	} else {
		if req.Type == "system" {
			internal = true
			userID = 0
		} else {
			internal = false
			if req.UserID <= 0 {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
				return
			}
			userID = req.UserID
		}
	}

	extra := map[string]interface{}{
		"match_mode": req.MatchMode,
		"blacklist":  req.Blacklist,
	}
	if len(req.Auth) > 0 {
		extra["auth"] = req.Auth
	}
	extraBytes, _ := json.Marshal(extra)

	filter := models.CCFilter{
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Remark,
		Type:         req.Action,
		WithinSecond: req.WithinSecond,
		MaxReq:       req.MaxReq,
		MaxReqPerUri: req.MaxReqPerURI,
		Extra:        string(extraBytes),
		Internal:     internal,
		Enable:       req.Enable,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createQuery := db.DB.Omit("TaskID")
	if internal {
		createQuery = db.DB.Omit("UserID", "TaskID")
	}
	if err := createQuery.Create(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to create filter")})
		return
	}
	services.BumpConfigVersion("cc_filter", []int64{filter.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateFilter Updates an existing filter
// PUT /api/v1/admin/rules/cc/filters/:id
func (c *RuleController) UpdateFilter(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	var req struct {
		Type         string                 `json:"type"`
		Name         string                 `json:"name"`
		Remark       string                 `json:"remark"`
		Enable       bool                   `json:"enable"`
		Action       string                 `json:"action"`
		MatchMode    string                 `json:"match_mode"`
		Blacklist    bool                   `json:"blacklist"`
		WithinSecond int                    `json:"within_second"`
		MaxReq       int                    `json:"max_req"`
		MaxReqPerURI int                    `json:"max_req_per_uri"`
		Auth         map[string]interface{} `json:"auth"`
		UserID       int64                  `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("Invalid request")})
		return
	}

	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("filter not found")})
		return
	}

	if msgKey := services.GuardCCFilterModify(filter); msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || filter.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
		req.Type = "user"
	} else if req.Type == "system" {
		filter.Internal = true
		filter.UserID = 0
	} else {
		if req.UserID <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": T("user_id is required")})
			return
		}
		filter.Internal = false
		filter.UserID = req.UserID
	}

	if msgKey, err := services.GuardCCFilterDisable(filter.ID, filter.Enable, req.Enable); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update filter")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	extra := map[string]interface{}{
		"match_mode": req.MatchMode,
		"blacklist":  req.Blacklist,
	}
	if len(req.Auth) > 0 {
		extra["auth"] = req.Auth
	}
	extraBytes, _ := json.Marshal(extra)

	filter.Name = req.Name
	filter.Description = req.Remark
	filter.Type = req.Action
	filter.WithinSecond = req.WithinSecond
	filter.MaxReq = req.MaxReq
	filter.MaxReqPerUri = req.MaxReqPerURI
	filter.Extra = string(extraBytes)
	filter.Enable = req.Enable
	if isUserRequest(ctx) {
		filter.Internal = false
	}
	filter.UpdatedAt = time.Now()

	if !isUserRequest(ctx) && req.Type == "system" {
		if err := db.DB.Model(&models.CCFilter{}).Where("id = ?", filter.ID).Updates(map[string]interface{}{
			"uid":             nil,
			"name":            filter.Name,
			"des":             filter.Description,
			"type":            filter.Type,
			"within_second":   filter.WithinSecond,
			"max_req":         filter.MaxReq,
			"max_req_per_uri": filter.MaxReqPerUri,
			"extra":           filter.Extra,
			"enable":          filter.Enable,
			"internal":        true,
			"update_at":       filter.UpdatedAt,
		}).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update filter")})
			return
		}
	} else if err := db.DB.Omit("CreatedAt", "TaskID").Save(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to update filter")})
		return
	}
	services.BumpConfigVersion("cc_filter", []int64{filter.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// GetFilter Retrieves details of a filter
// GET /api/v1/admin/rules/cc/filters/:id
func (c *RuleController) GetFilter(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("filter not found")})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (filter.UserID != 0 && filter.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	extra := map[string]interface{}{}
	if filter.Extra != "" {
		_ = json.Unmarshal([]byte(filter.Extra), &extra)
	}
	auth, _ := extra["auth"].(map[string]interface{})

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              filter.ID,
			"user_id":         filter.UserID,
			"uid":             filter.UserID,
			"type":            mapRuleType(filter.UserID, filter.Internal),
			"type_label":      ruleTypeLabel(filter.Internal || filter.UserID == 0),
			"is_system":       filter.Internal || filter.UserID == 0,
			"in_use":          ccFilterInUse(filter.ID),
			"name":            filter.Name,
			"remark":          filter.Description,
			"enable":          filter.Enable,
			"is_on":           filter.Enable,
			"action":          filter.Type,
			"match_mode":      extra["match_mode"],
			"blacklist":       extra["blacklist"],
			"within_second":   filter.WithinSecond,
			"max_req":         filter.MaxReq,
			"max_req_per_uri": filter.MaxReqPerUri,
			"auth":            auth,
		},
	})
}

// DeleteFilter Deletes a filter
// DELETE /api/v1/admin/rules/cc/filters/:id
func (c *RuleController) DeleteFilter(ctx *gin.Context) {
	if !ensureCustomCCRuleAllowed(ctx) {
		return
	}
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": T("invalid id")})
		return
	}
	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": T("filter not found")})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || filter.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": T("forbidden")})
			return
		}
	}

	if services.IsCCFilterInternal(filter) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("cc_filter.system_protected")})
		return
	}
	if msgKey, err := services.GuardCCFilterDelete(filter.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete filter")})
		return
	} else if msgKey != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(msgKey)})
		return
	}

	if err := db.DB.Delete(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": T("Failed to delete filter")})
		return
	}
	services.BumpConfigVersion("cc_filter", []int64{id})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// GetRuleGroup Retrieves details of a rule group

func mapRuleType(userID int64, internal bool) string {
	if userID == 0 || internal {
		return "system"
	}
	return "user"
}

func ruleTypeLabel(isSystem bool) string {
	if isSystem {
		return "系统规则"
	}
	return "用户规则"
}

func ccRuleInUse(ruleID int64) bool {
	inUse, _ := services.IsCCRuleGroupInUse(ruleID)
	return inUse
}

func ccMatcherInUse(matcherID int64) bool {
	inUse, _ := services.IsCCMatcherInUse(matcherID)
	return inUse
}

func ccFilterInUse(filterID int64) bool {
	inUse, _ := services.IsCCFilterInUse(filterID)
	return inUse
}

func loadUserNameMapFromRules(items []models.CCRule) (map[int64]string, error) {
	ids := uniqueUserIDsFromRules(items)
	return loadUsersByIDs(ids)
}

func loadUserNameMapFromMatchers(items []models.CCMatch) (map[int64]string, error) {
	ids := uniqueUserIDsFromMatchers(items)
	return loadUsersByIDs(ids)
}

func loadUserNameMapFromFilters(items []models.CCFilter) (map[int64]string, error) {
	ids := uniqueUserIDsFromFilters(items)
	return loadUsersByIDs(ids)
}

func uniqueUserIDsFromRules(items []models.CCRule) []int64 {
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.UserID == 0 {
			continue
		}
		seen[item.UserID] = struct{}{}
	}
	return mapKeysToSlice(seen)
}

func uniqueUserIDsFromMatchers(items []models.CCMatch) []int64 {
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.UserID == 0 {
			continue
		}
		seen[item.UserID] = struct{}{}
	}
	return mapKeysToSlice(seen)
}

func uniqueUserIDsFromFilters(items []models.CCFilter) []int64 {
	seen := map[int64]struct{}{}
	for _, item := range items {
		if item.UserID == 0 {
			continue
		}
		seen[item.UserID] = struct{}{}
	}
	return mapKeysToSlice(seen)
}

func mapKeysToSlice(m map[int64]struct{}) []int64 {
	ids := make([]int64, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}

func loadUsersByIDs(ids []int64) (map[int64]string, error) {
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var users []models.User
	if err := db.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		result[u.ID] = u.Name
	}
	return result, nil
}
