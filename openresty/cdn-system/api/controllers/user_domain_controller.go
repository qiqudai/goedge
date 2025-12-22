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

type UserDomainController struct{}

// ListDomains
// GET /api/v1/user/domains
func (ctr *UserDomainController) ListDomains(c *gin.Context) {
	userID, _ := c.Get("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var domains []models.Domain
	query := db.DB.Model(&models.Domain{}).Preload("Origins").Where("user_id = ?", userID)
	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ?", keywordLike)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  domains,
			"total": total,
		},
	})
}

// AdminListDomains
// GET /api/v1/admin/domains
func (ctr *UserDomainController) AdminListDomains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var domains []models.Domain
	query := db.DB.Model(&models.Domain{}).Preload("Origins") // No User Filter

	if keyword != "" {
		keywordLike := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("lower(name) LIKE ?", keywordLike)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  domains,
			"total": total,
		},
	})
}

// CreateDomain
// POST /api/v1/user/domains
func (ctr *UserDomainController) CreateDomain(c *gin.Context) {
	var req struct {
		Name    string                `json:"name"`
		Origins []models.DomainOrigin `json:"origins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Params"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Domain name is required"})
		return
	}
	userID, _ := c.Get("userID")
	uid := userID.(int64)

	var existing models.Domain
	if err := db.DB.Where("user_id = ? AND name = ?", uid, req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Domain already exists"})
		return
	}

	domain := models.Domain{
		UserID:    uid,
		Name:      req.Name,
		Cname:     req.Name + ".cdn.node.com",
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&domain).Error; err != nil {
			return err
		}
		if len(req.Origins) > 0 {
			for i := range req.Origins {
				req.Origins[i].DomainID = domain.ID
				if req.Origins[i].Port == 0 {
					req.Origins[i].Port = 80
				}
				if req.Origins[i].Weight == 0 {
					req.Origins[i].Weight = 1
				}
				if req.Origins[i].Protocol == "" {
					req.Origins[i].Protocol = "http"
				}
			}
			if err := tx.Create(&req.Origins).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Create Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Domain Created", "data": domain})
}

// GetConfig
// GET /api/v1/user/domains/:id/config
func (ctr *UserDomainController) GetConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	userID, _ := c.Get("userID")

	var domain models.Domain
	if err := db.DB.Preload("Origins").Where("id = ? AND user_id = ?", id, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Domain Not Found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"domain":   domain.Name,
			"origins":  domain.Origins,
			"https_on": true,
			"cache_rules": []gin.H{
				{"ext": ".jpg", "ttl": 3600},
			},
		},
	})
}
