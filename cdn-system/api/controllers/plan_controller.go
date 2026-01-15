package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	L2Origin        bool   `json:"l2_origin"`
	PriceMonthly    int64  `json:"price_monthly"`
	PriceQuarterly  int64  `json:"price_quarterly"`
	PriceYearly     int64  `json:"price_yearly"`
	SortOrder       int    `json:"sort_order"`
	Status          bool   `json:"status"`
}

type planDetail struct {
	planItem
	HTTPPort           int64      `json:"http_port"`
	StreamPort         int64      `json:"stream_port"`
	CnameDomain        string     `json:"cname_domain"`
	CnameHostname2     string     `json:"cname_hostname2"`
	CnameMode          string     `json:"cname_mode"`
	BuyNumLimit        int64      `json:"buy_num_limit"`
	BackendIPLimit     string     `json:"backend_ip_limit"`
	IDVerify           bool       `json:"id_verify"`
	BeforeExpDaysRenew int64      `json:"before_exp_days_renew"`
	ExpireAt           *time.Time `json:"expire"`
	Owner              string     `json:"owner"`
}

type userPlanItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	UserName        string    `json:"user_name"`
	PackageID       int64     `json:"package_id"`
	PackageName     string    `json:"package_name"`
	PlanName        string    `json:"plan_name"`
	RecordID        string    `json:"record_id"`
	RegionID        int64     `json:"region_id"`
	NodeGroupID     int64     `json:"node_group_id"`
	BackupGroupID   int64     `json:"backup_group_id"`
	EnableBackup    bool      `json:"enable_backup_group"`
	Traffic         int64     `json:"traffic"`
	Bandwidth       string    `json:"bandwidth"`
	Connection      int64     `json:"connection"`
	DomainLimit     int64     `json:"domain"`
	MainDomainLimit int64     `json:"main_domain_limit"`
	HTTPPort        int64     `json:"http_port"`
	StreamPort      int64     `json:"stream_port"`
	CustomCCRule    bool      `json:"custom_cc_rule"`
	Websocket       bool      `json:"websocket"`
	CnameDomain     string    `json:"cname_domain"`
	CnameHostname   string    `json:"cname_hostname"`
	CnameHostname2  string    `json:"cname_hostname2"`
	CnameMode       string    `json:"cname_mode"`
	StartAt         time.Time `json:"start_at"`
	EndAt           time.Time `json:"end_at"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListPlans - GET /api/v1/plans
func (ctr *PlanController) ListPlans(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	var packages []models.Package
	if err := db.DB.Order("sort asc, id desc").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
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
			L2Origin:        p.L2Origin,
			PriceMonthly:    p.MonthPrice,
			PriceQuarterly:  p.QuarterPrice,
			PriceYearly:     p.YearPrice,
			SortOrder:       p.Sort,
			Status:          p.Enable,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": len(list)}})
}

// GetPlan - GET /api/v1/plans/:id
func (ctr *PlanController) GetPlan(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	var pkg models.Package
	if err := db.DB.Where("id = ?", id).First(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Plan not found")})
		return
	}

	detail := planDetail{
		planItem: planItem{
			ID:              pkg.ID,
			Name:            pkg.Name,
			Desc:            pkg.Description,
			Group:           "default",
			Region:          pkg.RegionID,
			LineGroup:       pkg.NodeGroupID,
			BackupGroup:     pkg.BackupNode,
			TrafficLimit:    pkg.Traffic,
			BandwidthLimit:  pkg.Bandwidth,
			ConnectionLimit: pkg.Connection,
			DomainLimit:     pkg.DomainLimit,
			CustomCCRules:   pkg.CustomCCRule,
			Websocket:       pkg.Websocket,
			L2Origin:        pkg.L2Origin,
			PriceMonthly:    pkg.MonthPrice,
			PriceQuarterly:  pkg.QuarterPrice,
			PriceYearly:     pkg.YearPrice,
			SortOrder:       pkg.Sort,
			Status:          pkg.Enable,
		},
		HTTPPort:           pkg.HttpPort,
		StreamPort:         pkg.StreamPort,
		CnameDomain:        pkg.CnameDomain,
		CnameHostname2:     pkg.CnameHost2,
		CnameMode:          pkg.CnameMode,
		BuyNumLimit:        pkg.BuyNumLimit,
		BackendIPLimit:     pkg.BackendIPLimit,
		IDVerify:           pkg.IDVerify,
		BeforeExpDaysRenew: pkg.BeforeExpDaysRenew,
		ExpireAt:           pkg.ExpireAt,
		Owner:              pkg.Owner,
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": detail})
}

// CreatePlan - POST /api/v1/plans
func (ctr *PlanController) CreatePlan(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid JSON")})
		return
	}

	name, _ := getStringValue(payload, "name")
	if strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Name is required")})
		return
	}

	pkg := models.Package{
		Name:               name,
		Description:        getString(payload, "desc"),
		RegionID:           getInt64(payload, "region"),
		NodeGroupID:        getInt64(payload, "line_group"),
		BackupNode:         getInt64(payload, "backup_group"),
		CnameDomain:        getString(payload, "cname_domain"),
		CnameHost2:         getString(payload, "cname_hostname2"),
		CnameMode:          getString(payload, "cname_mode"),
		MonthPrice:         getInt64(payload, "price_monthly"),
		QuarterPrice:       getInt64(payload, "price_quarterly"),
		YearPrice:          getInt64(payload, "price_yearly"),
		Traffic:            getInt64(payload, "traffic_limit"),
		Bandwidth:          getString(payload, "bandwidth_limit"),
		Connection:         getInt64(payload, "connection_limit"),
		DomainLimit:        getInt64(payload, "domain_limit"),
		HttpPort:           getInt64(payload, "http_port"),
		StreamPort:         getInt64(payload, "stream_port"),
		ExpireAt:           getTimePtr(payload, "expire"),
		BuyNumLimit:        getInt64(payload, "buy_num_limit"),
		BackendIPLimit:     getString(payload, "backend_ip_limit"),
		IDVerify:           getBool(payload, "id_verify"),
		BeforeExpDaysRenew: getInt64(payload, "before_exp_days_renew"),
		Websocket:          getBool(payload, "websocket"),
		CustomCCRule:       getBool(payload, "custom_cc_rules"),
		L2Origin:           getBool(payload, "l2_origin"),
		Sort:               int(getInt64(payload, "sort_order")),
		Owner:              getString(payload, "owner"),
		Enable:             getBool(payload, "status"),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := db.DB.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Create Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Plan Created")})
}

// UpdatePlan - PUT /api/v1/plans/:id
func (ctr *PlanController) UpdatePlan(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid ID")})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid JSON")})
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
	if hasKey(payload, "cname_domain") {
		updates["cname_domain"] = getString(payload, "cname_domain")
	}
	if hasKey(payload, "cname_hostname2") {
		updates["cname_hostname2"] = getString(payload, "cname_hostname2")
	}
	if hasKey(payload, "cname_mode") {
		updates["cname_mode"] = getString(payload, "cname_mode")
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
		updates["traffic"] = int32(getInt64(payload, "traffic_limit"))
	}
	if hasKey(payload, "bandwidth_limit") {
		updates["bandwidth"] = getString(payload, "bandwidth_limit")
	}
	if hasKey(payload, "connection_limit") {
		updates["connection"] = int32(getInt64(payload, "connection_limit"))
	}
	if hasKey(payload, "domain_limit") {
		updates["domain"] = int32(getInt64(payload, "domain_limit"))
	}
	if hasKey(payload, "http_port") {
		updates["http_port"] = int32(getInt64(payload, "http_port"))
	}
	if hasKey(payload, "stream_port") {
		updates["stream_port"] = int32(getInt64(payload, "stream_port"))
	}
	if hasKey(payload, "expire") {
		updates["expire"] = getTimeUpdateValue(payload, "expire")
	}
	if hasKey(payload, "buy_num_limit") {
		updates["buy_num_limit"] = getInt64(payload, "buy_num_limit")
	}
	if hasKey(payload, "backend_ip_limit") {
		updates["backend_ip_limit"] = getString(payload, "backend_ip_limit")
	}
	if hasKey(payload, "id_verify") {
		updates["id_verify"] = getBool(payload, "id_verify")
	}
	if hasKey(payload, "before_exp_days_renew") {
		updates["before_exp_days_renew"] = getInt64(payload, "before_exp_days_renew")
	}
	if hasKey(payload, "websocket") {
		updates["websocket"] = getBool(payload, "websocket")
	}
	if hasKey(payload, "custom_cc_rules") {
		updates["custom_cc_rule"] = getBool(payload, "custom_cc_rules")
	}
	if hasKey(payload, "l2_origin") {
		updates["l2_origin"] = getBool(payload, "l2_origin")
	}
	if hasKey(payload, "sort_order") {
		updates["sort"] = int(getInt64(payload, "sort_order"))
	}
	if hasKey(payload, "owner") {
		updates["owner"] = getString(payload, "owner")
	}
	if hasKey(payload, "status") {
		updates["enable"] = getBool(payload, "status")
	}

	if err := db.DB.Model(&models.Package{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Plan Updated")})
}

// DeletePlan - DELETE /api/v1/plans/:id
func (ctr *PlanController) DeletePlan(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete Failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Plan Deleted")})
}

// ListUserPlans - GET /api/v1/user_plans
func (ctr *PlanController) ListUserPlans(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	if err := ensureUserPackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	var userPlans []models.UserPackage
	if err := db.DB.Order("id desc").Find(&userPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("DB Error")})
		return
	}
	userIDs := make([]int64, 0, len(userPlans))
	packageIDs := make([]int64, 0, len(userPlans))
	seenUsers := map[int64]struct{}{}
	seenPackages := map[int64]struct{}{}
	for _, p := range userPlans {
		if p.UserID != 0 {
			uid := int64(p.UserID)
			if _, ok := seenUsers[uid]; !ok {
				seenUsers[uid] = struct{}{}
				userIDs = append(userIDs, uid)
			}
		}
		if p.PackageID != 0 {
			pkgID := int64(p.PackageID)
			if _, ok := seenPackages[pkgID]; !ok {
				seenPackages[pkgID] = struct{}{}
				packageIDs = append(packageIDs, pkgID)
			}
		}
	}

	userNameMap := map[int64]string{}
	if len(userIDs) > 0 {
		var users []models.User
		if err := db.DB.Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				name := strings.TrimSpace(u.Name)
				if name == "" {
					name = strings.TrimSpace(u.Email)
				}
				if name == "" {
					name = strings.TrimSpace(u.Phone)
				}
				if name == "" {
					name = strings.TrimSpace(u.QQ)
				}
				userNameMap[u.ID] = name
			}
		}
	}
	packageNameMap := map[int64]string{}
	if len(packageIDs) > 0 {
		var packages []models.Package
		if err := db.DB.Where("id IN ?", packageIDs).Find(&packages).Error; err == nil {
			for _, p := range packages {
				packageNameMap[p.ID] = p.Name
			}
		}
	}

	now := time.Now()
	list := make([]userPlanItem, 0, len(userPlans))
	for _, p := range userPlans {
		status := "active"
		if !p.EndAt.IsZero() && p.EndAt.Before(now) {
			status = "expired"
		}
		recordID := strings.TrimSpace(p.RecordID)
		if recordID == "" {
			if newID, err := generateUniqueRecordID(); err == nil {
				recordID = newID
				_ = db.DB.Model(&models.UserPackage{}).
					Where("id = ?", p.ID).
					Update("record_id", newID).Error
			}
		}
		packageName := packageNameMap[int64(p.PackageID)]
		if packageName == "" {
			packageName = p.Name
		}
		startAt := p.StartAt
		if startAt.IsZero() {
			startAt = p.CreatedAt
		}
		list = append(list, userPlanItem{
			ID:              p.ID,
			UserID:          int64(p.UserID),
			UserName:        userNameMap[int64(p.UserID)],
			PackageID:       int64(p.PackageID),
			PackageName:     packageName,
			PlanName:        p.Name,
			RecordID:        recordID,
			RegionID:        p.RegionID,
			NodeGroupID:     p.NodeGroupID,
			BackupGroupID:   p.BackupNodeGroup,
			EnableBackup:    p.EnableBackup,
			Traffic:         int64(p.Traffic),
			Bandwidth:       p.Bandwidth,
			Connection:      int64(p.Connection),
			DomainLimit:     int64(p.DomainLimit),
			MainDomainLimit: int64(p.MainDomainLimit),
			HTTPPort:        int64(p.HTTPPortLimit),
			StreamPort:      int64(p.StreamPortLimit),
			CustomCCRule:    p.CustomCCRule,
			Websocket:       p.Websocket,
			CnameDomain:     p.CnameDomain,
			CnameHostname:   p.CnameHostname,
			CnameHostname2:  p.CnameHostname2,
			CnameMode:       p.CnameMode,
			StartAt:         startAt,
			EndAt:           p.EndAt,
			Status:          status,
			CreatedAt:       p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

// AssignUserPlan - POST /api/v1/user_plans/assign
func (ctr *PlanController) AssignUserPlan(c *gin.Context) {
	if err := ensurePackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	if err := ensureUserPackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	var req struct {
		PlanID         int64  `json:"plan_id"`
		UserID         int64  `json:"user_id"`
		DurationMonths int    `json:"duration_months"`
		EndAt          string `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid params")})
		return
	}
	if req.PlanID == 0 || req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("plan_id and user_id are required")})
		return
	}

	var user models.User
	if err := db.DB.Where("id = ?", req.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("User not found")})
		return
	}

	var pkg models.Package
	if err := db.DB.Where("id = ?", req.PlanID).First(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Plan not found")})
		return
	}

	now := time.Now()
	endAt := time.Time{}
	if strings.TrimSpace(req.EndAt) != "" {
		if tm := parseTimeString(req.EndAt); tm != nil {
			endAt = *tm
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid end_at")})
			return
		}
	} else {
		months := req.DurationMonths
		if months <= 0 {
			months = 1
		}
		endAt = now.AddDate(0, months, 0)
	}
	if endAt.Before(now) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("end_at must be in the future")})
		return
	}

	userPkg := models.UserPackage{
		UserID:          int32(req.UserID),
		Name:            pkg.Name,
		PackageID:       int32(pkg.ID),
		RegionID:        pkg.RegionID,
		NodeGroupID:     pkg.NodeGroupID,
		BackupNodeGroup: pkg.BackupNode,
		EnableBackup:    false,
		CnameDomain:     pkg.CnameDomain,
		CnameHostname2:  pkg.CnameHost2,
		CnameMode:       pkg.CnameMode,
		Traffic:         int32(pkg.Traffic),
		Bandwidth:       pkg.Bandwidth,
		Connection:      int32(pkg.Connection),
		DomainLimit:     int32(pkg.DomainLimit),
		HTTPPortLimit:   int32(pkg.HttpPort),
		StreamPortLimit: int32(pkg.StreamPort),
		CustomCCRule:    pkg.CustomCCRule,
		Websocket:       pkg.Websocket,
		L2Origin:        pkg.L2Origin,
		MonthPrice:      pkg.MonthPrice,
		QuarterPrice:    pkg.QuarterPrice,
		YearPrice:       pkg.YearPrice,
		CreatedAt:       now,
		StartAt:         now,
		EndAt:           endAt,
	}
	if strings.TrimSpace(userPkg.RecordID) == "" {
		recordID, err := generateUniqueRecordID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Failed to generate record id")})
			return
		}
		userPkg.RecordID = recordID
	}

	if err := db.DB.Create(&userPkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Assign failed")})
		return
	}

	// Trigger Sync
	if err := services.NewUserPackageService().SyncUserPackage(userPkg.ID, "update"); err != nil {
		fmt.Printf("[WARN] SyncUserPackage (Assign) Failed: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Assigned")})
}

