package models

import "time"

// Node 对应数据库中的 `node` 表
type Node struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	PID         int64  `json:"pid"` // Process ID or Parent ID? usually Parent ID for cluster
	RegionID    int64  `json:"region_id"`
	Name        string `json:"name"`
	Description string `json:"des" gorm:"column:des"`
	IP          string `json:"ip" gorm:"index"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	IsMgmt      bool   `json:"is_mgmt"`
	Enable      bool   `json:"enable"`

	// Monitoring & Health Check
	CheckOn       bool   `json:"check_on"`
	CheckProtocol string `json:"check_protocol"`
	CheckPort     int    `json:"check_port"`
	CheckHost     string `json:"check_host"`
	CheckPath     string `json:"check_path"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (Node) TableName() string {
	return "node"
}
