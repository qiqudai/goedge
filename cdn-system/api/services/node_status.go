package services

import (
	"cdn-api/db"
	"cdn-api/models"
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
	if db.DB == nil {
		return toOffline
	}

	var enabledIDs []int64
	if err := db.DB.Model(&models.Node{}).
		Where("pid = 0 AND enable = ?", true).
		Pluck("id", &enabledIDs).Error; err != nil {
		return toOffline
	}

	successIDs := make([]int64, 0, len(enabledIDs))
	failedIDs := make([]int64, 0, len(enabledIDs))

	nodeStatusStore.mu.Lock()
	for _, nodeID := range enabledIDs {
		last, ok := nodeStatusStore.last[nodeID]
		onlineNow := ok && now.Sub(last) <= interval
		if onlineNow {
			nodeStatusStore.fail[nodeID] = 0
			nodeStatusStore.offline[nodeID] = false
			successIDs = append(successIDs, nodeID)
			continue
		}

		nodeStatusStore.fail[nodeID]++
		failedIDs = append(failedIDs, nodeID)
		if nodeStatusStore.fail[nodeID] >= maxFails && !nodeStatusStore.offline[nodeID] {
			nodeStatusStore.offline[nodeID] = true
			toOffline = append(toOffline, nodeID)
		}
	}
	nodeStatusStore.mu.Unlock()

	if len(successIDs) > 0 {
		WriteNodeMonitorLogs(successIDs, "heartbeat", true, map[int64]string{})
	}
	if len(failedIDs) > 0 {
		WriteNodeMonitorLogs(failedIDs, "heartbeat", false, map[int64]string{})
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
	// Node recovery does not change line membership or package CNAME policy.
	// DNS is refreshed only when line membership, node enable/delete, or node group
	// resolution names change.
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
