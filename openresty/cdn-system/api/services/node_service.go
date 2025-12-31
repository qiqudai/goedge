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

// GetUserNodes returns all node IDs associated with a user's packages
func (s *NodeService) GetUserNodes(userID int64) ([]int64, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}

	// 1. Get User Packages
	var userPkgs []models.UserPackage
	// TODO: check expiry
	if err := db.DB.Where("uid = ?", userID).Find(&userPkgs).Error; err != nil {
		return nil, err
	}
	if len(userPkgs) == 0 {
		return []int64{}, nil
	}

	pkgIDs := make([]int64, 0)
	for _, up := range userPkgs {
		pkgIDs = append(pkgIDs, up.PackageID)
	}

	// 2. Get Packages to find Regions/Groups
	var pkgs []models.Package
	if err := db.DB.Where("id IN ?", pkgIDs).Find(&pkgs).Error; err != nil {
		return nil, err
	}

	regionIDs := make([]int64, 0)
	groupIDs := make([]int64, 0)

	for _, p := range pkgs {
		if p.RegionID > 0 {
			regionIDs = append(regionIDs, p.RegionID)
		}
		if p.NodeGroupID > 0 {
			groupIDs = append(groupIDs, p.NodeGroupID)
		}
	}

	// 3. Find Nodes (Active)
	var nodes []models.Node
	query := db.DB.Where("enable = ?", true)
	
	conds := db.DB.Where("1=0") // Start with False
	if len(regionIDs) > 0 {
		conds = conds.Or("region_id IN ?", regionIDs)
	}
	if len(groupIDs) > 0 {
		conds = conds.Or("group_id IN ?", groupIDs)
	}
	// Note: If no regions/groups, conds is False?
	// If a user has a package with NO region/group (e.g. global), does it mean ALL nodes?
	// Usually Package enforces constraint. Assuming strict binding.
	if len(regionIDs) == 0 && len(groupIDs) == 0 {
		return []int64{}, nil
	}

	if err := query.Where(conds).Find(&nodes).Error; err != nil {
		return nil, err
	}

	nodeIDs := make([]int64, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID)
	}
	return nodeIDs, nil
}
