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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load rules"})
		return
	}
	var items []models.CCRule
	if err := query.Order("sort asc, id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load rules"})
		return
	}

	userMap, _ := loadUserNameMapFromRules(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":          item.ID,
			"user":        userMap[item.UserID],
			"name":        item.Name,
			"is_system":   item.Internal || item.UserID == 0,
			"is_on":       item.Enable,
			"is_show":     item.IsShow,
			"status":      "normal",
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
	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		Rules  []map[string]interface{} `json:"rules"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	userID := int64(0)
	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		req.Type = "user"
	}

	// Prepare JSON data for 'data' column
	dataMap := map[string]interface{}{
		"rules": req.Rules,
	}
	dataBytes, _ := json.Marshal(dataMap)

	ccRule := models.CCRule{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Remark,
		Data:        string(dataBytes),
		Enable:      true,
		IsShow:      true,
		Internal:    req.Type == "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Type == "user" {
		// Simplified logic: assume admin wants to create for a specific user if uid passed?
		// For now, if "type" is user but no uid, assume it's a template for users or system default
		// Adjust based on requirements. The UI sends type='system' or 'user'.
		// If 'user', maybe we set Internal=false. UserID is 0 for admin created templates.
		ccRule.Internal = false
	} else {
		ccRule.Internal = true
	}

	if err := db.DB.Create(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule group"})
		return
	}
	services.BumpConfigVersion("cc_rule", []int64{ccRule.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateCCRuleGroup Updates an existing CC rule group
// PUT /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) UpdateCCRuleGroup(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		Rules  []map[string]interface{} `json:"rules"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var ccRule models.CCRule
	if err := db.DB.Where("id = ?", id).First(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "rule group not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || ccRule.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		req.Type = "user"
	}

	dataMap := map[string]interface{}{
		"rules": req.Rules,
	}
	dataBytes, _ := json.Marshal(dataMap)

	ccRule.Name = req.Name
	ccRule.Description = req.Remark
	ccRule.Data = string(dataBytes)
	ccRule.Internal = (req.Type == "system")
	ccRule.UpdatedAt = time.Now()

	if err := db.DB.Save(&ccRule).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule group"})
		return
	}
	services.BumpConfigVersion("cc_rule", []int64{ccRule.ID})

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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load matchers"})
		return
	}
	var items []models.CCMatch
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load matchers"})
		return
	}

	// Correctly call loadUserNameMapFromMatchers
	userMap, _ := loadUserNameMapFromMatchers(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":          item.ID,
			"user":        userMap[item.UserID],
			"name":        item.Name,
			"is_system":   item.Internal || item.UserID == 0,
			"status":      "normal",
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
	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		IsOn   bool                     `json:"is_on"`
		Rules  []map[string]interface{} `json:"rules"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	userID := int64(0)
	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		req.Type = "user"
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
		Internal:    req.Type == "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.DB.Create(&matcher).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create matcher"})
		return
	}
	services.BumpConfigVersion("cc_match", []int64{matcher.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateMatcher Updates an existing matcher
// PUT /api/v1/admin/rules/cc/matchers/:id
func (c *RuleController) UpdateMatcher(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Type   string                   `json:"type"`
		Name   string                   `json:"name"`
		Remark string                   `json:"remark"`
		IsOn   bool                     `json:"is_on"`
		Rules  []map[string]interface{} `json:"rules"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var matcher models.CCMatch
	if err := db.DB.Where("id = ?", id).First(&matcher).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "matcher not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || matcher.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		req.Type = "user"
	}

	dataMap := map[string]interface{}{
		"rules": req.Rules,
	}
	dataBytes, _ := json.Marshal(dataMap)

	matcher.Name = req.Name
	matcher.Description = req.Remark
	matcher.Data = string(dataBytes)
	matcher.Enable = req.IsOn
	matcher.Internal = (req.Type == "system")
	matcher.UpdatedAt = time.Now()

	if err := db.DB.Save(&matcher).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update matcher"})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var matcher models.CCMatch
	if err := db.DB.Where("id = ?", id).First(&matcher).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "matcher not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (matcher.UserID != 0 && matcher.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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
			"id":        matcher.ID,
			"name":      matcher.Name,
			"remark":    matcher.Description,
			"is_system": matcher.Internal || matcher.UserID == 0,
			"is_on":     matcher.Enable,
			"type":      mapRuleType(matcher.UserID, matcher.Internal),
			"rules":     rules,
		},
	})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load filters"})
		return
	}
	var items []models.CCFilter
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load filters"})
		return
	}

	// Correctly call loadUserNameMapFromFilters
	userMap, _ := loadUserNameMapFromFilters(items)
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":          item.ID,
			"user":        userMap[item.UserID],
			"name":        item.Name,
			"is_system":   item.Internal || item.UserID == 0,
			"type":        item.Type,
			"status":      "normal",
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
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	userID := int64(0)
	if isUserRequest(ctx) {
		userID = parseUserID(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		req.Type = "user"
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
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Remark,
		Type:          req.Action,
		WithinSecond:  req.WithinSecond,
		MaxReq:        req.MaxReq,
		MaxReqPerUri:  req.MaxReqPerURI,
		Extra:         string(extraBytes),
		Internal:      req.Type == "system",
		Enable:        req.Enable,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if req.Type == "user" {
		filter.Internal = false
	}

	if err := db.DB.Create(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create filter"})
		return
	}
	services.BumpConfigVersion("cc_filter", []int64{filter.ID})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateFilter Updates an existing filter
// PUT /api/v1/admin/rules/cc/filters/:id
func (c *RuleController) UpdateFilter(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || filter.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		req.Type = "user"
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
	filter.Internal = req.Type == "system"
	if req.Type == "user" {
		filter.Internal = false
	}
	filter.UpdatedAt = time.Now()

	if err := db.DB.Save(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update filter"})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (filter.UserID != 0 && filter.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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
			"id":             filter.ID,
			"type":           mapRuleType(filter.UserID, filter.Internal),
			"name":           filter.Name,
			"remark":         filter.Description,
			"enable":         filter.Enable,
			"action":         filter.Type,
			"match_mode":     extra["match_mode"],
			"blacklist":      extra["blacklist"],
			"within_second":  filter.WithinSecond,
			"max_req":        filter.MaxReq,
			"max_req_per_uri": filter.MaxReqPerUri,
			"auth":           auth,
		},
	})
}

