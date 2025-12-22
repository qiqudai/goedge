package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type RuleController struct{}

// ================= CC Rules =================

// ListCCRuleGroups Lists CC rule groups
// GET /api/v1/admin/rules/cc/groups
func (c *RuleController) ListCCRuleGroups(ctx *gin.Context) {
	type RuleGroup struct {
		ID         int    `json:"id"`
		User       string `json:"user"` // system or user
		Name       string `json:"name"`
		IsSystem   bool   `json:"is_system"`
		IsOn       bool   `json:"is_on"`
		Status     string `json:"status"` // normal
		SortOrder  int    `json:"sort_order"`
		CreateTime string `json:"create_time"`
	}

	// Mock Data
	list := []RuleGroup{
		{10002, "系统", "关闭", true, true, "正常", 1, "2025-04-02 09:10:16"},
		{6, "系统", "宽松", true, true, "正常", 2, "2025-04-02 09:10:16"},
		{1, "系统", "JS验证", true, true, "正常", 3, "2025-04-02 09:10:15"},
		{10001, "系统", "5秒盾", true, true, "正常", 4, "2025-04-02 09:10:16"},
		{10000, "系统", "点击验证", true, true, "正常", 5, "2025-04-02 09:10:16"},
		{2, "系统", "滑块验证", true, true, "正常", 6, "2025-04-02 09:10:15"},
		{4, "系统", "验证码", true, true, "正常", 7, "2025-04-02 09:10:16"},
		{10003, "系统", "旋转图片", true, true, "正常", 8, "2025-04-02 09:10:16"},
		{10004, "系统", "点击验证(简单)", true, true, "正常", 9, "2025-04-02 09:10:16"},
		{10005, "系统", "滑块验证(简单)", true, true, "正常", 10, "2025-04-02 09:10:16"},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": len(list),
		},
	})
}

// ListMatchers Lists available matchers
// GET /api/v1/admin/rules/cc/matchers
func (c *RuleController) ListMatchers(ctx *gin.Context) {
	type Matcher struct {
		ID         int    `json:"id"`
		User       string `json:"user"`
		Name       string `json:"name"`
		IsSystem   bool   `json:"is_system"`
		Status     string `json:"status"`
		CreateTime string `json:"create_time"`
		Rules      string `json:"rules"` // Simplified for list
	}

	// Mock Matchers
	list := []Matcher{
		{5, "系统", "匹配内置资源", true, "正常", "2025-04-02 09:10:15", ""},
		{4, "系统", "匹配静态资源", true, "正常", "2025-04-02 09:10:15", ""},
		{3, "系统", "匹配所有资源", true, "正常", "2025-04-02 09:10:15", ""},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0, "data": gin.H{"list": list, "total": len(list)},
	})
}

// ListFilters Lists available filters
// GET /api/v1/admin/rules/cc/filters
func (c *RuleController) ListFilters(ctx *gin.Context) {
	type Filter struct {
		ID         int    `json:"id"`
		User       string `json:"user"`
		Name       string `json:"name"`
		IsSystem   bool   `json:"is_system"`
		Type       string `json:"type"` // e.g., "滑动验证(简单)"
		Status     string `json:"status"`
		CreateTime string `json:"create_time"`
	}

	// Mock Filters based on screenshot/requirements
	list := []Filter{
		{10004, "系统", "滑动过滤(简单)60-5", true, "滑动验证(简单)", "正常", "2025-04-02 09:10:15"},
		{10003, "系统", "点击过滤(简单)60-5", true, "点击验证(简单)", "正常", "2025-04-02 09:10:15"},
		{10002, "系统", "旋转图片60-5", true, "旋转图片", "正常", "2025-04-02 09:10:15"},
		{10001, "系统", "5秒盾60-5", true, "5秒盾", "正常", "2025-04-02 09:10:15"},
		{10000, "系统", "点击验证60-5", true, "点击验证", "正常", "2025-04-02 09:10:15"},
		{12, "系统", "临时白名单专用2", true, "请求频率", "正常", "2025-04-02 09:10:15"},
		{11, "系统", "临时白名单专用1", true, "请求频率", "正常", "2025-04-02 09:10:15"},
		{10, "系统", "请求速率5-200-50", true, "请求速率", "正常", "2025-04-02 09:10:15"},
		{9, "系统", "请求速率5-300-50", true, "请求速率", "正常", "2025-04-02 09:10:15"},
		{8, "系统", "浏览器特征60-5", true, "无感验证", "正常", "2025-04-02 09:10:15"},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0, "data": gin.H{"list": list, "total": len(list)},
	})
}

// GetRuleGroup Retrieves details of a rule group
// GET /api/v1/admin/rules/cc/groups/:id
func (c *RuleController) GetRuleGroup(ctx *gin.Context) {
	// id := ctx.Param("id")

	// Mock a detailed response for editing
	// Rules structure: Matcher + Filters + Action
	type Rule struct {
		ID          int    `json:"id"`
		MatcherID   int    `json:"matcher_id"`
		MatcherName string `json:"matcher_name"`
		Filter1ID   int    `json:"filter1_id"`
		Filter1Name string `json:"filter1_name"`
		Filter2ID   int    `json:"filter2_id"`
		Filter2Name string `json:"filter2_name"`
		Action      string `json:"action"` // block, allow, etc.
		Mode        string `json:"mode"`   // next, stop
		IsOn        bool   `json:"is_on"`
	}

	rules := []Rule{
		{1, 1, "匹配所有资源", 1, "滑动过滤(简单)60-5", 0, "", "block", "next", true},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":        10002,
			"name":      "关闭",
			"remark":    "内置规则",
			"is_system": true,
			"type":      "system", // system or user
			"rules":     rules,
		},
	})
}
