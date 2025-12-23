package models

type APIKey struct {
	ID        int64  `json:"id" gorm:"primaryKey"`
	UserID    int64  `json:"user_id" gorm:"column:uid;index"`
	APIKey    string `json:"api_key" gorm:"column:api_key"`
	APISecret string `json:"api_secret" gorm:"column:api_secret"`
	APIIP     string `json:"api_ip" gorm:"column:api_ip"`
}

func (APIKey) TableName() string {
	return "api_key"
}
