package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"strconv"
	"time"
)

type UserPackageService struct{}

func NewUserPackageService() *UserPackageService {
	return &UserPackageService{}
}

// SyncUserPackage creates tasks and jobs to sync user package config to nodes
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
		PackageID:       up.ID,
		UID:             up.UserID,
		Version:         up.Version,
		Status:          "active",
		RegionID:        up.RegionID,
		NodeGroupID:     up.NodeGroupID,
		BackupNodeGroup: up.BackupNodeGroup,
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
	if trigger == "expire" {
		agentConfig.Status = "expired"
	} else if time.Now().After(up.EndAt) {
		agentConfig.Status = "expired"
		// If explicit expire trigger wasn't sent but it is expired, maybe set trigger?
		// But for now follow input.
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
	// task.data: packages: [{package_id, version, node_ids, node_group_ids}]
	taskPayload := map[string]interface{}{
		"trigger": trigger,
		"packages": []map[string]interface{}{
			{
				"package_id":     up.ID,
				"version":        up.Version,
				"node_ids":       nodeIDs,
				"node_group_ids": groupIDs,
			},
		},
	}
	taskDataBytes, _ := json.Marshal(taskPayload)

	task := models.Task{
		Type:     "套餐同步",
		Name:     "同步套餐 package_id=" + strconv.FormatInt(up.ID, 10), // Simplification: + " (trigger=" + trigger + ")",
		Data:     string(taskDataBytes),
		State:    "waiting",
		Enable:   true,
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return err
	}

	// 6. Create Jobs
	// job.data: packages: [{package_id, version, config: {...}}]
	jobs := make([]models.Job, 0, len(nodes))
	jobPayload := map[string]interface{}{
		"packages": []map[string]interface{}{
			{
				"package_id": up.ID,
				"version":    up.Version,
				"config":     agentConfig,
			},
		},
	}
	jobDataBytes, _ := json.Marshal(jobPayload)

	for _, node := range nodes {
		jobs = append(jobs, models.Job{
			TaskID:      task.ID,
			NodeID:      node.ID,
			NodeGroupID: node.GroupID,
			UID:         up.UserID,
			Type:        "套餐同步",
			State:       "waiting",
			Data:        string(jobDataBytes),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	if len(jobs) > 0 {
		if err := db.DB.Create(&jobs).Error; err != nil {
			return err
		}
	}

	return nil
}
