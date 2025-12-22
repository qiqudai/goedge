package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskController struct{}

// Create handles purge/preheat task creation
func (c *TaskController) Create(ctx *gin.Context) {
	var input struct {
		Type string `json:"type"` // refresh_url, refresh_dir, preheat
		URLs string `json:"urls"` // newline separated
	}
	if err := ctx.BindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid params"})
		return
	}

	if input.URLs == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "URLs cannot be empty"})
		return
	}

	task := models.Task{
		Type:     input.Type,
		Data:     input.URLs,
		State:    "waiting",
		CreateAt: time.Now(),
		Enable:   true,
	}

	if err := db.DB.Create(&task).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to create task"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

// List returns tasks list
func (c *TaskController) List(ctx *gin.Context) {
	var list []models.Task
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	keyword := ctx.Query("keyword")
	taskType := ctx.Query("type")

	query := db.DB.Model(&models.Task{})

	if keyword != "" {
		query = query.Where("data LIKE ?", "%"+keyword+"%")
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}

	var total int64
	query.Count(&total)

	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to fetch list"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"list":  list,
		"total": total,
		"page":  page,
	})
}
