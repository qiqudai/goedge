package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// startHeartbeat sends status to API at the configured interval.
func startHeartbeat() {
	sendHeartbeat()
	ticker := time.NewTicker(HEARTBEAT_INT)
	for range ticker.C {
		sendHeartbeat()
	}
}

func sendHeartbeat() {
	data := map[string]interface{}{
		"node_id":   NodeID,
		"timestamp": time.Now().Unix(),
		"status":    "active",
		// Add Load/CPU/Mem stats here later
	}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/heartbeat", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	readBody := true
	respBody, status, err := doRequest(req, 5*time.Second, readBody)
	if err != nil {
		log.Printf("[Error] Heartbeat Failed: %v", err)
		return
	}
	debugLogInteraction("POST", req.URL.String(), status, jsonData, respBody)

	if status == 200 {
		log.Println("[Info] Heartbeat OK")
		var resp struct {
			SyncAction string `json:"sync_action"`
		}
		if len(respBody) > 0 && json.Unmarshal(respBody, &resp) == nil {
			if action := strings.ToLower(strings.TrimSpace(resp.SyncAction)); action != "" {
				if err := applyNodeSync(action); err != nil {
					log.Printf("[Error] Sync node status failed: %v", err)
				}
			}
		}
	} else {
		log.Printf("[Warn] Heartbeat Status: %d", status)
	}
}

func applyNodeSync(action string) error {
	switch action {
	case "enable":
		if err := startNginx(); err != nil {
			if reloadErr := executeReload(); reloadErr != nil {
				_ = reportNodeSync(action, false)
				return err
			}
		}
	case "disable":
		if err := stopNginx(); err != nil {
			_ = reportNodeSync(action, false)
			return err
		}
	default:
		return nil
	}
	return reportNodeSync(action, true)
}

func reportNodeSync(action string, success bool) error {
	payload := map[string]interface{}{
		"node_id": NodeID,
		"action":  action,
		"success": success,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/node/sync", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	respBody, status, err := doRequest(req, 10*time.Second, DebugMode)
	if err != nil {
		return err
	}
	debugLogInteraction("POST", req.URL.String(), status, body, respBody)
	if status != 200 {
		return fmt.Errorf("sync status failed: %d", status)
	}
	return nil
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

type l2NodesResponse struct {
	Nodes []l2NodeInfo `json:"nodes"`
}

func startL2Monitor() {
	ticker := time.NewTicker(L2_CHECK_INT)
	for range ticker.C {
		checkL2Nodes()
	}
}

func checkL2Nodes() {
	nodes, err := fetchL2Nodes()
	if err != nil {
		log.Printf("[Error] L2 Monitor fetch failed: %v", err)
		return
	}
	if len(nodes) == 0 {
		return
	}
	online := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		if isL2Alive(node) {
			online = append(online, node.ID)
		}
	}
	if len(online) == 0 {
		return
	}
	if err := reportL2Heartbeat(online); err != nil {
		log.Printf("[Error] L2 Monitor report failed: %v", err)
		return
	}
	if DebugMode {
		log.Printf("[Debug] L2 Monitor OK: %d/%d online", len(online), len(nodes))
	}
}

func fetchL2Nodes() ([]l2NodeInfo, error) {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/l2/nodes?node_id="+NodeID, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	body, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("unexpected status: %d", status)
	}
	var resp l2NodesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Nodes, nil
}

func reportL2Heartbeat(nodes []int64) error {
	payload := map[string]interface{}{
		"nodes": nodes,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/l2/heartbeat", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	_, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("unexpected status: %d", status)
	}
	return nil
}

func isL2Alive(node l2NodeInfo) bool {
	timeout := time.Duration(node.CheckTimeout)
	if timeout <= 0 {
		timeout = 5
	}
	timeout *= time.Second
	protocol := strings.ToLower(strings.TrimSpace(node.CheckProtocol))
	port := node.CheckPort
	if port <= 0 {
		if node.Port > 0 {
			port = node.Port
		} else if protocol == "https" {
			port = 443
		} else {
			port = 80
		}
	}
	checkHost := strings.TrimSpace(node.CheckHost)
	if checkHost == "" {
		checkHost = node.IP
	}
	path := strings.TrimSpace(node.CheckPath)
	if path == "" {
		path = "/"
	}

	switch protocol {
	case "http", "https":
		target := fmt.Sprintf("%s://%s:%d%s", protocol, node.IP, port, path)
		client := &http.Client{Timeout: timeout}
		req, _ := http.NewRequest("GET", target, nil)
		if checkHost != "" {
			req.Host = checkHost
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode >= 200 && resp.StatusCode < 400
	default:
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", node.IP, port), timeout)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}
}
