package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlanController struct{}

// ListPlans - GET /api/v1/plans
func (ctr *PlanController) ListPlans(c *gin.Context) {
	var plans []models.Plan
	// Simple list for now, maybe add pagination later
	if err := db.DB.Order("sort_order asc, id desc").Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Database Error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": plans, "total": len(plans)}})
}

// CreatePlan - POST /api/v1/plans
func (ctr *PlanController) CreatePlan(c *gin.Context) {
	var plan models.Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}
	
	if err := db.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Create Failed"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Created"})
}

// UpdatePlan - PUT /api/v1/plans/:id
func (ctr *PlanController) UpdatePlan(c *gin.Context) {
    idStr := c.Param("id")
    id, _ := strconv.Atoi(idStr)
    
	var plan models.Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}
	
    // Update fields
    // GORM updates non-zero fields by default with Model(&dest).Updates(...)
    // To ensure all fields (including bool false) are updated, we use the struct save or map
    plan.ID = uint(id)
    
	if err := db.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Update Failed"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Updated"})
}

// DeletePlan - DELETE /api/v1/plans/:id
func (ctr *PlanController) DeletePlan(c *gin.Context) {
    id := c.Param("id")
    if err := db.DB.Delete(&models.Plan{}, id).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "Delete Failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Plan Deleted"})
}

// ListUserPlans - GET /api/v1/user_plans
func (ctr *PlanController) ListUserPlans(c *gin.Context) {
    var userPlans []models.UserPlan
    if err := db.DB.Order("id desc").Find(&userPlans).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
        return
    }
     c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": userPlans}}) // Structure matching standard list response
}
