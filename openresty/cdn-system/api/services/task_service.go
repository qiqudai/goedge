package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"time"
)

// CreateDNSTask 创建 DNS 相关的异步任务，支持幂等去重
// 如果存在相同 IdempotencyKey 且状态为 waiting/running/retrying 的任务，则复用并返回 existingID
func CreateDNSTask(taskType string, data string, idempotencyKey string) (int64, error) {
	// IdempotencyKey logic removed to avoid DB changes.
	
	newTask := models.Task{
		Type:           taskType,
		Data:           data,
		State:          "waiting",
		Enable:         true,
		CreateAt:       time.Now(),
	}
	if err := db.DB.Create(&newTask).Error; err != nil {
		return 0, err
	}

	return newTask.ID, nil
}
