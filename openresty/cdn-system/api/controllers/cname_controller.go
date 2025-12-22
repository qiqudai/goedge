package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CnameController struct{}

func (c *CnameController) ListDomains(ctx *gin.Context) {
	var list []models.CnameDomain
	result := db.DB.Find(&list)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to fetch List"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list}})
}

func (c *CnameController) CreateDomain(ctx *gin.Context) {
	var input struct {
		Domain string `json:"domain"`
		Note   string `json:"note"`
	}
	if err := ctx.BindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid params"})
		return
	}

	model := models.CnameDomain{
		Domain: input.Domain,
		Note:   input.Note,
	}

	if err := db.DB.Create(&model).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to create"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

func (c *CnameController) DeleteDomain(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := db.DB.Delete(&models.CnameDomain{}, id).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "Failed to delete"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}
