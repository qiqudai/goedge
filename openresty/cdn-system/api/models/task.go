package models

import "time"

// Task 对应数据库中的 `task` 表
type Task struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	PID       int64     `json:"pid"`
	Pry       int       `json:"pry"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // refresh_url, refresh_dir, preheat
	Res       string    `json:"res"`
	Data      string    `json:"data"` // URLs
	Depend    string    `json:"depend"`
	CreateAt  time.Time `json:"create_at" gorm:"column:create_at"`
	StartAt   time.Time `json:"start_at" gorm:"column:start_at"`
	EndAt     time.Time `json:"end_at" gorm:"column:end_at"`
	Ret       string    `json:"ret"`
	Enable    bool      `json:"enable"`
	State     string    `json:"state"` // running, done, fail, waiting
	ErrTimes  int       `json:"err_times"`
	RetryAt   time.Time `json:"retry_at" gorm:"column:retry_at"`
	Progress  string    `json:"progress"`
}

func (Task) TableName() string {
	return "task"
}
