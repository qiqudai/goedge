package models

import (
	"time"
)

type Cert struct {
	ID          int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      int        `json:"uid" gorm:"column:uid"`
	Name        string     `json:"name"`
	Description string     `json:"des" gorm:"column:des"`
	Type        string     `json:"type"` // e.g., "upload", "lets"
	Domain      string     `json:"domain"`
	DNSAPI      int        `json:"dnsapi" gorm:"column:dnsapi"`
	Cert        string     `json:"cert"`
	Key         string     `json:"key"`
	StartTime   *time.Time `json:"start_time"`
	ExpireTime  *time.Time `json:"expire_time"`
	AutoRenew   bool       `json:"auto_renew"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
	Enable      bool       `json:"enable"`
	TaskID      int64      `json:"task_id"`
	IssueTaskID int64      `json:"issue_task_id"`
	Version     int        `json:"version"`
}

func (Cert) TableName() string {
	return "cert"
}
