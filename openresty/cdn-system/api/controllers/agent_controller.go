package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	// TODO: Update node status in DB
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

func (ctr *AgentController) GetTasks(c *gin.Context) {
	nodeID := ""
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			nodeID = s
		}
	}
	if nodeID == "" {
		nodeID = c.Query("node_id")
	}

	var tasks []models.Task
	if err := db.DB.Where("enable = ? AND state IN ?", true, []string{"waiting", "running"}).Order("id asc").Limit(100).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tasks"})
		return
	}
	filtered := make([]models.Task, 0, len(tasks))
	now := time.Now()
	for _, task := range tasks {
		if task.RetryAt.After(now) {
			continue
		}
		if nodeID == "" || !taskProgressHasNode(task.Progress, nodeID) {
			filtered = append(filtered, task)
		}
	}
	if len(filtered) > 0 && nodeID != "" {
		for _, task := range filtered {
			progress := updateTaskProgress(task.Progress, nodeID, "running")
			db.DB.Model(&models.Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"state":    "running",
				"start_at": time.Now(),
				"progress": progress,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"tasks": filtered})
}

func (ctr *AgentController) FinishTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	nodeID := ""
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			nodeID = s
		}
	}
	var req struct {
		State string `json:"state"`
		Ret   string `json:"ret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.State == "" {
		req.State = "done"
	}
	var task models.Task
	if err := db.DB.Where("id = ?", id).First(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load task"})
		return
	}
	progress := updateTaskProgress(task.Progress, nodeID, req.State)
	updates := map[string]interface{}{
		"ret":      req.Ret,
		"progress": progress,
	}
	if req.State == "fail" {
		nextErrTimes := task.ErrTimes + 1
		maxRetries := 3
		updates["err_times"] = nextErrTimes
		if nextErrTimes >= maxRetries {
			updates["state"] = "fail"
			updates["end_at"] = time.Now()
		} else {
			updates["state"] = "waiting"
			updates["retry_at"] = time.Now().Add(time.Duration(nextErrTimes*30) * time.Second)
		}
	} else {
		updates["state"] = deriveTaskState(progress)
		if updates["state"] == "done" {
			updates["end_at"] = time.Now()
		}
	}
	if err := db.DB.Model(&models.Task{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func taskProgressHasNode(raw string, nodeID string) bool {
	if nodeID == "" || raw == "" {
		return false
	}
	var progress map[string]string
	if err := json.Unmarshal([]byte(raw), &progress); err != nil {
		return false
	}
	state, ok := progress[nodeID]
	if !ok {
		return false
	}
	return state != "fail"
}

func updateTaskProgress(raw string, nodeID string, state string) string {
	if nodeID == "" {
		return raw
	}
	progress := map[string]string{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &progress)
	}
	progress[nodeID] = state
	b, _ := json.Marshal(progress)
	return string(b)
}

func deriveTaskState(progressRaw string) string {
	if progressRaw == "" {
		return "running"
	}
	var progress map[string]string
	if err := json.Unmarshal([]byte(progressRaw), &progress); err != nil {
		return "running"
	}
	if len(progress) == 0 {
		return "running"
	}
	for _, v := range progress {
		if v != "done" {
			return "running"
		}
	}
	return "done"
}
