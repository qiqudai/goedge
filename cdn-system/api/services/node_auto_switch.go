package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	nodeAutoSwitchInterval    = 10 * time.Second
	nodeAutoSwitchConfigKey   = "node_monitor_config"
	autoWeightTagPrefix       = "auto_weight:"
	autoWeightEmptyTag        = "__empty__"
	defaultAutoSwitchPercent  = 90
	defaultAutoSwitchDuration = 30
	defaultAutoSwitchRecover  = 300
	defaultAutoSwitchWeight   = 1
)

type nodeAutoSwitchConfig struct {
	AutoSwitchEnable    bool `json:"auto_switch_enable"`
	AutoSwitchThreshold int  `json:"auto_switch_threshold"`
	AutoSwitchDuration  int  `json:"auto_switch_duration"`
	AutoSwitchRecover   int  `json:"auto_switch_recover"`
	AutoSwitchMinWeight int  `json:"auto_switch_min_weight"`
}

type nodeAutoSwitchState struct {
	OverSince    time.Time
	RecoverSince time.Time
	Applied      bool
}

var nodeAutoSwitchStore = struct {
	mu    sync.Mutex
	nodes map[int64]*nodeAutoSwitchState
}{
	nodes: map[int64]*nodeAutoSwitchState{},
}

var nodeAutoSwitchWarnOnce sync.Once

func StartNodeAutoSwitchWorker() {
	go func() {
		runNodeAutoSwitch()
		ticker := time.NewTicker(nodeAutoSwitchInterval)
		defer ticker.Stop()
		for range ticker.C {
			runNodeAutoSwitch()
		}
	}()
}

func runNodeAutoSwitch() {
	if db.DB == nil {
		return
	}
	cfg := loadNodeAutoSwitchConfig()
	if !cfg.AutoSwitchEnable {
		return
	}
	if !db.ClickHouseEnabled() {
		nodeAutoSwitchWarnOnce.Do(func() {
			log.Printf("[AutoSwitch] skipped: ClickHouse disabled")
		})
		return
	}

	nodes := loadAutoSwitchNodes()
	if len(nodes) == 0 {
		return
	}

	window := time.Duration(cfg.AutoSwitchDuration) * time.Second
	if window <= 0 {
		window = defaultAutoSwitchDuration * time.Second
	}
	since := time.Now().Add(-window)
	threshold := float64(cfg.AutoSwitchThreshold) / 100
	if threshold <= 0 || threshold > 1 {
		threshold = float64(defaultAutoSwitchPercent) / 100
	}
	minWeight := cfg.AutoSwitchMinWeight
	if minWeight <= 0 {
		minWeight = defaultAutoSwitchWeight
	}

	cpuUsage := loadNodeCPUUsage(since)
	memUsage := loadNodeMemUsage(since)
	bwRate := loadNodeBandwidthRate(since, window)
	lineMap := loadNodeAutoSwitchLines(nodes)

	now := time.Now()
	nodeSet := map[int64]struct{}{}
	toApply := make([]int64, 0)
	toRestore := make([]int64, 0)

	nodeAutoSwitchStore.mu.Lock()
	for _, node := range nodes {
		nodeSet[node.ID] = struct{}{}
		st := nodeAutoSwitchStore.nodes[node.ID]
		if st == nil {
			st = &nodeAutoSwitchState{}
			nodeAutoSwitchStore.nodes[node.ID] = st
		}

		cpuVal, cpuOk := cpuUsage[node.ID]
		memVal, memOk := memUsage[node.ID]
		bwVal, bwOk := bwRate[node.ID]
		bwLimit := parseBandwidthMbps(node.BwLimit)
		bwLimitBytes := float64(mbpsToLimitRate(bwLimit))

		hasMetric := cpuOk || memOk || (bwOk && bwLimitBytes > 0)
		if !hasMetric {
			continue
		}

		over := false
		if cpuOk && cpuVal >= threshold {
			over = true
		}
		if memOk && memVal >= threshold {
			over = true
		}
		if bwOk && bwLimitBytes > 0 && bwVal >= bwLimitBytes*threshold {
			over = true
		}

		if over {
			st.RecoverSince = time.Time{}
			if st.OverSince.IsZero() {
				st.OverSince = now
			}
		} else {
			st.OverSince = time.Time{}
			if st.Applied {
				if st.RecoverSince.IsZero() {
					st.RecoverSince = now
				}
			} else {
				st.RecoverSince = time.Time{}
			}
		}

		if over && !st.Applied && !st.OverSince.IsZero() && now.Sub(st.OverSince) >= time.Duration(cfg.AutoSwitchDuration)*time.Second {
			toApply = append(toApply, node.ID)
		}
		if !over && st.Applied && !st.RecoverSince.IsZero() && now.Sub(st.RecoverSince) >= time.Duration(cfg.AutoSwitchRecover)*time.Second {
			toRestore = append(toRestore, node.ID)
		}
	}
	for nodeID := range nodeAutoSwitchStore.nodes {
		if _, ok := nodeSet[nodeID]; !ok {
			delete(nodeAutoSwitchStore.nodes, nodeID)
		}
	}
	nodeAutoSwitchStore.mu.Unlock()

	applied := map[int64]bool{}
	for _, nodeID := range toApply {
		lines := lineMap[nodeID]
		if len(lines) == 0 {
			continue
		}
		if changed, err := reduceNodeLineWeight(nodeID, lines, minWeight); err != nil {
			log.Printf("[AutoSwitch] weight down failed node=%d err=%v", nodeID, err)
		} else if changed {
			applied[nodeID] = true
		}
	}

	restored := map[int64]bool{}
	for _, nodeID := range toRestore {
		lines := lineMap[nodeID]
		if len(lines) == 0 {
			continue
		}
		if changed, err := restoreNodeLineWeight(nodeID, lines); err != nil {
			log.Printf("[AutoSwitch] weight restore failed node=%d err=%v", nodeID, err)
		} else if changed {
			restored[nodeID] = true
		}
	}

	if len(applied) == 0 && len(restored) == 0 {
		return
	}

	nodeAutoSwitchStore.mu.Lock()
	for nodeID := range applied {
		if st := nodeAutoSwitchStore.nodes[nodeID]; st != nil {
			st.Applied = true
			st.RecoverSince = time.Time{}
		}
	}
	for nodeID := range restored {
		if st := nodeAutoSwitchStore.nodes[nodeID]; st != nil {
			st.Applied = false
			st.OverSince = time.Time{}
			st.RecoverSince = time.Time{}
		}
	}
	nodeAutoSwitchStore.mu.Unlock()
}

