package models

import "time"

// NodeGroup 对应数据库中的 `node_group` 表
type NodeGroup struct {
	ID                 int64  `json:"id" gorm:"primaryKey"`
	RegionID           int64  `json:"region_id"`
	CnameHostname      string `json:"cname_hostname"`
	Name               string `json:"name"`
	Description        string `json:"des" gorm:"column:des"`
	BackupSwitchType   string `json:"backup_switch_type"`
	BackupSwitchPolicy string `json:"backup_switch_policy"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (NodeGroup) TableName() string {
	return "node_group"
}
