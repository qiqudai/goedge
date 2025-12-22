package models

import "time"

type Domain struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"uniqueIndex:idx_user_domain"`
	Name      string    `json:"name" gorm:"uniqueIndex:idx_user_domain"`
	Cname     string    `json:"cname"`
	Status    int       `json:"status"` // 1: Active, 0: Disabled
	
	// Legacy storage (if exists in DB)
	OriginsRaw string         `json:"-" gorm:"column:origins"`
	Origins    []DomainOrigin `json:"origins" gorm:"foreignKey:DomainID"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
