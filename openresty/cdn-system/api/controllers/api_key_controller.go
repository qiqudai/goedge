package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIKeyController struct{}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)[:n*2]
}

func ensureAPIKey(userID int64) (*models.APIKey, error) {
	var key models.APIKey
	if err := db.DB.Where("uid = ?", userID).First(&key).Error; err == nil {
		return &key, nil
	}
	key = models.APIKey{
		UserID:    userID,
		APIKey:    randomHex(8),  // 16 chars
		APISecret: randomHex(15), // 30 chars
		APIIP:     "",
	}
	if err := db.DB.Create(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// GetKey
// GET /api/v1/user/api_key
func (ctr *APIKeyController) GetKey(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	key, err := ensureAPIKey(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"api_key":    key.APIKey,
			"api_secret": key.APISecret,
			"api_ip":     key.APIIP,
		},
	})
}

// UpdateKey
// PUT /api/v1/user/api_key
func (ctr *APIKeyController) UpdateKey(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	var req struct {
		APIIP string `json:"api_ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid JSON"})
		return
	}

	key, err := ensureAPIKey(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	if err := db.DB.Model(&models.APIKey{}).Where("id = ?", key.ID).Update("api_ip", req.APIIP).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Updated"})
}

// ResetSecret
// POST /api/v1/user/api_key/reset
func (ctr *APIKeyController) ResetSecret(c *gin.Context) {
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(int64)

	key, err := ensureAPIKey(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	newSecret := randomHex(15)
	if err := db.DB.Model(&models.APIKey{}).Where("id = ?", key.ID).Update("api_secret", newSecret).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "DB Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"api_secret": newSecret,
		},
	})
}
