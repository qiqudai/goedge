package models

import "time"

type DNSAPI struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"uid" gorm:"column:uid;index"`
	Name      string    `json:"name"`
	Remark    string    `json:"remark" gorm:"column:des"`
	Type      string    `json:"type"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (DNSAPI) TableName() string {
	return "dnsapi"
}
