package models

import (
	"time"
)

// SysConfig corresponds to the `config` table in db.sql
type SysConfig struct {
	Name      string    `gorm:"column:name" json:"name"`
	Value     string    `gorm:"column:value" json:"value"` // Stored as JSON string
	Type      string    `gorm:"column:type" json:"type"`
	ScopeID   int       `gorm:"column:scope_id" json:"scope_id"`
	ScopeName string    `gorm:"column:scope_name" json:"scope_name"`
	CreatedAt time.Time `gorm:"column:create_at" json:"create_at"`
	UpdatedAt time.Time `gorm:"column:update_at" json:"update_at"`
	Enable    bool      `gorm:"column:enable" json:"enable"`
	TaskID    *int64    `gorm:"column:task_id" json:"task_id"`
}

// TableName overrides the table name to `config`
func (SysConfig) TableName() string {
	return "config"
}
