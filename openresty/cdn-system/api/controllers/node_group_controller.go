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
	L2Config        string `json:"l2_config"`
	SortOrder       int    `json:"sort_order"`
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
	query := db.DB.Model(&models.NodeGroup{})
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR cname_hostname LIKE ? OR des LIKE ?", like, like, like)
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
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	var groups []models.NodeGroup
	if err := query.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
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
			NodeGroup:   group,
			NodeCount:   counts[group.ID],
			SiteCount:   siteCounts[group.ID],
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
	var req models.NodeGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	req.BackupSwitchPolicy = buildNodeGroupPolicy(&req, "")

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
	}
	services.BumpConfigVersion("node_group", []int64{req.ID})
}

// UpdateNodeGroup
// PUT /api/v1/admin/node-groups/:id
func (ctr *NodeGroupController) UpdateNodeGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req models.NodeGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	backupPolicy := buildNodeGroupPolicy(&req, req.BackupSwitchPolicy)

	updates := map[string]interface{}{
		"name":                 req.Name,
		"region_id":            req.RegionID,
		"cname_hostname":       req.CnameHostname,  // maps to resolution
		"des":                  req.Description,    // maps to remark
		"backup_switch_type":   req.BackupSwitchType, // maps to spare_ip_switch
		"backup_switch_policy": backupPolicy,
		"update_at":            time.Now(),
	}
	if err := db.DB.Model(&models.NodeGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}
	services.BumpConfigVersion("node_group", []int64{id})

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Updated",
	})
}

// DeleteNodeGroup
// DELETE /api/v1/admin/node-groups/:id
func (ctr *NodeGroupController) DeleteNodeGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if err := db.DB.Delete(&models.NodeGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Delete Failed"})
		return
	}
	services.BumpConfigVersion("node_group", []int64{id})
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Deleted",
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
	Name                string `json:"name"`
	IP                  string `json:"ip"`
	Online              bool   `json:"online"`
	Enabled             bool   `json:"enabled"`
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
	Action string `json:"action"`
	IDs    []int64 `json:"ids"`
	Value  string `json:"value"`
}

// GetResolution
// GET /api/v1/admin/node-groups/:id/resolution
func (ctr *NodeGroupController) GetResolution(c *gin.Context) {
	groupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid group id"})
		return
	}

	var group models.NodeGroup
	if err := db.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "group not found"})
		return
	}
	applyNodeGroupPolicy(&group)

	lineID := strings.TrimSpace(c.DefaultQuery("line_id", "default"))
	if lineID == "" {
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
	if err := db.DB.Where("node_group_id = ? AND line_id = ?", groupID, lineID).Find(&lines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "load lines failed"})
		return
	}

	assigned, assignedIPIDs := buildAssignedLineItems(lines)
	available, err := buildAvailableLineItems(group, assignedIPIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "load nodes failed"})
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid group id"})
		return
	}

	var req lineAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "items required"})
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "no valid items"})
		return
	}
	enabledNodes := map[int64]bool{}
	var nodes []models.Node
	if err := db.DB.Select("id", "enable").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "load nodes failed"})
		return
	}
	for _, node := range nodes {
		if node.Enable {
			enabledNodes[node.ID] = true
		}
	}
	for _, item := range req.Items {
		id := item.NodeID
		if id == 0 {
			id = item.NodeIPID
		}
		if id != 0 && !enabledNodes[id] {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "node disabled"})
			return
		}
	}

	now := time.Now()
	createItems := make([]models.Line, 0, len(req.Items))
	assignedIPIDs := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		if item.NodeID == 0 {
			item.NodeID = item.NodeIPID
		}
		if item.NodeID == 0 || item.NodeIPID == 0 {
			continue
		}
		assignedIPIDs = append(assignedIPIDs, item.NodeIPID)
		createItems = append(createItems, models.Line{
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
		c.JSON(http.StatusBadRequest, gin.H{"msg": "no valid items"})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range createItems {
			var existing models.Line
			if err := tx.Where("node_group_id = ? AND line_id = ? AND node_ip_id = ?", item.NodeGroupID, item.LineID, item.NodeIPID).First(&existing).Error; err == nil {
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "assign failed"})
		return
	}

	services.BumpConfigVersion("line", []int64{groupID})
	_ = dns.SyncLineRecords(groupID, lineID, lineName, "add", assignedIPIDs)
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// LineResolutionAction
// POST /api/v1/admin/node-groups/:id/resolution/action
func (ctr *NodeGroupController) LineResolutionAction(c *gin.Context) {
	groupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid group id"})
		return
	}
	var req lineActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "ids required"})
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "action required"})
		return
	}

	var targetLines []models.Line
	if action == "enable" || action == "disable" || action == "delete" {
		_ = db.DB.Where("id IN ?", req.IDs).Find(&targetLines).Error
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
			value := strings.TrimSpace(req.Value)
			if value == "" {
				return errors.New("weight required")
			}
			return tx.Model(&models.Line{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
				"weight":    value,
				"update_at": time.Now(),
			}).Error
		case "set_sort":
			value := strings.TrimSpace(req.Value)
			if value == "" {
				return errors.New("sort required")
			}
			sortVal, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
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
		default:
			return errors.New("unknown action")
		}
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	services.BumpConfigVersion("line", []int64{groupID})
	if len(targetLines) > 0 {
		groupIDs := map[int64][]int64{}
		lineIDs := map[int64]string{}
		lineNames := map[int64]string{}
		for _, line := range targetLines {
			groupIDs[line.NodeGroupID] = append(groupIDs[line.NodeGroupID], line.NodeIPID)
			lineIDs[line.NodeGroupID] = line.LineID
			lineNames[line.NodeGroupID] = line.LineName
		}
		for gid, ipIDs := range groupIDs {
			_ = dns.SyncLineRecords(gid, lineIDs[gid], lineNames[gid], action, ipIDs)
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
			Name:                node.Name,
			IP:                  nodeIP.IP,
			Online:              services.IsNodeOnline(line.NodeID, 90*time.Second),
			Enabled:             line.Enable,
			IsBackup:            line.IsBackup,
			IsBackupDefaultLine: line.IsBackupDefaultLine,
			Weight:              line.Weight,
			SortOrder:           node.Sort,
		})
	}
	return items, ipIDs
}

func buildAvailableLineItems(group models.NodeGroup, assignedIPIDs map[int64]struct{}) ([]lineIPItem, error) {
	query := db.DB.Model(&models.Node{}).Where("enable = ?", true)
	if group.RegionID != nil && *group.RegionID > 0 {
		query = query.Where("region_id = ?", *group.RegionID)
	}
	var nodes []models.Node
	if err := query.Find(&nodes).Error; err != nil {
		return nil, err
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
		L2Config:        strings.TrimSpace(req.L2Config),
		SortOrder:       req.SortOrder,
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return fallback
	}
	return string(b)
}
