package models

import "time"

type CnameDomain struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex" json:"domain"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:update_at"`
}

func (CnameDomain) TableName() string {
	return "cname_domains"
}
