package models

import "time"

// Job 对应数据库中的 `job` 表 (节点任务)
type Job struct {
	ID          int32     `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID      int64     `json:"task_id" gorm:"column:task_id;index"`
	NodeID      int64     `json:"node_id" gorm:"column:node_id;index"`
	NodeGroupID int32     `json:"node_group_id" gorm:"column:node_group_id"`
	UID         int32     `json:"uid" gorm:"column:uid"`
	Type        string    `json:"type" gorm:"column:type;type:varchar(255);index:type_idx"`
	State       string    `json:"state"` // waiting, running, success, fail
	Data        string    `json:"data" gorm:"column:data;type:text"`
	Ret         string    `json:"ret"`
	CreatedAt   time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Job) TableName() string {
	return "job"
}
