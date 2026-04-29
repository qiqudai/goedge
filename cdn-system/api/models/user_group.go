package models

import "time"

// UserGroup maps to the `user_group` table.
type UserGroup struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"index"`
	Des       string    `json:"des" gorm:"column:des"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (UserGroup) TableName() string {
	return "user_group"
}
