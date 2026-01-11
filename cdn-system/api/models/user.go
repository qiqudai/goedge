package models

import "time"

// User maps to the `user` table.
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
	Company      string `json:"company"`
	TeaCode      string `json:"tea_code"` // Social Credit Code

	// Secondary Auth
	SecondaryAuth         bool   `json:"secondary_auth"`
	SecondaryAuthDeadline string `json:"secondary_auth_deadline"` // Format: 2006-01-02 15:04:05
	SecondaryAuthAction   string `json:"secondary_auth_action"`   // empty or "lock"
	SecondaryAuthStatus   string `json:"secondary_auth_status"`

	// Security
	WhiteIP      string `json:"white_ip"`
	LoginCaptcha string `json:"login_captcha"`
	Password     string `json:"-"` // Never return password

	// Finance
	Balance int64 `json:"balance"` // In cents? cdnfly use int64, assume lowest unit
	Freeze  int64 `json:"freeze"`

	Enable  bool `json:"enable"`
	Type    int  `json:"type" gorm:"index"` // 1: Admin? 2: User?
	GroupID int  `json:"group_id"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
}

// TableName returns the table name.
func (User) TableName() string {
	return "user"
}
