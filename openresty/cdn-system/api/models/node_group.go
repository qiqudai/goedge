package models

import "time"

// NodeGroup 对应数据库中的 `node_group` 表
type NodeGroup struct {
	ID               int64  `json:"id" gorm:"primaryKey"`
	RegionID         *int64 `json:"region_id"`
	Name             string `json:"name"`
	CnameHostname    string `json:"resolution"`                 // Frontend: resolution
	Ipv4Resolution   string `json:"ipv4_resolution"`            // New
	Description      string `json:"remark" gorm:"column:des"`   // Frontend: remark -> DB: des
	SortOrder        int    `json:"sort_order" gorm:"default:100"` // New
	L2Config         string `json:"l2_config"`                  // New
	BackupSwitchType string `json:"spare_ip_switch"`            // Frontend: spare_ip_switch (string)
	BackupSwitchPolicy string `json:"backup_switch_policy"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (NodeGroup) TableName() string {
	return "node_group"
}
