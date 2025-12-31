package controllers

import (
	"net/http"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type AgentCertController struct{}

func (ctr *AgentCertController) ReceiveIssued(c *gin.Context) {
	var req struct {
		CertID       int64  `json:"cert_id"`
		CertPEM      string `json:"cert"`
		KeyPEM       string `json:"key"`
		IssueTaskID  int64  `json:"issue_task_id"`
		RateLimited  bool   `json:"rate_limited"`
		RateCooldown int    `json:"rate_cooldown"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.CertID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cert_id is required"})
		return
	}
	nodeID := resolveAgentNodeID(c)
	if req.RateLimited && nodeID != 0 {
		cooldown := time.Minute * 10
		if req.RateCooldown > 0 {
			cooldown = time.Duration(req.RateCooldown) * time.Second
		}
		services.MarkNodeRateLimited(nodeID, cooldown)
	}
	if strings.TrimSpace(req.CertPEM) == "" || strings.TrimSpace(req.KeyPEM) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cert and key are required"})
		return
	}
	notBefore, notAfter, err := services.ParseCertTimes(req.CertPEM)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certificate"})
		return
	}
	// Fetch current cert to increment version
	var existingCert models.Cert
	if err := db.DB.First(&existingCert, req.CertID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cert not found"})
		return
	}
	// Encrypt key before update (Agent sends plaintext key?)
	// Note: Agent might send encrypted if configured? Assuming Agent sends PEM (plaintext).
	// UpdateIssuedCert expects ENCRYPTED KEY since my change in Step 222 (issueCertLocal).
	// Wait! `UpdateIssuedCert` arg is `keyCipher string`.
	// So I MUST ENCRYPT IT HERE if `req.KeyPEM` is plaintext.
	encryptedKey, err := services.Crypto.Encrypt(req.KeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	if err := services.UpdateIssuedCert(req.CertID, req.CertPEM, encryptedKey, notBefore, notAfter, req.IssueTaskID, existingCert.Version+1); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
