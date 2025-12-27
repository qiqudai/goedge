package services

import (
	"sync"
	"time"
)

var nodeStatusStore = struct {
	mu   sync.RWMutex
	last map[int64]time.Time
}{
	last: map[int64]time.Time{},
}

func MarkNodeOnline(nodeID int64, at time.Time) {
	if nodeID <= 0 {
		return
	}
	nodeStatusStore.mu.Lock()
	nodeStatusStore.last[nodeID] = at
	nodeStatusStore.mu.Unlock()
}

func IsNodeOnline(nodeID int64, ttl time.Duration) bool {
	if nodeID <= 0 {
		return false
	}
	nodeStatusStore.mu.RLock()
	last, ok := nodeStatusStore.last[nodeID]
	nodeStatusStore.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(last) <= ttl
}
