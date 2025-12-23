package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PlanController struct{}

type planItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Desc            string `json:"desc"`
	Group           string `json:"group"`
	Region          int64  `json:"region"`
	LineGroup       int64  `json:"line_group"`
	BackupGroup     int64  `json:"backup_group"`
	TrafficLimit    int64  `json:"traffic_limit"`
	BandwidthLimit  string `json:"bandwidth_limit"`
	ConnectionLimit int64  `json:"connection_limit"`
	DomainLimit     int64  `json:"domain_limit"`
	CustomCCRules   bool   `json:"custom_cc_rules"`
	Websocket       bool   `json:"websocket"`
	PriceMonthly    int64  `json:"price_monthly"`
	PriceQuarterly  int64  `json:"price_quarterly"`
	PriceYearly     int64  `json:"price_yearly"`
	SortOrder       int    `json:"sort_order"`
	Status          bool   `json:"status"`
}

type userPlanItem struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	PlanName  string    `json:"plan_name"`
	ExpireAt  time.Time `json:"expire_at"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ListPlans - GET /api/v1/plans
func (ctr *PlanController) ListPlans(c *gin.Context) {
	var packages []models.Package
	if err := db.DB.Order("sort asc, id desc").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}
	list := make([]planItem, 0, len(packages))
	for _, p := range packages {
		list = append(list, planItem{
			ID:              p.ID,
			Name:            p.Name,
			Desc:            p.Description,
			Group:           "default",
			Region:          p.RegionID,
			LineGroup:       p.NodeGroupID,
			BackupGroup:     p.BackupNode,
			TrafficLimit:    p.Traffic,
			BandwidthLimit:  p.Bandwidth,
			ConnectionLimit: p.Connection,
			DomainLimit:     p.DomainLimit,
			CustomCCRules:   p.CustomCCRule,
			Websocket:       p.Websocket,
			PriceMonthly:    p.MonthPrice,
			PriceQuarterly:  p.QuarterPrice,
			PriceYearly:     p.YearPrice,
			SortOrder:       p.Sort,
			Status:          p.Enable,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": len(list)}})
}

// CreatePlan - POST /api/v1/plans
func (ctr *PlanController) CreatePlan(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	name, _ := getStringValue(payload, "name")
	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Name is required"})
		return
	}

	pkg := models.Package{
		Name:         name,
		Description:  getString(payload, "desc"),
		RegionID:     getInt64(payload, "region"),
		NodeGroupID:  getInt64(payload, "line_group"),
		BackupNode:   getInt64(payload, "backup_group"),
		MonthPrice:   getInt64(payload, "price_monthly"),
		QuarterPrice: getInt64(payload, "price_quarterly"),
		YearPrice:    getInt64(payload, "price_yearly"),
		Traffic:      getInt64(payload, "traffic_limit"),
		Bandwidth:    getString(payload, "bandwidth_limit"),
		Connection:   getInt64(payload, "connection_limit"),
		DomainLimit:  getInt64(payload, "domain_limit"),
		Websocket:    getBool(payload, "websocket"),
		CustomCCRule: getBool(payload, "custom_cc_rules"),
		Sort:         int(getInt64(payload, "sort_order")),
		Enable:       getBool(payload, "status"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.DB.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Created"})
}

// UpdatePlan - PUT /api/v1/plans/:id
func (ctr *PlanController) UpdatePlan(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	updates := map[string]interface{}{"update_at": time.Now()}
	if v, ok := getStringValue(payload, "name"); ok {
		updates["name"] = v
	}
	if v, ok := getStringValue(payload, "desc"); ok {
		updates["des"] = v
	}
	if hasKey(payload, "region") {
		updates["region_id"] = getInt64(payload, "region")
	}
	if hasKey(payload, "line_group") {
		updates["node_group_id"] = getInt64(payload, "line_group")
	}
	if hasKey(payload, "backup_group") {
		updates["backup_node_group"] = getInt64(payload, "backup_group")
	}
	if hasKey(payload, "price_monthly") {
		updates["month_price"] = getInt64(payload, "price_monthly")
	}
	if hasKey(payload, "price_quarterly") {
		updates["quarter_price"] = getInt64(payload, "price_quarterly")
	}
	if hasKey(payload, "price_yearly") {
		updates["year_price"] = getInt64(payload, "price_yearly")
	}
	if hasKey(payload, "traffic_limit") {
		updates["traffic"] = getInt64(payload, "traffic_limit")
	}
	if hasKey(payload, "bandwidth_limit") {
		updates["bandwidth"] = getString(payload, "bandwidth_limit")
	}
	if hasKey(payload, "connection_limit") {
		updates["connection"] = getInt64(payload, "connection_limit")
	}
	if hasKey(payload, "domain_limit") {
		updates["domain"] = getInt64(payload, "domain_limit")
	}
	if hasKey(payload, "websocket") {
		updates["websocket"] = getBool(payload, "websocket")
	}
	if hasKey(payload, "custom_cc_rules") {
		updates["custom_cc_rule"] = getBool(payload, "custom_cc_rules")
	}
	if hasKey(payload, "sort_order") {
		updates["sort"] = int(getInt64(payload, "sort_order"))
	}
	if hasKey(payload, "status") {
		updates["enable"] = getBool(payload, "status")
	}

	if err := db.DB.Model(&models.Package{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Updated"})
}

// DeletePlan - DELETE /api/v1/plans/:id
func (ctr *PlanController) DeletePlan(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Deleted"})
}

// ListUserPlans - GET /api/v1/user_plans
func (ctr *PlanController) ListUserPlans(c *gin.Context) {
	var userPlans []models.UserPackage
	if err := db.DB.Order("id desc").Find(&userPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}
	now := time.Now()
	list := make([]userPlanItem, 0, len(userPlans))
	for _, p := range userPlans {
		status := "active"
		if !p.EndAt.IsZero() && p.EndAt.Before(now) {
			status = "expired"
		}
		list = append(list, userPlanItem{
			ID:        p.ID,
			UserID:    p.UserID,
			PlanName:  p.Name,
			ExpireAt:  p.EndAt,
			Status:    status,
			CreatedAt: p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

// AssignUserPlan - POST /api/v1/user_plans/assign
func (ctr *PlanController) AssignUserPlan(c *gin.Context) {
	var req struct {
		PlanID int64 `json:"plan_id"`
		UserID int64 `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid params"})
		return
	}
	if req.PlanID == 0 || req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "plan_id and user_id are required"})
		return
	}

	var pkg models.Package
	if err := db.DB.Where("id = ?", req.PlanID).First(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Plan not found"})
		return
	}

	now := time.Now()
	userPkg := models.UserPackage{
		UserID:      req.UserID,
		Name:        pkg.Name,
		PackageID:   pkg.ID,
		RegionID:    pkg.RegionID,
		NodeGroupID: pkg.NodeGroupID,
		Traffic:     pkg.Traffic,
		Bandwidth:   pkg.Bandwidth,
		Connection:  pkg.Connection,
		DomainLimit: pkg.DomainLimit,
		CreatedAt:   now,
		StartAt:     now,
		EndAt:       now.AddDate(0, 1, 0),
	}

	if err := db.DB.Create(&userPkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Assign failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Assigned"})
}

