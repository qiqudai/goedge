package controllers

import (
  "cdn-api/db"
  "cdn-api/models"
  "net/http"
  "time"

  "github.com/gin-gonic/gin"
)

type ForwardGroupController struct{}

func (ctrl *ForwardGroupController) List(c *gin.Context) {
  var groups []models.ForwardGroup
  if err := db.DB.Order("id desc").Find(&groups).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"list": groups})
}

func (ctrl *ForwardGroupController) Create(c *gin.Context) {
  var req struct {
    Name   string `json:"name"`
    Remark string `json:"remark"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
    return
  }
  group := &models.ForwardGroup{
    Name:      req.Name,
    Remark:    req.Remark,
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
  }
  if err := db.DB.Create(group).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "created", "data": group})
}

func (ctrl *ForwardGroupController) Update(c *gin.Context) {
  var req struct {
    ID     int64  `json:"id"`
    Name   string `json:"name"`
    Remark string `json:"remark"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
    return
  }
  updates := map[string]interface{}{"update_at": time.Now()}
  if req.Name != "" {
    updates["name"] = req.Name
  }
  if req.Remark != "" {
    updates["remark"] = req.Remark
  }
  if err := db.DB.Model(&models.ForwardGroup{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (ctrl *ForwardGroupController) Delete(c *gin.Context) {
  var req struct {
    ID int64 `json:"id"`
  }
  if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
    return
  }
  if err := db.DB.Where("id = ?", req.ID).Delete(&models.ForwardGroup{}).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
