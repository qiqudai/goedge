package services

import (
	"errors"
	"strconv"
	"strings"

	"cdn-api/db"
	"cdn-api/models"
)

const (
	NodeConfigParentGroupID   = "parent_group_id"
	NodeConfigParentFetchMode = "parent_fetch_mode"
	NodeConfigL1RespectL2     = "l1_respect_l2"
)

const (
	ParentFetchOrigin = "origin"
	ParentFetchL1     = "l1"
	ParentFetchL2     = "l2"
)

type ParentFetchConfig struct {
	ParentGroupID   int64
	ParentFetchMode string
	L1RespectL2     bool
}

func NormalizeParentFetchMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case ParentFetchL1, ParentFetchL2:
		return mode
	default:
		return ParentFetchOrigin
	}
}

func LoadParentFetchConfig(nodeID int64) ParentFetchConfig {
	cfg := ParentFetchConfig{
		ParentFetchMode: ParentFetchOrigin,
		L1RespectL2:     true,
	}
	if nodeID == 0 {
		return cfg
	}
	if raw, err := GetNodeConfigValue(nodeID, NodeConfigParentGroupID); err == nil {
		if id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
			cfg.ParentGroupID = id
		}
	}
	if raw, err := GetNodeConfigValue(nodeID, NodeConfigParentFetchMode); err == nil {
		cfg.ParentFetchMode = NormalizeParentFetchMode(raw)
	}
	if raw, err := GetNodeConfigValue(nodeID, NodeConfigL1RespectL2); err == nil && strings.TrimSpace(raw) != "" {
		cfg.L1RespectL2 = ParseBoolFlag(raw)
	}
	return cfg
}

func SaveParentFetchConfig(nodeID int64, cfg ParentFetchConfig) error {
	if nodeID == 0 {
		return nil
	}
	mode := NormalizeParentFetchMode(cfg.ParentFetchMode)
	if err := UpsertNodeConfigItem(nodeID, NodeConfigParentFetchMode, mode); err != nil {
		return err
	}
	groupRaw := ""
	if cfg.ParentGroupID > 0 {
		groupRaw = strconv.FormatInt(cfg.ParentGroupID, 10)
	}
	if err := UpsertNodeConfigItem(nodeID, NodeConfigParentGroupID, groupRaw); err != nil {
		return err
	}
	l1Respect := "true"
	if !cfg.L1RespectL2 {
		l1Respect = "false"
	}
	return UpsertNodeConfigItem(nodeID, NodeConfigL1RespectL2, l1Respect)
}

func AttachParentFetchToNodes(nodes []models.Node) {
	if len(nodes) == 0 {
		return
	}
	groupMap, _ := GetNodeConfigMap(NodeConfigParentGroupID)
	modeMap, _ := GetNodeConfigMap(NodeConfigParentFetchMode)
	respectMap, _ := GetNodeConfigMap(NodeConfigL1RespectL2)
	for i := range nodes {
		if nodes[i].Level != 3 {
			continue
		}
		if raw, ok := groupMap[nodes[i].ID]; ok {
			if id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
				nodes[i].ParentGroupID = id
			}
		}
		if raw, ok := modeMap[nodes[i].ID]; ok {
			nodes[i].ParentFetchMode = NormalizeParentFetchMode(raw)
		} else {
			nodes[i].ParentFetchMode = ParentFetchOrigin
		}
		if raw, ok := respectMap[nodes[i].ID]; ok && strings.TrimSpace(raw) != "" {
			nodes[i].L1RespectL2 = ParseBoolFlag(raw)
		} else {
			nodes[i].L1RespectL2 = true
		}
	}
}

func ValidateParentFetchConfig(level int, cfg ParentFetchConfig) error {
	if level != 3 {
		return nil
	}
	mode := NormalizeParentFetchMode(cfg.ParentFetchMode)
	if mode == ParentFetchOrigin {
		return nil
	}
	if cfg.ParentGroupID <= 0 {
		return ErrParentGroupRequired
	}
	return nil
}

var ErrParentGroupRequired = errors.New("parent_group_id is required for l1/l2 parent fetch mode")

func CountParentGroupReferences(groupID int64) (int64, error) {
	if groupID <= 0 || db.DB == nil {
		return 0, nil
	}
	groupRaw := strconv.FormatInt(groupID, 10)
	var count int64
	err := db.DB.Model(&models.ConfigItem{}).
		Where("type = ? AND scope_name = ? AND name = ? AND value = ?", "node_config", "node", NodeConfigParentGroupID, groupRaw).
		Count(&count).Error
	return count, err
}
