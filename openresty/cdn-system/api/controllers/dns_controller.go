package controllers

import (
	"cdn-api/models"
	"cdn-api/services/dns"

	// Register providers
	_ "cdn-api/services/dns/providers"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DnsController struct{}

var (
	mockProviders      = []models.DNSProvider{}
	mockProvidersMutex sync.Mutex
	mockProviderNextID int64 = 1
)

func init() {
	// Add some dummy data
	mockProviders = append(mockProviders, models.DNSProvider{
		ID:        mockProviderNextID,
		Name:      "My Cloudflare",
		Type:      "cloudflare",
		CreatedAt: time.Now(),
	})
	mockProviderNextID++
}

// ListProviders
func (ctr *DnsController) ListProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": mockProviders,
		},
	})
}

// GetProviderTypes
func (ctr *DnsController) GetProviderTypes(c *gin.Context) {
	types := []gin.H{
		{"type": "cloudflare", "name": "Cloudflare", "fields": []string{"email", "api_key"}},
		{"type": "aliyun", "name": "Aliyun", "fields": []string{"access_key_id", "access_key_secret"}},
		{"type": "dnspod", "name": "DNSPod", "fields": []string{"id", "token"}},
		{"type": "godaddy", "name": "GoDaddy", "fields": []string{"key", "secret"}},
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"types": types,
		},
	})
}

// CreateProvider
func (ctr *DnsController) CreateProvider(c *gin.Context) {
	var req models.DNSProvider
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Request"})
		return
	}

	// Validate Credentials with Factory
	_, err := dns.GetProvider(req.Type, req.Credentials)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid Credentials: " + err.Error()})
		return
	}

	mockProvidersMutex.Lock()
	req.ID = mockProviderNextID
	req.CreatedAt = time.Now()
	mockProviders = append(mockProviders, req)
	mockProviderNextID++
	mockProvidersMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}

// DeleteProvider
func (ctr *DnsController) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	mockProvidersMutex.Lock()
	defer mockProvidersMutex.Unlock()

	newProviders := []models.DNSProvider{}
	for _, p := range mockProviders {
		if p.ID != id {
			newProviders = append(newProviders, p)
		}
	}
	mockProviders = newProviders

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Success"})
}
