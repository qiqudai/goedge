package models

import "time"

// UserPackage 对应数据库中的 `user_package` 表 (用户已购实例)
type UserPackage struct {
	ID        int64  `json:"id" gorm:"primaryKey"`
	UserID    int64  `json:"uid" gorm:"column:uid"`
	Name      string `json:"name"`
	PackageID int64  `json:"package_id" gorm:"column:package"`

	// Runtime Config (copied from Package or customized)
	RegionID        int64  `json:"region_id"`
	NodeGroupID     int64  `json:"node_group_id"`
	BackupNodeGroup int64  `json:"backup_node_group" gorm:"column:backup_node_group"`
	EnableBackup    bool   `json:"enable_backup_group" gorm:"column:enable_backup_group"`
	CnameDomain     string `json:"cname_domain" gorm:"column:cname_domain"`
	CnameHostname2  string `json:"cname_hostname2" gorm:"column:cname_hostname2"`
	CnameHostname   string `json:"cname_hostname" gorm:"column:cname_hostname"`
	CnameMode       string `json:"cname_mode" gorm:"column:cname_mode"`
	RecordID        string `json:"record_id" gorm:"column:record_id"`

	// Resource Usage/Quota
	Traffic         int64  `json:"traffic"`
	Bandwidth       string `json:"bandwidth"`
	Connection      int64  `json:"connection"`
	DomainLimit     int64  `json:"domain" gorm:"column:domain"`
	HTTPPortLimit   int64  `json:"http_port" gorm:"column:http_port"`
	StreamPortLimit int64  `json:"stream_port" gorm:"column:stream_port"`
	CustomCCRule    bool   `json:"custom_cc_rule" gorm:"column:custom_cc_rule"`
	Websocket       bool   `json:"websocket" gorm:"column:websocket"`
	MonthPrice      int64  `json:"month_price" gorm:"column:month_price"`
	QuarterPrice    int64  `json:"quarter_price" gorm:"column:quarter_price"`
	YearPrice       int64  `json:"year_price" gorm:"column:year_price"`

	// Validity
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	TaskID    int64     `json:"task_id" gorm:"column:task_id"`

	// Status (Derived from time & state)
	// No explicit status column in db.sql, usually checked via EndAt > Now()
}

// TableName 指定表名
func (UserPackage) TableName() string {
	return "user_package"
}
