package services

import (
	"sync"
	"time"
)

type nodeCooldown struct {
	until time.Time
}

var nodeCooldowns = struct {
	mu    sync.RWMutex
	items map[int64]nodeCooldown
}{
	items: map[int64]nodeCooldown{},
}

func MarkNodeRateLimited(nodeID int64, cooldown time.Duration) {
	if nodeID == 0 || cooldown <= 0 {
		return
	}
	nodeCooldowns.mu.Lock()
	defer nodeCooldowns.mu.Unlock()
	nodeCooldowns.items[nodeID] = nodeCooldown{until: time.Now().Add(cooldown)}
}

func IsNodeRateLimited(nodeID int64) bool {
	if nodeID == 0 {
		return false
	}
	nodeCooldowns.mu.RLock()
	item, ok := nodeCooldowns.items[nodeID]
	nodeCooldowns.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(item.until) {
		nodeCooldowns.mu.Lock()
		delete(nodeCooldowns.items, nodeID)
		nodeCooldowns.mu.Unlock()
		return false
	}
	return true
}
