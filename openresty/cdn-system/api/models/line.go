package models

import "time"

// Line 对应数据库中的 `line` 表
// 用于关联 Node 和 NodeGroup
type Line struct {
	ID                      int64     `json:"id" gorm:"primaryKey"`
	NodeGroupID             int64     `json:"node_group_id"`
	NodeID                  int64     `json:"node_id"`
	NodeIPID                int64     `json:"node_ip_id"`
	LineID                  string    `json:"line_id"`   // e.g. "unicom", "telecom"
	LineName                string    `json:"line_name"` // e.g. "联通", "电信"
	Weight                  string    `json:"weight"`
	RecordID                string    `json:"record_id"`
	TaskID                  int64     `json:"task_id"`
	Enable                  bool      `json:"enable"`
	IsBackup                bool      `json:"is_backup"`
	EnableBackup            bool      `json:"enable_backup"`
	IsBackupDefaultLine     bool      `json:"is_backup_default_line"`
	EnableBackupDefaultLine bool      `json:"enable_backup_default_line"`
	SwitchAt                time.Time `json:"switch_at"`
	DisableBy               string    `json:"disable_by"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (Line) TableName() string {
	return "line"
}
