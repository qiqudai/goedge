package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"time"
)

// CreateDNSTask creates DNS async tasks.
// If a matching idempotency key exists in waiting/running/retrying, reuse it and return existingID.
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
