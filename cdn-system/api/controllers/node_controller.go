package controllers

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-api/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeController struct {
	NodeService *services.NodeService
}

var errNodeSubIPChangeBlocked = errors.New("node.subip_change_blocked")

const defaultNodeSSHUser = "root"

type nodeRequest struct {
	ID             int64              `json:"id"`
	RegionID       *int64             `json:"region_id"`
	Name           string             `json:"name"`
	Remark         string             `json:"remark"`
	IP             string             `json:"ip"`
	Host           string             `json:"host"`
	Port           int                `json:"port"`
	HttpProxy      string             `json:"http_proxy"`
	IsMgmt         bool               `json:"is_mgmt"`
	Enable         bool               `json:"enable"`
	CheckOn        bool               `json:"check_on"`
	CheckProtocol  string             `json:"check_protocol"`
	CheckTimeout   int                `json:"check_timeout"`
	CheckPort      int                `json:"check_port"`
	CheckHost      string             `json:"check_host"`
	CheckPath      string             `json:"check_path"`
	CheckNodeGroup string             `json:"check_node_group"`
	CheckAction    string             `json:"check_action"`
	BwLimit        string             `json:"bw_limit"`
	Level          int                `json:"type"`
	Sort           int                `json:"sort_order"`
	CacheDir       string             `json:"cache_dir"`
	MaxCacheSize   int                `json:"cache_limit"`
	LogDir         string             `json:"log_dir"`
	SSHHost        string             `json:"ssh_host"`
	SSHPort        int                `json:"ssh_port"`
	SSHUser        string             `json:"ssh_user"`
	SSHAuthType    string             `json:"ssh_auth_type"`
	SSHPassword    string             `json:"ssh_password"`
	SSHKey         string             `json:"ssh_key"`
	WorkDir        string             `json:"work_dir"`
	AutoInstall    bool               `json:"auto_install"`
	SubIPs         []models.NodeSubIP `json:"sub_ips"`
}

