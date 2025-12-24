package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type NodeService struct{}

func NewNodeService() *NodeService {
	return &NodeService{}
}

// SyncNodeToRedis caches node data and updates IP mappings
func (s *NodeService) SyncNodeToRedis(node *models.Node) error {
	if db.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// 1. Cache Node Data
	nodeKey := fmt.Sprintf("CDN:NODE:DATA:%d", node.ID)
	data, _ := json.Marshal(node)
	db.Redis.Set(ctx, nodeKey, data, 0)

	// 2. Identify Active Node ID
	// If this is a sub-node (PID > 0), we map its IP to the PARENT node?
	// User said: "用子IP 则享受和主IP一样的配置" which implies they share config.
	// But `GetConfig` uses the passed NodeID.
	// If mapped IP -> SubNodeID, and SubNodeID has PID=ParentID, then GetConfig logic needs to handle that.
	// Let's assume we map IP -> The Node ID representing accurate Identity.
	// We map SubIP -> SubNodeID.
	// And `GetConfig` should check if PID > 0, then fetch Parent's config? Or fetch SubNode's config?
	// Usually SubNodes are just IP aliases.
	// Let's stick to IP -> NodeID (the ID of the record in DB).

	targetID := node.ID

	// 3. Update Main IP
	if node.IP != "" {
		ipKey := fmt.Sprintf("CDN:NODE:IP:%s", node.IP)
		db.Redis.Set(ctx, ipKey, targetID, 0)
	}

	// 4. Update Sub IPs
	// We need to fetch/parse SubIPs if they are not in the struct
	// Assuming SubIPs are passed in `node.SubIPs` (which is gorm:"-")
	// If caller didn't populate it, we might miss them.
	// For robustness, maybe Fetch from DB `replaceSubIPs` logic creates actual Node records with PID > 0.
	// So `SyncNodeToRedis` called on Parent won't see SubNodes unless we query them.
	// BUT, `node_controller` Create/Update Logic actually creates separate Node records for SubIPs.
	// So when we create a SubNode, `SyncNodeToRedis` will be called for THAT SubNode (if we hook it up right).

	// Wait, if SubNodes are real records in DB with PID>0.
	// Then `SyncNodeToRedis` called on SubNode will do:
	// IP -> SubNodeID.

	return nil
}

// GetNodeIDByIP resolves IP to Node ID from Redis
func (s *NodeService) GetNodeIDByIP(ip string) (int64, error) {
	if db.Redis == nil {
		return 0, fmt.Errorf("redis unavailable")
	}
	key := fmt.Sprintf("CDN:NODE:IP:%s", ip)
	val, err := db.Redis.Get(context.Background(), key).Result()
	if err != nil {
		return 0, err
	}
	id, _ := strconv.ParseInt(val, 10, 64)
	return id, nil
}
