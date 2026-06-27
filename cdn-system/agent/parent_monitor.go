package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	fsutil "cdn-common/io"
)

type parentNodeInfo struct {
	ID            int64  `json:"id"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	Level         int    `json:"level"`
	CheckProtocol string `json:"check_protocol"`
	CheckPort     int    `json:"check_port"`
	CheckHost     string `json:"check_host"`
	CheckPath     string `json:"check_path"`
	CheckTimeout  int    `json:"check_timeout"`
}

type parentNodesResponse struct {
	L1Nodes []parentNodeInfo `json:"l1_nodes"`
	L2Nodes []parentNodeInfo `json:"l2_nodes"`
}

var parentHealthStore = struct {
	mu           sync.Mutex
	l1States     map[int64]*l2HealthState
	l2States     map[int64]*l2HealthState
	lastL1Snap   map[string]bool
	lastL2Snap   map[string]bool
}{
	l1States:   map[int64]*l2HealthState{},
	l2States:   map[int64]*l2HealthState{},
	lastL1Snap: map[string]bool{},
	lastL2Snap: map[string]bool{},
}

func startParentMonitor() {
	checkParentNodes()
	ticker := time.NewTicker(L2_CHECK_INT)
	for range ticker.C {
		checkParentNodes()
	}
}

func currentNodeLevel() int {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return 0
	}
	path := strings.TrimSpace(CONFIG_PATH)
	if path == "" {
		path = rootDir + "/conf/cdn_config.json"
	}
	var cfg struct {
		NodeLevel int `json:"node_level"`
	}
	if err := fsutil.ReadJSONFile(path, &cfg); err != nil {
		return 0
	}
	return cfg.NodeLevel
}

func fetchParentNodesHTTP(timeout time.Duration) (*parentNodesResponse, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	endpoint := strings.TrimRight(API_BaseURL, "/") + "/api/v1/agent/parent/nodes?node_id=" + url.QueryEscape(NodeID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	body, status, err := doRequest(req, timeout, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("parent nodes status: %d", status)
	}
	var resp parentNodesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func fetchParentNodes() (*parentNodesResponse, error) {
	return fetchParentNodesHTTP(5 * time.Second)
}

func isParentNodeAlive(node parentNodeInfo) bool {
	return isL2Alive(l2NodeInfo{
		ID:            node.ID,
		IP:            node.IP,
		Port:          node.Port,
		CheckProtocol: node.CheckProtocol,
		CheckPort:     node.CheckPort,
		CheckHost:     node.CheckHost,
		CheckPath:     node.CheckPath,
		CheckTimeout:  node.CheckTimeout,
	})
}

func updateTierHealth(nodes []parentNodeInfo, states map[int64]*l2HealthState) map[string]bool {
	nodeSet := map[int64]struct{}{}
	snapshot := map[string]bool{}
	for _, node := range nodes {
		nodeSet[node.ID] = struct{}{}
		alive := isParentNodeAlive(node)
		state := states[node.ID]
		if state == nil {
			state = &l2HealthState{Online: true}
			states[node.ID] = state
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
	for id := range states {
		if _, ok := nodeSet[id]; !ok {
			delete(states, id)
		}
	}
	return snapshot
}

func writeParentStatusSnapshot(l1Snap, l2Snap map[string]bool) {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return
	}
	path := filepath.Join(rootDir, "conf", "parent_status.json")
	payload := map[string]interface{}{
		"updated_at": time.Now().Unix(),
		"l1":         l1Snap,
		"l2":         l2Snap,
	}
	if err := fsutil.WriteJSONAtomic(path, payload, true); err != nil {
		log.Printf("[Error] Parent status write failed: %v", err)
	}
}

func checkParentNodes() {
	if currentNodeLevel() != 3 {
		return
	}
	resp, err := fetchParentNodes()
	if err != nil {
		log.Printf("[Error] Parent Monitor fetch failed: %v", err)
		return
	}
	parentHealthStore.mu.Lock()
	l1Snap := updateTierHealth(resp.L1Nodes, parentHealthStore.l1States)
	l2Snap := updateTierHealth(resp.L2Nodes, parentHealthStore.l2States)
	l1Changed := !l2SnapshotEqual(l1Snap, parentHealthStore.lastL1Snap)
	l2Changed := !l2SnapshotEqual(l2Snap, parentHealthStore.lastL2Snap)
	if l1Changed {
		parentHealthStore.lastL1Snap = l1Snap
	}
	if l2Changed {
		parentHealthStore.lastL2Snap = l2Snap
	}
	parentHealthStore.mu.Unlock()
	if l1Changed || l2Changed {
		writeParentStatusSnapshot(l1Snap, l2Snap)
		refreshStreamConfigForParentStatus(l1Snap, l2Snap)
	}
}

func refreshStreamConfigForParentStatus(l1Snap, l2Snap map[string]bool) {
	rootDir := runtimeRoot()
	if rootDir == "" || CONFIG_PATH == "" {
		return
	}
	l1Status := map[int64]bool{}
	l2Status := map[int64]bool{}
	for key, val := range l1Snap {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			l1Status[id] = val
		}
	}
	for key, val := range l2Snap {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			l2Status[id] = val
		}
	}
	data, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		return
	}
	var cfg edgeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	if len(cfg.Streams) == 0 {
		return
	}
	hasParentStreams := false
	for _, stream := range cfg.Streams {
		mode := strings.ToLower(strings.TrimSpace(stream.ParentFetchMode))
		if stream.UseListenPort && (mode == "l1" || mode == "l2") {
			hasParentStreams = true
			break
		}
	}
	if !hasParentStreams {
		return
	}
	content := renderStreamConfig(cfg.Streams, streamStatusSnapshot{ParentL1: l1Status, ParentL2: l2Status})
	confPath := filepath.Join(rootDir, "conf", "dynamic", "stream.conf")
	existing, err := ioutil.ReadFile(confPath)
	if err == nil && string(existing) == content {
		return
	}
	if err := ioutil.WriteFile(confPath, []byte(content), 0644); err != nil {
		log.Printf("[Warn] Parent stream refresh failed: %v", err)
		return
	}
	if err := reloadNginxWithRollback(); err != nil {
		log.Printf("[Warn] Parent stream reload failed: %v", err)
	}
}
