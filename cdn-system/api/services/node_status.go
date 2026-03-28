package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"log"
	"sync"
	"time"
)

var nodeStatusStore = struct {
	mu      sync.RWMutex
	last    map[int64]time.Time
	fail    map[int64]int
	offline map[int64]bool
}{
	last:    map[int64]time.Time{},
	fail:    map[int64]int{},
	offline: map[int64]bool{},
}

var lastMissingCheck time.Time
const missingCheckInterval = 30 * time.Second

func MarkNodeOnline(nodeID int64, at time.Time) {
	if nodeID <= 0 {
		return
	}
	nodeStatusStore.mu.Lock()
	wasOffline := nodeStatusStore.offline[nodeID]
	nodeStatusStore.last[nodeID] = at
	nodeStatusStore.fail[nodeID] = 0
	nodeStatusStore.offline[nodeID] = false
	nodeStatusStore.mu.Unlock()
	if wasOffline {
		go HandleNodeRecover(nodeID)
	}
}

func IsNodeOnline(nodeID int64, ttl time.Duration) bool {
	if nodeID <= 0 {
		return false
	}
	nodeStatusStore.mu.RLock()
	last, ok := nodeStatusStore.last[nodeID]
	offline := nodeStatusStore.offline[nodeID]
	nodeStatusStore.mu.RUnlock()
	if !ok {
		return false
	}
	if offline {
		return false
	}
	return time.Since(last) <= ttl
}

func EvaluateNodeHealth(interval time.Duration, maxFails int) []int64 {
	now := time.Now()
	toOffline := make([]int64, 0)
	nodeStatusStore.mu.Lock()
	for nodeID, last := range nodeStatusStore.last {
		if now.Sub(last) <= interval {
			nodeStatusStore.fail[nodeID] = 0
			nodeStatusStore.offline[nodeID] = false
			continue
		}
		nodeStatusStore.fail[nodeID]++
		if nodeStatusStore.fail[nodeID] >= maxFails && !nodeStatusStore.offline[nodeID] {
			nodeStatusStore.offline[nodeID] = true
			toOffline = append(toOffline, nodeID)
		}
	}
	nodeStatusStore.mu.Unlock()
	if db.DB != nil && (lastMissingCheck.IsZero() || now.Sub(lastMissingCheck) >= missingCheckInterval) {
		lastMissingCheck = now
		var ids []int64
		if err := db.DB.Model(&models.Node{}).
			Where("pid = 0 AND enable = ?", true).
			Pluck("id", &ids).Error; err == nil {
			nodeStatusStore.mu.Lock()
			for _, nodeID := range ids {
				if _, ok := nodeStatusStore.last[nodeID]; ok {
					continue
				}
				if nodeStatusStore.offline[nodeID] {
					continue
				}
				nodeStatusStore.offline[nodeID] = true
				toOffline = append(toOffline, nodeID)
			}
			nodeStatusStore.mu.Unlock()
		}
	}
	return toOffline
}

func HandleNodeOffline(nodeID int64) {
	if nodeID <= 0 || db.DB == nil {
		return
	}
	// Intentionally skip DNS record removal when a node goes offline.
}

func HandleNodeRecover(nodeID int64) {
	if nodeID <= 0 || db.DB == nil {
		return
	}
	go func(id int64) {
		if err := SyncPackageCnameForNodes([]int64{id}, "add"); err != nil {
			log.Printf("[DNS] package cname recover sync failed node=%d err=%v", id, err)
		}
	}(nodeID)
	var lines []models.Line
	if err := db.DB.Where("(node_ip_id = ? OR node_id = ?) AND enable = ?", nodeID, nodeID, true).Find(&lines).Error; err != nil {
		return
	}
	if len(lines) == 0 {
		return
	}
	type key struct {
		groupID  int64
		lineID   string
		lineName string
	}
	groupIPIDs := map[key][]int64{}
	for _, line := range lines {
		k := key{groupID: line.NodeGroupID, lineID: line.LineID, lineName: line.LineName}
		if line.NodeIPID != 0 {
			groupIPIDs[k] = append(groupIPIDs[k], line.NodeIPID)
		}
	}
	for k, ipIDs := range groupIPIDs {
		ipIDs = uniqueInt64List(ipIDs)
		if len(ipIDs) == 0 {
			continue
		}
		_ = dns.SyncLineRecords(k.groupID, k.lineID, k.lineName, "add", ipIDs)
	}
}

func uniqueInt64List(items []int64) []int64 {
	if len(items) == 0 {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	for _, id := range items {
		if id != 0 {
			seen[id] = struct{}{}
		}
	}
	result := make([]int64, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result
}
