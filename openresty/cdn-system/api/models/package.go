package models

import "time"

// Package 对应数据库中的 `package` 表 (产品套餐)
type Package struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	Name        string `json:"name"`
	Description string `json:"des" gorm:"column:des"`
	RegionID    int64  `json:"region_id"`
	NodeGroupID int64  `json:"node_group_id" gorm:"column:node_group_id"`
	BackupNode  int64  `json:"backup_node_group" gorm:"column:backup_node_group"`
	CnameDomain string `json:"cname_domain"`
	CnameHost2  string `json:"cname_hostname2" gorm:"column:cname_hostname2"`
	CnameMode   string `json:"cname_mode"`

	// Price (cents)
	MonthPrice   int64 `json:"month_price"`
	QuarterPrice int64 `json:"quarter_price"`
	YearPrice    int64 `json:"year_price"`

	// Limits
	Traffic     int64  `json:"traffic"`   // GB?
	Bandwidth   string `json:"bandwidth"` // e.g. "100M"
	Connection  int64  `json:"connection"`
	DomainLimit int64  `json:"domain" gorm:"column:domain"` // domain count limit
	HttpPort    int64  `json:"http_port"`
	StreamPort  int64  `json:"stream_port"`
	ExpireAt    *time.Time `json:"expire" gorm:"column:expire"`
	BuyNumLimit int64     `json:"buy_num_limit"`
	BackendIPLimit string `json:"backend_ip_limit" gorm:"column:backend_ip_limit"`
	IDVerify    bool   `json:"id_verify"`
	BeforeExpDaysRenew int64 `json:"before_exp_days_renew"`

	// Features
	Websocket    bool `json:"websocket"`
	CustomCCRule bool `json:"custom_cc_rule"`

	Sort   int    `json:"sort"`
	Owner  string `json:"owner"`
	Enable bool   `json:"enable"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (Package) TableName() string {
	return "package"
}
