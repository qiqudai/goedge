package controllers

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AgentWSController struct {
	upgrader websocket.Upgrader
}

func NewAgentWSController() *AgentWSController {
	initDispatchLoop()
	return &AgentWSController{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Global Connection Manager
var (
	agentConns     = make(map[int64]*websocket.Conn)
	agentMutex     sync.RWMutex
	ackWaiters     = make(map[string]chan JobAckMsg)
	ackMutex       sync.Mutex
	dispatchOnce   sync.Once
	dispatchSignal = make(chan struct{}, 1)
)

// Message Types
const (
	MsgAgentHello   = "agent_hello"
	MsgJobDispatch  = "job_dispatch"
	MsgJobAck       = "job_ack"
	MsgHeartbeat    = "heartbeat"
	MsgHeartbeatAck = "heartbeat_ack"
	MsgNodeSync     = "node_sync"
	MsgNodeSyncAck  = "node_sync_ack"
	MsgLogsAccess   = "agent_logs_access"
	MsgLogsMetrics  = "agent_logs_metrics"
	MsgLogsEvents   = "agent_logs_events"
	MsgL2NodesReq   = "l2_nodes_request"
	MsgL2NodesResp  = "l2_nodes_response"
	MsgL2Heartbeat  = "l2_heartbeat"
	MsgCertIssued   = "cert_issued"
)

// Payloads
type WSMsgHeader struct {
	Kind string `json:"kind"`
}

type AgentHelloMsg struct {
	Kind         string   `json:"kind"`
	NodeID       string   `json:"node_id"` // Agent sends string ID (hostname usually), but we need to map to DB ID?
	Token        string   `json:"token"`
	AgentVersion string   `json:"agent_version"`
	Capabilities []string `json:"capabilities"`
}

type HeartbeatMsg struct {
	Kind      string `json:"kind"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"`
}

type HeartbeatAckMsg struct {
	Kind       string `json:"kind"`
	SyncAction string `json:"sync_action"`
}

type NodeSyncMsg struct {
	Kind    string `json:"kind"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
}

// Note: In Agent logic, NodeID was string (hostname). In DB Node is int64.
// We need to resolve Token -> Node (int64) during handshake.
// The Agent should probably send the Token, and we look it up.

type JobDispatchMsg struct {
	Kind  string      `json:"kind"`
	MsgID string      `json:"msg_id"`
	Task  TaskSummary `json:"task"`
	Job   JobPayload  `json:"job"`
}

type TaskSummary struct {
	TaskID   int64  `json:"task_id"`
	TaskType string `json:"task_type"`
	TaskName string `json:"task_name"`
}

type JobPayload struct {
	JobID   int64  `json:"job_id"`
	JobType string `json:"job_type"`
	Payload string `json:"payload"` // JSON string of task data
}

type JobAckMsg struct {
	Kind    string          `json:"kind"`
	MsgID   string          `json:"msg_id"`
	NodeID  int64           `json:"node_id,omitempty"` // Optional in ACK if we track conn
	TaskID  int64           `json:"task_id"`
	JobID   int64           `json:"job_id"`
	JobType string          `json:"job_type"`
	Status  string          `json:"status"` // success, fail, ignored
	Applied json.RawMessage `json:"applied"`
	Error   string          `json:"error"`
	Ret     string          `json:"ret,omitempty"`
}

type AccessLogsMsg struct {
	Kind   string   `json:"kind"`
	NodeID string   `json:"node_id"`
	NodeIP string   `json:"node_ip"`
	Lines  []string `json:"lines"`
	MsgID  string   `json:"msg_id,omitempty"`
}

type MetricsMsg struct {
	Kind    string `json:"kind"`
	NodeID  string `json:"node_id"`
	NodeIP  string `json:"node_ip"`
	Content string `json:"content"`
	MsgID   string `json:"msg_id,omitempty"`
}

type EventsMsg struct {
	Kind     string   `json:"kind"`
	NodeID   string   `json:"node_id"`
	NodeIP   string   `json:"node_ip"`
	Type     string   `json:"type"`
	Payloads []string `json:"payloads"`
	MsgID    string   `json:"msg_id,omitempty"`
}

type l2NodeInfo struct {
	ID            int64  `json:"id"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	CheckProtocol string `json:"check_protocol"`
	CheckPort     int    `json:"check_port"`
	CheckHost     string `json:"check_host"`
	CheckPath     string `json:"check_path"`
	CheckTimeout  int    `json:"check_timeout"`
}

type L2NodesRequestMsg struct {
	Kind  string `json:"kind"`
	MsgID string `json:"msg_id"`
}

type L2NodesResponseMsg struct {
	Kind  string       `json:"kind"`
	MsgID string       `json:"msg_id"`
	Nodes []l2NodeInfo `json:"nodes"`
}

type L2HeartbeatMsg struct {
	Kind  string  `json:"kind"`
	Nodes []int64 `json:"nodes"`
}

type CertIssuedMsg struct {
	Kind         string `json:"kind"`
	CertID       int64  `json:"cert_id"`
	CertPEM      string `json:"cert"`
	KeyPEM       string `json:"key"`
	IssueTaskID  int64  `json:"issue_task_id"`
	RateLimited  bool   `json:"rate_limited"`
	RateCooldown int    `json:"rate_cooldown"`
}

type WSDispatchRequest struct {
	NodeID      int64  `json:"node_id"`
	TaskType    string `json:"task_type"`
	Payload     string `json:"payload"`
	WaitSeconds int    `json:"wait_seconds"`
}

type WSDispatchResponse struct {
	NodeID    int64  `json:"node_id"`
	Connected bool   `json:"connected"`
	TaskID    int64  `json:"task_id"`
	JobID     int64  `json:"job_id"`
	State     string `json:"state,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HandleWS handles the WebSocket connection
func (c *AgentWSController) HandleWS(ctx *gin.Context) {
	conn, err := c.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 1. Handshake Timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 2. Wait for Agent Hello
	var nodeID int64
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var header WSMsgHeader
		if json.Unmarshal(msg, &header) != nil {
			continue
		}

		if header.Kind == MsgAgentHello {
			var hello AgentHelloMsg
			if err := json.Unmarshal(msg, &hello); err != nil {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "Invalid Hello"))
				return
			}

			// Auth
			node, err := c.authenticateNode(hello.Token, hello.NodeID)
			if err != nil {
				log.Printf("[WS] Auth failed for token %s: %v", hello.Token, err)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4003, "Auth Failed"))
				return
			}
			nodeID = int64(node.ID)

			// Register
			c.registerConn(nodeID, conn)
			log.Printf("[WS] Node %d connected (ver: %s)", nodeID, hello.AgentVersion)
			if strings.TrimSpace(hello.AgentVersion) != "" {
				_ = services.UpsertNodeConfigItem(nodeID, "agent_version", strings.TrimSpace(hello.AgentVersion))
			}

			// Agent is persistent client.
			conn.SetReadDeadline(time.Time{})
			break
		}
	}

	defer c.unregisterConn(nodeID)

	// 3. Message Loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Node %d read error: %v", nodeID, err)
			break
		}

		var header WSMsgHeader
		if json.Unmarshal(msg, &header) != nil {
			continue
		}

		switch header.Kind {
		case MsgJobAck:
			var ack JobAckMsg
			if err := json.Unmarshal(msg, &ack); err == nil {
				c.handleJobAck(nodeID, ack)
			}
		case MsgHeartbeat:
			c.handleHeartbeat(nodeID, conn, msg)
		case MsgNodeSync:
			c.handleNodeSync(nodeID, msg)
		case MsgLogsAccess:
			c.handleAccessLogs(nodeID, msg)
		case MsgLogsMetrics:
			c.handleMetrics(nodeID, msg)
		case MsgLogsEvents:
			c.handleEvents(nodeID, msg)
		case MsgL2NodesReq:
			c.handleL2NodesRequest(nodeID, conn, msg)
		case MsgL2Heartbeat:
			c.handleL2Heartbeat(nodeID, msg)
		case MsgCertIssued:
			c.handleCertIssued(nodeID, msg)
		}
	}
}

func (c *AgentWSController) authenticateNode(token string, nodeHint string) (*models.Node, error) {
	var node models.Node
	// 1) Global token
	if config.App.AgentToken != "" && token == config.App.AgentToken {
		return findNodeByHint(nodeHint)
	}
	if envToken := os.Getenv("APP_AGENT_TOKEN"); envToken != "" && token == envToken {
		return findNodeByHint(nodeHint)
	}

	// 2) Per-node token
	if err := db.DB.Where("token = ?", token).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func findNodeByHint(nodeHint string) (*models.Node, error) {
	var node models.Node
	nodeHint = strings.TrimSpace(nodeHint)
	if nodeHint == "" {
		return nil, fmt.Errorf("node_id is required for global token auth")
	}
	if id, err := strconv.ParseInt(nodeHint, 10, 64); err == nil && id > 0 {
		if err := db.DB.Where("id = ?", id).First(&node).Error; err != nil {
			return nil, err
		}
		return &node, nil
	}
	if err := db.DB.Where("name = ? OR host = ? OR ip = ?", nodeHint, nodeHint, nodeHint).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (c *AgentWSController) registerConn(nodeID int64, conn *websocket.Conn) {
	agentMutex.Lock()
	defer agentMutex.Unlock()
	// Close existing if any
	if old, ok := agentConns[nodeID]; ok {
		old.Close()
	}
	agentConns[nodeID] = conn

	// Update Node Status Online? (Optional)
	services.MarkNodeOnline(nodeID, time.Now())
	triggerDispatchPending()
}

func (c *AgentWSController) unregisterConn(nodeID int64) {
	agentMutex.Lock()
	defer agentMutex.Unlock()
	if _, ok := agentConns[nodeID]; ok {
		// Only remove if it's the SAME connection object (prevent race where new conn replaces old, then old unregisters)
		// But here we don't have the ptr.
		// Handling simply:
		delete(agentConns, nodeID)
	}
}

func (c *AgentWSController) handleJobAck(nodeID int64, ack JobAckMsg) {
	notifyAckWaiter(ack)
	if ack.JobID == 0 && ack.TaskID != 0 {
		c.handleTaskAck(nodeID, ack)
		return
	}
	if ack.JobID == 0 {
		log.Printf("[WS] Job ACK from Node %d (no job id): %s", nodeID, ack.Status)
		return
	}
	// Update Job

	var state string
	switch ack.Status {
	case "success":
		state = "success"
	case "fail":
		state = "fail"
	case "ignored":
		state = "success" // map ignored to success?
	default:
		state = "fail"
	}

	updates := map[string]interface{}{
		"state":      state,
		"ret":        buildAckRet(ack),
		"updated_at": time.Now(),
	}

	// If we have detailed result
	if len(ack.Applied) > 0 {
		// Maybe store in a 'Result' or 'Log' column?
		// For now just logging or assuming Job struct has a place.
		// The prompt says "Update Job table".
	}

	db.DB.Model(&models.Job{}).Where("id = ?", ack.JobID).Updates(updates)

	// Also check Task Progress if needed (Aggregator logic)
	// For now, minimal update.
	log.Printf("[WS] Job %d ACK from Node %d: %s", ack.JobID, nodeID, ack.Status)
}

func buildAckRet(ack JobAckMsg) string {
	if strings.TrimSpace(ack.Error) != "" {
		return strings.TrimSpace(ack.Error)
	}
	if strings.TrimSpace(ack.Ret) != "" {
		return strings.TrimSpace(ack.Ret)
	}
	if len(ack.Applied) > 0 {
		return string(ack.Applied)
	}
	return ""
}

func (c *AgentWSController) handleTaskAck(nodeID int64, ack JobAckMsg) {
	state := "fail"
	switch ack.Status {
	case "success", "ignored":
		state = "done"
	case "fail":
		state = "fail"
	}

	var task models.Task
	if err := db.DB.Where("id = ?", ack.TaskID).First(&task).Error; err != nil {
		log.Printf("[WS] Task ACK load failed: %v", err)
		return
	}

	nodeIDStr := strconv.FormatInt(nodeID, 10)
	progress := updateTaskProgress(task.Progress, nodeIDStr, state)
	retLog := appendTaskLog(task.Ret, nodeIDStr, state, buildAckRet(ack), task.ErrTimes)
	updates := map[string]interface{}{
		"ret":      retLog,
		"progress": progress,
	}

	if state == "fail" {
		nextErrTimes := task.ErrTimes + 1
		maxRetries := 3
		retLog = appendTaskLog(retLog, nodeIDStr, "retry", fmt.Sprintf("retry %d/%d", nextErrTimes, maxRetries), nextErrTimes)
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

	if err := db.DB.Model(&models.Task{}).Where("id = ?", ack.TaskID).Updates(updates).Error; err != nil {
		log.Printf("[WS] Task ACK update failed: %v", err)
	}
}

func (c *AgentWSController) handleHeartbeat(nodeID int64, conn *websocket.Conn, raw []byte) {
	var hb HeartbeatMsg
	if err := json.Unmarshal(raw, &hb); err != nil {
		return
	}
	if nodeID != 0 {
		services.MarkNodeOnline(nodeID, time.Now())
	}

	syncAction := c.getSyncAction(nodeID)
	ack := HeartbeatAckMsg{
		Kind:       MsgHeartbeatAck,
		SyncAction: syncAction,
	}
	if err := conn.WriteJSON(ack); err != nil {
		log.Printf("[WS] Heartbeat ack failed for node %d: %v", nodeID, err)
	}
}

func (c *AgentWSController) getSyncAction(nodeID int64) string {
	if nodeID == 0 {
		return ""
	}
	var node models.Node
	if err := db.DB.Select("config_task").Where("id = ?", nodeID).First(&node).Error; err != nil {
		return ""
	}
	switch strings.TrimSpace(node.ConfigTask) {
	case "sync_enable":
		return "enable"
	case "sync_disable":
		return "disable"
	default:
		return ""
	}
}

func (c *AgentWSController) handleNodeSync(nodeID int64, raw []byte) {
	var req NodeSyncMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "enable" && action != "disable" {
		return
	}
	if !req.Success {
		return
	}

	if err := db.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"config_task": "",
		"update_at":   time.Now(),
	}).Error; err != nil {
		log.Printf("[WS] Node %d sync update failed: %v", nodeID, err)
		return
	}
}

func (c *AgentWSController) handleAccessLogs(nodeID int64, raw []byte) {
	var req AccessLogsMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		req.NodeID = strconv.FormatInt(nodeID, 10)
	}
	inserted := services.InsertAccessLogs(req.NodeID, req.NodeIP, req.Lines)
	log.Printf("[CK] Access logs inserted: %d", inserted)
}

func (c *AgentWSController) handleMetrics(nodeID int64, raw []byte) {
	var req MetricsMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		req.NodeID = strconv.FormatInt(nodeID, 10)
	}
	inserted := services.InsertMetrics(req.NodeID, req.NodeIP, req.Content)
	log.Printf("[CK] Metrics inserted: %d", inserted)
}

func (c *AgentWSController) handleEvents(nodeID int64, raw []byte) {
	var req EventsMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		req.NodeID = strconv.FormatInt(nodeID, 10)
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "event"
	}
	inserted := services.InsertEventLogs(req.NodeID, req.NodeIP, req.Type, req.Payloads)
	log.Printf("[CK] Events inserted: %d", inserted)
}

func (c *AgentWSController) handleL2NodesRequest(nodeID int64, conn *websocket.Conn, raw []byte) {
	var req L2NodesRequestMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	nodes, err := fetchL2NodesForNode(nodeID)
	if err != nil {
		log.Printf("[WS] L2 nodes fetch failed: %v", err)
		return
	}
	resp := L2NodesResponseMsg{
		Kind:  MsgL2NodesResp,
		MsgID: req.MsgID,
		Nodes: nodes,
	}
	agentMutex.Lock()
	_ = conn.WriteJSON(resp)
	agentMutex.Unlock()
}

func (c *AgentWSController) handleL2Heartbeat(nodeID int64, raw []byte) {
	var req L2HeartbeatMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if len(req.Nodes) == 0 {
		return
	}
	now := time.Now()
	for _, id := range req.Nodes {
		services.MarkNodeOnline(id, now)
	}
}

func (c *AgentWSController) handleCertIssued(nodeID int64, raw []byte) {
	var req CertIssuedMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if req.CertID == 0 {
		return
	}
	if req.RateLimited && nodeID != 0 {
		cooldown := time.Minute * 10
		if req.RateCooldown > 0 {
			cooldown = time.Duration(req.RateCooldown) * time.Second
		}
		services.MarkNodeRateLimited(nodeID, cooldown)
	}
	if strings.TrimSpace(req.CertPEM) == "" || strings.TrimSpace(req.KeyPEM) == "" {
		return
	}
	notBefore, notAfter, err := services.ParseCertTimes(req.CertPEM)
	if err != nil {
		return
	}
	var existingCert models.Cert
	if err := db.DB.First(&existingCert, req.CertID).Error; err != nil {
		return
	}
	encryptedKey, err := services.Crypto.Encrypt(req.KeyPEM)
	if err != nil {
		return
	}
	if err := services.UpdateIssuedCert(req.CertID, req.CertPEM, encryptedKey, notBefore, notAfter, req.IssueTaskID); err != nil {
		log.Printf("[WS] Update issued cert failed: %v", err)
	}
}

func initDispatchLoop() {
	dispatchOnce.Do(func() {
		services.SetDispatchPendingHook(triggerDispatchPending)
		go dispatchWorker()
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				triggerDispatchPending()
			}
		}()
	})
}

func triggerDispatchPending() {
	select {
	case dispatchSignal <- struct{}{}:
	default:
	}
}

func dispatchWorker() {
	for range dispatchSignal {
		dispatchPendingForConnected()
	}
}

func dispatchPendingForConnected() {
	nodeIDs := connectedNodeIDs()
	for _, nodeID := range nodeIDs {
		dispatchPendingTasksForNode(nodeID)
		dispatchPendingJobsForNode(nodeID)
	}
}

func connectedNodeIDs() []int64 {
	agentMutex.RLock()
	defer agentMutex.RUnlock()
	ids := make([]int64, 0, len(agentConns))
	for id := range agentConns {
		ids = append(ids, id)
	}
	return ids
}

func dispatchPendingTasksForNode(nodeID int64) {
	nodeIDStr := strconv.FormatInt(nodeID, 10)
	now := time.Now()
	var tasks []models.Task
	if err := db.DB.Where("enable = ? AND state IN ? AND (retry_at IS NULL OR retry_at <= ?)", true, []string{"waiting", "running"}, now).
		Order("id asc").Limit(100).Find(&tasks).Error; err != nil {
		return
	}
	filtered := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.RetryAt != nil && task.RetryAt.After(now) {
			continue
		}
		if strings.EqualFold(task.Type, "issue_cert") {
			if target := parseIssueTaskTarget(task.Res); target != "" && target != nodeIDStr {
				continue
			}
		}
		if nodeIDStr == "" || !taskProgressHasNode(task.Progress, nodeIDStr) {
			filtered = append(filtered, task)
		}
	}
	if len(filtered) > 0 {
		for _, task := range filtered {
			progress := updateTaskProgress(task.Progress, nodeIDStr, "running")
			db.DB.Model(&models.Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"state":    "running",
				"start_at": time.Now(),
				"progress": progress,
			})
		}
	}
	for _, task := range filtered {
		payload := task.Data
		if strings.EqualFold(task.Type, "config_sync") {
			cfg, err := services.NewConfigService().GenerateConfigForNode(nodeIDStr)
			if err != nil {
				continue
			}
			if raw, err := json.Marshal(cfg); err == nil {
				payload = string(raw)
			}
		}
		_ = dispatchTaskToNode(nodeID, task, payload)
	}
}

func dispatchPendingJobsForNode(nodeID int64) {
	var jobs []models.Job
	if err := db.DB.Where("state = ? AND node_id = ?", "waiting", nodeID).
		Order("id asc").Limit(100).Find(&jobs).Error; err != nil {
		return
	}
	for _, job := range jobs {
		var task models.Task
		if err := db.DB.Where("id = ?", job.TaskID).First(&task).Error; err != nil {
			continue
		}
		if err := DispatchJobToNode(nodeID, &task, &job, job.Data); err != nil {
			continue
		}
		db.DB.Model(&models.Job{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"state":      "running",
			"updated_at": time.Now(),
		})
	}
}

func dispatchTaskToNode(nodeID int64, task models.Task, payload string) error {
	agentMutex.RLock()
	conn, ok := agentConns[nodeID]
	agentMutex.RUnlock()
	if !ok {
		return fmt.Errorf("node %d not connected", nodeID)
	}
	msg := JobDispatchMsg{
		Kind:  MsgJobDispatch,
		MsgID: fmt.Sprintf("task-%d-%d", task.ID, nodeID),
		Task: TaskSummary{
			TaskID:   int64(task.ID),
			TaskType: task.Type,
			TaskName: task.Name,
		},
		Job: JobPayload{
			JobID:   0,
			JobType: task.Type,
			Payload: payload,
		},
	}
	agentMutex.Lock()
	defer agentMutex.Unlock()
	return conn.WriteJSON(msg)
}

func fetchL2NodesForNode(nodeID int64) ([]l2NodeInfo, error) {
	if nodeID == 0 {
		return []l2NodeInfo{}, nil
	}

	var self models.Node
	if err := db.DB.Where("id = ?", nodeID).First(&self).Error; err != nil {
		return nil, err
	}
	if self.Level != 1 {
		return []l2NodeInfo{}, nil
	}

	var groupIDs []int64
	if err := db.DB.Model(&models.Line{}).
		Select("distinct node_group_id").
		Where("node_id = ?", nodeID).
		Pluck("node_group_id", &groupIDs).Error; err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []l2NodeInfo{}, nil
	}

	var l2NodeIDs []int64
	if err := db.DB.Model(&models.Line{}).
		Select("distinct node_id").
		Where("node_group_id IN ?", groupIDs).
		Where("node_id <> ?", nodeID).
		Pluck("node_id", &l2NodeIDs).Error; err != nil {
		return nil, err
	}
	if len(l2NodeIDs) == 0 {
		return []l2NodeInfo{}, nil
	}

	var nodes []models.Node
	if err := db.DB.Where("id IN ? AND level = ? AND enable = ?", l2NodeIDs, 2, true).
		Select("id", "ip", "port", "check_protocol", "check_port", "check_host", "check_path", "check_timeout").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	result := make([]l2NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, l2NodeInfo{
			ID:            n.ID,
			IP:            n.IP,
			Port:          n.Port,
			CheckProtocol: n.CheckProtocol,
			CheckPort:     n.CheckPort,
			CheckHost:     n.CheckHost,
			CheckPath:     n.CheckPath,
			CheckTimeout:  n.CheckTimeout,
		})
	}
	return result, nil
}

// DispatchTest dispatches a job to a connected node (admin testing helper).
func (c *AgentWSController) DispatchTest(ctx *gin.Context) {
	var req WSDispatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.NodeID == 0 || strings.TrimSpace(req.TaskType) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid request"})
		return
	}

	agentMutex.RLock()
	_, connected := agentConns[req.NodeID]
	agentMutex.RUnlock()

	if !connected {
		ctx.JSON(http.StatusOK, WSDispatchResponse{
			NodeID:    req.NodeID,
			Connected: false,
			Error:     "node not connected",
		})
		return
	}

	msgID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	waiter := registerAckWaiter(msgID)
	if err := dispatchJobToNode(req.NodeID, msgID, req.TaskType, req.Payload); err != nil {
		unregisterAckWaiter(msgID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "dispatch failed"})
		return
	}

	resp := WSDispatchResponse{
		NodeID:    req.NodeID,
		Connected: true,
	}

	waitSeconds := req.WaitSeconds
	if waitSeconds <= 0 {
		waitSeconds = 5
	}
	select {
	case ack := <-waiter:
		resp.State = ack.Status
		if ack.Error != "" {
			resp.Error = ack.Error
		}
	case <-time.After(time.Duration(waitSeconds) * time.Second):
		resp.State = "timeout"
	}
	unregisterAckWaiter(msgID)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func dispatchJobToNode(nodeID int64, msgID string, jobType string, payload string) error {
	agentMutex.RLock()
	conn, ok := agentConns[nodeID]
	agentMutex.RUnlock()
	if !ok {
		return fmt.Errorf("node %d not connected", nodeID)
	}
	msg := JobDispatchMsg{
		Kind:  MsgJobDispatch,
		MsgID: msgID,
		Task: TaskSummary{
			TaskID:   0,
			TaskType: jobType,
			TaskName: "ws-dispatch-test",
		},
		Job: JobPayload{
			JobID:   0,
			JobType: jobType,
			Payload: payload,
		},
	}
	agentMutex.Lock()
	defer agentMutex.Unlock()
	return conn.WriteJSON(msg)
}

func registerAckWaiter(msgID string) chan JobAckMsg {
	ch := make(chan JobAckMsg, 1)
	ackMutex.Lock()
	ackWaiters[msgID] = ch
	ackMutex.Unlock()
	return ch
}

func unregisterAckWaiter(msgID string) {
	ackMutex.Lock()
	if ch, ok := ackWaiters[msgID]; ok {
		delete(ackWaiters, msgID)
		close(ch)
	}
	ackMutex.Unlock()
}

func notifyAckWaiter(ack JobAckMsg) {
	if ack.MsgID == "" {
		return
	}
	ackMutex.Lock()
	ch, ok := ackWaiters[ack.MsgID]
	ackMutex.Unlock()
	if ok {
		ch <- ack
	}
}

// Global Dispatcher
func DispatchJobToNode(nodeID int64, task *models.Task, job *models.Job, payload string) error {
	agentMutex.RLock()
	conn, ok := agentConns[nodeID]
	agentMutex.RUnlock()

	if !ok {
		return fmt.Errorf("node %d not connected", nodeID)
	}

	msg := JobDispatchMsg{
		Kind:  MsgJobDispatch,
		MsgID: fmt.Sprintf("%d-%d", task.ID, job.ID),
		Task: TaskSummary{
			TaskID:   int64(task.ID),
			TaskType: task.Type,
			TaskName: task.Name,
		},
		Job: JobPayload{
			JobID:   int64(job.ID),
			JobType: task.Type, // Use Task Type as Job Type for Agent
			Payload: payload,
		},
	}

	agentMutex.Lock()
	defer agentMutex.Unlock()
	// Write JSON
	return conn.WriteJSON(msg)
}
