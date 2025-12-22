package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeController struct{}

// ListNodes
// GET /api/v1/admin/nodes
func (ctr *NodeController) ListNodes(c *gin.Context) {
	// 1. Parse Params
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// 2. Query DB
	var nodes []models.Node
	query := db.DB.Model(&models.Node{})
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ? OR ip LIKE ?", keywordLike, keywordLike)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	// Order by ID desc
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
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

	// Set Defaults
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	// Default Enable is true if not specified?
	// If the JSON didn't include "enable", it defaults to false.
	// But usually we want enabled by default.
	// Let's assume if user sends "enable": false explicitly, it stays false.
	// But Request binding zero value is false.
	// We can't distinguish missing vs false easily without pointer.
	// For now, let's force true if it's a new node, unless we change model to *bool.
	req.Enable = true

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
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

	updates := map[string]interface{}{
		"name":           req.Name,
		"des":            req.Description,
		"ip":             req.IP,
		"region_id":      req.RegionID,
		"enable":         req.Enable,
		"check_on":       req.CheckOn,
		"check_protocol": req.CheckProtocol,
		"check_port":     req.CheckPort,
		"check_host":     req.CheckHost,
		"check_path":     req.CheckPath,
		"update_at":      time.Now(),
	}

	if err := db.DB.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
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
		Action string  `json:"action"` // start, stop, delete
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
		if err := db.DB.Model(&models.Node{}).Where("id IN ?", req.Ids).Update("enable", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
			return
		}
	case "stop":
		if err := db.DB.Model(&models.Node{}).Where("id IN ?", req.Ids).Update("enable", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
			return
		}
	case "delete":
		// Check references in 'line' table? For now just delete node.
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			// Delete related lines?
			if err := tx.Where("node_id IN ?", req.Ids).Delete(&models.Line{}).Error; err != nil {
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
