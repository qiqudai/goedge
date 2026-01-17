package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeGroupController struct{}

type nodeGroupPolicy struct {
	Ipv4Resolution string `json:"ipv4_resolution"`
	L2Config       string `json:"l2_config"`
	SortOrder      int    `json:"sort_order"`
}

type nodeGroupCount struct {
	NodeGroupID int64 `gorm:"column:node_group_id"`
	Count       int64 `gorm:"column:cnt"`
}

type nodeGroupView struct {
	models.NodeGroup
	NodeCount    int64 `json:"node_count"`
	SiteCount    int64 `json:"site_count"`
	ForwardCount int64 `json:"forward_count"`
}

// ListNodeGroups
// GET /api/v1/admin/node-groups
func (ctr *NodeGroupController) ListNodeGroups(c *gin.Context) {
	if err := ensureNodeGroupCnameDomainColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	query := db.DB.Model(&models.NodeGroup{})
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword != "" {
		// If keyword is a number, try to search by ID as well
		if id, err := strconv.ParseInt(keyword, 10, 64); err == nil && id > 0 {
			query = query.Where("id = ? OR name LIKE ? OR cname_hostname LIKE ? OR des LIKE ?", id, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		} else {
			like := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR cname_hostname LIKE ? OR des LIKE ?", like, like, like)
		}
	}
	if regionStr := strings.TrimSpace(c.Query("region_id")); regionStr != "" {
		if regionID, err := strconv.ParseInt(regionStr, 10, 64); err == nil && regionID > 0 {
			query = query.Where("region_id = ?", regionID)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	var groups []models.NodeGroup
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	for i := range groups {
		applyNodeGroupPolicy(&groups[i])
	}

	views := make([]nodeGroupView, 0, len(groups))
	counts := loadNodeGroupCounts(groups)
	forwardCounts := loadForwardCounts(groups)
	siteCounts := loadSiteCounts(groups)

	for _, group := range groups {
		views = append(views, nodeGroupView{
			NodeGroup:    group,
			NodeCount:    counts[group.ID],
			SiteCount:    siteCounts[group.ID],
			ForwardCount: forwardCounts[group.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  views,
			"total": total,
		},
	})
}

// CreateNodeGroup
// POST /api/v1/admin/node-groups
func (ctr *NodeGroupController) CreateNodeGroup(c *gin.Context) {
	if err := ensureNodeGroupCnameDomainColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	var req models.NodeGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	domain, err := resolveNodeGroupCnameDomain(strings.TrimSpace(req.CnameDomain))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T(err.Error())})
		return
	}
	req.CnameDomain = domain

	req.CnameHostname = normalizeGroupHostname(req.CnameHostname, req.CnameDomain)
	if strings.TrimSpace(req.CnameHostname) == "" {
		hostname, err := generateUniqueGroupHostname()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to generate resolution")})
			return
		}
		req.CnameHostname = hostname
	}
	req.Ipv4Resolution = strings.TrimSpace(req.Ipv4Resolution)
	if req.Ipv4Resolution == "" {
		if token, err := randomToken(8); err == nil {
			req.Ipv4Resolution = token
		}
	}

	req.BackupSwitchPolicy = buildNodeGroupPolicy(&req, "")

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Create Failed")})
		return
	}
	services.BumpConfigVersion("node_group", []int64{req.ID})
}

