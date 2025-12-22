package models

import "time"

type CnameDomain struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex" json:"domain"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
