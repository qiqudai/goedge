package models

import "time"

type SiteGroup struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"uid" gorm:"column:uid;index"`
	Name      string    `json:"name"`
	Remark    string    `json:"remark" gorm:"column:des"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (SiteGroup) TableName() string {
	return "site_group"
}

type SiteGroupRelation struct {
	SiteID  int64 `json:"site_id" gorm:"column:site_id;primaryKey"`
	GroupID int64 `json:"group_id" gorm:"column:group_id;primaryKey"`
}

func (SiteGroupRelation) TableName() string {
	return "merge_site_group"
}
