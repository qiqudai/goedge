package models

import "time"

// LineDeleteQueue stores delayed config removal for node-group associations.
type LineDeleteQueue struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	NodeID      int64     `json:"node_id" gorm:"column:node_id"`
	NodeGroupID int64     `json:"node_group_id" gorm:"column:node_group_id"`
	LineID      string    `json:"line_id" gorm:"column:line_id"`
	LineName    string    `json:"line_name" gorm:"column:line_name"`
	DeleteAt    time.Time `json:"delete_at" gorm:"column:delete_at"`
	CreatedAt   time.Time `json:"create_at" gorm:"column:create_at"`
}

func (LineDeleteQueue) TableName() string {
	return "line_delete_queue"
}
