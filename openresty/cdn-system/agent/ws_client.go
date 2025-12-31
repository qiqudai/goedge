package main

import (
	"encoding/json"
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
		case "pong":
			// ignore
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
