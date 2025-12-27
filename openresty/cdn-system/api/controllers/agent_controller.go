package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	var payload struct {
		NodeID    string `json:"node_id"`
		Timestamp int64  `json:"timestamp"`
		Status    string `json:"status"`
	}
	_ = c.ShouldBindJSON(&payload)
	nodeID := resolveHeartbeatNodeID(c, payload.NodeID)
	if nodeID != 0 {
		services.MarkNodeOnline(nodeID, time.Now())
	}
	syncAction := ""
	if nodeID != 0 {
		var node models.Node
		if err := db.DB.Select("config_task").Where("id = ?", nodeID).First(&node).Error; err == nil {
			switch strings.TrimSpace(node.ConfigTask) {
			case "sync_enable":
				syncAction = "enable"
			case "sync_disable":
				syncAction = "disable"
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "pong", "sync_action": syncAction})
}

func resolveHeartbeatNodeID(c *gin.Context, payloadID string) int64 {
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil && id > 0 {
				return id
			}
		}
	}

	if payloadID != "" {
		if id, err := strconv.ParseInt(payloadID, 10, 64); err == nil && id > 0 {
			return id
		}
		var node models.Node
		if err := db.DB.Where("name = ? AND pid = 0", payloadID).First(&node).Error; err == nil {
			return node.ID
		}
	}

	clientIP := c.ClientIP()
	if clientIP == "" {
		return 0
	}
	var node models.Node
	if err := db.DB.Where("ip = ? AND pid = 0", clientIP).First(&node).Error; err == nil {
		return node.ID
	}
	return 0
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

func (ctr *AgentController) GetL2Nodes(c *gin.Context) {
	nodeID := resolveAgentNodeID(c)
	if nodeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	var self models.Node
	if err := db.DB.Where("id = ?", nodeID).First(&self).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if self.Level != 1 {
		c.JSON(http.StatusOK, gin.H{"nodes": []gin.H{}})
		return
	}

	var groupIDs []int64
	if err := db.DB.Model(&models.Line{}).
		Select("distinct node_group_id").
		Where("node_id = ?", nodeID).
		Pluck("node_group_id", &groupIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load node groups"})
		return
	}
	if len(groupIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"nodes": []gin.H{}})
		return
	}

	var l2NodeIDs []int64
	if err := db.DB.Model(&models.Line{}).
		Select("distinct node_id").
		Where("node_group_id IN ?", groupIDs).
		Where("node_id <> ?", nodeID).
		Pluck("node_id", &l2NodeIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load l2 nodes"})
		return
	}
	if len(l2NodeIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"nodes": []gin.H{}})
		return
	}

	var nodes []models.Node
	if err := db.DB.Where("id IN ? AND level = ? AND enable = ?", l2NodeIDs, 2, true).
		Select("id", "ip", "port", "check_protocol", "check_port", "check_host", "check_path", "check_timeout").
		Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load l2 nodes"})
		return
	}

	result := make([]gin.H, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, gin.H{
			"id":             n.ID,
			"ip":             n.IP,
			"port":           n.Port,
			"check_protocol": n.CheckProtocol,
			"check_port":     n.CheckPort,
			"check_host":     n.CheckHost,
			"check_path":     n.CheckPath,
			"check_timeout":  n.CheckTimeout,
		})
	}
	c.JSON(http.StatusOK, gin.H{"nodes": result})
}