func generateUniqueRecordID() (string, error) {
	for i := 0; i < 5; i++ {
		id, err := randomToken(8)
		if err != nil {
			return "", err
		}
		var count int64
		if err := db.DB.Model(&models.UserPackage{}).Where("record_id = ?", id).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return id, nil
		}
	}
	return "", errors.New("failed to allocate unique record id")
}

func randomToken(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf), nil
}

// UpdateUserPlan - PUT /api/v1/admin/user_plans/:id
func (ctr *PlanController) UpdateUserPlan(c *gin.Context) {
	if err := ensureUserPackageL2OriginColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Database Error")})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid id")})
		return
	}
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid params")})
		return
	}

	updates := map[string]interface{}{}
	if hasKey(payload, "name") {
		updates["name"] = getString(payload, "name")
	}
	if hasKey(payload, "end_at") {
		if tm := getTimePtr(payload, "end_at"); tm != nil {
			updates["end_at"] = tm
		} else {
			// If empty string, means no expiration? or error?
			// Existing logic flagged error or ignored. Let's assume ignore if empty or error for now unless imperative.
			// Previous code: if invalid, return 400.
			if v, ok := getStringValue(payload, "end_at"); ok && v != "" {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("invalid end_at")})
				return
			}
		}
	}
	if hasKey(payload, "region_id") {
		updates["region_id"] = getInt64(payload, "region_id")
	}
	if hasKey(payload, "node_group_id") {
		updates["node_group_id"] = getInt64(payload, "node_group_id")
	}
	if hasKey(payload, "backup_group_id") {
		backupGroupID := getInt64(payload, "backup_group_id")
		updates["backup_node_group"] = backupGroupID
		if !hasKey(payload, "enable_backup_group") {
			updates["enable_backup_group"] = backupGroupID > 0
		}
	}
	if hasKey(payload, "enable_backup_group") {
		updates["enable_backup_group"] = getBool(payload, "enable_backup_group")
	}
	if hasKey(payload, "main_domain_limit") {
		updates["main_domain_limit"] = getInt64(payload, "main_domain_limit")
	}
	// CNAME Fields support
	if hasKey(payload, "cname_domain") {
		updates["cname_domain"] = getString(payload, "cname_domain")
	}
	if hasKey(payload, "cname_hostname") {
		updates["cname_hostname"] = getString(payload, "cname_hostname")
	}
	if hasKey(payload, "cname_mode") {
		updates["cname_mode"] = getString(payload, "cname_mode")
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("no updates")})
		return
	}

	if err := db.DB.Model(&models.UserPackage{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Update Failed")})
		return
	}

	// Trigger Sync
	if err := services.NewUserPackageService().SyncUserPackage(id, "update"); err != nil {
		fmt.Printf("[WARN] SyncUserPackage (Update) Failed: %v\n", err)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

// DeleteUserPlans - DELETE /api/v1/admin/user_plans
func (ctr *PlanController) DeleteUserPlans(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid params")})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("ids is required")})
		return
	}
	if err := db.DB.Where("id IN ?", payload.IDs).Delete(&models.UserPackage{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("Delete failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Deleted")})
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

func parseTimeString(input string) *time.Time {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil
	}
	if tm, err := time.Parse(time.RFC3339, s); err == nil {
		return &tm
	}
	if tm, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return &tm
	}
	if tm, err := time.Parse("2006-01-02", s); err == nil {
		return &tm
	}
	return nil
}
func getTimePtr(payload map[string]interface{}, key string) *time.Time {
	if v, ok := payload[key]; ok {
		switch t := v.(type) {
		case string:
			t = strings.TrimSpace(t)
			if t == "" {
				return nil
			}
			if t == "0000-00-00" || t == "0000-00-00 00:00:00" {
				return nil
			}
			if tm, err := time.Parse(time.RFC3339, t); err == nil {
				return &tm
			}
			if tm, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
				return &tm
			}
			if tm, err := time.Parse("2006-01-02", t); err == nil {
				return &tm
			}
		}
	}
	return nil
}

func getTimeUpdateValue(payload map[string]interface{}, key string) interface{} {
	if !hasKey(payload, key) {
		return nil
	}
	if tm := getTimePtr(payload, key); tm != nil {
		return tm
	}
	return gorm.Expr("NULL")
}
