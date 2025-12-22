package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type NodeGroupController struct{}

// ListNodeGroups
// GET /api/v1/admin/node-groups
func (ctr *NodeGroupController) ListNodeGroups(c *gin.Context) {
	var groups []models.NodeGroup
	// Removed sort_order as it's not in new schema
	if err := db.DB.Order("id desc").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  groups,
			"total": len(groups),
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

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Created",
		"data": req,
	})
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

	updates := map[string]interface{}{
		"name":                 req.Name,
		"region_id":            req.RegionID,
		"cname_hostname":       req.CnameHostname,
		"des":                  req.Description,
		"backup_switch_type":   req.BackupSwitchType,
		"backup_switch_policy": req.BackupSwitchPolicy,
		"update_at":            time.Now(),
	}
	if err := db.DB.Model(&models.NodeGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}

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
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Deleted",
	})
}
