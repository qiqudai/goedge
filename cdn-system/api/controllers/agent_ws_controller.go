package controllers

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"crypto/md5"
	"encoding/hex"
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
	agentConns      = make(map[int64]*websocket.Conn)
	agentMutex      sync.RWMutex
	ackWaiters      = make(map[string]chan TaskAckMsg)
	ackMutex        sync.Mutex
	taskUpdateLocks sync.Map
	heartbeatReport sync.Map
	dispatchOnce    sync.Once
	dispatchSignal  = make(chan struct{}, 1)
	dispatchPool    sync.Once
	dispatchQueue   chan dispatchRequest
)

// Message Types
const (
	MsgAgentHello   = "agent_hello"
	MsgTaskDispatch = "task_dispatch"
	MsgTaskAck      = "task_ack"
	MsgHeartbeat    = "heartbeat"
	MsgHeartbeatAck = "heartbeat_ack"
	MsgNodeSync     = "node_sync"
	MsgNodeSyncAck  = "node_sync_ack"
	MsgLogsAccess   = "agent_logs_access"
	MsgLogsStream   = "agent_logs_stream"
	MsgLogsMetrics  = "agent_logs_metrics"
	MsgLogsEvents   = "agent_logs_events"
	MsgL2NodesReq   = "l2_nodes_request"
	MsgL2NodesResp  = "l2_nodes_response"
	MsgL2Heartbeat  = "l2_heartbeat"
	MsgCertIssued   = "cert_issued"
)

const dispatchWorkerCount = 10

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
	Kind           string                 `json:"kind"`
	Timestamp      int64                  `json:"timestamp"`
	Status         string                 `json:"status"`
	ReportedConfig map[string]interface{} `json:"reported_config"`
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

type TaskDispatchMsg struct {
	Kind  string      `json:"kind"`
	MsgID string      `json:"msg_id"`
	Task  TaskPayload `json:"task"`
}

type TaskPayload struct {
	TaskID   int64  `json:"task_id"`
	TaskType string `json:"task_type"`
	TaskName string `json:"task_name"`
	Payload  string `json:"payload,omitempty"`
}

