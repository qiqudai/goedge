// Command dns_robustness runs node-group resolution chaos tests against a local admin API.
//
// Usage (local/docker only — do NOT point at production domains):
//
//	export ADMIN_URL=http://127.0.0.1:8080
//	export ADMIN_TOKEN=<jwt>
//	export GROUP_ID=999
//	export LINE_ID=default
//	go run ./scripts/dns_robustness
//
// Optional:
//   CYCLES=20          number of chaos rounds (default 10)
//   NODES_PER_CYCLE=10 nodes to assign per round
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type apiResp struct {
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func apiOK(code int) bool {
	return code == 0 || code == 200
}

type resolutionView struct {
	Assigned []assignedItem `json:"assigned"`
}

type assignedItem struct {
	ID       int64  `json:"id"`
	NodeIPID int64  `json:"node_ip_id"`
	IP       string `json:"ip"`
	IsOn     bool   `json:"is_on"`
	LineID   string `json:"line_id"`
}

func main() {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("ADMIN_URL")), "/")
	token := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	groupID := strings.TrimSpace(os.Getenv("GROUP_ID"))
	lineID := strings.TrimSpace(os.Getenv("LINE_ID"))
	if baseURL == "" || token == "" || groupID == "" {
		fmt.Fprintln(os.Stderr, "ADMIN_URL, ADMIN_TOKEN and GROUP_ID are required")
		os.Exit(2)
	}
	if lineID == "" {
		lineID = "default"
	}
	cycles := envInt("CYCLES", 10)
	nodesPerCycle := envInt("NODES_PER_CYCLE", 10)

	client := &http.Client{Timeout: 30 * time.Second}
	if err := runChaos(client, baseURL, token, groupID, lineID, cycles, nodesPerCycle); err != nil {
		fmt.Fprintf(os.Stderr, "chaos test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("dns robustness chaos test passed")
}

func runChaos(client *http.Client, baseURL, token, groupID, lineID string, cycles, nodesPerCycle int) error {
	for round := 1; round <= cycles; round++ {
		fmt.Printf("round %d/%d: fetch resolution\n", round, cycles)
		view, err := fetchResolution(client, baseURL, token, groupID, lineID)
		if err != nil {
			return err
		}
		lineItems := filterLine(view.Assigned, lineID)
		if len(lineItems) == 0 {
			return fmt.Errorf("round %d: no assigned nodes on line %s", round, lineID)
		}
		if len(lineItems) > nodesPerCycle {
			lineItems = lineItems[:nodesPerCycle]
		}

		ids := make([]int64, 0, len(lineItems))
		wantIPs := map[string]struct{}{}
		for _, item := range lineItems {
			ids = append(ids, item.ID)
			if item.IP != "" && item.IsOn {
				wantIPs[item.IP] = struct{}{}
			}
		}

		fmt.Printf("round %d: disable %d nodes\n", round, len(ids))
		if err := resolutionAction(client, baseURL, token, groupID, "disable", ids); err != nil {
			return fmt.Errorf("round %d disable: %w", round, err)
		}
		time.Sleep(300 * time.Millisecond)

		fmt.Printf("round %d: delete %d nodes\n", round, len(ids))
		if err := resolutionAction(client, baseURL, token, groupID, "delete", ids); err != nil {
			return fmt.Errorf("round %d delete: %w", round, err)
		}
		time.Sleep(300 * time.Millisecond)

		assignItems := make([]map[string]int64, 0, len(lineItems))
		for _, item := range lineItems {
			assignItems = append(assignItems, map[string]int64{
				"node_id":    item.NodeIPID,
				"node_ip_id": item.NodeIPID,
			})
		}
		fmt.Printf("round %d: re-assign %d nodes\n", round, len(assignItems))
		if err := assignNodes(client, baseURL, token, groupID, lineID, lineID, assignItems); err != nil {
			return fmt.Errorf("round %d assign: %w", round, err)
		}
		time.Sleep(500 * time.Millisecond)

		view, err = fetchResolution(client, baseURL, token, groupID, lineID)
		if err != nil {
			return err
		}
		gotIPs := map[string]struct{}{}
		for _, item := range filterLine(view.Assigned, lineID) {
			if item.IsOn && item.IP != "" {
				gotIPs[item.IP] = struct{}{}
			}
		}
		if len(gotIPs) != len(wantIPs) {
			return fmt.Errorf("round %d: assigned count mismatch got=%d want=%d", round, len(gotIPs), len(wantIPs))
		}
		for ip := range wantIPs {
			if _, ok := gotIPs[ip]; !ok {
				return fmt.Errorf("round %d: missing ip %s after re-assign", round, ip)
			}
		}
	}
	return nil
}

func fetchResolution(client *http.Client, baseURL, token, groupID, lineID string) (*resolutionView, error) {
	url := fmt.Sprintf("%s/api/v1/admin/node-groups/%s/resolution?line_id=%s", baseURL, groupID, lineID)
	body, err := doJSON(client, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if !apiOK(resp.Code) {
		msg := resp.Msg
		if msg == "" {
			msg = resp.Message
		}
		return nil, fmt.Errorf("list resolution: code=%d msg=%s", resp.Code, msg)
	}
	var view resolutionView
	if err := json.Unmarshal(resp.Data, &view); err != nil {
		return nil, err
	}
	return &view, nil
}

func resolutionAction(client *http.Client, baseURL, token, groupID, action string, ids []int64) error {
	url := fmt.Sprintf("%s/api/v1/admin/node-groups/%s/resolution/action", baseURL, groupID)
	payload := map[string]interface{}{
		"action": action,
		"ids":    ids,
	}
	body, err := doJSON(client, http.MethodPost, url, token, payload)
	if err != nil {
		return err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if !apiOK(resp.Code) {
		msg := resp.Msg
		if msg == "" {
			msg = resp.Message
		}
		return fmt.Errorf("action=%s code=%d msg=%s", action, resp.Code, msg)
	}
	return nil
}

func assignNodes(client *http.Client, baseURL, token, groupID, lineID, lineName string, items []map[string]int64) error {
	url := fmt.Sprintf("%s/api/v1/admin/node-groups/%s/resolution/assign", baseURL, groupID)
	payload := map[string]interface{}{
		"line_id":   lineID,
		"line_name": lineName,
		"items":     items,
	}
	body, err := doJSON(client, http.MethodPost, url, token, payload)
	if err != nil {
		return err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if !apiOK(resp.Code) {
		msg := resp.Msg
		if msg == "" {
			msg = resp.Message
		}
		return fmt.Errorf("assign code=%d msg=%s", resp.Code, msg)
	}
	return nil
}

func doJSON(client *http.Client, method, url, token string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func filterLine(items []assignedItem, lineID string) []assignedItem {
	out := make([]assignedItem, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.LineID), lineID) {
			out = append(out, item)
		}
	}
	return out
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
