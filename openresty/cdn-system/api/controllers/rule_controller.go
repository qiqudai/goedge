package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RuleController struct{}

// ================= CC Rules =================

// ListCCRuleGroups Lists CC rule groups
// GET /api/v1/admin/rules/cc/groups
func (c *RuleController) ListCCRuleGroups(ctx *gin.Context) {
	query := db.DB.Model(&models.CCRule{})
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

// ListMatchers Lists available matchers
// GET /api/v1/admin/rules/cc/matchers
func (c *RuleController) ListMatchers(ctx *gin.Context) {
	query := db.DB.Model(&models.CCMatch{})
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

// ListFilters Lists available filters
// GET /api/v1/admin/rules/cc/filters
func (c *RuleController) ListFilters(ctx *gin.Context) {
	query := db.DB.Model(&models.CCFilter{})
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
