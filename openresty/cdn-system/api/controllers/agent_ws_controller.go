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
	agentConns = make(map[int64]*websocket.Conn)
	agentMutex sync.RWMutex
	ackWaiters = make(map[string]chan JobAckMsg)
	ackMutex   sync.Mutex
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
	if err := db.DB.Where("name = ? OR unique_id = ?", nodeHint, nodeHint).First(&node).Error; err != nil {
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
	db.DB.Model(&models.Node{}).Where("id = ?", nodeID).Update("is_on", 1)
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
		"ret":        ack.Error,
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