// UpdateStatus toggles node enable status.
// PUT /api/v1/admin/nodes/:id/status
func (ctr *NodeController) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid ID")})
		return
	}

	var req struct {
		Enable *bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enable == nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	syncTask := "sync_disable"
	if *req.Enable {
		syncTask = "sync_enable"
	}

	if err := db.DB.Model(&models.Node{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"enable":      *req.Enable,
			"config_task": syncTask,
			"update_at":   time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}

	_ = db.DB.Model(&models.Node{}).
		Where("pid = ?", id).
		Updates(map[string]interface{}{
			"enable":    *req.Enable,
			"update_at": time.Now(),
		}).Error

	if ctr.NodeService != nil {
		var fullNode models.Node
		if err := db.DB.First(&fullNode, id).Error; err == nil {
			ctr.NodeService.SyncNodeToRedis(&fullNode)
		}
		var subNodes []models.Node
		db.DB.Where("pid = ?", id).Find(&subNodes)
		for _, sub := range subNodes {
			ctr.NodeService.SyncNodeToRedis(&sub)
		}
	}
	recordNodeIPSwitchLogs(id, actionLabel(*req.Enable))
	dnsAction := "delete"
	if *req.Enable {
		dnsAction = "add"
	}
	if err := services.SyncPackageCnameForNodes([]int64{id}, dnsAction); err != nil {
		log.Printf("[DNS] package cname sync failed action=%s node=%d err=%v", dnsAction, id, err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

// UpdateAntiBlocking toggles anti-blocking behavior for a node.
// PUT /api/v1/admin/nodes/:id/anti_blocking
func (ctr *NodeController) UpdateAntiBlocking(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid ID")})
		return
	}

	var req struct {
		Enable *bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enable == nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	var count int64
	if err := db.DB.Model(&models.Node{}).Where("id = ?", id).Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("node not found")})
		return
	}

	value := "0"
	if *req.Enable {
		value = "1"
	}
	if err := services.UpsertNodeConfigItem(id, "anti_blocking", value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}

	services.TriggerNodeConfigSync(id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Updated")})
}

// ListNodes
// GET /api/v1/admin/nodes
func (ctr *NodeController) ListNodes(c *gin.Context) {
	keyword := c.Query("keyword")
	regionIDStr := strings.TrimSpace(c.Query("region_id"))
	status := strings.TrimSpace(c.Query("status"))
	nodeTypeStr := strings.TrimSpace(c.Query("node_type"))
	page, pageSize := parsePageParams(c, 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var nodes []models.Node
	query := db.DB.Model(&models.Node{}).Where("pid = 0")
	if regionIDStr != "" {
		if regionID, err := strconv.ParseInt(regionIDStr, 10, 64); err == nil && regionID > 0 {
			query = query.Where("region_id = ?", regionID)
		}
	}
	if status != "" {
		switch strings.ToLower(status) {
		case "enabled":
			query = query.Where("enable = ?", true)
		case "disabled":
			query = query.Where("enable = ?", false)
		}
	}
	if nodeTypeStr != "" {
		if nodeType, err := strconv.Atoi(nodeTypeStr); err == nil && nodeType > 0 {
			query = query.Where("level = ?", nodeType)
		}
	}
	if keyword != "" {
		if id, err := strconv.ParseInt(keyword, 10, 64); err == nil && id > 0 {
			query = query.Where("id = ? OR lower(name) LIKE ? OR ip LIKE ?", id, "%"+strings.ToLower(keyword)+"%", "%"+keyword+"%")
		} else {
			keywordLike := "%" + strings.ToLower(keyword) + "%"
			query = query.Where("lower(name) LIKE ? OR ip LIKE ?", keywordLike, keywordLike)
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
		Find(&nodes).Error; err != nil {
		log.Println("[Error] ListNodes DB Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	if len(nodes) > 0 {
		parentIDs := make([]int64, 0, len(nodes))
		for _, node := range nodes {
			parentIDs = append(parentIDs, node.ID)
		}

		var subNodes []models.Node
		if err := db.DB.Select("id", "pid", "ip").Where("pid IN ?", parentIDs).Find(&subNodes).Error; err == nil {
			subMap := make(map[int64][]models.NodeSubIP)
			for _, sub := range subNodes {
				subMap[sub.PID] = append(subMap[sub.PID], models.NodeSubIP{ID: sub.ID, IP: sub.IP})
			}
			for i := range nodes {
				nodes[i].SubIPs = subMap[nodes[i].ID]
			}
		}

		var lineCounts []struct {
			NodeID int64 `gorm:"column:node_id"`
			Count  int64 `gorm:"column:cnt"`
		}
		_ = db.DB.Model(&models.Line{}).
			Select("node_id, count(*) as cnt").
			Where("node_id IN ?", parentIDs).
			Group("node_id").
			Scan(&lineCounts).Error
		lineCountMap := make(map[int64]int64, len(lineCounts))
		for _, row := range lineCounts {
			lineCountMap[row.NodeID] = row.Count
		}
		for i := range nodes {
			nodes[i].LineCount = lineCountMap[nodes[i].ID]
		}

		// Load Regions
		regionIDs := make([]int64, 0)
		for _, node := range nodes {
			if node.RegionID != nil && *node.RegionID > 0 {
				regionIDs = append(regionIDs, *node.RegionID)
			}
		}
		if len(regionIDs) > 0 {
			var regions []models.Region
			if err := db.DB.Select("id", "name").Find(&regions, regionIDs).Error; err == nil {
				regionMap := make(map[int64]string)
				for _, r := range regions {
					regionMap[r.ID] = r.Name
				}
				for i := range nodes {
					if nodes[i].RegionID != nil {
						nodes[i].RegionName = regionMap[*nodes[i].RegionID]
					}
				}
			}
		}

		progressMap, _ := services.FetchInstallProgress(parentIDs)
		if len(progressMap) > 0 {
			for i := range nodes {
				if progress, ok := progressMap[nodes[i].ID]; ok {
					nodes[i].InstallStage = progress.Stage
					nodes[i].InstallProgress = progress.Percent
					nodes[i].InstallProgressBytes = progress.CurrentBytes
					nodes[i].InstallProgressTotal = progress.TotalBytes
				}
			}
		}
	}

	for i := range nodes {
		nodes[i].Online = services.IsNodeOnline(nodes[i].ID, 30*time.Second)
	}
	antiBlockingMap, _ := services.GetNodeConfigMap("anti_blocking")
	reportedConfigMap, _ := services.GetNodeConfigMap("reported_config")
	for i := range nodes {
		nodes[i].AntiBlocking = true
		if raw, ok := antiBlockingMap[nodes[i].ID]; ok && strings.TrimSpace(raw) != "" {
			nodes[i].AntiBlocking = services.ParseBoolFlag(raw)
		}
		reportedAntiBlocking := extractReportedAntiBlocking(reportedConfigMap[nodes[i].ID])
		if reportedAntiBlocking != nil {
			reportedVal := *reportedAntiBlocking
			nodes[i].ReportedAntiBlocking = &reportedVal
			if reportedVal != nodes[i].AntiBlocking {
				nodes[i].ConfigDrift = true
				nodes[i].ConfigDriftFields = []string{"anti_blocking"}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  nodes,
			"total": total,
		},
	})
}

func extractReportedAntiBlocking(raw string) *bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var reported map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &reported); err != nil {
		return nil
	}
	value, ok := reported["anti_blocking"]
	if !ok {
		return nil
	}
	parsed, ok := toReportedBool(value)
	if !ok {
		return nil
	}
	result := parsed
	return &result
}

func toReportedBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		return services.ParseBoolFlag(v), true
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	default:
		return false, false
	}
}

// ListMonitorLogs
// GET /api/v1/admin/nodes/:id/monitor_logs
func (ctr *NodeController) ListMonitorLogs(c *gin.Context) {
	idStr := c.Param("id")
	nodeID, _ := strconv.ParseInt(idStr, 10, 64)
	if nodeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid ID")})
		return
	}

	page, pageSize := parsePageParams(c, 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	logType := strings.TrimSpace(c.DefaultQuery("type", "availability"))
	switch logType {
	case "", "availability":
		logType = "heartbeat"
	case "availability_switch":
		logType = "switch"
	case "bandwidth":
		logType = "bandwidth"
	case "bandwidth_switch":
		logType = "bandwidth_switch"
	}
	timeRange := c.QueryArray("timeRange[]")
	if len(timeRange) == 0 {
		timeRange = c.QueryArray("timeRange")
	}
	startStr := strings.TrimSpace(c.DefaultQuery("start", ""))
	endStr := strings.TrimSpace(c.DefaultQuery("end", ""))

	var startAt time.Time
	var endAt time.Time
	layout := "2006-01-02 15:04:05"
	if len(timeRange) >= 2 {
		if t, err := time.ParseInLocation(layout, timeRange[0], time.Local); err == nil {
			startAt = t
		}
		if t, err := time.ParseInLocation(layout, timeRange[1], time.Local); err == nil {
			endAt = t
		}
	} else if startStr != "" && endStr != "" {
		if t, err := time.ParseInLocation(layout, startStr, time.Local); err == nil {
			startAt = t
		}
		if t, err := time.ParseInLocation(layout, endStr, time.Local); err == nil {
			endAt = t
		}
	}

	baseQuery := db.DB.Model(&models.NodeMonitorLog{}).Where("node_id = ?", nodeID)
	if logType != "" {
		baseQuery = baseQuery.Where("type = ?", logType)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		baseQuery = baseQuery.Where("create_at BETWEEN ? AND ?", startAt, endAt)
	}

	bucketExpr := "FLOOR(UNIX_TIMESTAMP(create_at) / 30)"
	grouped := baseQuery.Select(bucketExpr + " as bucket").Group("bucket")
	var total int64
	if err := db.DB.Table("(?) as logs", grouped).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	type logSummary struct {
		CheckedAt  string `json:"checked_at"`
		FailCount  int64  `json:"fail_count"`
		TotalCount int64  `json:"total_count"`
	}
	var list []logSummary
	if err := baseQuery.
		Select("DATE_FORMAT(FROM_UNIXTIME(" + bucketExpr + " * 30), '%Y-%m-%d %H:%i:%s') as checked_at, SUM(CASE WHEN success = '1' THEN 0 ELSE 1 END) as fail_count, COUNT(*) as total_count").
		Group("bucket").
		Order("bucket desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// CreateNode
// POST /api/v1/admin/nodes
func (ctr *NodeController) CreateNode(c *gin.Context) {
	var req nodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.IP) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Name and IP are required")})
		return
	}

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}
	normalizeNodeSSHDefaults(&req)
	req.WorkDir = "/www/node"

	token := strings.TrimSpace(config.App.AgentToken)
	if token == "" {
		token = utils.GenerateNodeToken()
		if token == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Failed to generate token")})
			return
		}
	}

	node := models.Node{
		PID:            0,
		RegionID:       req.RegionID,
		Name:           req.Name,
		Remark:         req.Remark,
		IP:             req.IP,
		Token:          token,
		Host:           req.Host,
		Port:           req.Port,
		HttpProxy:      req.HttpProxy,
		IsMgmt:         req.IsMgmt,
		Enable:         req.Enable,
		CheckOn:        req.CheckOn,
		CheckProtocol:  req.CheckProtocol,
		CheckTimeout:   req.CheckTimeout,
		CheckPort:      req.CheckPort,
		CheckHost:      req.CheckHost,
		CheckPath:      req.CheckPath,
		CheckNodeGroup: req.CheckNodeGroup,
		CheckAction:    req.CheckAction,
		BwLimit:        req.BwLimit,
		Level:          req.Level,
		Sort:           req.Sort,
		CacheDir:       req.CacheDir,
		MaxCacheSize:   req.MaxCacheSize,
		LogDir:         req.LogDir,
		SSHHost:        req.SSHHost,
		SSHPort:        req.SSHPort,
		SSHUser:        req.SSHUser,
		SSHAuthType:    req.SSHAuthType,
		SSHPassword:    req.SSHPassword,
		SSHKey:         req.SSHKey,
		WorkDir:        req.WorkDir,
		AutoInstall:    req.AutoInstall,
		InstallStatus:  resolveInitialInstallStatus(req.AutoInstall),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if !node.Enable {
		node.Enable = true
	}

	if err := db.DB.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Create Failed")})
		return
	}

	if err := replaceSubIPs(db.DB, node.ID, node, req.SubIPs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Create Sub IPs Failed")})
		return
	}

	// Sync node metadata (no-op if no cache layer)
	if ctr.NodeService != nil {
		// We should re-fetch full node with SubIPs or just assume replaceSubIPs did DB work.
		// For simplicity, let's sync what we have, but SubIPs in req might need to be refreshed if we want logic inside Sync.
		// Actually SyncNodeToRedis implementation assumes `node.IP` is main IP.
		// We also need to loop SubIPs and sync them.

		// Let NodeService handle SubIPs if we pass them.
		// Currently NodeService.SyncNodeToRedis doesn't iterate SubIPs.
		// I should update NodeService first to handle SubIP iteration or handle iteration here.

		// Let's handle iteration here for now to avoid re-editing NodeService multiple times.
		ctr.NodeService.SyncNodeToRedis(&node)

		// Also sync sub nodes.
		// replaceSubIPs creates new Node records with PID=req.ID.
		// We should really fetch them to sync properly.
		var subNodes []models.Node
		db.DB.Where("pid = ?", node.ID).Find(&subNodes)
		for _, sub := range subNodes {
			ctr.NodeService.SyncNodeToRedis(&sub)
		}
	}
	if node.Enable {
		if err := services.SyncPackageCnameForNodes([]int64{node.ID}, "add"); err != nil {
			log.Printf("[DNS] package cname sync failed action=add node=%d err=%v", node.ID, err)
		}
	}

	if node.AutoInstall {
		apiBase := services.ResolveAPIBaseURL(c.Request)
		if err := updateInstallStatus(node.ID, "running", "", time.Now()); err != nil {
			log.Printf("[Install] update running status failed node=%d err=%v", node.ID, err)
		}
		startNodeInstallAsync(node, apiBase)
		node.InstallStatus = "running"
		node.InstallError = ""
		now := time.Now()
		node.InstallAt = &now
	}

	resp := gin.H{
		"code": 0,
		"msg":  T("Node Created"),
		"data": node,
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateNode
// PUT /api/v1/admin/nodes/:id
func (ctr *NodeController) UpdateNode(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req nodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}
	normalizeNodeSSHDefaults(&req)
	req.WorkDir = "/www/node"

	var existing models.Node
	_ = db.DB.Select("enable", "region_id").Where("id = ?", id).First(&existing).Error
	syncTask := ""
	if req.Enable != existing.Enable {
		if req.Enable {
			syncTask = "sync_enable"
		} else {
			syncTask = "sync_disable"
		}
	}
	existingRegion := int64(0)
	if existing.RegionID != nil {
		existingRegion = *existing.RegionID
	}
	reqRegion := int64(0)
	if req.RegionID != nil {
		reqRegion = *req.RegionID
	}
	if existingRegion != reqRegion {
		var lineCount int64
		if err := db.DB.Model(&models.Line{}).
			Where("node_id = ? OR node_ip_id = ?", id, id).
			Count(&lineCount).Error; err == nil && lineCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.region_change_blocked")})
			return
		}
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"name":             req.Name,
			"des":              req.Remark,
			"ip":               req.IP,
			"region_id":        req.RegionID,
			"host":             req.Host,
			"port":             req.Port,
			"http_proxy":       req.HttpProxy,
			"is_mgmt":          req.IsMgmt,
			"enable":           req.Enable,
			"check_on":         req.CheckOn,
			"check_protocol":   req.CheckProtocol,
			"check_timeout":    req.CheckTimeout,
			"check_port":       req.CheckPort,
			"check_host":       req.CheckHost,
			"check_path":       req.CheckPath,
			"check_node_group": req.CheckNodeGroup,
			"check_action":     req.CheckAction,
			"bw_limit":         req.BwLimit,
			"level":            req.Level,
			"sort":             req.Sort,
			"cache_dir":        req.CacheDir,
			"max_cache_size":   req.MaxCacheSize,
			"log_dir":          req.LogDir,
			"ssh_host":         req.SSHHost,
			"ssh_port":         req.SSHPort,
			"ssh_user":         req.SSHUser,
			"ssh_auth_type":    req.SSHAuthType,
			"work_dir":         req.WorkDir,
			"auto_install":     req.AutoInstall,
			"update_at":        time.Now(),
		}
		if strings.TrimSpace(req.SSHPassword) != "" {
			updates["ssh_password"] = req.SSHPassword
		}
		if strings.TrimSpace(req.SSHKey) != "" {
			updates["ssh_key"] = req.SSHKey
		}
		if syncTask != "" {
			updates["config_task"] = syncTask
		}

		if err := tx.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		parent := models.Node{
			RegionID:  req.RegionID,
			Name:      req.Name,
			Remark:    req.Remark,
			IP:        req.IP,
			Host:      req.Host,
			Port:      req.Port,
			HttpProxy: req.HttpProxy,
			IsMgmt:    req.IsMgmt,
			Enable:    req.Enable,
		}

		var oldSubNodes []models.Node
		if err := tx.Select("id", "ip").Where("pid = ?", id).Find(&oldSubNodes).Error; err != nil {
			return err
		}
		if !sameSubIPs(oldSubNodes, req.SubIPs) {
			if len(oldSubNodes) > 0 {
				oldIDs := make([]int64, 0, len(oldSubNodes))
				for _, n := range oldSubNodes {
					oldIDs = append(oldIDs, n.ID)
				}
				var lineCount int64
				if err := tx.Model(&models.Line{}).
					Where("node_ip_id IN ?", oldIDs).
					Count(&lineCount).Error; err != nil {
					return err
				}
				if lineCount > 0 {
					return errNodeSubIPChangeBlocked
				}
			}
			if err := replaceSubIPs(tx, id, parent, req.SubIPs); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, errNodeSubIPChangeBlocked) {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "该节点子IP已被线路引用，不能直接修改，请先解绑线路后重试"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
		return
	}

	if ctr.NodeService != nil {
		// Update cached node metadata if used
		var fullNode models.Node
		db.DB.First(&fullNode, id)
		ctr.NodeService.SyncNodeToRedis(&fullNode)

		// Sync SubNodes
		var subNodes []models.Node
		db.DB.Where("pid = ?", id).Find(&subNodes)
		for _, sub := range subNodes {
			ctr.NodeService.SyncNodeToRedis(&sub)
		}
	}
	if err := services.SyncPackageCnameForNodes([]int64{id}, "resync"); err != nil {
		log.Printf("[DNS] package cname resync failed node=%d err=%v", id, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  T("Node Updated Successfully"),
	})
}

