package controllers

import (
	"cdn-api/services"
	"net/http"

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
	nodeID := c.Query("node_id")
	if nodeID == "" {
		if v, ok := c.Get("nodeID"); ok {
			if s, ok := v.(string); ok {
				nodeID = s
			}
		}
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

	c.JSON(http.StatusOK, config)
}