// DeleteFilter Deletes a filter
// DELETE /api/v1/admin/rules/cc/filters/:id
func (c *RuleController) DeleteFilter(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var filter models.CCFilter
	if err := db.DB.Where("id = ?", id).First(&filter).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || filter.UserID != userID {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	if err := db.DB.Delete(&filter).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete filter"})
		return
	}
	services.BumpConfigVersion("cc_filter", []int64{id})

	ctx.JSON(http.StatusOK, gin.H{"code": 0})
}

// GetRuleGroup Retrieves details of a rule group
// GET /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) GetRuleGroup(ctx *gin.Context) {
	id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var rule models.CCRule
	if err := db.DB.Where("id = ?", id).First(&rule).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	if isUserRequest(ctx) {
		userID := parseUserID(mustGet(ctx, "userID"))
		if userID == 0 || (rule.UserID != 0 && rule.UserID != userID) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	rules := []gin.H{}
	if rule.Data != "" {
		var parsed struct {
			Rules []map[string]interface{} `json:"rules"`
		}
		if err := json.Unmarshal([]byte(rule.Data), &parsed); err == nil {
			for _, r := range parsed.Rules {
				item := gin.H{}
				if v, ok := r["matcher_id"]; ok {
					item["matcher_id"] = v
				}
				if v, ok := r["matcher_name"]; ok {
					item["matcher_name"] = v
				}
				if v, ok := r["filter1_id"]; ok {
					item["filter1_id"] = v
				}
				if v, ok := r["filter1_name"]; ok {
					item["filter1_name"] = v
				}
				if v, ok := r["action"]; ok {
					item["action"] = v
				}
				if v, ok := r["mode"]; ok {
					item["mode"] = v
				}
				if len(item) > 0 {
					rules = append(rules, item)
				}
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":        rule.ID,
			"name":      rule.Name,
			"remark":    rule.Description,
			"is_system": rule.Internal || rule.UserID == 0,
			"type":      mapRuleType(rule.UserID, rule.Internal),
			"rules":     rules,
		},
	})
}

func mapRuleType(userID int64, internal bool) string {
	if userID == 0 || internal {
		return "system"
	}
	return "user"
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
