package controllers

import (
	"net/http"
	"strings"
	"time"

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
	if err := services.UpdateIssuedCert(req.CertID, req.CertPEM, req.KeyPEM, notBefore, notAfter, req.IssueTaskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
