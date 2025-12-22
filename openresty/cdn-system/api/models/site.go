package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Site 对应数据库中的 `site` 表
type Site struct {
	ID                int64 `json:"id" gorm:"primaryKey"`
	UserID            int64 `json:"uid" gorm:"column:uid;index"`
	UserPackageID     int64 `json:"user_package_id" gorm:"column:user_package"`
	RegionID          int64 `json:"region_id" gorm:"column:region_id"`
	NodeGroupID       int64 `json:"node_group_id" gorm:"column:node_group_id"`
	BackupNodeGroupID int64 `json:"backup_node_group_id" gorm:"column:backup_node_group"`
	EnableBackupGroup bool  `json:"enable_backup_group"`
	DNSProviderID     int64 `json:"dns_provider_id" gorm:"column:dns_provider_id;index"`

	// CNAME 相关
	CnameDomain    string `json:"cname_domain"`
	CnameHostname  string `json:"cname_hostname"`
	CnameHostname2 string `json:"cname_hostname_2" gorm:"column:cname_hostname2"`
	CnameMode      string `json:"cname_mode"`

	// 域名列表
	DomainRaw string   `json:"-" gorm:"column:domain;type:text"`
	Domains   []string `json:"domains" gorm:"-"`

	// 监听配置
	HttpListenRaw  string `json:"-" gorm:"column:http_listen;type:text"`
	HttpsListenRaw string `json:"-" gorm:"column:https_listen;type:text"`
	HttpListen     []string `json:"http_listen" gorm:"-"`
	HttpsListen    []string `json:"https_listen" gorm:"-"`

	// 源站配置
	BackendRaw      string `json:"-" gorm:"column:backend;type:text"`
	BackendProtocol string `json:"backend_protocol"` // http, https
	BalanceWay      string `json:"balance_way"`      // ip_hash, rr
	Backends        []string `json:"backends" gorm:"-"`

	// 安全配置
	CcDefaultRule  int64  `json:"cc_default_rule"`
	CcSwitchRaw    string `json:"-" gorm:"column:cc_switch;type:text"`
	BlackIPRaw     string `json:"-" gorm:"column:black_ip;type:text"`
	WhiteIPRaw     string `json:"-" gorm:"column:white_ip;type:text"`
	BlockRegionRaw string `json:"-" gorm:"column:block_region;type:text"`
	SettingsRaw    string `json:"-" gorm:"column:settings;type:longtext"`
	Settings       map[string]interface{} `json:"settings" gorm:"-"`

	// 状态
	State     string    `json:"state"` // running, stop
	Enable    bool      `json:"enable"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

// TableName 指定表名
func (Site) TableName() string {
	return "site"
}

// BeforeSave GORM hook to serialize fields
func (s *Site) BeforeSave(tx *gorm.DB) (err error) {
	if len(s.Domains) > 0 {
		b, _ := json.Marshal(s.Domains)
		s.DomainRaw = string(b)
	}
	if len(s.HttpListen) > 0 {
		b, _ := json.Marshal(s.HttpListen)
		s.HttpListenRaw = string(b)
	}
	if len(s.HttpsListen) > 0 {
		b, _ := json.Marshal(s.HttpsListen)
		s.HttpsListenRaw = string(b)
	}
	if len(s.Backends) > 0 {
		b, _ := json.Marshal(s.Backends)
		s.BackendRaw = string(b)
	}
	if s.Settings != nil {
		b, _ := json.Marshal(s.Settings)
		s.SettingsRaw = string(b)
	}
	// Add other serialization logic here
	return
}

// AfterFind GORM hook to deserialize fields
func (s *Site) AfterFind(tx *gorm.DB) (err error) {
	if s.DomainRaw != "" {
		json.Unmarshal([]byte(s.DomainRaw), &s.Domains)
	}
	if s.HttpListenRaw != "" {
		json.Unmarshal([]byte(s.HttpListenRaw), &s.HttpListen)
	}
	if s.HttpsListenRaw != "" {
		json.Unmarshal([]byte(s.HttpsListenRaw), &s.HttpsListen)
	}
	if s.BackendRaw != "" {
		json.Unmarshal([]byte(s.BackendRaw), &s.Backends)
	}
	if s.SettingsRaw != "" {
		json.Unmarshal([]byte(s.SettingsRaw), &s.Settings)
	}
	return
}
