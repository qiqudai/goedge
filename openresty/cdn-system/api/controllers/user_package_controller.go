package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserPackageController struct{}

// ListUserPackages - GET /api/v1/admin/user_packages?user_id=xx
func (ctr *UserPackageController) ListUserPackages(c *gin.Context) {
	var packs []models.UserPackage
	query := db.DB.Model(&models.UserPackage{})
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			query = query.Where("uid = ?", uid)
		}
	}
	if err := query.Order("id desc").Find(&packs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": packs}})
}
