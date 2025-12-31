package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"time"
)

// CreateDNSTask 创建 DNS 相关的异步任务，支持幂等去重
// 如果存在相同 IdempotencyKey 且状态为 waiting/running/retrying 的任务，则复用并返回 existingID
func CreateDNSTask(taskType string, data string, idempotencyKey string) (int64, error) {
	if idempotencyKey == "" {
		// 如果没有幂等键，则作为普通任务插入
		task := models.Task{
			Type:     taskType,
			Data:     data,
			State:    "waiting",
			Enable:   true,
			CreateAt: time.Now(),
		}
		err := db.DB.Create(&task).Error
		return task.ID, err
	}

	// 检查是否存在活跃任务
	var existingTask models.Task
	err := db.DB.Where("idempotency_key = ? AND state IN ?", idempotencyKey, []string{"waiting", "running", "retrying"}).First(&existingTask).Error
	if err == nil {
		// 存在活跃任务，直接返回 ID
		return existingTask.ID, nil
	}

	// 不存在活跃任务，创建新任务
	newTask := models.Task{
		Type:           taskType,
		Data:           data,
		State:          "waiting",
		Enable:         true,
		CreateAt:       time.Now(),
		IdempotencyKey: idempotencyKey,
	}
	if err := db.DB.Create(&newTask).Error; err != nil {
		return 0, err
	}

	return newTask.ID, nil
}