func loadAutoSwitchNodes() []models.Node {
	var nodes []models.Node
	_ = db.DB.Select("id", "bw_limit", "enable").
		Where("pid = 0 AND enable = ?", true).
		Find(&nodes).Error
	return nodes
}

func loadNodeAutoSwitchConfig() nodeAutoSwitchConfig {
	cfg := nodeAutoSwitchConfig{
		AutoSwitchEnable:    false,
		AutoSwitchThreshold: defaultAutoSwitchPercent,
		AutoSwitchDuration:  defaultAutoSwitchDuration,
		AutoSwitchRecover:   defaultAutoSwitchRecover,
		AutoSwitchMinWeight: defaultAutoSwitchWeight,
	}
	if db.DB == nil {
		return cfg
	}
	var sysConfig models.SysConfig
	if err := db.DB.Where("name = ? AND type = ?", nodeAutoSwitchConfigKey, "system").First(&sysConfig).Error; err != nil {
		return cfg
	}
	_ = json.Unmarshal([]byte(sysConfig.Value), &cfg)
	if cfg.AutoSwitchThreshold <= 0 || cfg.AutoSwitchThreshold > 100 {
		cfg.AutoSwitchThreshold = defaultAutoSwitchPercent
	}
	if cfg.AutoSwitchDuration <= 0 {
		cfg.AutoSwitchDuration = defaultAutoSwitchDuration
	}
	if cfg.AutoSwitchRecover < defaultAutoSwitchRecover {
		cfg.AutoSwitchRecover = defaultAutoSwitchRecover
	}
	if cfg.AutoSwitchMinWeight <= 0 {
		cfg.AutoSwitchMinWeight = defaultAutoSwitchWeight
	}
	return cfg
}

func loadNodeCPUUsage(since time.Time) map[int64]float64 {
	result := map[int64]float64{}
	if !db.ClickHouseEnabled() {
		return result
	}
	query := `
SELECT node_id,
       sumIf(delta, labels LIKE '%mode="idle"%') AS idle_delta,
       sum(delta) AS total_delta
FROM (
  SELECT node_id, labels, argMax(value, ts) - argMin(value, ts) AS delta
  FROM node_metrics
  WHERE metric = 'node_cpu_seconds_total' AND ts >= ?
  GROUP BY node_id, labels
)
GROUP BY node_id`
	rows, err := db.CK.Query(query, since)
	if err != nil {
		log.Printf("[AutoSwitch] cpu query failed: %v", err)
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var nodeIDStr string
		var idleDelta float64
		var totalDelta float64
		if err := rows.Scan(&nodeIDStr, &idleDelta, &totalDelta); err != nil {
			continue
		}
		nodeID, err := strconv.ParseInt(strings.TrimSpace(nodeIDStr), 10, 64)
		if err != nil || nodeID <= 0 || totalDelta <= 0 {
			continue
		}
		usage := 1 - idleDelta/totalDelta
		if usage < 0 {
			usage = 0
		}
		if usage > 1 {
			usage = 1
		}
		result[nodeID] = usage
	}
	return result
}

