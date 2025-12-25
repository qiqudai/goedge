package models

type DNSAPI struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"uid" gorm:"column:uid;index"`
	Name      string    `json:"name"`
	Remark    string    `json:"remark" gorm:"column:des"`
	Type      string    `json:"type"`
	Auth      string    `json:"auth"`
	IsDefault bool      `json:"is_default" gorm:"column:is_default"`
}

func (DNSAPI) TableName() string {
	return "dnsapi"
}
