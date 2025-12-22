package models

import "time"

// CCRule 对应数据库中的 `cc_rule` 表
type CCRule struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Sort        int       `json:"sort"`
	UserID      int64     `json:"uid" gorm:"column:uid"`
	Name        string    `json:"name"`
	Description string    `json:"des" gorm:"column:des"`
	Data        string    `json:"data"` // JSON content
	Internal    bool      `json:"internal"`
	Enable      bool      `json:"enable"`
	IsShow      bool      `json:"is_show"`
	TaskID      int64     `json:"task_id"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt   time.Time `json:"update_at" gorm:"column:update_at"`
}

func (CCRule) TableName() string {
	return "cc_rule"
}

// CCMatch 对应数据库中的 `cc_match` 表
type CCMatch struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	UserID      int64     `json:"uid" gorm:"column:uid"`
	Name        string    `json:"name"`
	Description string    `json:"des" gorm:"column:des"`
	Data        string    `json:"data"` // MEDIUMTEXT JSON
	Internal    bool      `json:"internal"`
	Enable      bool      `json:"enable"`
	TaskID      int64     `json:"task_id"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt   time.Time `json:"update_at" gorm:"column:update_at"`
}

func (CCMatch) TableName() string {
	return "cc_match"
}

// CCFilter 对应数据库中的 `cc_filter` 表
type CCFilter struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	UserID          int64     `json:"uid" gorm:"column:uid"`
	Name            string    `json:"name"`
	Description     string    `json:"des" gorm:"column:des"`
	Type            string    `json:"type"`
	WithinSecond    int       `json:"within_second"`
	MaxReq          int       `json:"max_req"`
	MaxReqPerUri    int       `json:"max_req_per_uri"`
	Extra           string    `json:"extra"` // JSON or string
	Internal        bool      `json:"internal"`
	Enable          bool      `json:"enable"`
	TaskID          int64     `json:"task_id"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt       time.Time `json:"update_at" gorm:"column:update_at"`
}

func (CCFilter) TableName() string {
	return "cc_filter"
}
