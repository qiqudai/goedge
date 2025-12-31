package models

import "time"

// Job 对应数据库中的 `job` 表 (节点任务)
type Job struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	TaskID      int64     `json:"task_id" gorm:"index"`
	NodeID      int64     `json:"node_id" gorm:"index"`
	NodeGroupID int64     `json:"node_group_id"`
	UID         int64     `json:"uid"`
	Type        string    `json:"type"`  // e.g. "套餐同步"
	State       string    `json:"state"` // waiting, running, success, fail
	Data        string    `json:"data" gorm:"type:longtext"`
	Ret         string    `json:"ret"`
	CreatedAt   time.Time `json:"create_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Job) TableName() string {
	return "job"
}
