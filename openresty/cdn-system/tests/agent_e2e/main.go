package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type loginResponse struct {
	Token string `json:"token"`
}

type dispatchResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Connected bool   `json:"connected"`
		State     string `json:"state"`
		Error     string `json:"error"`
	} `json:"data"`
}

type checkResult struct {
	Name   string
	Method string
	URL    string
	Status int
	Err    error
}

func main() {
	var (
		apiBase    = flag.String("api", "http://127.0.0.1:8080", "API base URL")
		adminUser  = flag.String("admin-user", "", "Admin username")
		adminPass  = flag.String("admin-pass", "", "Admin password")
		agentToken = flag.String("agent-token", "", "Agent token (Bearer)")
		nodeID     = flag.String("node-id", "", "Agent node ID")
		timeoutSec = flag.Int("timeout", 10, "HTTP timeout seconds")
	)
	flag.Parse()

	if *adminUser == "" || *adminPass == "" || *agentToken == "" || *nodeID == "" {
		fmt.Println("Missing required flags: -admin-user -admin-pass -agent-token -node-id")
		os.Exit(2)
	}

	client := &http.Client{Timeout: time.Duration(*timeoutSec) * time.Second}

	adminToken, err := login(client, *apiBase, *adminUser, *adminPass)
	if err != nil {
		fmt.Printf("Admin login failed: %v\n", err)
		os.Exit(1)
	}

	if err := wsDispatch(client, *apiBase, adminToken, *nodeID); err != nil {
		fmt.Printf("WS dispatch failed: %v\n", err)
		os.Exit(1)
	}

	checks := []struct {
		name   string
		method string
		url    string
		token  string
		body   interface{}
	}{
		{"agent config", http.MethodGet, fmt.Sprintf("%s/api/v1/agent/config?node_id=%s", *apiBase, *nodeID), *agentToken, nil},
		{"agent tasks", http.MethodGet, fmt.Sprintf("%s/api/v1/agent/tasks", *apiBase), *agentToken, nil},
		{"agent l2 nodes", http.MethodGet, fmt.Sprintf("%s/api/v1/agent/l2/nodes?node_id=%s", *apiBase, *nodeID), *agentToken, nil},
		{"agent heartbeat", http.MethodPost, fmt.Sprintf("%s/api/v1/agent/heartbeat", *apiBase), *agentToken, map[string]interface{}{
			"node_id":   *nodeID,
			"timestamp": time.Now().Unix(),
			"status":    "ok",
		}},
	}

	results := make([]checkResult, 0, len(checks))
	for _, c := range checks {
		status, err := doRequest(client, c.method, c.url, c.token, c.body)
		results = append(results, checkResult{
			Name:   c.name,
			Method: c.method,
			URL:    c.url,
			Status: status,
			Err:    err,
		})
	}

	failures := 0
	for _, r := range results {
		ok := r.Err == nil && r.Status >= 200 && r.Status < 300
		if !ok {
			failures++
		}
		fmt.Printf("%-15s %-4s status=%d ok=%v\n", r.Name, r.Method, r.Status, ok)
		if r.Err != nil {
			fmt.Printf("  error: %v\n", r.Err)
		}
	}

	if failures > 0 {
		fmt.Printf("Agent e2e checks failed: %d\n", failures)
		os.Exit(1)
	}
	fmt.Println("Agent e2e checks passed.")
}

func login(client *http.Client, apiBase, user, pass string) (string, error) {
	body := map[string]string{"username": user, "password": pass}
	url := fmt.Sprintf("%s/api/v1/admin/login", apiBase)
	status, data, err := doJSON(client, http.MethodPost, url, "", body)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("login status %d", status)
	}
	var resp loginResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("login parse: %w", err)
	}
	if resp.Token == "" {
		return "", fmt.Errorf("login token empty")
	}
	return resp.Token, nil
}

func wsDispatch(client *http.Client, apiBase, adminToken, nodeID string) error {
	body := map[string]interface{}{
		"node_id":      parseNodeID(nodeID),
		"task_type":    "config_sync",
		"payload":      "",
		"wait_seconds": 8,
	}
	url := fmt.Sprintf("%s/api/v1/admin/ws/dispatch", apiBase)
	status, data, err := doJSON(client, http.MethodPost, url, adminToken, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("dispatch status %d", status)
	}
	var resp dispatchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("dispatch parse: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("dispatch code %d: %s", resp.Code, resp.Msg)
	}
	if !resp.Data.Connected {
		return fmt.Errorf("node not connected")
	}
	if resp.Data.State != "" && resp.Data.State != "success" {
		return fmt.Errorf("dispatch state %s: %s", resp.Data.State, resp.Data.Error)
	}
	return nil
}

func parseNodeID(raw string) int64 {
	var id int64
	_, _ = fmt.Sscanf(raw, "%d", &id)
	if id <= 0 {
		return 0
	}
	return id
}

func doRequest(client *http.Client, method, url, token string, body interface{}) (int, error) {
	status, _, err := doJSON(client, method, url, token, body)
	return status, err
}

func doJSON(client *http.Client, method, url, token string, body interface{}) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}