func loadNodeMemUsage(since time.Time) map[int64]float64 {
	type memValues struct {
		total float64
		avail float64
	}
	values := map[int64]*memValues{}
	if !db.ClickHouseEnabled() {
		return map[int64]float64{}
	}
	query := `
SELECT node_id, metric, argMax(value, ts) AS val
FROM node_metrics
WHERE metric IN ('node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes') AND ts >= ?
GROUP BY node_id, metric`
	rows, err := db.CK.Query(query, since)
	if err != nil {
		log.Printf("[AutoSwitch] mem query failed: %v", err)
		return map[int64]float64{}
	}
	defer rows.Close()
	for rows.Next() {
		var nodeIDStr string
		var metric string
		var val float64
		if err := rows.Scan(&nodeIDStr, &metric, &val); err != nil {
			continue
		}
		nodeID, err := strconv.ParseInt(strings.TrimSpace(nodeIDStr), 10, 64)
		if err != nil || nodeID <= 0 {
			continue
		}
		item := values[nodeID]
		if item == nil {
			item = &memValues{}
			values[nodeID] = item
		}
		switch metric {
		case "node_memory_MemTotal_bytes":
			item.total = val
		case "node_memory_MemAvailable_bytes":
			item.avail = val
		}
	}
	result := map[int64]float64{}
	for nodeID, item := range values {
		if item.total <= 0 {
			continue
		}
		usage := 1 - item.avail/item.total
		if usage < 0 {
			usage = 0
		}
		if usage > 1 {
			usage = 1
		}
		result[nodeID] = usage
	}
	return result
}

func loadNodeBandwidthRate(since time.Time, window time.Duration) map[int64]float64 {
	result := map[int64]float64{}
	if !db.ClickHouseEnabled() {
		return result
	}
	windowSeconds := window.Seconds()
	if windowSeconds <= 0 {
		windowSeconds = float64(defaultAutoSwitchDuration)
	}
	query := `
SELECT node_id, metric, sum(delta) AS delta_bytes
FROM (
  SELECT node_id, metric, labels, argMax(value, ts) - argMin(value, ts) AS delta
  FROM node_metrics
  WHERE metric IN ('node_network_receive_bytes_total', 'node_network_transmit_bytes_total')
    AND ts >= ?
    AND labels NOT LIKE '%device="lo"%'
  GROUP BY node_id, metric, labels
)
GROUP BY node_id, metric`
	rows, err := db.CK.Query(query, since)
	if err != nil {
		log.Printf("[AutoSwitch] bandwidth query failed: %v", err)
		return result
	}
	defer rows.Close()
	type bwDelta struct {
		rx float64
		tx float64
	}
	deltas := map[int64]*bwDelta{}
	for rows.Next() {
		var nodeIDStr string
		var metric string
		var delta float64
		if err := rows.Scan(&nodeIDStr, &metric, &delta); err != nil {
			continue
		}
		nodeID, err := strconv.ParseInt(strings.TrimSpace(nodeIDStr), 10, 64)
		if err != nil || nodeID <= 0 {
			continue
		}
		if delta < 0 {
			delta = 0
		}
		item := deltas[nodeID]
		if item == nil {
			item = &bwDelta{}
			deltas[nodeID] = item
		}
		switch metric {
		case "node_network_receive_bytes_total":
			item.rx += delta
		case "node_network_transmit_bytes_total":
			item.tx += delta
		}
	}
	for nodeID, item := range deltas {
		result[nodeID] = (item.rx + item.tx) / windowSeconds
	}
	return result
}

func loadNodeAutoSwitchLines(nodes []models.Node) map[int64][]models.Line {
	result := map[int64][]models.Line{}
	if db.DB == nil || len(nodes) == 0 {
		return result
	}
	parentIDs := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		if node.ID != 0 {
			parentIDs = append(parentIDs, node.ID)
		}
	}
	parentIDs = uniqueInt64List(parentIDs)
	if len(parentIDs) == 0 {
		return result
	}

	var subNodes []models.Node
	_ = db.DB.Select("id", "pid").
		Where("pid IN ?", parentIDs).
		Find(&subNodes).Error
	parentMap := map[int64]int64{}
	subIDs := make([]int64, 0, len(subNodes))
	for _, sub := range subNodes {
		if sub.ID == 0 || sub.PID == 0 {
			continue
		}
		parentMap[sub.ID] = sub.PID
		subIDs = append(subIDs, sub.ID)
	}

	nodeIDs := append([]int64{}, parentIDs...)
	nodeIDs = append(nodeIDs, subIDs...)
	nodeIDs = uniqueInt64List(nodeIDs)
	if len(nodeIDs) == 0 {
		return result
	}

	var lines []models.Line
	_ = db.DB.Where("node_id IN ? OR node_ip_id IN ?", nodeIDs, nodeIDs).Find(&lines).Error
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		rootID := nodeID
		if parent, ok := parentMap[nodeID]; ok {
			rootID = parent
		}
		if rootID == 0 {
			continue
		}
		result[rootID] = append(result[rootID], line)
	}
	return result
}

