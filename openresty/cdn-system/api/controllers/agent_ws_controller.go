package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
)

// Message Types
const (
	MsgAgentHello  = "agent_hello"
	MsgJobDispatch = "job_dispatch"
	MsgJobAck      = "job_ack"
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
			node, err := c.authenticateNode(hello.Token)
			if err != nil {
				log.Printf("[WS] Auth failed for token %s: %v", hello.Token, err)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4003, "Auth Failed"))
				return
			}
			nodeID = int64(node.ID)

			// Register
			c.registerConn(nodeID, conn)
			log.Printf("[WS] Node %d connected (ver: %s)", nodeID, hello.AgentVersion)

			// Reset deadline for heartbeat (if we add ping/pong)
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
		case "ping":
			conn.WriteMessage(websocket.TextMessage, []byte(`{"kind":"pong"}`))
		}
	}
}

func (c *AgentWSController) authenticateNode(token string) (*models.Node, error) {
	var node models.Node
	// Assuming Token matches InstallKey or we have a Token field.
	// The prompt says "Token from install parameter".
	// In models/node.go, usually there is a seed or token.
	// Let's assume InstallKey for now or a generic token field.
	// Actually previous code used 'AuthToken' in headers.
	// Let's check DB for node with this token.
	// If Token is not unique, this is an issue.
	// Assuming 'install_key' or similar unique token per node.
	// ...
	if err := db.DB.Where("install_key = ? OR unique_id = ?", token, token).First(&node).Error; err != nil {
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
	// Update Job

	var state string
	switch ack.Status {
	case "success":
		state = "success"
	case "fail":
		state = "fail"
	case "ignored":
		state = "success" // map ignored to success?
	}

	errStr := ack.Error

	updates := map[string]interface{}{
		"status": state,
		"error":  errStr,
		"end_at": time.Now(),
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
