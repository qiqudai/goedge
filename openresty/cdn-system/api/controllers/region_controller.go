package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RegionController struct{}

// ListRegions
// GET /api/v1/admin/regions
func (ctr *RegionController) ListRegions(c *gin.Context) {
	var regions []models.Region
	if err := db.DB.Order("id asc").Find(&regions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Database Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  regions,
			"total": len(regions),
		},
	})
}
