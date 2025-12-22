package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// AgentConfig Configuration
const (
	API_URL       = "http://127.0.0.1:8080/api/v1/agent" // Control Plane URL
	HEARTBEAT_INT = 30 * time.Second
	CONFIG_PATH   = "/usr/local/openresty/nginx/conf/cdn_config.json"
	CONFIG_BAK    = "/usr/local/openresty/nginx/conf/cdn_config.json.bak"
)

var (
	NodeID    = "" // Unique Node ID
	AuthToken = "" // Token from install parameter
)

func main() {
	// 1. Argument Parsing (Simplified)
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./cdn-agent <auth-token>")
	}
	AuthToken = os.Args[1]

	// Assume hostname as NodeID for now
	hostname, _ := os.Hostname()
	NodeID = hostname

	log.Printf("Starting Edge Agent on %s...", NodeID)

	// 2. Start Tickers
	go startHeartbeat()
	go startConfigPull()

	// 3. Keep Alive
	select {}
}

// startHeartbeat sends status to API every 30s
func startHeartbeat() {
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

	req, _ := http.NewRequest("POST", API_URL+"/heartbeat", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Heartbeat Failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("[Info] Heartbeat OK")
	} else {
		log.Printf("[Warn] Heartbeat Status: %d", resp.StatusCode)
	}
}

// startConfigPull checks for config updates
func startConfigPull() {
	// Pull every minute or use HTTP Long-Polling / Websocket in production
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		pullConfig()
	}
}

func pullConfig() {
	req, _ := http.NewRequest("GET", API_URL+"/config?node_id="+NodeID, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Config Pull Failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		newVersion := extractVersion(body)
		currentVersion := readLocalVersion()
		if newVersion != 0 && newVersion <= currentVersion {
			log.Printf("[Info] Config unchanged (version=%d). Skipping reload.", currentVersion)
			return
		}

		if err := writeConfigWithBackup(body); err != nil {
			log.Printf("[Error] Failed to write config file: %v", err)
			return
		}

		log.Printf("[Info] Config Updated (version=%d, %d bytes). Reloading Nginx...", newVersion, len(body))
		if err := executeReload(); err != nil {
			log.Printf("[Error] Reload Nginx Failed: %v", err)
			if restoreErr := restoreBackup(); restoreErr != nil {
				log.Printf("[Error] Failed to restore backup: %v", restoreErr)
				return
			}
			if retryErr := executeReload(); retryErr != nil {
				log.Printf("[Error] Reload after rollback failed: %v", retryErr)
			} else {
				log.Println("[Warn] Rolled back to previous config")
			}
		}
	}
}

func executeReload() error {
	// Check if nginx is running first to avoid errors
	cmd := exec.Command("nginx", "-t")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("nginx", "-s", "reload")
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println("[Success] Nginx Reloaded")
	return nil
}

func extractVersion(body []byte) int64 {
	var payload struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}
	return payload.Version
}

func readLocalVersion() int64 {
	data, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		return 0
	}
	return extractVersion(data)
}

func writeConfigWithBackup(body []byte) error {
	if current, err := ioutil.ReadFile(CONFIG_PATH); err == nil {
		_ = ioutil.WriteFile(CONFIG_BAK, current, 0644)
	}
	tmpPath := CONFIG_PATH + ".tmp"
	if err := ioutil.WriteFile(tmpPath, body, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, CONFIG_PATH)
}

func restoreBackup() error {
	data, err := ioutil.ReadFile(CONFIG_BAK)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(CONFIG_PATH, data, 0644); err != nil {
		return err
	}
	return nil
}
