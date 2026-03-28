package models

import "time"

// NodeMonitorLog maps to the `node_monitor_log` table.
type NodeMonitorLog struct {
	CreateAt time.Time `json:"create_at" gorm:"column:create_at"`
	Type     string    `json:"type" gorm:"column:type"`
	EventID  string    `json:"event_id" gorm:"column:event_id"`
	IP       string    `json:"ip" gorm:"column:ip"`
	Success  string    `json:"success" gorm:"column:success"`
	NodeID   int64     `json:"node_id" gorm:"column:node_id"`
}

func (NodeMonitorLog) TableName() string {
	return "node_monitor_log"
}
