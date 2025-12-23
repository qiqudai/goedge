package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AnnouncementController struct{}

type announcementRow struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	IsShow    bool   `json:"is_show"`
	IsRed     bool   `json:"is_red"`
	IsBold    bool   `json:"is_bold"`
	CreatedAt string `json:"created_at"`
}

const announcementType = "announcement"

// List
// GET /api/v1/admin/announcements
func (ctr *AnnouncementController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))

	query := db.DB.Model(&models.Message{}).Where("type = ?", announcementType)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	var listData []models.Message
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&listData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	list := make([]announcementRow, 0, len(listData))
	for _, item := range listData {
		list = append(list, announcementRow{
			ID:        item.ID,
			Title:     item.Title,
			Content:   item.Content,
			IsShow:    item.IsShow,
			IsRed:     item.IsRed,
			IsBold:    item.IsBold,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// Create
// POST /api/v1/admin/announcements
func (ctr *AnnouncementController) Create(c *gin.Context) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		IsShow  bool   `json:"is_show"`
		IsRed   bool   `json:"is_red"`
		IsBold  bool   `json:"is_bold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	item := models.Message{
		Type:      announcementType,
		Receive:   0,
		Title:     req.Title,
		Content:   req.Content,
		IsShow:    req.IsShow,
		IsRed:     req.IsRed,
		IsBold:    req.IsBold,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Created"})
}

// Update
// PUT /api/v1/admin/announcements/:id
func (ctr *AnnouncementController) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		IsShow  bool   `json:"is_show"`
		IsRed   bool   `json:"is_red"`
		IsBold  bool   `json:"is_bold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	updates := map[string]interface{}{
		"title":     req.Title,
		"content":   req.Content,
		"is_show":   req.IsShow,
		"is_red":    req.IsRed,
		"is_bold":   req.IsBold,
		"update_at": time.Now(),
	}

	if err := db.DB.Model(&models.Message{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

// Delete
// DELETE /api/v1/admin/announcements/:id
func (ctr *AnnouncementController) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid ID"})
		return
	}

	if err := db.DB.Delete(&models.Message{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Deleted"})
}