// DeleteNode
// DELETE /api/v1/admin/nodes/:id
func (ctr *NodeController) DeleteNode(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid ID")})
		return
	}
	inUse, err := hasLineBindings(db.DB, []int64{id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
		return
	}
	if inUse {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.delete_in_use")})
		return
	}
	recordNodeIPSwitchLogs(id, "delete")
	if err := services.SyncPackageCnameForNodes([]int64{id}, "delete"); err != nil {
		log.Printf("[DNS] package cname delete sync failed node=%d err=%v", id, err)
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var subIDs []int64
		if err := tx.Model(&models.Node{}).Where("pid = ?", id).Pluck("id", &subIDs).Error; err != nil {
			return err
		}

		ids := make([]int64, 0, 1+len(subIDs))
		ids = append(ids, id)
		ids = append(ids, subIDs...)

		if err := tx.Where("pid = ?", id).Delete(&models.Node{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&models.Node{}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Delete Failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Deleted")})
}

// BatchAction
// POST /api/v1/admin/nodes/batch
func (ctr *NodeController) BatchAction(c *gin.Context) {
	var req struct {
		Action string  `json:"action"`
		Ids    []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid Params")})
		return
	}

	if len(req.Ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("ids is required")})
		return
	}

	switch strings.ToLower(req.Action) {
	case "start":
		if err := db.DB.Model(&models.Node{}).
			Where("id IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":      true,
				"config_task": "sync_enable",
				"update_at":   time.Now(),
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
			return
		}
		_ = db.DB.Model(&models.Node{}).
			Where("pid IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":    true,
				"update_at": time.Now(),
			}).Error
		recordBatchNodeIPSwitchLogs(req.Ids, "enable")
	case "stop":
		if err := db.DB.Model(&models.Node{}).
			Where("id IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":      false,
				"config_task": "sync_disable",
				"update_at":   time.Now(),
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Update Failed")})
			return
		}
		_ = db.DB.Model(&models.Node{}).
			Where("pid IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":    false,
				"update_at": time.Now(),
			}).Error
		recordBatchNodeIPSwitchLogs(req.Ids, "disable")
	case "delete":
		inUse, err := hasLineBindings(db.DB, req.Ids)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Database Error")})
			return
		}
		if inUse {
			c.JSON(http.StatusBadRequest, gin.H{"msg": T("node.delete_in_use")})
			return
		}
		recordBatchNodeIPSwitchLogs(req.Ids, "delete")
		err = db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("pid IN ?", req.Ids).Delete(&models.Node{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.Ids).Delete(&models.Node{}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": T("Delete Failed")})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Unknown action")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": fmt.Sprintf(T("Batch action executed on %d nodes"), len(req.Ids))})
}