// UpdateNodeGroup
// PUT /api/v1/admin/node-groups/:id
func (ctr *NodeGroupController) UpdateNodeGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if err := ensureNodeGroupCnameDomainColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	var req models.NodeGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	var existing models.NodeGroup
	if err := db.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("group not found")})
		return
	}
	applyNodeGroupPolicy(&existing)

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	domainInput := strings.TrimSpace(req.CnameDomain)
	if domainInput == "" {
		domainInput = strings.TrimSpace(existing.CnameDomain)
	}
	domain, err := resolveNodeGroupCnameDomain(domainInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T(err.Error())})
		return
	}
	req.CnameDomain = domain

	req.CnameHostname = normalizeGroupHostname(req.CnameHostname, req.CnameDomain)
	if strings.TrimSpace(req.CnameHostname) == "" {
		req.CnameHostname = normalizeGroupHostname(existing.CnameHostname, req.CnameDomain)
	}
	req.Ipv4Resolution = strings.TrimSpace(req.Ipv4Resolution)
	if req.Ipv4Resolution == "" {
		req.Ipv4Resolution = strings.TrimSpace(existing.Ipv4Resolution)
	}

	backupPolicy := buildNodeGroupPolicy(&req, req.BackupSwitchPolicy)

	updates := map[string]interface{}{
		"name":                 req.Name,
		"region_id":            req.RegionID,
		"cname_hostname":       req.CnameHostname, // maps to resolution
		"cname_domain":         req.CnameDomain,
		"des":                  req.Description,      // maps to remark
		"backup_switch_type":   req.BackupSwitchType, // maps to spare_ip_switch
		"backup_switch_policy": backupPolicy,
		"update_at":            time.Now(),
	}
	if err := db.DB.Model(&models.NodeGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}
	services.BumpConfigVersion("node_group", []int64{id})

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  T("Updated"),
	})
}

// DeleteNodeGroup
// DELETE /api/v1/admin/node-groups/:id
func (ctr *NodeGroupController) DeleteNodeGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid id")})
		return
	}
	var nodeCount int64
	if err := db.DB.Model(&models.Line{}).Where("node_group_id = ?", id).Count(&nodeCount).Error; err == nil {
		if nodeCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node_group.has_nodes")})
			return
		}
	}
	var pkgCount int64
	if err := db.DB.Model(&models.Package{}).
		Where("node_group_id = ? OR backup_node_group = ?", id, id).
		Count(&pkgCount).Error; err == nil {
		if pkgCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node_group.has_packages")})
			return
		}
	}
	var userPkgCount int64
	if err := db.DB.Model(&models.UserPackage{}).
		Where("node_group_id = ? OR backup_node_group = ?", id, id).
		Count(&userPkgCount).Error; err == nil {
		if userPkgCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node_group.has_packages")})
			return
		}
	}
	if err := db.DB.Delete(&models.NodeGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Delete Failed")})
		return
	}
	services.BumpConfigVersion("node_group", []int64{id})
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  T("Deleted"),
	})
}

