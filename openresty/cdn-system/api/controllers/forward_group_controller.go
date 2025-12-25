package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ForwardGroupController struct{}

func (ctrl *ForwardGroupController) List(c *gin.Context) {
	var groups []models.ForwardGroup
	query := db.DB.Order("id desc")
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if err := query.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to fetch groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": groups}})
}

func (ctrl *ForwardGroupController) Create(c *gin.Context) {
	var req struct {
		UserID int64  `json:"user_id"`
		Name   string `json:"name"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "name is required"})
		return
	}
	if req.UserID == 0 {
		req.UserID = parseInt64(mustGet(c, "userID"))
	}
	group := &models.ForwardGroup{
		UserID:    req.UserID,
		Name:      req.Name,
		Remark:    req.Remark,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.DB.Create(group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to create group"})
		return
	}
	services.BumpConfigVersion("forward_group", []int64{group.ID})
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": group})
}

func (ctrl *ForwardGroupController) Update(c *gin.Context) {
	var req struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "id is required"})
		return
	}
	updates := map[string]interface{}{"update_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["des"] = req.Remark
	if err := db.DB.Model(&models.ForwardGroup{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to update group"})
		return
	}
	services.BumpConfigVersion("forward_group", []int64{req.ID})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "updated"})
}

func (ctrl *ForwardGroupController) Delete(c *gin.Context) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "id is required"})
		return
	}
	if err := db.DB.Where("id = ?", req.ID).Delete(&models.ForwardGroup{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Failed to delete group"})
		return
	}
	services.BumpConfigVersion("forward_group", []int64{req.ID})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "deleted"})
}