// InstallNode
// POST /api/v1/admin/nodes/:id/install
func (ctr *NodeController) InstallNode(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": T("Invalid ID")})
		return
	}
	var node models.Node
	if err := db.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": T("node not found")})
		return
	}
	apiBase := services.ResolveAPIBaseURL(c.Request)
	if err := services.ValidateNodeInstallConfig(&node, apiBase); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T(err.Error())})
		return
	}
	if err := updateInstallStatus(node.ID, "running", "", time.Now()); err != nil {
		log.Printf("[Install] update running status failed node=%d err=%v", node.ID, err)
	}
	startNodeInstallAsync(node, apiBase)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": T("Success"), "install_status": "running"})
}

func actionLabel(enable bool) string {
	if enable {
		return "enable"
	}
	return "disable"
}

func recordNodeIPSwitchLogs(nodeID int64, action string) {
	if nodeID == 0 {
		return
	}
	var subIDs []int64
	_ = db.DB.Model(&models.Node{}).Where("pid = ?", nodeID).Pluck("id", &subIDs).Error
	ids := append([]int64{nodeID}, subIDs...)
	var lines []models.Line
	_ = db.DB.Where("node_id IN ? OR node_ip_id IN ?", ids, ids).Find(&lines).Error
	if len(lines) > 0 {
		services.WriteIPSwitchLogsForLines(lines, action, "node")
		return
	}
	var node models.Node
	if err := db.DB.Select("id", "ip").Where("id = ?", nodeID).First(&node).Error; err == nil {
		services.WriteIPSwitchLogForNode(node, action, "node", "")
	}
}

