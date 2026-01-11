package models

import "time"

type Region struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Desc      string    `json:"des" gorm:"column:des"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (Region) TableName() string {
	return "region"
}