func (ctr *AgentController) ReportL2Heartbeat(c *gin.Context) {
	var req struct {
		Nodes []int64 `json:"nodes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if len(req.Nodes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	now := time.Now()
	for _, id := range req.Nodes {
		services.MarkNodeOnline(id, now)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (ctr *AgentController) SyncNodeStatus(c *gin.Context) {
	var req struct {
		NodeID  string `json:"node_id"`
		Action  string `json:"action"`
		Success bool   `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	nodeID := resolveAgentNodeID(c)
	if nodeID == 0 && strings.TrimSpace(req.NodeID) != "" {
		if id, err := strconv.ParseInt(req.NodeID, 10, 64); err == nil {
			nodeID = id
		} else {
			var node models.Node
			if err := db.DB.Where("name = ? AND pid = 0", req.NodeID).First(&node).Error; err == nil {
				nodeID = node.ID
			}
		}
	}
	if nodeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "enable" && action != "disable" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}
	if !req.Success {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	if err := db.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"config_task": "",
		"update_at":   time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func resolveAgentNodeID(c *gin.Context) int64 {
	if v, ok := c.Get("nodeID"); ok {
		if s, ok := v.(string); ok {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil && id > 0 {
				return id
			}
		}
	}
	nodeID := strings.TrimSpace(c.Query("node_id"))
	if nodeID == "" {
		return 0
	}
	if id, err := strconv.ParseInt(nodeID, 10, 64); err == nil && id > 0 {
		return id
	}
	var node models.Node
	if err := db.DB.Where("name = ? AND pid = 0", nodeID).First(&node).Error; err == nil {
		return node.ID
	}
	return 0
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
	if err := db.DB.Where("enable = ? AND state IN ? AND (retry_at IS NULL OR retry_at <= ?)", true, []string{"waiting", "running"}, time.Now()).
		Order("id asc").Limit(100).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tasks"})
		return
	}
	filtered := make([]models.Task, 0, len(tasks))
	now := time.Now()
	for _, task := range tasks {
		if task.RetryAt.After(now) {
			continue
		}
		if strings.EqualFold(task.Type, "issue_cert") && nodeID != "" {
			if target := parseIssueTaskTarget(task.Res); target != "" && target != nodeID {
				continue
			}
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
	if strings.EqualFold(task.Type, "issue_cert") {
		if strings.Contains(strings.ToLower(req.Ret), "429") {
			if nid, err := strconv.ParseInt(nodeID, 10, 64); err == nil {
				services.MarkNodeRateLimited(nid, 15*time.Minute)
			}
		}
	}
	progress := updateTaskProgress(task.Progress, nodeID, req.State)
	retLog := appendTaskLog(task.Ret, nodeID, req.State, req.Ret, task.ErrTimes)
	updates := map[string]interface{}{
		"ret":      retLog,
		"progress": progress,
	}
	if req.State == "fail" {
		nextErrTimes := task.ErrTimes + 1
		maxRetries := 3
		retLog = appendTaskLog(retLog, nodeID, "retry", fmt.Sprintf("retry %d/%d", nextErrTimes, maxRetries), nextErrTimes)
		updates["ret"] = retLog
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
	if nextState, ok := updates["state"].(string); ok {
		if nextState != task.State && (nextState == "done" || nextState == "fail") {
			notifyTaskCompletion(task, nextState, req.Ret)
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type taskMeta struct {
	UserID int64 `json:"user_id"`
}

type issueTaskMeta struct {
	TargetNodeID int64 `json:"target_node_id"`
}

func notifyTaskCompletion(task models.Task, state string, ret string) {
	userID := parseTaskUserID(task.Res)
	if userID == 0 {
		return
	}
	phone, email, ok := lookupMessageSubscription(userID, task.Type)
	if !ok {
		return
	}
	title := buildTaskTitle(task.Type, state)
	content := buildTaskContent(task.Type, state, task.Data, ret)

	msg := models.Message{
		Type:          task.Type,
		Receive:       userID,
		Title:         title,
		Content:       content,
		PhoneContent:  content,
		IsShow:        true,
		EmailNeedSend: email,
		PhoneNeedSend: phone,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_ = db.DB.Create(&msg).Error
}

func parseTaskUserID(raw string) int64 {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	var meta taskMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return 0
	}
	return meta.UserID
}

func parseIssueTaskTarget(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var meta issueTaskMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return ""
	}
	if meta.TargetNodeID <= 0 {
		return ""
	}
	return strconv.FormatInt(meta.TargetNodeID, 10)
}

func lookupMessageSubscription(userID int64, msgType string) (bool, bool, bool) {
	if userID == 0 || msgType == "" {
		return false, false, false
	}
	var sub models.MessageSub
	err := db.DB.Where("uid = ? AND msg_type = ?", userID, msgType).First(&sub).Error
	if err == nil {
		return sub.Phone, sub.Email, true
	}
	if !errorsIsRecordNotFound(err) {
		return false, false, false
	}
	var count int64
	if err := db.DB.Model(&models.MessageSub{}).Where("uid = ?", userID).Count(&count).Error; err != nil {
		return false, false, false
	}
	if count == 0 {
		return true, true, true
	}
	return false, false, false
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func buildTaskTitle(taskType string, state string) string {
	label := taskType
	switch taskType {
	case "refresh_url":
		label = "刷新URL"
	case "refresh_dir":
		label = "刷新目录"
	case "preheat":
		label = "预热"
	}
	if state == "fail" {
		return label + "任务失败"
	}
	return label + "任务完成"
}

func buildTaskContent(taskType string, state string, data string, ret string) string {
	result := "执行成功"
	if state == "fail" {
		result = "执行失败"
	}
	if strings.TrimSpace(ret) != "" {
		result = result + "，原因：" + ret
	}
	if strings.TrimSpace(data) == "" {
		return result
	}
	return result + "，URL：" + data
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

type taskLogEntry struct {
	Time    string `json:"time"`
	NodeID  string `json:"node_id"`
	State   string `json:"state"`
	Message string `json:"message"`
	Attempt int    `json:"attempt"`
}

func appendTaskLog(raw string, nodeID string, state string, message string, attempt int) string {
	entry := taskLogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		NodeID:  nodeID,
		State:   state,
		Message: message,
		Attempt: attempt,
	}
	logs := make([]taskLogEntry, 0)
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &logs); err != nil {
			logs = []taskLogEntry{{
				Time:    time.Now().Format("2006-01-02 15:04:05"),
				NodeID:  "",
				State:   "legacy",
				Message: raw,
				Attempt: 0,
			}}
		}
	}
	logs = append(logs, entry)
	b, _ := json.Marshal(logs)
	return string(b)
}
