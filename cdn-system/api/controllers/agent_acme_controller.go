package controllers

import (
	"net/http"
	"strings"
	"time"

	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type AgentAcmeController struct{}

type acmeTokenRequest struct {
	Token string `json:"token"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

func (ctr *AgentAcmeController) PutToken(c *gin.Context) {
	var req acmeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("token is required")})
		return
	}
	value := strings.TrimSpace(req.Value)
	if value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("value is required")})
		return
	}
	ttl := 15 * time.Minute
	if req.TTL > 0 {
		ttl = time.Duration(req.TTL) * time.Second
	}
	services.AcmeTokens.Put(token, value, ttl)
	c.JSON(http.StatusOK, gin.H{"status": T("status.ok")})
}

func (ctr *AgentAcmeController) DeleteToken(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("token is required")})
		return
	}
	services.AcmeTokens.Delete(token)
	c.JSON(http.StatusOK, gin.H{"status": T("status.ok")})
}
