package models

import "time"

type UserLoginLog struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	Region    string    `json:"region"`
	Status    int       `json:"status"` // 1: Success, 0: Failed
	CreatedAt time.Time `json:"created_at"`
}

type UserOperationLog struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	Action      string    `json:"action"`      // e.g. "Create Node"
	Description string    `json:"description"` // e.g. "Created node HK-01"
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}
