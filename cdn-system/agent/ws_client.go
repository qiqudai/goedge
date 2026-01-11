package main

import (
	"cdn-common/i18n"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wsConn      *websocket.Conn
	wsWriteLock sync.Mutex
	l2Waiters   = make(map[string]chan L2NodesResponseMsg)
	l2WaiterMu  sync.Mutex
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
		case "task_dispatch":
			handleTaskDispatch(msg)
		case "heartbeat_ack":
			handleHeartbeatAck(msg)
		case "l2_nodes_response":
			handleL2NodesResponse(msg)
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
		"capabilities": []string{
			i18n.T("agent.capability.sync_package"),
			i18n.T("agent.capability.acl_publish"),
			i18n.T("agent.capability.cc_publish"),
		},
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
			err := sendWSJSON(map[string]interface{}{
				"kind":      "heartbeat",
				"timestamp": time.Now().Unix(),
				"status":    "active",
			})
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
	msg := map[string]interface{}{
		"kind":    "node_sync",
		"action":  action,
		"success": success,
	}
	if err := sendWSJSON(msg); err != nil {
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
	if err := sendWSJSON(msg); err != nil {
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
// We need to route task_dispatch -> processTask
// And then send ACK.

type TaskDispatchMsg struct {
	Kind  string `json:"kind"`
	MsgID string `json:"msg_id"`
	Task  struct {
		TaskID   int64  `json:"task_id"`
		TaskType string `json:"task_type"`
		TaskName string `json:"task_name"`
		Payload  string `json:"payload,omitempty"`
	} `json:"task"`
}

func handleTaskDispatch(raw []byte) {
	var msg TaskDispatchMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("[WS] Invalid dispatch msg: %v", err)
		return
	}

	taskType := strings.TrimSpace(msg.Task.TaskType)
	payload := msg.Task.Payload
	taskID := msg.Task.TaskID
	runID := taskID

	log.Printf("[WS] Received Task %d (Type: %s)", runID, taskType)

	// Execute
	// We call the existing processTask function in tasks.go
	ret, err := processTask(runID, taskType, payload)

	status := "success"
	errMsg := ""
	if err != nil {
		status = "fail"
		errMsg = err.Error()
		log.Printf("[WS] Task %d Failed: %v", runID, err)
	} else {
		// If ret is "skipped" based on logic?
		// processTask returns string.
		// For synUserPackage, it returns JSON `{"applied":...}`.
	}

	// ACK
	// ret usually contains the JSON result if success
	// We need to parse ret to put into 'applied' if it's JSON?
	// The ACK struct expects `applied` as RawMessage.
	sendTaskAck(msg.MsgID, taskID, taskType, status, ret, errMsg)
}

func sendTaskAck(msgID string, taskID int64, taskType, status, ret, errorMsg string) {
	// Try to treat ret as JSON object if possible, else string
	var applied json.RawMessage
	retValue := ""
	if ret != "" && (ret[0] == '{' || ret[0] == '[') {
		applied = json.RawMessage(ret)
	} else {
		retValue = ret
	}

	ack := map[string]interface{}{
		"kind":      "task_ack",
		"msg_id":    msgID,
		"node_id":   0, // API knows who we are
		"task_id":   taskID,
		"task_type": taskType,
		"status":    status,
		"applied":   applied,
		"ret":       retValue,
		"error":     errorMsg,
	}

	if err := sendWSJSON(ack); err != nil {
		log.Printf("[WS] Failed to send ACK: %v", err)
	}
}

type L2NodesResponseMsg struct {
	Kind  string       `json:"kind"`
	MsgID string       `json:"msg_id"`
	Nodes []l2NodeInfo `json:"nodes"`
}

func sendAccessLogs(lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	msg := map[string]interface{}{
		"kind":    "agent_logs_access",
		"node_id": NodeID,
		"node_ip": "",
		"lines":   lines,
	}
	return sendWSJSON(msg)
}

func sendMetrics(content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	msg := map[string]interface{}{
		"kind":    "agent_logs_metrics",
		"node_id": NodeID,
		"node_ip": "",
		"content": content,
	}
	return sendWSJSON(msg)
}

func sendEvents(eventType string, payloads []string) error {
	if strings.TrimSpace(eventType) == "" || len(payloads) == 0 {
		return nil
	}
	msg := map[string]interface{}{
		"kind":     "agent_logs_events",
		"node_id":  NodeID,
		"node_ip":  "",
		"type":     eventType,
		"payloads": payloads,
	}
	return sendWSJSON(msg)
}

func requestL2Nodes(timeout time.Duration) ([]l2NodeInfo, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	msgID := fmt.Sprintf("l2-%d", time.Now().UnixNano())
	waiter := registerL2Waiter(msgID)
	if err := sendWSJSON(map[string]interface{}{
		"kind":   "l2_nodes_request",
		"msg_id": msgID,
	}); err != nil {
		unregisterL2Waiter(msgID)
		return nil, err
	}
	select {
	case resp := <-waiter:
		unregisterL2Waiter(msgID)
		return resp.Nodes, nil
	case <-time.After(timeout):
		unregisterL2Waiter(msgID)
		return nil, errors.New("l2 nodes request timeout")
	}
}

func handleL2NodesResponse(raw []byte) {
	var resp L2NodesResponseMsg
	if err := json.Unmarshal(raw, &resp); err != nil {
		return
	}
	if resp.MsgID == "" {
		return
	}
	l2WaiterMu.Lock()
	ch, ok := l2Waiters[resp.MsgID]
	l2WaiterMu.Unlock()
	if ok {
		ch <- resp
	}
}

func registerL2Waiter(msgID string) chan L2NodesResponseMsg {
	ch := make(chan L2NodesResponseMsg, 1)
	l2WaiterMu.Lock()
	l2Waiters[msgID] = ch
	l2WaiterMu.Unlock()
	return ch
}

func unregisterL2Waiter(msgID string) {
	l2WaiterMu.Lock()
	if ch, ok := l2Waiters[msgID]; ok {
		delete(l2Waiters, msgID)
		close(ch)
	}
	l2WaiterMu.Unlock()
}

func sendL2Heartbeat(nodes []int64) error {
	if len(nodes) == 0 {
		return nil
	}
	msg := map[string]interface{}{
		"kind":  "l2_heartbeat",
		"nodes": nodes,
	}
	return sendWSJSON(msg)
}

func sendCertIssued(taskID int64, certID int64, certPEM string, keyPEM string, rateLimited bool, rateCooldown int) error {
	msg := map[string]interface{}{
		"kind":          "cert_issued",
		"cert_id":       certID,
		"cert":          certPEM,
		"key":           keyPEM,
		"issue_task_id": taskID,
	}
	if rateLimited {
		msg["rate_limited"] = true
		if rateCooldown > 0 {
			msg["rate_cooldown"] = rateCooldown
		}
	}
	return sendWSJSON(msg)
}

func sendWSJSON(msg interface{}) error {
	wsWriteLock.Lock()
	defer wsWriteLock.Unlock()
	if wsConn == nil {
		return errors.New("ws not connected")
	}
	return wsConn.WriteJSON(msg)
}
