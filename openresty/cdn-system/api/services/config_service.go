package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"time"
)

type ConfigService struct{}

// NewConfigService creates a new instance
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// GenerateConfigForNode constructs the configuration for a specific node
func (s *ConfigService) GenerateConfigForNode(nodeID string) (*models.EdgeConfig, error) {
	node, err := findNode(nodeID)
	if err != nil {
		return nil, err
	}

	payload := &models.EdgeConfig{
		Version:   0,
		NodeID:    strconv.FormatInt(node.ID, 10),
		Domains:   make([]models.EdgeDomain, 0),
		Upstreams: make([]models.EdgeUpstream, 0),
	}

	// 1. Find Node Groups this node belongs to
	// Query 'line' table for node_group_id
	var lines []models.Line
	if err := db.DB.Where("node_id = ?", node.ID).Find(&lines).Error; err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		// Node not assigned to any group? Return empty config
		payload.Version = hashConfigVersion(payload)
		return payload, nil
	}

	var groupIDs []int64
	for _, l := range lines {
		groupIDs = append(groupIDs, l.NodeGroupID)
	}

	// 2. Find Sites assigned to these Node Groups
	// In db.sql, 'site' table has 'node_group_id'
	// It also has 'backup_node_group' - we might want to include those too?
	// For now, let's just stick to primary node_group_id
	var sites []models.Site
	if err := db.DB.Where("node_group_id IN ?", groupIDs).Find(&sites).Error; err != nil {
		return nil, err
	}

	for _, site := range sites {
		if !site.Enable {
			continue
		}

		// NOTE: Site config in db.sql is complex (json fields).
		// We need to parse fields like Backend, HTTPListen, etc.
		// For this refactor, I will create a basic valid structure.
		// Real implementation needs robust JSON parsing for Site fields.

		upstreamKey := fmt.Sprintf("upstream_%d", site.ID)

		// Parse Backend JSON to get targets
		// db.sql: backend text
		// We'll need a struct for that in future. For now, assuming empty or simple.

		// MOCK: Add a dummy target for compilation/testing if backend is empty
		// In real world, we parse site.Backend
		targets := []models.EdgeUpstreamTarget{}
		// TODO: Parse site.Backend -> targets

		if len(targets) > 0 {
			payload.Upstreams = append(payload.Upstreams, models.EdgeUpstream{
				ID:      upstreamKey,
				Targets: targets,
			})
		}

		// Build Server Config
		// site.Domain is also JSON/text in db.sql? "domain text"
		// We need to parse it.

		domainConfig := models.EdgeDomain{
			Name:              site.CnameHostname, // Or parse site.Domain
			UpstreamKey:       upstreamKey,
			LoadBalancePolicy: "round_robin", // site.BalanceWay
			Status:            "active",
		}
		payload.Domains = append(payload.Domains, domainConfig)
	}

	// Stable version based on config content to avoid unnecessary reloads
	payload.Version = hashConfigVersion(payload)

	return payload, nil
}

func hashConfigVersion(cfg *models.EdgeConfig) int64 {
	clone := *cfg
	clone.Version = 0
	b, err := json.Marshal(clone)
	if err != nil {
		return time.Now().Unix()
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return int64(h.Sum64())
}

func findNode(nodeID string) (*models.Node, error) {
	if nodeID == "" {
		return nil, errors.New("node_id is required")
	}

	var node models.Node
	if id, err := strconv.ParseInt(nodeID, 10, 64); err == nil {
		if err := db.DB.Where("id = ?", id).First(&node).Error; err == nil {
			return &node, nil
		}
	}

	if err := db.DB.Where("name = ?", nodeID).First(&node).Error; err != nil {
		return nil, errors.New("node not found")
	}
	return &node, nil
}
