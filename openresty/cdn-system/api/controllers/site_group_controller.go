package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type SiteGroupController struct{}

func (ctr *SiteGroupController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var groups []models.SiteGroup
	var total int64
	query := db.DB.Model(&models.SiteGroup{})

	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid != 0 {
			query = query.Where("uid = ?", uid)
		}
	}

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}

	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": groups, "total": total}})
}

func (ctr *SiteGroupController) Create(c *gin.Context) {
	var req models.SiteGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}
	if req.UserID == 0 {
		if uid, ok := c.Get("userID"); ok {
			switch v := uid.(type) {
			case float64:
				req.UserID = int64(v)
			case int:
				req.UserID = int64(v)
			case int64:
				req.UserID = v
			case string:
				if id, err := strconv.ParseInt(v, 10, 64); err == nil {
					req.UserID = id
				}
			}
		}
	}
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Created", "data": req})
}

func (ctr *SiteGroupController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	var req models.SiteGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Params"})
		return
	}
	req.UpdatedAt = time.Now()
	query := db.DB.Model(&models.SiteGroup{}).Where("id = ?", id)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
			return
		}
		query = query.Where("uid = ?", uid)
	}
	if err := query.Updates(map[string]interface{}{
		"name":      req.Name,
		"des":       req.Remark,
		"update_at": req.UpdatedAt,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

func (ctr *SiteGroupController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if isUserRequest(c) {
		uid := parseUserID(mustGet(c, "userID"))
		if uid == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "Forbidden"})
			return
		}
		var group models.SiteGroup
		if err := db.DB.Where("id = ? AND uid = ?", id, uid).First(&group).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Not Found"})
			return
		}
	}
	if err := db.DB.Where("group_id = ?", id).Delete(&models.SiteGroupRelation{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
		return
	}
	if err := db.DB.Delete(&models.SiteGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Deleted"})
}
