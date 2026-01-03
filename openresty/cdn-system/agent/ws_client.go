package main

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wsConn      *websocket.Conn
	wsWriteLock sync.Mutex
)

// StartWebSocketClient initiates the persistent connection
func startWebSocketClient() {
	backoff := time.Second
	maxBackoff := 60 * time.Second

	for {
		if err := connectWS(); err != nil {
			log.Printf("[WS] Connect failed: %v. Retrying in %v...", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		// Reset backoff on successful close (or if validation passed but connection dropped later)
		// Usually we reset after a successful handshake.
		backoff = time.Second
	}
}

func connectWS() error {
	log.Printf("[WS] Connecting to %s/ws/agent...", API_BaseURL)

	// Convert http -> ws
	// We assume API_BaseURL is http(s)://...
	// If http -> ws, https -> wss
	// Actually URL parsing is better but simple string replacement works for now provided BaseURL is standard.
	// But `gorilla/websocket.Dial` expects proper URL.
	// Let's assume API_BaseURL is "http://127.0.0.1:8080"

	var wsURL string
	if len(API_BaseURL) > 5 && API_BaseURL[:5] == "https" {
		wsURL = "wss" + API_BaseURL[5:] + "/ws/agent"
	} else if len(API_BaseURL) > 4 && API_BaseURL[:4] == "http" {
		wsURL = "ws" + API_BaseURL[4:] + "/ws/agent"
	} else {
		// Fallback or raw domain
		wsURL = "ws://" + API_BaseURL + "/ws/agent"
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	defer conn.Close()

	// Set Global Conn
	setWSConn(conn)
	defer setWSConn(nil)

	// 1. Send Hello
	if err := sendAgentHello(conn); err != nil {
		log.Printf("[WS] Handshake failed: %v", err)
		return err
	}

	log.Println("[WS] Connected and Authenticated")

	heartbeatDone := make(chan struct{})
	go startWSHeartbeat(conn, heartbeatDone)
	defer close(heartbeatDone)

	// 2. Read Loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Read error: %v", err)
			return err
		}

		var header struct {
			Kind string `json:"kind"`
		}
		if json.Unmarshal(msg, &header) != nil {
			continue
		}

		switch header.Kind {
		case "job_dispatch":
			handleJobDispatch(msg)
		case "heartbeat_ack":
			handleHeartbeatAck(msg)
		}
	}
}

func setWSConn(conn *websocket.Conn) {
	wsWriteLock.Lock()
	wsConn = conn
	wsWriteLock.Unlock()
}

func sendAgentHello(conn *websocket.Conn) error {
	msg := map[string]interface{}{
		"kind":          "agent_hello",
		"node_id":       NodeID, // Hostname
		"token":         AuthToken,
		"agent_version": "1.0.0",
		"capabilities":  []string{"套餐同步", "ACL发布", "CC发布"},
	}
	return conn.WriteJSON(msg)
}

func startWSHeartbeat(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			wsWriteLock.Lock()
			err := conn.WriteJSON(map[string]interface{}{
				"kind":      "heartbeat",
				"timestamp": time.Now().Unix(),
				"status":    "active",
			})
			wsWriteLock.Unlock()
			if err != nil {
				log.Printf("[WS] Heartbeat failed: %v", err)
				return
			}
			retryPendingNodeSync(conn)
		}
	}
}

