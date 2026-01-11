package models

import "time"

type DomainOrigin struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	DomainID  int64     `json:"domain_id" gorm:"index"`
	Addr      string    `json:"addr"`
	Port      int       `json:"port"`
	Weight    int       `json:"weight"`
	Protocol  string    `json:"protocol"` // http or https
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
