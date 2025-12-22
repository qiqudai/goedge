package models

import (
	"time"
)

// SysConfig stores system-wide configurations as JSON strings
type SysConfig struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value"` // Stored as JSON string
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