func reduceNodeLineWeight(nodeID int64, lines []models.Line, minWeight int) (bool, error) {
	if len(lines) == 0 || minWeight <= 0 {
		return false, nil
	}
	now := time.Now()
	targets := make([]models.Line, 0)
	groupLineMap := map[int64]map[lineKey]struct{}{}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range lines {
			if !line.Enable {
				continue
			}
			if _, ok := parseAutoWeightTag(line.DisableBy); ok {
				continue
			}
			origWeight := strings.TrimSpace(line.Weight)
			if origWeight == strconv.Itoa(minWeight) {
				continue
			}
			marker := buildAutoWeightTag(origWeight)
			if err := tx.Model(&models.Line{}).Where("id = ?", line.ID).Updates(map[string]interface{}{
				"weight":     strconv.Itoa(minWeight),
				"disable_by": marker,
				"update_at":  now,
			}).Error; err != nil {
				return err
			}
			targets = append(targets, line)
			addGroupLineKey(groupLineMap, line)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if len(targets) == 0 {
		return false, nil
	}

	WriteIPSwitchLogsForLines(targets, "auto_weight_down", "line")
	syncAutoWeightLines(groupLineMap)
	log.Printf("[AutoSwitch] node=%d weight down lines=%d", nodeID, len(targets))
	return true, nil
}

func restoreNodeLineWeight(nodeID int64, lines []models.Line) (bool, error) {
	if len(lines) == 0 {
		return false, nil
	}
	now := time.Now()
	targets := make([]models.Line, 0)
	groupLineMap := map[int64]map[lineKey]struct{}{}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range lines {
			origWeight, ok := parseAutoWeightTag(line.DisableBy)
			if !ok {
				continue
			}
			if err := tx.Model(&models.Line{}).Where("id = ?", line.ID).Updates(map[string]interface{}{
				"weight":     origWeight,
				"disable_by": "",
				"update_at":  now,
			}).Error; err != nil {
				return err
			}
			targets = append(targets, line)
			addGroupLineKey(groupLineMap, line)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if len(targets) == 0 {
		return false, nil
	}

	WriteIPSwitchLogsForLines(targets, "auto_weight_restore", "line")
	syncAutoWeightLines(groupLineMap)
	log.Printf("[AutoSwitch] node=%d weight restored lines=%d", nodeID, len(targets))
	return true, nil
}

func syncAutoWeightLines(groupLineMap map[int64]map[lineKey]struct{}) {
	groupIDs := make([]int64, 0, len(groupLineMap))
	for groupID := range groupLineMap {
		groupIDs = append(groupIDs, groupID)
	}
	groupIDs = uniqueInt64List(groupIDs)
	if len(groupIDs) > 0 {
		BumpConfigVersion("line", groupIDs)
	}
	for groupID, lineSet := range groupLineMap {
		for key := range lineSet {
			if err := dns.SyncLineRecords(groupID, key.ID, key.Name, "resync", nil); err != nil {
				log.Printf("[AutoSwitch] dns sync failed group=%d line=%s err=%v", groupID, key.ID, err)
			}
			if err := SyncPackageCnameForLineChange(groupID, key.ID, key.Name, nil, "resync"); err != nil {
				log.Printf("[AutoSwitch] package cname sync failed group=%d line=%s err=%v", groupID, key.ID, err)
			}
		}
	}
}

func buildAutoWeightTag(orig string) string {
	orig = strings.TrimSpace(orig)
	if orig == "" {
		orig = autoWeightEmptyTag
	}
	return autoWeightTagPrefix + orig
}

func parseAutoWeightTag(tag string) (string, bool) {
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, autoWeightTagPrefix) {
		return "", false
	}
	orig := strings.TrimPrefix(tag, autoWeightTagPrefix)
	if orig == autoWeightEmptyTag {
		return "", true
	}
	return orig, true
}

func addGroupLineKey(groupLineMap map[int64]map[lineKey]struct{}, line models.Line) {
	if groupLineMap == nil || line.NodeGroupID == 0 {
		return
	}
	key := lineKey{ID: strings.TrimSpace(line.LineID), Name: strings.TrimSpace(line.LineName)}
	if groupLineMap[line.NodeGroupID] == nil {
		groupLineMap[line.NodeGroupID] = map[lineKey]struct{}{}
	}
	groupLineMap[line.NodeGroupID][key] = struct{}{}
}
