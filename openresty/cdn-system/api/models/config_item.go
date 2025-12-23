package models

import "time"

// ConfigItem maps to config table in db.sql.
type ConfigItem struct {
  Name      string    `json:"name" gorm:"column:name"`
  Value     string    `json:"value" gorm:"column:value"`
  Type      string    `json:"type" gorm:"column:type"`
  ScopeID   int64     `json:"scope_id" gorm:"column:scope_id"`
  ScopeName string    `json:"scope_name" gorm:"column:scope_name"`
  Enable    bool      `json:"enable" gorm:"column:enable"`
  CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
  UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (ConfigItem) TableName() string {
  return "config"
}
