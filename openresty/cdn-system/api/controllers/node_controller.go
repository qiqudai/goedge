package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeController struct {
	NodeService *services.NodeService
}

// ListNodes
// GET /api/v1/admin/nodes
func (ctr *NodeController) ListNodes(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var nodes []models.Node
	query := db.DB.Model(&models.Node{}).Where("pid = 0")
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ? OR ip LIKE ?", keywordLike, keywordLike)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&nodes).Error; err != nil {
		log.Println("[Error] ListNodes DB Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error: " + err.Error()})
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
	}

	for i := range nodes {
		nodes[i].Online = services.IsNodeOnline(nodes[i].ID, 30*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  nodes,
			"total": total,
		},
	})
}

// CreateNode
// POST /api/v1/admin/nodes
func (ctr *NodeController) CreateNode(c *gin.Context) {
	var req models.Node
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.IP) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Name and IP are required"})
		return
	}

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	req.PID = 0
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	if !req.Enable {
		req.Enable = true
	}

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
	}

	if err := replaceSubIPs(db.DB, req.ID, req, req.SubIPs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Sub IPs Failed"})
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
		ctr.NodeService.SyncNodeToRedis(&req)
		
		// Also sync sub nodes.
		// replaceSubIPs creates new Node records with PID=req.ID.
		// We should really fetch them to sync properly.
        var subNodes []models.Node
        db.DB.Where("pid = ?", req.ID).Find(&subNodes)
        for _, sub := range subNodes {
             ctr.NodeService.SyncNodeToRedis(&sub)
        }
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Node Created",
		"data": req,
	})
}

// UpdateNode
// PUT /api/v1/admin/nodes/:id
func (ctr *NodeController) UpdateNode(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var req models.Node
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	if req.RegionID != nil && *req.RegionID == 0 {
		req.RegionID = nil
	}

	var existing models.Node
	_ = db.DB.Select("enable").Where("id = ?", id).First(&existing).Error
	syncTask := ""
	if req.Enable != existing.Enable {
		if req.Enable {
			syncTask = "sync_enable"
		} else {
			syncTask = "sync_disable"
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
			"update_at":        time.Now(),
		}
		if syncTask != "" {
			updates["config_task"] = syncTask
		}

		if err := tx.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		if err := replaceSubIPs(tx, id, req, req.SubIPs); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Node Updated Successfully",
	})
}

// BatchAction
// POST /api/v1/admin/nodes/batch
func (ctr *NodeController) BatchAction(c *gin.Context) {
	var req struct {
		Action string  `json:"action"`
		Ids    []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	if len(req.Ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "ids is required"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
			return
		}
		_ = db.DB.Model(&models.Node{}).
			Where("pid IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":    true,
				"update_at": time.Now(),
			}).Error
	case "stop":
		if err := db.DB.Model(&models.Node{}).
			Where("id IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":      false,
				"config_task": "sync_disable",
				"update_at":   time.Now(),
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
			return
		}
		_ = db.DB.Model(&models.Node{}).
			Where("pid IN ?", req.Ids).
			Updates(map[string]interface{}{
				"enable":    false,
				"update_at": time.Now(),
			}).Error
	case "delete":
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("node_id IN ?", req.Ids).Delete(&models.Line{}).Error; err != nil {
				return err
			}
			if err := tx.Where("pid IN ?", req.Ids).Delete(&models.Node{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.Ids).Delete(&models.Node{}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Delete Failed"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Unknown action"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Batch " + req.Action + " executed on " + strconv.Itoa(len(req.Ids)) + " nodes"})
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
