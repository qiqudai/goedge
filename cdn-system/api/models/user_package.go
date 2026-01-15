package models

import "time"

// UserPackage maps to the `user_package` table (purchased instance).
type UserPackage struct {
	ID        int64  `json:"id" gorm:"primaryKey"`
	UserID    int32  `json:"uid" gorm:"column:uid"`
	Name      string `json:"name"`
	PackageID int32  `json:"package_id" gorm:"column:package"`

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
	Traffic         int32  `json:"traffic"`
	Bandwidth       string `json:"bandwidth"`
	Connection      int32  `json:"connection"`
	DomainLimit     int32  `json:"domain" gorm:"column:domain"`
	MainDomainLimit int32  `json:"main_domain_limit" gorm:"column:main_domain_limit"`
	HTTPPortLimit   int32  `json:"http_port" gorm:"column:http_port"`
	StreamPortLimit int32  `json:"stream_port" gorm:"column:stream_port"`
	CustomCCRule    bool   `json:"custom_cc_rule" gorm:"column:custom_cc_rule"`
	Websocket       bool   `json:"websocket" gorm:"column:websocket"`
	L2Origin        bool   `json:"l2_origin" gorm:"column:l2_origin"`
	MonthPrice      int64  `json:"month_price" gorm:"column:month_price"`
	QuarterPrice    int64  `json:"quarter_price" gorm:"column:quarter_price"`
	YearPrice       int64  `json:"year_price" gorm:"column:year_price"`

	// Validity
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	TaskID    *int64    `json:"task_id" gorm:"column:task_id"`
	Version   int       `json:"version" gorm:"column:version;default:1"` // Sync Version
	IsExpired bool      `json:"is_expired" gorm:"column:is_expired"`

	// Status (Derived from time & state)
	// No explicit status column in db.sql, usually checked via EndAt > Now()
}

// TableName returns the table name.
func (UserPackage) TableName() string {
	return "user_package"
}
