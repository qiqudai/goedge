package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"sync"
	"time"
)

var lineDeleteQueueOnce sync.Once
var lineDeleteQueueErr error

func ensureLineDeleteQueueTable() error {
	lineDeleteQueueOnce.Do(func() {
		if db.DB == nil {
			return
		}
		if db.DB.Migrator().HasTable(&models.LineDeleteQueue{}) {
			return
		}
		lineDeleteQueueErr = db.DB.AutoMigrate(&models.LineDeleteQueue{})
	})
	return lineDeleteQueueErr
}

func QueueLineConfigDeletion(nodeID, groupID int64, lineID, lineName string, delay time.Duration) {
	if nodeID == 0 || groupID == 0 || delay <= 0 {
		return
	}
	if err := ensureLineDeleteQueueTable(); err != nil {
		return
	}
	now := time.Now()
	record := models.LineDeleteQueue{
		NodeID:      nodeID,
		NodeGroupID: groupID,
		LineID:      lineID,
		LineName:    lineName,
		DeleteAt:    now.Add(delay),
		CreatedAt:   now,
	}
	_ = db.DB.Create(&record).Error
}

func LoadPendingGroupIDs(nodeID int64) []int64 {
	if nodeID == 0 {
		return nil
	}
	if err := ensureLineDeleteQueueTable(); err != nil {
		return nil
	}
	now := time.Now()
	_ = db.DB.Where("delete_at <= ?", now).Delete(&models.LineDeleteQueue{}).Error
	var rows []models.LineDeleteQueue
	if err := db.DB.Where("node_id = ? AND delete_at > ?", nodeID, now).Find(&rows).Error; err != nil {
		return nil
	}
	seen := map[int64]struct{}{}
	result := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.NodeGroupID == 0 {
			continue
		}
		if _, ok := seen[row.NodeGroupID]; ok {
			continue
		}
		seen[row.NodeGroupID] = struct{}{}
		result = append(result, row.NodeGroupID)
	}
	return result
}
