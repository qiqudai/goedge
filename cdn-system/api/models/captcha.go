package models

import "time"

// Captcha stores email/SMS verification codes.
type Captcha struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Code      string    `json:"captcha" gorm:"column:captcha"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
}

func (Captcha) TableName() string {
	return "captcha"
}