func hasKey(payload map[string]interface{}, key string) bool {
	_, ok := payload[key]
	return ok
}

func getStringValue(payload map[string]interface{}, key string) (string, bool) {
	if !hasKey(payload, key) {
		return "", false
	}
	return getString(payload, key), true
}

func getString(payload map[string]interface{}, key string) string {
	if v, ok := payload[key]; ok {
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		case float64:
			if t == float64(int64(t)) {
				return strconv.FormatInt(int64(t), 10)
			}
			return strconv.FormatFloat(t, 'f', -1, 64)
		case int:
			return strconv.Itoa(t)
		case int64:
			return strconv.FormatInt(t, 10)
		}
	}
	return ""
}

func getInt64(payload map[string]interface{}, key string) int64 {
	if v, ok := payload[key]; ok {
		switch t := v.(type) {
		case float64:
			return int64(t)
		case int:
			return int64(t)
		case int64:
			return t
		case string:
			if t == "" {
				return 0
			}
			if i, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

func getBool(payload map[string]interface{}, key string) bool {
	if v, ok := payload[key]; ok {
		switch t := v.(type) {
		case bool:
			return t
		case float64:
			return t != 0
		case int:
			return t != 0
		case int64:
			return t != 0
		case string:
			t = strings.ToLower(strings.TrimSpace(t))
			return t == "1" || t == "true" || t == "yes" || t == "on"
		}
	}
	return false
}
