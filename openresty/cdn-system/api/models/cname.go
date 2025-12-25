package models

import "time"

type CnameDomain struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Domain    string    `json:"domain" gorm:"column:domain;size:255;uniqueIndex"`
	Note      string    `json:"note" gorm:"column:note;size:255"`
	CreatedAt time.Time `json:"created_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:update_at"`
}

func (CnameDomain) TableName() string {
	return "cname_domains"
}
