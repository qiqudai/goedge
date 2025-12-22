package models

import "time"

// ACL 对应数据库中的 `acl` 表
type ACL struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	UserID        int64     `json:"uid" gorm:"column:uid"`
	Name          string    `json:"name"`
	Description   string    `json:"des" gorm:"column:des"`
	DefaultAction string    `json:"default_action"`
	Data          string    `json:"data"` // MEDIUMTEXT JSON
	Enable        bool      `json:"enable"`
	TaskID        int64     `json:"task_id"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt     time.Time `json:"update_at" gorm:"column:update_at"`
}

func (ACL) TableName() string {
	return "acl"
}