type TaskAckMsg struct {
	Kind     string          `json:"kind"`
	MsgID    string          `json:"msg_id"`
	NodeID   int64           `json:"node_id,omitempty"` // Optional in ACK if we track conn
	TaskID   int64           `json:"task_id"`
	TaskType string          `json:"task_type,omitempty"`
	Status   string          `json:"status"` // success, fail, ignored
	Applied  json.RawMessage `json:"applied"`
	Error    string          `json:"error"`
	Ret      string          `json:"ret,omitempty"`
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
		case MsgTaskAck:
			var ack TaskAckMsg
			if err := json.Unmarshal(msg, &ack); err == nil {
				c.handleTaskAck(nodeID, ack)
			}
		case MsgHeartbeat:
			c.handleHeartbeat(nodeID, conn, msg)
		case MsgNodeSync:
			c.handleNodeSync(nodeID, msg)
		case MsgLogsAccess:
			c.handleAccessLogs(nodeID, msg)
		case MsgLogsStream:
			c.handleStreamLogs(nodeID, msg)
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

func (c *AgentWSController) handleTaskAck(nodeID int64, ack TaskAckMsg) {
	notifyAckWaiter(ack)
	if ack.TaskID == 0 {
		log.Printf("[WS] Task ACK from Node %d (no task id): %s", nodeID, ack.Status)
		return
	}

	var task models.Task
	if err := db.DB.Where("id = ?", ack.TaskID).First(&task).Error; err != nil {
		log.Printf("[WS] Task ACK load failed: %v", err)
		return
	}

	if strings.TrimSpace(task.TargetsJSON) != "" {
		c.handleTargetsTaskAck(nodeID, ack, task)
		return
	}
	c.handleProgressTaskAck(nodeID, ack, task)
}

func buildAckRet(ack TaskAckMsg) string {
	errText := strings.TrimSpace(ack.Error)
	retText := strings.TrimSpace(ack.Ret)
	if errText != "" && retText != "" {
		return errText + "\n" + retText
	}
	if errText != "" {
		return errText
	}
	if retText != "" {
		return retText
	}
	if len(ack.Applied) > 0 {
		return string(ack.Applied)
	}
	return ""
}

type taskProgressPayload struct {
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

func parseTaskProgress(ack TaskAckMsg) (int, string) {
	var payload taskProgressPayload
	if strings.TrimSpace(ack.Ret) != "" {
		if json.Unmarshal([]byte(ack.Ret), &payload) == nil {
			return payload.Progress, strings.TrimSpace(payload.Message)
		}
	}
	if len(ack.Applied) > 0 {
		if json.Unmarshal(ack.Applied, &payload) == nil {
			return payload.Progress, strings.TrimSpace(payload.Message)
		}
	}
	return 0, ""
}

func (c *AgentWSController) handleProgressTaskAck(nodeID int64, ack TaskAckMsg, task models.Task) {
	state := "fail"
	switch ack.Status {
	case "success", "ignored":
		state = "done"
	case "fail":
		state = "fail"
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
	if strings.EqualFold(task.Type, "issue_cert") && state == "fail" {
		services.MarkIssueTaskFailed(task.ID, buildAckRet(ack))
	}
	if strings.EqualFold(task.Type, services.TaskTypeHTTPSProbe) && (updates["state"] == "done" || updates["state"] == "fail") {
		services.HandleHTTPSProbeTaskFinished(task.ID)
	}
}

func (c *AgentWSController) handleTargetsTaskAck(nodeID int64, ack TaskAckMsg, task models.Task) {
	withTaskUpdateLock(task.ID, func() {
		var current models.Task
		if err := db.DB.Select("id", "targets_json", "ret", "start_at").Where("id = ?", task.ID).First(&current).Error; err != nil {
			log.Printf("[WS] Task ACK reload failed: %v", err)
			return
		}
		nodeIDStr := strconv.FormatInt(nodeID, 10)
		now := time.Now()
		targets := services.ParseTaskTargets(current.TargetsJSON)

		retMessage := buildAckRet(ack)
		state := "fail"
		switch ack.Status {
		case "progress":
			state = "running"
		case "success", "ignored":
			state = "done"
		case "fail":
			state = "fail"
		}

		retLog := current.Ret
		var attempt int
		if ack.Status == "progress" {
			progress, message := parseTaskProgress(ack)
			target := targets.EnsureNode(nodeIDStr)
			target.State = services.TargetStateRunning
			if progress > 0 {
				target.Progress = progress
			}
			if message != "" {
				target.Message = message
			}
			if strings.TrimSpace(retMessage) != "" {
				target.Ret = retMessage
			}
			target.LastAt = now.Unix()
			updates := map[string]interface{}{
				"targets_json": targets.Marshal(),
				"state":        "running",
			}
			if current.StartAt == nil {
				updates["start_at"] = now
			}
			if err := db.DB.Model(&models.Task{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
				log.Printf("[WS] Task progress update failed: %v", err)
			}
			return
		}
		if state == "done" {
			attempt = targets.MarkSuccess(nodeIDStr, retMessage, now)
			retLog = appendTaskLog(retLog, nodeIDStr, state, retMessage, attempt)
		} else {
			finalFail, retryAt, attemptCount := targets.MarkFailure(nodeIDStr, retMessage, now, 3)
			attempt = attemptCount
			retLog = appendTaskLog(retLog, nodeIDStr, "fail", retMessage, attempt)
			if finalFail {
				retLog = appendTaskLog(retLog, nodeIDStr, "failed_final", "max retries reached", attempt)
			} else {
				retLog = appendTaskLog(retLog, nodeIDStr, "retry", fmt.Sprintf("retry at %s", retryAt.Format("2006-01-02 15:04:05")), attempt)
			}
		}

		nextState := deriveTargetsState(targets)
		updates := map[string]interface{}{
			"targets_json": targets.Marshal(),
			"ret":          retLog,
			"state":        nextState,
		}
		if current.StartAt == nil && nextState == "running" {
			updates["start_at"] = now
		}
		if nextState == "done" || nextState == "fail" {
			updates["end_at"] = now
		}

		if err := db.DB.Model(&models.Task{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
			log.Printf("[WS] Task target ACK update failed: %v", err)
		}
		if strings.EqualFold(task.Type, services.TaskTypeHTTPSProbe) && (nextState == "done" || nextState == "fail") {
			services.HandleHTTPSProbeTaskFinished(task.ID)
		}
	})
}

func deriveTargetsState(targets *services.TaskTargets) string {
	if targets == nil || len(targets.Nodes) == 0 {
		return "done"
	}
	hasPending := false
	hasRunning := false
	hasFailedFinal := false
	for _, target := range targets.Nodes {
		switch target.State {
		case services.TargetStateSuccess:
			continue
		case services.TargetStateFailedFinal:
			hasFailedFinal = true
		default:
			hasPending = true
			if target.State == services.TargetStateRunning {
				hasRunning = true
			}
		}
	}
	if hasPending {
		if hasRunning {
			return "running"
		}
		return "waiting"
	}
	if hasFailedFinal {
		return "fail"
	}
	return "done"
}

func (c *AgentWSController) handleHeartbeat(nodeID int64, conn *websocket.Conn, raw []byte) {
	var hb HeartbeatMsg
	if err := json.Unmarshal(raw, &hb); err != nil {
		return
	}
	if nodeID != 0 {
		services.MarkNodeOnline(nodeID, time.Now())
		if err := upsertHeartbeatReportedConfig(nodeID, hb.ReportedConfig); err != nil {
			log.Printf("[WS] Persist reported_config failed for node %d: %v", nodeID, err)
		}
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

func upsertHeartbeatReportedConfig(nodeID int64, reported map[string]interface{}) error {
	if nodeID == 0 || len(reported) == 0 {
		return nil
	}
	raw, err := json.Marshal(reported)
	if err != nil {
		return err
	}
	sum := md5.Sum(raw)
	digest := hex.EncodeToString(sum[:])
	if prev, ok := heartbeatReport.Load(nodeID); ok {
		if prevDigest, ok := prev.(string); ok && prevDigest == digest {
			return nil
		}
	}
	if err := services.UpsertNodeConfigItem(nodeID, "reported_config", string(raw)); err != nil {
		return err
	}
	heartbeatReport.Store(nodeID, digest)
	return nil
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
	services.WriteNodeMonitorLog(nodeID, "sync", req.Success, "")
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

func (c *AgentWSController) handleStreamLogs(nodeID int64, raw []byte) {
	var req AccessLogsMsg
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		req.NodeID = strconv.FormatInt(nodeID, 10)
	}
	inserted := services.InsertStreamLogs(req.NodeID, req.NodeIP, req.Lines)
	log.Printf("[CK] Stream logs inserted: %d", inserted)
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
	services.WriteNodeMonitorLogs(req.Nodes, "l2_beat", true, nil)
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
	keyToStore := req.KeyPEM
	if encryptedKey, err := services.Crypto.Encrypt(req.KeyPEM); err == nil {
		keyToStore = encryptedKey
	} else {
		log.Printf("[WS] Encrypt key failed, storing plaintext: %v", err)
	}
	if err := services.UpdateIssuedCert(req.CertID, req.CertPEM, keyToStore, notBefore, notAfter, req.IssueTaskID); err != nil {
		log.Printf("[WS] Update issued cert failed: %v", err)
	}
}

func initDispatchLoop() {
	dispatchOnce.Do(func() {
		initDispatchPool()
		services.SetDispatchPendingHook(triggerDispatchPending)
		services.SetConnectedNodeProvider(connectedNodeIDs)
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
		dispatchPendingTargetsForNode(nodeID)
		dispatchPendingTasksForNode(nodeID)
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

func dispatchPendingTargetsForNode(nodeID int64) {
	nodeIDStr := strconv.FormatInt(nodeID, 10)
	now := time.Now()
	var tasks []models.Task
	if err := db.DB.Where("enable = ? AND state IN ? AND targets_json IS NOT NULL AND targets_json <> ''", true, []string{"waiting", "running"}).
		Order("id asc").Limit(100).Find(&tasks).Error; err != nil {
		return
	}
	for _, task := range tasks {
		taskID := task.ID
		var req *dispatchRequest
		withTaskUpdateLock(taskID, func() {
			var current models.Task
			if err := db.DB.Select("id", "type", "name", "data", "targets_json", "ret", "start_at").Where("id = ?", taskID).First(&current).Error; err != nil {
				return
			}
			targets := services.ParseTaskTargets(current.TargetsJSON)
			if target := targets.Nodes[nodeIDStr]; target != nil && target.State == services.TargetStateRunning && target.LastAt > 0 {
				lastAt := time.Unix(target.LastAt, 0)
				if now.Sub(lastAt) > 30*time.Second {
					retLog := appendTaskLog(current.Ret, nodeIDStr, "timeout", "ack timeout", target.Tries)
					finalFail, retryAt, _ := targets.MarkFailure(nodeIDStr, "ack timeout", now, 3)
					if finalFail {
						retLog = appendTaskLog(retLog, nodeIDStr, "failed_final", "max retries reached", target.Tries)
					} else {
						retLog = appendTaskLog(retLog, nodeIDStr, "retry", fmt.Sprintf("retry at %s", retryAt.Format("2006-01-02 15:04:05")), target.Tries)
					}
					nextState := deriveTargetsState(targets)
					timeoutUpdates := map[string]interface{}{
						"targets_json": targets.Marshal(),
						"ret":          retLog,
						"state":        nextState,
					}
					if nextState == "done" || nextState == "fail" {
						timeoutUpdates["end_at"] = now
					}
					if err := db.DB.Model(&models.Task{}).Where("id = ?", current.ID).Updates(timeoutUpdates).Error; err != nil {
						log.Printf("[WS] Task timeout update failed: %v", err)
					}
					return
				}
			}
			if !targets.ShouldDispatch(nodeIDStr, now) {
				return
			}
			targets.MarkRunning(nodeIDStr, now)
			updates := map[string]interface{}{
				"targets_json": targets.Marshal(),
				"state":        "running",
			}
			if current.StartAt == nil {
				updates["start_at"] = now
			}
			if err := db.DB.Model(&models.Task{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
				return
			}

			payload := current.Data
			if strings.EqualFold(current.Type, "config_sync") {
				cfg, err := services.NewConfigService().GenerateConfigForNode(nodeIDStr)
				if err != nil {
					return
				}
				if raw, err := json.Marshal(cfg); err == nil {
					payload = string(raw)
				}
			}

			taskCopy := current
			req = &dispatchRequest{
				nodeID:  nodeID,
				task:    taskCopy,
				payload: payload,
				onError: func(err error) {
					withTaskUpdateLock(taskCopy.ID, func() {
						var latest models.Task
						if err := db.DB.Select("id", "targets_json", "ret").Where("id = ?", taskCopy.ID).First(&latest).Error; err != nil {
							log.Printf("[WS] Task reload failed: %v", err)
							return
						}
						latestTargets := services.ParseTaskTargets(latest.TargetsJSON)
						attempt := 0
						if target := latestTargets.Nodes[nodeIDStr]; target != nil {
							attempt = target.Tries
						}
						retLog := appendTaskLog(latest.Ret, nodeIDStr, "fail", err.Error(), attempt)
						finalFail, retryAt, _ := latestTargets.MarkFailure(nodeIDStr, err.Error(), time.Now(), 3)
						if finalFail {
							retLog = appendTaskLog(retLog, nodeIDStr, "failed_final", "max retries reached", attempt)
						} else {
							retLog = appendTaskLog(retLog, nodeIDStr, "retry", fmt.Sprintf("retry at %s", retryAt.Format("2006-01-02 15:04:05")), attempt)
						}
						nextState := deriveTargetsState(latestTargets)
						failureUpdates := map[string]interface{}{
							"targets_json": latestTargets.Marshal(),
							"ret":          retLog,
							"state":        nextState,
						}
						if nextState == "done" || nextState == "fail" {
							failureUpdates["end_at"] = time.Now()
						}
						if err := db.DB.Model(&models.Task{}).Where("id = ?", latest.ID).Updates(failureUpdates).Error; err != nil {
							log.Printf("[WS] Task dispatch failure update failed: %v", err)
						}
					})
				},
			}
		})
		if req != nil {
			enqueueDispatch(*req)
		}
	}
}

func dispatchPendingTasksForNode(nodeID int64) {
	nodeIDStr := strconv.FormatInt(nodeID, 10)
	now := time.Now()
	var tasks []models.Task
	if err := db.DB.Where("enable = ? AND state IN ? AND (retry_at IS NULL OR retry_at <= ?) AND (targets_json IS NULL OR targets_json = '')", true, []string{"waiting", "running"}, now).
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
			if strings.EqualFold(task.Type, "issue_cert") {
				_ = db.DB.Model(&models.Cert{}).
					Where("issue_task_id = ? AND state IN ?", task.ID, []string{"waiting", "fail"}).
					Update("state", "issuing").Error
			}
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
		enqueueDispatch(dispatchRequest{
			nodeID:  nodeID,
			task:    task,
			payload: payload,
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
	msg := TaskDispatchMsg{
		Kind:  MsgTaskDispatch,
		MsgID: fmt.Sprintf("task-%d-%d", task.ID, nodeID),
		Task: TaskPayload{
			TaskID:   int64(task.ID),
			TaskType: task.Type,
			TaskName: task.Name,
			Payload:  payload,
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
		Select("id", "ip", "port", "region_id", "check_protocol", "check_port", "check_host", "check_path", "check_timeout").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	metaMap := services.LoadRegionMetaMap()
	result := make([]l2NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		checkPort := n.CheckPort
		if checkPort == 0 {
			checkPort = services.ResolveRegionL2CheckPort(metaMap, n.RegionID)
		}
		checkProtocol := strings.TrimSpace(n.CheckProtocol)
		if checkProtocol == "" {
			checkProtocol = "tcp"
		}
		result = append(result, l2NodeInfo{
			ID:            n.ID,
			IP:            n.IP,
			Port:          n.Port,
			CheckProtocol: checkProtocol,
			CheckPort:     checkPort,
			CheckHost:     n.CheckHost,
			CheckPath:     n.CheckPath,
			CheckTimeout:  n.CheckTimeout,
		})
	}
	return result, nil
}

// DispatchTest dispatches a task to a connected node (admin testing helper).
func (c *AgentWSController) DispatchTest(ctx *gin.Context) {
	var req WSDispatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.NodeID == 0 || strings.TrimSpace(req.TaskType) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": T("Invalid request")})
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
	if err := dispatchTaskTestToNode(req.NodeID, msgID, req.TaskType, req.Payload); err != nil {
		unregisterAckWaiter(msgID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": T("dispatch failed")})
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

func dispatchTaskTestToNode(nodeID int64, msgID string, taskType string, payload string) error {
	agentMutex.RLock()
	conn, ok := agentConns[nodeID]
	agentMutex.RUnlock()
	if !ok {
		return fmt.Errorf("node %d not connected", nodeID)
	}
	msg := TaskDispatchMsg{
		Kind:  MsgTaskDispatch,
		MsgID: msgID,
		Task: TaskPayload{
			TaskID:   0,
			TaskType: taskType,
			TaskName: "ws-dispatch-test",
			Payload:  payload,
		},
	}
	agentMutex.Lock()
	defer agentMutex.Unlock()
	return conn.WriteJSON(msg)
}

func registerAckWaiter(msgID string) chan TaskAckMsg {
	ch := make(chan TaskAckMsg, 1)
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

func notifyAckWaiter(ack TaskAckMsg) {
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

func withTaskUpdateLock(taskID int64, fn func()) {
	if taskID == 0 {
		fn()
		return
	}
	lockValue, _ := taskUpdateLocks.LoadOrStore(taskID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	fn()
}

type dispatchRequest struct {
	nodeID  int64
	task    models.Task
	payload string
	onError func(error)
}

func initDispatchPool() {
	dispatchPool.Do(func() {
		dispatchQueue = make(chan dispatchRequest, 200)
		for i := 0; i < dispatchWorkerCount; i++ {
			go dispatchQueueWorker()
		}
	})
}

func dispatchQueueWorker() {
	for req := range dispatchQueue {
		if err := dispatchTaskToNode(req.nodeID, req.task, req.payload); err != nil {
			if req.onError != nil {
				req.onError(err)
			} else {
				log.Printf("[WS] Dispatch failed for node %d: %v", req.nodeID, err)
			}
		}
	}
}

func enqueueDispatch(req dispatchRequest) {
	dispatchQueue <- req
}
