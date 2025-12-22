package models

import "time"

// UserPackage 对应数据库中的 `user_package` 表 (用户已购实例)
type UserPackage struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	UserID          int64     `json:"uid" gorm:"column:uid"`
	Name            string    `json:"name"`
	PackageID       int64     `json:"package_id" gorm:"column:package"`
	
	// Runtime Config (copied from Package or customized)
	RegionID        int64     `json:"region_id"`
	NodeGroupID     int64     `json:"node_group_id"`
	
	// Resource Usage/Quota
	Traffic         int64     `json:"traffic"`
	Bandwidth       string    `json:"bandwidth"`
	Connection      int64     `json:"connection"`
	DomainLimit     int64     `json:"domain" gorm:"column:domain"`
	
	// Validity
	StartAt         time.Time `json:"start_at"`
	EndAt           time.Time `json:"end_at"`
	CreatedAt       time.Time `json:"create_at" gorm:"column:create_at"`
	
	// Status (Derived from time & state)
	// No explicit status column in db.sql, usually checked via EndAt > Now()
}

// TableName 指定表名
func (UserPackage) TableName() string {
	return "user_package"
}
