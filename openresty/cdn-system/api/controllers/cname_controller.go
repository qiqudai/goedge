package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strings"
	"time"

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

	domain := normalizeDomainInput(input.Domain)
	if domain == "" || !isValidDomain(domain) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Invalid domain"})
		return
	}

	model := models.CnameDomain{
		Domain:    domain,
		Note:      input.Note,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

func normalizeDomainInput(input string) string {
	domain := strings.TrimSpace(strings.ToLower(input))
	if strings.HasPrefix(domain, "http://") {
		domain = strings.TrimPrefix(domain, "http://")
	} else if strings.HasPrefix(domain, "https://") {
		domain = strings.TrimPrefix(domain, "https://")
	}
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "#"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "?"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	domain = strings.TrimRight(domain, ".")
	return domain
}

func isValidDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" || len(part) > 63 {
			return false
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return false
		}
		for i := 0; i < len(part); i++ {
			ch := part[i]
			isLower := ch >= 'a' && ch <= 'z'
			isDigit := ch >= '0' && ch <= '9'
			if !(isLower || isDigit || ch == '-') {
				return false
			}
		}
	}
	return true
}