func recordBatchNodeIPSwitchLogs(nodeIDs []int64, action string) {
	for _, id := range nodeIDs {
		recordNodeIPSwitchLogs(id, action)
	}
}

func resolveInitialInstallStatus(autoInstall bool) string {
	if autoInstall {
		return "running"
	}
	return "idle"
}

func normalizeNodeSSHDefaults(req *nodeRequest) {
	if strings.TrimSpace(req.SSHUser) == "" {
		req.SSHUser = defaultNodeSSHUser
	} else {
		req.SSHUser = strings.TrimSpace(req.SSHUser)
	}
	if req.SSHPort <= 0 {
		req.SSHPort = 22
	}
}

func hasLineBindings(tx *gorm.DB, nodeIDs []int64) (bool, error) {
	if len(nodeIDs) == 0 {
		return false, nil
	}
	var subIDs []int64
	if err := tx.Model(&models.Node{}).Where("pid IN ?", nodeIDs).Pluck("id", &subIDs).Error; err != nil {
		return false, err
	}
	ids := make([]int64, 0, len(nodeIDs)+len(subIDs))
	ids = append(ids, nodeIDs...)
	ids = append(ids, subIDs...)
	var count int64
	if err := tx.Model(&models.Line{}).
		Where("node_id IN ? OR node_ip_id IN ?", ids, ids).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func updateInstallStatus(nodeID int64, status, errMsg string, at time.Time) error {
	updates := map[string]interface{}{
		"install_status": status,
		"install_error":  errMsg,
		"install_at":     at,
		"update_at":      time.Now(),
	}
	return db.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(updates).Error
}

func startNodeInstallAsync(node models.Node, apiBase string) {
	copyNode := node
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Install] panic node=%d err=%v", copyNode.ID, r)
			}
		}()
		if err := services.InstallNodeAgent(&copyNode, apiBase); err != nil {
			log.Printf("[Install] failed node=%d err=%v", copyNode.ID, err)
			_ = updateInstallStatus(copyNode.ID, "failed", err.Error(), time.Now())
			_ = services.UpdateInstallProgress(copyNode.ID, "failed", 0, 0, 0, err.Error())
			return
		}
		_ = updateInstallStatus(copyNode.ID, "success", "", time.Now())
		_ = services.UpdateInstallProgress(copyNode.ID, "success", 100, 0, 0, "")
	}()
}

