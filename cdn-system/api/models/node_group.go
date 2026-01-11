package models

import "time"

// NodeGroup maps to the `node_group` table.
type NodeGroup struct {
	ID               int64  `json:"id" gorm:"primaryKey"`
	RegionID         *int64 `json:"region_id"`
	Name             string `json:"name"`
	CnameHostname    string `json:"resolution"`                 // Frontend: resolution
	Ipv4Resolution   string `json:"ipv4_resolution" gorm:"-"`   // stored in backup_switch_policy
	Description      string `json:"remark" gorm:"column:des"`   // Frontend: remark -> DB: des
	SortOrder        int    `json:"sort_order" gorm:"-"`         // stored in backup_switch_policy
	L2Config         string `json:"l2_config" gorm:"-"`          // stored in backup_switch_policy
	BackupSwitchType string `json:"spare_ip_switch"`            // Frontend: spare_ip_switch (string)
	BackupSwitchPolicy string `json:"backup_switch_policy"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName returns the table name.
func (NodeGroup) TableName() string {
	return "node_group"
}
