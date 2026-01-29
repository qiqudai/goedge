package main

import (
	fsutil "cdn-common/io"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	if err := sendWSJSON(map[string]interface{}{
		"kind":      "heartbeat",
		"timestamp": time.Now().Unix(),
		"status":    "active",
	}); err != nil {
		log.Printf("[Error] Heartbeat Failed: %v", err)
	}
}

func applyNodeSync(action string) error {
	switch action {
	case "enable":
		if err := startNginx(); err != nil {
			if reloadErr := reloadNginx(); reloadErr != nil {
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
	if err := sendNodeSync(action, success); err != nil {
		return err
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

type l2HealthState struct {
	Online  bool
	Success int
	Fail    int
}

var l2HealthStore = struct {
	mu           sync.Mutex
	states       map[int64]*l2HealthState
	lastSnapshot map[string]bool
}{
	states:       map[int64]*l2HealthState{},
	lastSnapshot: map[string]bool{},
}

func startL2Monitor() {
	checkL2Nodes()
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
		empty := map[string]bool{}
		l2HealthStore.mu.Lock()
		changed := !l2SnapshotEqual(empty, l2HealthStore.lastSnapshot)
		if changed {
			l2HealthStore.lastSnapshot = empty
		}
		l2HealthStore.mu.Unlock()
		if changed {
			writeL2StatusSnapshot(empty)
		}
		return
	}

	onlineNow := make([]int64, 0, len(nodes))
	nodeSet := map[int64]struct{}{}
	snapshot := map[string]bool{}

	l2HealthStore.mu.Lock()
	for _, node := range nodes {
		nodeSet[node.ID] = struct{}{}
		alive := isL2Alive(node)
		if alive {
			onlineNow = append(onlineNow, node.ID)
		}
		state := l2HealthStore.states[node.ID]
		if state == nil {
			state = &l2HealthState{Online: true}
			l2HealthStore.states[node.ID] = state
		}
		if alive {
			state.Fail = 0
			state.Success++
			if state.Success > 3 {
				state.Success = 3
			}
			if !state.Online && state.Success >= 3 {
				state.Online = true
			}
		} else {
			state.Success = 0
			state.Fail++
			if state.Fail > 3 {
				state.Fail = 3
			}
			if state.Online && state.Fail >= 3 {
				state.Online = false
			}
		}
		snapshot[strconv.FormatInt(node.ID, 10)] = state.Online
	}
	for id := range l2HealthStore.states {
		if _, ok := nodeSet[id]; !ok {
			delete(l2HealthStore.states, id)
		}
	}

	changed := !l2SnapshotEqual(snapshot, l2HealthStore.lastSnapshot)
	if changed {
		l2HealthStore.lastSnapshot = snapshot
	}
	l2HealthStore.mu.Unlock()

	if changed {
		writeL2StatusSnapshot(snapshot)
	}
	if len(onlineNow) == 0 {
		return
	}
	if err := reportL2Heartbeat(onlineNow); err != nil {
		log.Printf("[Error] L2 Monitor report failed: %v", err)
		return
	}
	if DebugMode {
		log.Printf("[Debug] L2 Monitor OK: %d/%d online", len(onlineNow), len(nodes))
	}
}

func fetchL2Nodes() ([]l2NodeInfo, error) {
	return requestL2Nodes(5 * time.Second)
}

func reportL2Heartbeat(nodes []int64) error {
	return sendL2Heartbeat(nodes)
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

func l2SnapshotEqual(next map[string]bool, current map[string]bool) bool {
	if len(next) != len(current) {
		return false
	}
	for key, val := range next {
		if cur, ok := current[key]; !ok || cur != val {
			return false
		}
	}
	return true
}

func writeL2StatusSnapshot(snapshot map[string]bool) {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return
	}
	path := filepath.Join(rootDir, "conf", "l2_status.json")
	payload := map[string]interface{}{
		"updated_at": time.Now().Unix(),
		"nodes":      snapshot,
	}
	if err := fsutil.WriteJSONAtomic(path, payload, true); err != nil {
		log.Printf("[Error] L2 status write failed: %v", err)
		return
	}
	refreshStreamConfigForL2Status(snapshot)
}
