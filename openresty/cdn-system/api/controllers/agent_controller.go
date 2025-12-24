package controllers

import (
	"cdn-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AgentController struct {
	ConfigSvc *services.ConfigService
}

func NewAgentController() *AgentController {
	return &AgentController{
		ConfigSvc: services.NewConfigService(),
	}
}

func (ctr *AgentController) Heartbeat(c *gin.Context) {
	// TODO: Update node status in Redis/DB
	c.JSON(http.StatusOK, gin.H{"status": "pong"})
}

func (ctr *AgentController) GetConfig(c *gin.Context) {
	var nodeID string
	// Prioritize Authenticated Node ID
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			nodeID = s
		}
	}
	if nodeID == "" {
		nodeID = c.Query("node_id")
	}
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	config, err := ctr.ConfigSvc.GenerateConfigForNode(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate config"})
		return
	}

	if verStr := c.Query("version"); verStr != "" {
		if ver, err := strconv.ParseInt(verStr, 10, 64); err == nil && ver == config.Version {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.JSON(http.StatusOK, config)
}
