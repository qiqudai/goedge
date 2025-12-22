package models

import "time"

// UserNodeGroup maps users to node groups (many-to-many)
type UserNodeGroup struct {
	UserID      int64     `json:"user_id" gorm:"primaryKey;index"`
	NodeGroupID int64     `json:"node_group_id" gorm:"primaryKey;index"`
	CreatedAt   time.Time `json:"created_at"`
}
