package models

import "time"

// User 对应数据库中的 `user` 表
type User struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	Email       string `json:"email" gorm:"index"`
	Name        string `json:"name" gorm:"index"`
	Description string `json:"des" gorm:"column:des"`
	Phone       string `json:"phone"`
	QQ          string `json:"qq"`

	// Certification
	CertID       string `json:"cert_id"`
	CertName     string `json:"cert_name"`
	CertNo       string `json:"cert_no"`
	CertVerified bool   `json:"cert_verified"`

	// Security
	WhiteIP      string `json:"white_ip"`
	LoginCaptcha string `json:"login_captcha"`
	Password     string `json:"-"` // Never return password

	// Finance
	Balance int64 `json:"balance"` // In cents? cdnfly use int64, assume lowest unit
	Freeze  int64 `json:"freeze"`

	Enable bool `json:"enable"`
	Type   int  `json:"type" gorm:"index"` // 1: Admin? 2: User?

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "user"
}