func handleHeartbeatAck(raw []byte) {
	var resp struct {
		Kind       string `json:"kind"`
		SyncAction string `json:"sync_action"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return
	}
	if action := resp.SyncAction; action != "" {
		if err := applyNodeSync(action); err != nil {
			log.Printf("[Error] Sync node status failed: %v", err)
		}
	}
}

func sendNodeSync(action string, success bool) error {
	wsWriteLock.Lock()
	conn := wsConn
	wsWriteLock.Unlock()
	if conn == nil {
		recordPendingNodeSync(action, success, errors.New("ws not connected"))
		return errors.New("ws not connected")
	}

	msg := map[string]interface{}{
		"kind":    "node_sync",
		"action":  action,
		"success": success,
	}

	wsWriteLock.Lock()
	err := conn.WriteJSON(msg)
	wsWriteLock.Unlock()
	if err != nil {
		log.Printf("[WS] Failed to send node sync: %v", err)
		recordPendingNodeSync(action, success, err)
		return err
	}
	return nil
}

func recordPendingNodeSync(action string, success bool, err error) {
	localConfigMu.Lock()
	defer localConfigMu.Unlock()

	for i := range pendingNodeSyncs {
		if pendingNodeSyncs[i].Action == action && pendingNodeSyncs[i].Success == success {
			pendingNodeSyncs[i].Attempts++
			pendingNodeSyncs[i].LastError = err.Error()
			pendingNodeSyncs[i].LastAt = time.Now()
			return
		}
	}
	pendingNodeSyncs = append(pendingNodeSyncs, nodeSyncAck{
		Action:    action,
		Success:   success,
		Attempts:  1,
		LastError: err.Error(),
		LastAt:    time.Now(),
	})
	if len(pendingNodeSyncs) > 10 {
		pendingNodeSyncs = pendingNodeSyncs[len(pendingNodeSyncs)-10:]
	}
}

func retryPendingNodeSync(conn *websocket.Conn) {
	localConfigMu.Lock()
	if len(pendingNodeSyncs) == 0 {
		localConfigMu.Unlock()
		return
	}
	pending := pendingNodeSyncs[0]
	localConfigMu.Unlock()

	msg := map[string]interface{}{
		"kind":    "node_sync",
		"action":  pending.Action,
		"success": pending.Success,
	}

	wsWriteLock.Lock()
	err := conn.WriteJSON(msg)
	wsWriteLock.Unlock()
	if err != nil {
		log.Printf("[WS] Retry node sync failed: %v", err)
		recordPendingNodeSync(pending.Action, pending.Success, err)
		return
	}

	localConfigMu.Lock()
	if len(pendingNodeSyncs) > 0 {
		pendingNodeSyncs = pendingNodeSyncs[1:]
	}
	localConfigMu.Unlock()
}

// In main.go or tasks.go we have processTask
// We need to route job_dispatch -> processTask
// And then send ACK.

type JobDispatchMsg struct {
	Kind  string `json:"kind"`
	MsgID string `json:"msg_id"`
	Task  struct {
		TaskID   int64  `json:"task_id"`
		TaskType string `json:"task_type"`
	} `json:"task"`
	Job struct {
		JobID   int64  `json:"job_id"`
		JobType string `json:"job_type"`
		Payload string `json:"payload"`
	} `json:"job"`
}

func handleJobDispatch(raw []byte) {
	var msg JobDispatchMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("[WS] Invalid dispatch msg: %v", err)
		return
	}

	log.Printf("[WS] Received Job %d (Type: %s)", msg.Job.JobID, msg.Job.JobType)

	// Execute
	// We call the existing processTask function in tasks.go
	ret, err := processTask(msg.Job.JobID, msg.Job.JobType, msg.Job.Payload)

	status := "success"
	errMsg := ""
	if err != nil {
		status = "fail"
		errMsg = err.Error()
		log.Printf("[WS] Job %d Failed: %v", msg.Job.JobID, err)
	} else {
		// If ret is "skipped" based on logic?
		// processTask returns string.
		// For synUserPackage, it returns JSON `{"applied":...}`.
	}

	// ACK
	// ret usually contains the JSON result if success
	// We need to parse ret to put into 'applied' if it's JSON?
	// The ACK struct expects `applied` as RawMessage.

	sendJobAck(msg.MsgID, msg.Task.TaskID, msg.Job.JobID, msg.Job.JobType, status, ret, errMsg)
}

func sendJobAck(msgID string, taskID, jobID int64, jobType, status, ret, errorMsg string) {
	wsWriteLock.Lock()
	conn := wsConn
	wsWriteLock.Unlock()

	if conn == nil {
		return
	}

	// Try to treat ret as JSON object if possible, else string
	var applied json.RawMessage
	if ret != "" && (ret[0] == '{' || ret[0] == '[') {
		applied = json.RawMessage(ret)
	} else {
		// If plain string, maybe wrap it? Or just leave nil?
		// Spec says "applied": [ ... ].
		// If ret is not JSON, applied might be null.
	}

	ack := map[string]interface{}{
		"kind":     "job_ack",
		"msg_id":   msgID,
		"node_id":  0, // API knows who we are
		"task_id":  taskID,
		"job_id":   jobID,
		"job_type": jobType,
		"status":   status,
		"applied":  applied,
		"error":    errorMsg,
	}

	if err := conn.WriteJSON(ack); err != nil {
		log.Printf("[WS] Failed to send ACK: %v", err)
	}
}