func replaceSubIPs(tx *gorm.DB, parentID int64, parent models.Node, subIPs []models.NodeSubIP) error {
	if err := tx.Where("pid = ?", parentID).Delete(&models.Node{}).Error; err != nil {
		return err
	}

	if len(subIPs) == 0 {
		return nil
	}

	now := time.Now()
	nodes := make([]models.Node, 0, len(subIPs))
	for _, sub := range subIPs {
		ip := strings.TrimSpace(sub.IP)
		if ip == "" {
			continue
		}
		nodes = append(nodes, models.Node{
			PID:       parentID,
			RegionID:  parent.RegionID,
			Name:      parent.Name,
			Remark:    parent.Remark,
			IP:        ip,
			Host:      parent.Host,
			Port:      parent.Port,
			HttpProxy: parent.HttpProxy,
			IsMgmt:    parent.IsMgmt,
			Enable:    parent.Enable,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if len(nodes) == 0 {
		return nil
	}

	return tx.Create(&nodes).Error
}

func normalizeSubIPValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, v := range values {
		ip := strings.TrimSpace(v)
		if ip == "" {
			continue
		}
		normalized = append(normalized, ip)
	}
	sort.Strings(normalized)
	return normalized
}

func sameSubIPs(oldSubNodes []models.Node, newSubIPs []models.NodeSubIP) bool {
	oldValues := make([]string, 0, len(oldSubNodes))
	for _, n := range oldSubNodes {
		oldValues = append(oldValues, n.IP)
	}
	newValues := make([]string, 0, len(newSubIPs))
	for _, n := range newSubIPs {
		newValues = append(newValues, n.IP)
	}
	oldNorm := normalizeSubIPValues(oldValues)
	newNorm := normalizeSubIPValues(newValues)
	if len(oldNorm) != len(newNorm) {
		return false
	}
	for i := range oldNorm {
		if oldNorm[i] != newNorm[i] {
			return false
		}
	}
	return true
}
