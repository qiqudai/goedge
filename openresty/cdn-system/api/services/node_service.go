package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
)

type NodeService struct{}

func NewNodeService() *NodeService {
	return &NodeService{}
}

// SyncNodeToRedis is kept for compatibility; Redis-based sync is removed.
func (s *NodeService) SyncNodeToRedis(node *models.Node) error {
	return nil
}

// GetNodeIDByIP resolves IP to Node ID from DB.
func (s *NodeService) GetNodeIDByIP(ip string) (int64, error) {
	if db.DB == nil {
		return 0, errors.New("db unavailable")
	}
	var node models.Node
	if err := db.DB.Where("ip = ? AND enable = ?", ip, true).First(&node).Error; err != nil {
		return 0, err
	}
	return node.ID, nil
}
