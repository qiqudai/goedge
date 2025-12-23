package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConfigItemController struct{}

type configItemPayload struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Enable *bool  `json:"enable"`
}

type configItemUpsertRequest struct {
	Type      string              `json:"type"`
	ScopeName string              `json:"scope_name"`
	ScopeID   int64               `json:"scope_id"`
	Items     []configItemPayload `json:"items"`
}

func (ctrl *ConfigItemController) List(c *gin.Context) {
	cfgType := c.Query("type")
	scopeName := c.DefaultQuery("scope_name", "global")
	scopeID, _ := strconv.ParseInt(c.DefaultQuery("scope_id", "0"), 10, 64)

	var items []models.ConfigItem
	query := db.DB.Model(&models.ConfigItem{})
	if cfgType != "" {
		query = query.Where("type = ?", cfgType)
	}
	if scopeName != "" {
		query = query.Where("scope_name = ?", scopeName)
	}
	query = query.Where("scope_id = ?", scopeID)
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": items})
}

func (ctrl *ConfigItemController) Upsert(c *gin.Context) {
	var req configItemUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" || req.ScopeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items is required"})
		return
	}

	err := upsertConfigItems(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func (ctrl *ConfigItemController) ListUser(c *gin.Context) {
	uid := parseInt64(mustGet(c, "userID"))
	if uid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	cfgType := c.Query("type")
	var items []models.ConfigItem
	query := db.DB.Model(&models.ConfigItem{})
	if cfgType != "" {
		query = query.Where("type = ?", cfgType)
	}
	if err := query.Where("scope_name = ? AND scope_id = ?", "user", uid).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": items})
}

func (ctrl *ConfigItemController) UpsertUser(c *gin.Context) {
	uid := parseInt64(mustGet(c, "userID"))
	if uid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	var req configItemUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	req.ScopeName = "user"
	req.ScopeID = uid
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items is required"})
		return
	}

	if err := upsertConfigItems(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func upsertConfigItems(req configItemUpsertRequest) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if item.Name == "" {
				continue
			}
			enable := true
			if item.Enable != nil {
				enable = *item.Enable
			}
			var existing models.ConfigItem
			if err := tx.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", req.Type, req.ScopeName, req.ScopeID, item.Name).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					newItem := models.ConfigItem{
						Type:      req.Type,
						ScopeName: req.ScopeName,
						ScopeID:   req.ScopeID,
						Name:      item.Name,
						Value:     item.Value,
						Enable:    enable,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					if err := tx.Create(&newItem).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			existing.Value = item.Value
			existing.Enable = enable
			existing.UpdatedAt = time.Now()
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
