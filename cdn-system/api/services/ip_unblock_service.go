package services

import (
	"encoding/json"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
)

const ipUnblockQueueKey = "edge_ip_unblock_queue"

type ipUnblockQueue struct {
	Rev int64    `json:"rev"`
	IPs []string `json:"ips"`
}

func normalizeUnblockIPs(ips []string) []string {
	if len(ips) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ips))
	out := make([]string, 0, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	return out
}

func loadIPUnblockQueue() ipUnblockQueue {
	var cfg models.SysConfig
	if err := db.DB.Where("name = ? AND type = ?", ipUnblockQueueKey, "system").First(&cfg).Error; err != nil {
		return ipUnblockQueue{}
	}
	var queue ipUnblockQueue
	if strings.TrimSpace(cfg.Value) == "" {
		return queue
	}
	_ = json.Unmarshal([]byte(cfg.Value), &queue)
	queue.IPs = normalizeUnblockIPs(queue.IPs)
	return queue
}

func saveIPUnblockQueue(queue ipUnblockQueue) error {
	queue.IPs = normalizeUnblockIPs(queue.IPs)
	raw, err := json.Marshal(queue)
	if err != nil {
		return err
	}
	now := time.Now()
	var cfg models.SysConfig
	err = db.DB.Where("name = ? AND type = ?", ipUnblockQueueKey, "system").First(&cfg).Error
	if err != nil {
		cfg = models.SysConfig{
			Name:      ipUnblockQueueKey,
			Type:      "system",
			ScopeID:   0,
			ScopeName: "global",
			Value:     string(raw),
			Enable:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return db.DB.Create(&cfg).Error
	}
	return db.DB.Model(&models.SysConfig{}).
		Where("name = ? AND type = ?", ipUnblockQueueKey, "system").
		Updates(map[string]interface{}{
			"value":     string(raw),
			"update_at": now,
		}).Error
}

// EnqueueIPUnblock records IPs to remove from edge shared blacklist and bumps config version.
func EnqueueIPUnblock(ips []string) (int64, error) {
	ips = normalizeUnblockIPs(ips)
	if len(ips) == 0 {
		return 0, nil
	}
	queue := loadIPUnblockQueue()
	merged := normalizeUnblockIPs(append(queue.IPs, ips...))
	rev := time.Now().UnixNano()
	if rev <= queue.Rev {
		rev = queue.Rev + 1
	}
	next := ipUnblockQueue{Rev: rev, IPs: merged}
	if err := saveIPUnblockQueue(next); err != nil {
		return 0, err
	}
	BumpConfigVersion("ip_unblock", nil)
	createIPUnblockTask(next)
	return rev, nil
}

// SnapshotIPUnblock returns the current unblock directive for edge config payloads.
func SnapshotIPUnblock() *models.EdgeIPUnblock {
	queue := loadIPUnblockQueue()
	if queue.Rev == 0 || len(queue.IPs) == 0 {
		return nil
	}
	return &models.EdgeIPUnblock{
		Rev: queue.Rev,
		IPs: append([]string(nil), queue.IPs...),
	}
}

func createIPUnblockTask(queue ipUnblockQueue) {
	if queue.Rev == 0 || len(queue.IPs) == 0 {
		return
	}
	payload, err := json.Marshal(queue)
	if err != nil {
		return
	}
	now := time.Now()
	task := models.Task{
		Type:     "ip_unblock",
		State:    "waiting",
		Enable:   true,
		Data:     string(payload),
		CreateAt: now,
		StartAt:  &now,
		EndAt:    &now,
		RetryAt:  &now,
	}
	nodeIDs := ConnectedNodeIDs()
	if len(nodeIDs) > 0 {
		targets := NewTaskTargets(nodeIDs)
		task.TargetsJSON = targets.Marshal()
	}
	if err := db.DB.Create(&task).Error; err == nil {
		TriggerDispatchPending()
	}
}
