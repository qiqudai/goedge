package models

import "time"

type UserLoginLog struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	UserID      int64     `json:"user_id" gorm:"column:uid"`
	IP          string    `json:"ip"`
	Success     bool      `json:"success"`
	PostContent string    `json:"post_content"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:create_at"`
}

func (UserLoginLog) TableName() string {
	return "login_log"
}

type UserOperationLog struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"column:uid"`
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	Content   string    `json:"content"`
	Diff      string    `json:"diff"`
	IP        string    `json:"ip"`
	Process   string    `json:"process"`
	CreatedAt time.Time `json:"created_at" gorm:"column:create_at"`
}

func (UserOperationLog) TableName() string {
	return "op_log"
}
