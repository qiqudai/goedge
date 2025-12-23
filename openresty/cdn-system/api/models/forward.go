package models

import (
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ForwardOrigin struct {
	Address string `json:"address"`
	Weight  int    `json:"weight"`
	Enable  bool   `json:"enable"`
}

// Forward corresponds to `stream` table.
type Forward struct {
	ID                int64  `json:"id" gorm:"primaryKey"`
	UserID            int64  `json:"uid" gorm:"column:uid;index"`
	UserPackageID     int64  `json:"user_package_id" gorm:"column:user_package"`
	RegionID          int64  `json:"region_id" gorm:"column:region_id"`
	NodeGroupID       int64  `json:"node_group_id" gorm:"column:node_group_id"`
	BackupNodeGroup   int64  `json:"backup_node_group" gorm:"column:backup_node_group"`
	EnableBackupGroup bool   `json:"enable_backup_group" gorm:"column:enable_backup_group"`
	Enable            bool   `json:"enable" gorm:"column:enable"`
	State             string `json:"state" gorm:"column:state"`
	Remark            string `json:"remark" gorm:"-"`

	CnameDomain    string `json:"cname_domain" gorm:"column:cname_domain"`
	CnameHostname2 string `json:"cname_hostname2" gorm:"column:cname_hostname2"`
	CnameMode      string `json:"cname_mode" gorm:"column:cname_mode"`
	Cname          string `json:"cname" gorm:"column:cname_hostname"`

	ListenPortsRaw string          `json:"-" gorm:"column:listen;type:text"`
	ListenPorts    []string        `json:"listen_ports" gorm:"-"`
	OriginsRaw     string          `json:"-" gorm:"column:backend;type:text"`
	Origins        []ForwardOrigin `json:"origins" gorm:"-"`

	BackendPort   string `json:"backend_port" gorm:"column:backend_port"`
	BalanceWay    string `json:"balance_way" gorm:"column:balance_way"`
	ProxyProtocol bool   `json:"proxy_protocol" gorm:"column:proxy_protocol"`
	ConnLimit     string `json:"conn_limit" gorm:"column:conn_limit"`

	SettingsRaw string                 `json:"-" gorm:"column:acl;type:text"`
	Settings    map[string]interface{} `json:"settings" gorm:"-"`

	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (Forward) TableName() string {
	return "stream"
}

func (f *Forward) BeforeSave(tx *gorm.DB) (err error) {
	if len(f.ListenPorts) > 0 {
		f.ListenPortsRaw = encodeStringList(f.ListenPorts)
	}
	if len(f.Origins) > 0 {
		f.OriginsRaw = encodeOrigins(f.Origins)
	}
	if f.Settings != nil {
		// Keep existing settings
	} else if f.Remark != "" {
		f.Settings = map[string]interface{}{}
	}
	if f.Remark != "" {
		f.Settings["remark"] = f.Remark
	}
	if f.Settings != nil {
		b, _ := json.Marshal(f.Settings)
		f.SettingsRaw = string(b)
	}
	return
}

func (f *Forward) AfterFind(tx *gorm.DB) (err error) {
	if f.ListenPortsRaw != "" {
		f.ListenPorts = decodeStringList(f.ListenPortsRaw)
	}
	if f.OriginsRaw != "" {
		f.Origins = decodeOrigins(f.OriginsRaw)
	}
	if f.SettingsRaw != "" {
		_ = json.Unmarshal([]byte(f.SettingsRaw), &f.Settings)
		if v, ok := f.Settings["remark"]; ok {
			if s, ok := v.(string); ok {
				f.Remark = s
			}
		}
	}
	return
}

type ForwardGroup struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"uid" gorm:"column:uid"`
	Name      string    `json:"name" gorm:"column:name;type:varchar(64)"`
	Remark    string    `json:"remark" gorm:"column:des;type:varchar(255)"`
	CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (ForwardGroup) TableName() string {
	return "stream_group"
}

type ForwardGroupRelation struct {
	ID        int64 `json:"id" gorm:"primaryKey"`
	ForwardID int64 `json:"forward_id" gorm:"column:stream_id;index"`
	GroupID   int64 `json:"group_id" gorm:"column:group_id;index"`
}

func (ForwardGroupRelation) TableName() string {
	return "merge_stream_group"
}

func encodeStringList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func encodeOrigins(items []ForwardOrigin) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func decodeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &out); err == nil {
			return out
		}
	}
	return splitForwardFields(raw)
}

func decodeOrigins(raw string) []ForwardOrigin {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []ForwardOrigin
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &out); err == nil {
			return out
		}
	}
	fields := splitForwardFields(raw)
	out = make([]ForwardOrigin, 0, len(fields))
	for _, f := range fields {
		out = append(out, ForwardOrigin{Address: f, Weight: 1, Enable: true})
	}
	return out
}

func splitForwardFields(input string) []string {
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")
	input = strings.ReplaceAll(input, ",", " ")
	parts := strings.Fields(input)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