type lineIPItem struct {
	NodeID   int64  `json:"node_id"`
	NodeIPID int64  `json:"node_ip_id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Online   bool   `json:"online"`
}

type lineAssignedItem struct {
	ID                  int64  `json:"id"`
	NodeID              int64  `json:"node_id"`
	NodeIPID            int64  `json:"node_ip_id"`
	LineID              string `json:"line_id"`
	LineName            string `json:"line_name"`
	Name                string `json:"name"`
	IP                  string `json:"ip"`
	Online              bool   `json:"online"`
	IsOn                bool   `json:"is_on"`      // Resolution Status
	NodeIsOn            bool   `json:"node_is_on"` // Node Status
	IsBackup            bool   `json:"is_backup"`
	IsBackupDefaultLine bool   `json:"is_backup_default_line"`
	Weight              string `json:"weight"`
	SortOrder           int    `json:"sort_order"`
}

type lineAssignRequest struct {
	LineID   string       `json:"line_id"`
	LineName string       `json:"line_name"`
	Items    []lineIPItem `json:"items"`
}

type lineActionRequest struct {
	Action string  `json:"action"`
	IDs    []int64 `json:"ids"`
	Value  string  `json:"value"`
}

type packageLineKey struct {
	ID   string
	Name string
}

// GetResolution
// GET /api/v1/admin/node-groups/:id/resolution
func (ctr *NodeGroupController) GetResolution(c *gin.Context) {
	groupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid group id")})
		return
	}
	if err := ensureNodeGroupCnameDomainColumn(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	var group models.NodeGroup
	if err := db.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("group not found")})
		return
	}
	applyNodeGroupPolicy(&group)

	lineID := strings.TrimSpace(c.DefaultQuery("line_id", "default"))
	if lineID == "all" {
		lineID = ""
	}
	if lineID == "" && strings.TrimSpace(c.Query("line_id")) == "" {
		lineID = "default"
	}

	var regionName string
	if group.RegionID != nil {
		var region models.Region
		if err := db.DB.Where("id = ?", *group.RegionID).First(&region).Error; err == nil {
			regionName = region.Name
		}
	}

	var lines []models.Line
	lineQuery := db.DB.Where("node_group_id = ?", groupID)
	if lineID != "" {
		lineQuery = lineQuery.Where("line_id = ?", lineID)
	}
	if err := lineQuery.Find(&lines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("load lines failed")})
		return
	}

	assigned, assignedIPIDs := buildAssignedLineItems(lines)
	available, err := buildAvailableLineItems(group, assignedIPIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("load nodes failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"group": gin.H{
				"id":          group.ID,
				"name":        group.Name,
				"region_id":   group.RegionID,
				"region_name": regionName,
			},
			"line": gin.H{
				"id":   lineID,
				"name": lineID,
			},
			"available": available,
			"assigned":  assigned,
		},
	})
}

// AssignResolutionLines
// POST /api/v1/admin/node-groups/:id/resolution/assign
func (ctr *NodeGroupController) AssignResolutionLines(c *gin.Context) {
	groupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid group id")})
		return
	}
	var group models.NodeGroup
	if err := db.DB.Select("id", "region_id").Where("id = ?", groupID).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("group not found")})
		return
	}

	var req lineAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid params")})
		return
	}
	lineID := strings.TrimSpace(req.LineID)
	if lineID == "" {
		lineID = "default"
	}
	lineName := strings.TrimSpace(req.LineName)
	if lineName == "" {
		lineName = lineID
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("items required")})
		return
	}

	nodeIDs := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		id := item.NodeID
		if id == 0 {
			id = item.NodeIPID
		}
		if id != 0 {
			nodeIDs = append(nodeIDs, id)
		}
	}
	if len(nodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("no valid items")})
		return
	}
	var conflicts []models.Line
	if err := db.DB.Select("node_group_id", "node_id", "node_ip_id").
		Where("node_group_id <> ? AND (node_id IN ? OR node_ip_id IN ?)", groupID, nodeIDs, nodeIDs).
		Find(&conflicts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("load nodes failed")})
		return
	}
	if len(conflicts) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("node already assigned to another group")})
		return
	}
	var regionIDs []int64
	if err := db.DB.Model(&models.Region{}).Pluck("id", &regionIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	if len(regionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.region_required")})
		return
	}
	regionSet := make(map[int64]struct{}, len(regionIDs))
	for _, id := range regionIDs {
		regionSet[id] = struct{}{}
	}
	enabledNodes := map[int64]bool{}
	nodeRegions := map[int64]*int64{}
	var nodes []models.Node
	if err := db.DB.Select("id", "enable", "region_id").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("load nodes failed")})
		return
	}
	for _, node := range nodes {
		if node.Enable {
			enabledNodes[node.ID] = true
		}
		nodeRegions[node.ID] = node.RegionID
	}
	for _, item := range req.Items {
		id := item.NodeID
		if id == 0 {
			id = item.NodeIPID
		}
		if id != 0 && !enabledNodes[id] {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node disabled")})
			return
		}
		regionID, ok := nodeRegions[id]
		if id != 0 && (!ok || regionID == nil || *regionID == 0) {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.region_required")})
			return
		}
		if id != 0 {
			if _, exists := regionSet[*regionID]; !exists {
				c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.region_required")})
				return
			}
		}
		if id != 0 && group.RegionID != nil && *group.RegionID > 0 {
			if regionID == nil || *regionID != *group.RegionID {
				c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.region_mismatch")})
				return
			}
		}
	}

	now := time.Now()
	createItems := make([]*models.Line, 0, len(req.Items))
	assignedIPIDs := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		if item.NodeID == 0 {
			item.NodeID = item.NodeIPID
		}
		if item.NodeID == 0 || item.NodeIPID == 0 {
			continue
		}
		assignedIPIDs = append(assignedIPIDs, item.NodeIPID)
		createItems = append(createItems, &models.Line{
			NodeGroupID:             groupID,
			NodeID:                  item.NodeID,
			NodeIPID:                item.NodeIPID,
			LineID:                  lineID,
			LineName:                lineName,
			Weight:                  "1",
			Enable:                  true,
			IsBackup:                false,
			EnableBackup:            false,
			IsBackupDefaultLine:     false,
			EnableBackupDefaultLine: false,
			CreatedAt:               now,
			UpdatedAt:               now,
		})
	}
	if len(createItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("no valid items")})
		return
	}

	createdLines := make([]models.Line, 0, len(createItems))
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range createItems {
			var existing models.Line
			result := tx.Where("node_group_id = ? AND line_id = ? AND node_ip_id = ?", item.NodeGroupID, item.LineID, item.NodeIPID).Limit(1).Find(&existing)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				continue
			}
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			createdLines = append(createdLines, *item)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("assign failed")})
		return
	}

	if len(createdLines) > 0 {
		services.WriteIPSwitchLogsForLines(createdLines, "assign", "line")
	}

	services.BumpConfigVersion("line", []int64{groupID})
	if err := dns.SyncLineRecords(groupID, lineID, lineName, "add", assignedIPIDs); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns sync failed"), "error": T("dns sync failed")})
		return
	}
	if err := services.SyncPackageCnameForLineChange(groupID, lineID, lineName, assignedIPIDs, "add"); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns sync failed"), "error": T("dns sync failed")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// LineResolutionAction
// POST /api/v1/admin/node-groups/:id/resolution/action
func (ctr *NodeGroupController) LineResolutionAction(c *gin.Context) {
	groupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid group id")})
		return
	}
	var req lineActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("invalid params")})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("ids required")})
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("action required")})
		return
	}
	switch action {
	case "enable", "disable", "delete", "set_backup", "unset_backup", "set_backup_default", "unset_backup_default", "set_weight", "set_sort":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("unknown action")})
		return
	}

	value := strings.TrimSpace(req.Value)
	sortVal := 0
	if action == "set_weight" {
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("weight required")})
			return
		}
	}
	if action == "set_sort" {
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("sort required")})
			return
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid params")})
			return
		}
		sortVal = parsed
	}

	var targetLines []models.Line
	if action == "enable" || action == "disable" || action == "delete" || action == "set_weight" || action == "set_sort" {
		_ = db.DB.Where("id IN ?", req.IDs).Find(&targetLines).Error
	}
	if action == "delete" {
		delay := services.ResolveDeleteConfigDelay()
		if delay > 0 {
			for _, line := range targetLines {
				nodeID := line.NodeID
				if nodeID == 0 {
					nodeID = line.NodeIPID
				}
				services.QueueLineConfigDeletion(nodeID, line.NodeGroupID, line.LineID, line.LineName, delay)
			}
		}
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		switch action {
		case "enable":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"enable":    true,
				"update_at": time.Now(),
			}).Error
		case "disable":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"enable":    false,
				"update_at": time.Now(),
			}).Error
		case "delete":
			return tx.Where("id IN ?", req.IDs).Delete(&models.Line{}).Error
		case "set_backup":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"is_backup":     true,
				"enable_backup": true,
				"update_at":     time.Now(),
			}).Error
		case "unset_backup":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"is_backup":     false,
				"enable_backup": false,
				"update_at":     time.Now(),
			}).Error
		case "set_backup_default":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"is_backup_default_line":     true,
				"enable_backup_default_line": true,
				"update_at":                  time.Now(),
			}).Error
		case "unset_backup_default":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"is_backup_default_line":     false,
				"enable_backup_default_line": false,
				"update_at":                  time.Now(),
			}).Error
		case "set_weight":
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"weight":    value,
				"update_at": time.Now(),
			}).Error
		case "set_sort":
			var lineNodes []models.Line
			if err := tx.Select("node_id").Where("id IN ?", req.IDs).Find(&lineNodes).Error; err != nil {
				return err
			}
			nodeIDs := make([]int64, 0, len(lineNodes))
			for _, line := range lineNodes {
				if line.NodeID != 0 {
					nodeIDs = append(nodeIDs, line.NodeID)
				}
			}
			if len(nodeIDs) == 0 {
				return nil
			}
			return tx.Model(&models.Node{}).Where("id IN ?", nodeIDs).Updates(map[string]interface{}{
				"sort":      sortVal,
				"update_at": time.Now(),
			}).Error
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Update failed")})
		return
	}
	if len(targetLines) > 0 && (action == "enable" || action == "disable" || action == "delete") {
		services.WriteIPSwitchLogsForLines(targetLines, action, "line")
	}
	services.BumpConfigVersion("line", []int64{groupID})
	if len(targetLines) > 0 && (action == "enable" || action == "disable" || action == "delete" || action == "set_weight" || action == "set_sort") {
		groupLineNodes := map[int64]map[packageLineKey][]int64{}
		for _, line := range targetLines {
			gid := line.NodeGroupID
			key := packageLineKey{ID: strings.TrimSpace(line.LineID), Name: strings.TrimSpace(line.LineName)}
			if _, ok := groupLineNodes[gid]; !ok {
				groupLineNodes[gid] = make(map[packageLineKey][]int64)
			}
			nodeID := line.NodeIPID
			if nodeID == 0 {
				nodeID = line.NodeID
			}
			if nodeID != 0 {
				groupLineNodes[gid][key] = append(groupLineNodes[gid][key], nodeID)
			}
		}
		dnsAction := "resync"
		if action == "enable" {
			dnsAction = "add"
		} else if action == "disable" || action == "delete" {
			dnsAction = "delete"
		}
		for gid, lineMap := range groupLineNodes {
			for key, nodeIDs := range lineMap {
				ids := uniqueInt64List(nodeIDs)
				if dnsAction == "resync" {
					ids = loadLineNodeIDs(gid, key.ID)
				}
				if len(ids) == 0 && dnsAction != "resync" {
					continue
				}
				if err := dns.SyncLineRecords(gid, key.ID, key.Name, dnsAction, ids); err != nil {
					c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns sync failed"), "error": T("dns sync failed")})
					return
				}
				if err := services.SyncPackageCnameForLineChange(gid, key.ID, key.Name, ids, dnsAction); err != nil {
					c.JSON(http.StatusOK, gin.H{"code": 1, "msg": T("dns sync failed"), "error": T("dns sync failed")})
					return
				}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func buildAssignedLineItems(lines []models.Line) ([]lineAssignedItem, map[int64]struct{}) {
	items := make([]lineAssignedItem, 0, len(lines))
	ipIDs := make(map[int64]struct{})
	if len(lines) == 0 {
		return items, ipIDs
	}
	nodeIDs := make([]int64, 0, len(lines)*2)
	seen := map[int64]struct{}{}
	for _, line := range lines {
		if line.NodeID != 0 {
			if _, ok := seen[line.NodeID]; !ok {
				seen[line.NodeID] = struct{}{}
				nodeIDs = append(nodeIDs, line.NodeID)
			}
		}
		if line.NodeIPID != 0 {
			if _, ok := seen[line.NodeIPID]; !ok {
				seen[line.NodeIPID] = struct{}{}
				nodeIDs = append(nodeIDs, line.NodeIPID)
			}
			ipIDs[line.NodeIPID] = struct{}{}
		}
	}
	nodeMap := map[int64]models.Node{}
	if len(nodeIDs) > 0 {
		var nodes []models.Node
		_ = db.DB.Where("id IN ?", nodeIDs).Find(&nodes).Error
		for _, node := range nodes {
			nodeMap[node.ID] = node
		}
	}
	for _, line := range lines {
		node := nodeMap[line.NodeID]
		nodeIP := nodeMap[line.NodeIPID]
		if nodeIP.ID == 0 {
			nodeIP = node
		}
		items = append(items, lineAssignedItem{
			ID:                  line.ID,
			NodeID:              line.NodeID,
			NodeIPID:            line.NodeIPID,
			LineID:              line.LineID,
			LineName:            line.LineName,
			Name:                node.Name,
			IP:                  nodeIP.IP,
			Online:              services.IsNodeOnline(line.NodeID, 90*time.Second),
			IsOn:                line.Enable,
			NodeIsOn:            node.Enable,
			IsBackup:            line.IsBackup,
			IsBackupDefaultLine: line.IsBackupDefaultLine,
			Weight:              line.Weight,
			SortOrder:           node.Sort,
		})
	}
	return items, ipIDs
}

func uniqueInt64List(items []int64) []int64 {
	if len(items) == 0 {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	result := make([]int64, 0, len(items))
	for _, id := range items {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func loadLineNodeIDs(groupID int64, lineID string) []int64 {
	if groupID == 0 {
		return []int64{}
	}
	var lines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id = ? AND line_id = ? AND enable = ?", groupID, lineID, true).
		Find(&lines).Error; err != nil {
		return []int64{}
	}
	nodeIDs := make([]int64, 0, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID != 0 {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	return uniqueInt64List(nodeIDs)
}

func buildAvailableLineItems(group models.NodeGroup, assignedIPIDs map[int64]struct{}) ([]lineIPItem, error) {
	var regionIDs []int64
	if err := db.DB.Model(&models.Region{}).Pluck("id", &regionIDs).Error; err != nil {
		return nil, err
	}
	if len(regionIDs) == 0 {
		return []lineIPItem{}, nil
	}
	query := db.DB.Model(&models.Node{}).
		Where("enable = ?", true).
		Where("region_id IN ?", regionIDs)
	if group.RegionID != nil && *group.RegionID > 0 {
		query = query.Where("region_id = ?", *group.RegionID)
	}
	var nodes []models.Node
	if err := query.Find(&nodes).Error; err != nil {
		return nil, err
	}
	otherAssigned := map[int64]struct{}{}
	var otherLines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id <> ?", group.ID).
		Find(&otherLines).Error; err == nil {
		for _, line := range otherLines {
			if line.NodeID != 0 {
				otherAssigned[line.NodeID] = struct{}{}
			}
			if line.NodeIPID != 0 {
				otherAssigned[line.NodeIPID] = struct{}{}
			}
		}
	}
	nameMap := map[int64]string{}
	for _, node := range nodes {
		nameMap[node.ID] = node.Name
	}
	result := make([]lineIPItem, 0, len(nodes))
	for _, node := range nodes {
		parentID := node.ID
		name := node.Name
		if node.PID > 0 {
			parentID = node.PID
			if parentName, ok := nameMap[parentID]; ok && parentName != "" {
				name = parentName
			}
		}
		if _, exists := assignedIPIDs[node.ID]; exists {
			continue
		}
		if _, exists := otherAssigned[node.ID]; exists {
			continue
		}
		result = append(result, lineIPItem{
			NodeID:   parentID,
			NodeIPID: node.ID,
			Name:     name,
			IP:       node.IP,
			Online:   services.IsNodeOnline(parentID, 90*time.Second),
		})
	}
	return result, nil
}

func loadNodeGroupCounts(groups []models.NodeGroup) map[int64]int64 {
	result := map[int64]int64{}
	if len(groups) == 0 {
		return result
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	var rows []nodeGroupCount
	_ = db.DB.Model(&models.Line{}).
		Select("node_group_id, count(distinct node_id) as cnt").
		Where("node_group_id IN ?", groupIDs).
		Group("node_group_id").
		Scan(&rows).Error
	for _, row := range rows {
		result[row.NodeGroupID] = row.Count
	}
	return result
}

func loadForwardCounts(groups []models.NodeGroup) map[int64]int64 {
	result := map[int64]int64{}
	if len(groups) == 0 {
		return result
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	var rows []nodeGroupCount
	_ = db.DB.Model(&models.Forward{}).
		Select("node_group_id, count(*) as cnt").
		Where("node_group_id IN ?", groupIDs).
		Group("node_group_id").
		Scan(&rows).Error
	for _, row := range rows {
		result[row.NodeGroupID] = row.Count
	}
	return result
}

func loadSiteCounts(groups []models.NodeGroup) map[int64]int64 {
	result := map[int64]int64{}
	if len(groups) == 0 {
		return result
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	var rows []nodeGroupCount
	_ = db.DB.Model(&models.Site{}).
		Select("node_group_id, count(*) as cnt").
		Where("node_group_id IN ?", groupIDs).
		Group("node_group_id").
		Scan(&rows).Error
	for _, row := range rows {
		result[row.NodeGroupID] = row.Count
	}
	return result
}

func applyNodeGroupPolicy(group *models.NodeGroup) {
	if group == nil {
		return
	}
	if strings.TrimSpace(group.BackupSwitchPolicy) == "" {
		return
	}
	var policy nodeGroupPolicy
	if err := json.Unmarshal([]byte(group.BackupSwitchPolicy), &policy); err != nil {
		return
	}
	group.Ipv4Resolution = policy.Ipv4Resolution
	group.L2Config = policy.L2Config
	group.SortOrder = policy.SortOrder
}

func buildNodeGroupPolicy(req *models.NodeGroup, fallback string) string {
	policy := nodeGroupPolicy{
		Ipv4Resolution: strings.TrimSpace(req.Ipv4Resolution),
		L2Config:       strings.TrimSpace(req.L2Config),
		SortOrder:      req.SortOrder,
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return fallback
	}
	return string(b)
}

func ensureNodeGroupCnameDomainColumn() error {
	if db.DB == nil {
		return nil
	}
	if db.DB.Migrator().HasColumn(&models.NodeGroup{}, "cname_domain") {
		return nil
	}
	return db.DB.Migrator().AddColumn(&models.NodeGroup{}, "CnameDomain")
}

func resolveNodeGroupCnameDomain(input string) (string, error) {
	if err := ensureCnameTable(); err != nil {
		return "", err
	}
	domain := normalizeDomainInput(input)
	if domain == "" {
		var row models.CnameDomain
		if err := db.DB.Order("id asc").First(&row).Error; err != nil {
			return "", errors.New("cname domains not configured")
		}
		domain = normalizeDomainInput(row.Domain)
	}
	if domain == "" || !isValidDomain(domain) {
		return "", errors.New("invalid cname domain")
	}
	var existing models.CnameDomain
	if err := db.DB.Where("domain = ?", domain).First(&existing).Error; err != nil {
		return "", errors.New("cname domain not found")
	}
	return domain, nil
}

func normalizeGroupHostname(host, domain string) string {
	normalized := normalizeDomainInput(host)
	if normalized == "" {
		return ""
	}
	domain = normalizeDomainInput(domain)
	if domain != "" {
		if normalized == domain {
			return "@"
		}
		suffix := "." + domain
		if strings.HasSuffix(normalized, suffix) {
			normalized = strings.TrimSuffix(normalized, suffix)
		}
	}
	return strings.TrimSuffix(normalized, ".")
}

func generateUniqueGroupHostname() (string, error) {
	for i := 0; i < 5; i++ {
		token, err := randomToken(8)
		if err != nil {
			return "", err
		}
		var count int64
		if err := db.DB.Model(&models.NodeGroup{}).Where("cname_hostname = ?", token).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return token, nil
		}
	}
	return "", errors.New("failed to generate unique resolution")
}
