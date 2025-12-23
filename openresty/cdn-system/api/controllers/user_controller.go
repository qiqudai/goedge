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

type UserController struct{}

// ListUsers returns paginated user list
// GET /api/v1/admin/users?page=1&size=20
func (ctr *UserController) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var users []models.User
	query := db.DB.Model(&models.User{})
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		// Updated to use Name column instead of Username
		query = query.Where("lower(name) LIKE ? OR email LIKE ? OR phone LIKE ?", keywordLike, keywordLike, keywordLike)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"list":  users,
			"total": total,
		},
	})
}

// ToggleStatus enables or disables a user
// PUT /api/v1/admin/users/:id/status
func (ctr *UserController) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid params"})
		return
	}

	enabled := req.Status == 1
	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("enable", enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "User " + id + " status updated to " + strconv.Itoa(req.Status)})
}

// DeleteUser removes a user
// DELETE /api/v1/admin/users/:id
func (ctr *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Delete Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "User " + id + " deleted"})
}

// ListUserNodeGroups returns node groups assigned to a user
// GET /api/v1/admin/users/:id/node-groups
func (ctr *UserController) ListUserNodeGroups(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)

	var groups []models.NodeGroup
	if err := db.DB.Table("node_groups").
		Select("node_groups.*").
		Joins("JOIN user_node_groups ON user_node_groups.node_group_id = node_groups.id").
		Where("user_node_groups.user_id = ?", userID).
		Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": groups}})
}

// UpdateUserNodeGroups assigns node groups to a user
// PUT /api/v1/admin/users/:id/node-groups
func (ctr *UserController) UpdateUserNodeGroups(c *gin.Context) {
	idStr := c.Param("id")
	userID, _ := strconv.ParseInt(idStr, 10, 64)

	var req struct {
		NodeGroupIDs []int64 `json:"node_group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserNodeGroup{}).Error; err != nil {
			return err
		}
		if len(req.NodeGroupIDs) == 0 {
			return nil
		}
		mappings := make([]models.UserNodeGroup, 0, len(req.NodeGroupIDs))
		for _, gid := range req.NodeGroupIDs {
			mappings = append(mappings, models.UserNodeGroup{
				UserID:      userID,
				NodeGroupID: gid,
				CreatedAt:   time.Now(),
			})
		}
		return tx.Create(&mappings).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Update Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "User node groups updated"})
}
