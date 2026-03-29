package controllers

import (
	"log"
	"net/http"

	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type AgentLogController struct{}

func (ctr *AgentLogController) AccessLogs(c *gin.Context) {
	var req struct {
		NodeID string   `json:"node_id"`
		NodeIP string   `json:"node_ip"`
		Lines  []string `json:"lines"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
		return
	}
	applyAgentNodeIdentity(c, &req.NodeID, &req.NodeIP)
	inserted := services.InsertAccessLogs(req.NodeID, req.NodeIP, req.Lines)
	log.Printf("[CK] Access logs inserted: %d", inserted)
	c.JSON(http.StatusOK, gin.H{"status": T("status.ok")})
}

func (ctr *AgentLogController) Metrics(c *gin.Context) {
	var req struct {
		NodeID  string `json:"node_id"`
		NodeIP  string `json:"node_ip"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
		return
	}
	applyAgentNodeIdentity(c, &req.NodeID, &req.NodeIP)
	inserted := services.InsertMetrics(req.NodeID, req.NodeIP, req.Content)
	log.Printf("[CK] Metrics inserted: %d", inserted)
	c.JSON(http.StatusOK, gin.H{"status": T("status.ok")})
}

func (ctr *AgentLogController) Events(c *gin.Context) {
	var req struct {
		NodeID   string   `json:"node_id"`
		NodeIP   string   `json:"node_ip"`
		Type     string   `json:"type"`
		Payloads []string `json:"payloads"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": T("invalid request")})
		return
	}
	applyAgentNodeIdentity(c, &req.NodeID, &req.NodeIP)
	if req.Type == "" {
		req.Type = "event"
	}
	inserted := services.InsertEventLogs(req.NodeID, req.NodeIP, req.Type, req.Payloads)
	log.Printf("[CK] Events inserted: %d", inserted)
	c.JSON(http.StatusOK, gin.H{"status": T("status.ok")})
}
