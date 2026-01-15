package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
	"encoding/json"
	"strconv"
	"time"
)

type UserPackageService struct{}

func NewUserPackageService() *UserPackageService {
	return &UserPackageService{}
}

func (s *UserPackageService) UserHasCustomCCRule(userID int64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var packs []models.UserPackage
	if err := db.DB.Where("uid = ? AND custom_cc_rule = ?", userID, true).Find(&packs).Error; err != nil {
		return false, err
	}
	if len(packs) == 0 {
		return false, nil
	}
	now := time.Now()
	for _, pack := range packs {
		if pack.EndAt.IsZero() || pack.EndAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}

// SyncUserPackage creates tasks to sync user package config to nodes
func (s *UserPackageService) SyncUserPackage(userPackageID int64, trigger string) error {
	// 1. Get UserPackage
	var up models.UserPackage
	if err := db.DB.First(&up, userPackageID).Error; err != nil {
		return err
	}

	// 2. Increment Version
	up.Version++
	if err := db.DB.Model(&up).Update("version", up.Version).Error; err != nil {
		return err
	}

	// 3. Build Agent Config
	agentConfig := models.AgentPackageConfig{
		PackageID:       int32(up.ID),
		UID:             up.UserID,
		Version:         up.Version,
		Status:          "active",
		RegionID:        int32(up.RegionID),
		NodeGroupID:     int32(up.NodeGroupID),
		BackupNodeGroup: int32(up.BackupNodeGroup),
		EnableBackup:    0,
		Cname: models.AgentCnameInfo{
			Domain:    up.CnameDomain,
			Hostname:  up.CnameHostname,
			Hostname2: up.CnameHostname2,
			Mode:      up.CnameMode,
			RecordID:  up.RecordID,
		},
		Limits: models.AgentLimits{
			Traffic:    up.Traffic,
			Bandwidth:  up.Bandwidth,
			Connection: up.Connection,
			Domain:     up.DomainLimit,
		},
		Features: models.AgentFeatures{
			HTTPPort:     int(up.HTTPPortLimit),
			StreamPort:   int(up.StreamPortLimit),
			Websocket:    up.Websocket,
			CustomCCRule: up.CustomCCRule,
			L2Origin:     up.L2Origin,
		},
		Time: models.AgentTime{
			StartAt: up.StartAt.Format("2006-01-02 15:04:05"),
			EndAt:   up.EndAt.Format("2006-01-02 15:04:05"),
		},
	}
	if up.EnableBackup {
		agentConfig.EnableBackup = 1
	}

	// Expiry Check
	expireCloseEnabled := true
	if cfg, err := LoadSystemConfig(); err == nil {
		if val, ok := cfg["package_expire_close_site"]; ok {
			expireCloseEnabled = ParseBoolFlag(val)
		}
	}
	if expireCloseEnabled {
		if trigger == "expire" {
			agentConfig.Status = "expired"
		} else if time.Now().After(up.EndAt) {
			agentConfig.Status = "expired"
			// If explicit expire trigger wasn't sent but it is expired, maybe set trigger?
			// But for now follow input.
		}
	}

	// 4. Identify Nodes
	var groupIDs []int64
	if up.NodeGroupID > 0 {
		groupIDs = append(groupIDs, up.NodeGroupID)
	}
	if up.EnableBackup && up.BackupNodeGroup > 0 {
		groupIDs = append(groupIDs, up.BackupNodeGroup)
	}

	var nodes []models.Node
	if len(groupIDs) > 0 {
		if err := db.DB.Where("group_id IN ? AND enable = ?", groupIDs, true).Find(&nodes).Error; err != nil {
			return err
		}
	}

	nodeIDs := make([]int64, len(nodes))
	for i, n := range nodes {
		nodeIDs[i] = n.ID
	}

	// 5. Create Task
	// task.data: packages: [{package_id, version, config: {...}}]
	taskPayload := map[string]interface{}{
		"packages": []map[string]interface{}{
			{
				"package_id": up.ID,
				"version":    up.Version,
				"config":     agentConfig,
			},
		},
	}
	taskDataBytes, _ := json.Marshal(taskPayload)
	targets := NewTaskTargets(nodeIDs)
	now := time.Now()
	state := "waiting"
	var endAt *time.Time
	if targets.Total == 0 {
		state = "done"
		endAt = &now
	}
	task := models.Task{
		Type:        i18n.T("agent.task_sync_package"),
		Name:        i18n.T("task.sync_package_prefix") + strconv.FormatInt(up.ID, 10), // Simplification: + " (trigger=" + trigger + ")",
		Data:        string(taskDataBytes),
		TargetsJSON: targets.Marshal(),
		State:       state,
		Enable:      true,
		CreateAt:    now,
		EndAt:       endAt,
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return err
	}

	TriggerDispatchPending()
	return nil
}
