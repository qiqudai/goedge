package models

import "time"

type NodeSubIP struct {
	ID int64  `json:"id"`
	IP string `json:"ip"`
}

// Node maps to the `node` table.
type Node struct {
	ID             int64       `json:"id" gorm:"primaryKey"`
	PID            int64       `json:"pid" gorm:"column:pid"`
	RegionID       *int64      `json:"region_id"`
	Name           string      `json:"name"`
	Remark         string      `json:"remark" gorm:"column:des"`
	IP             string      `json:"ip" gorm:"index"`
	Host           string      `json:"host"`
	Port           int         `json:"port"`
	HttpProxy      string      `json:"http_proxy"`
	IsMgmt         bool        `json:"is_mgmt"`
	Enable         bool        `json:"enable"`
	DisableBy      string      `json:"disable_by"`
	ConfigTask     string      `json:"config_task"`
	CheckOn        bool        `json:"check_on"`
	CheckProtocol  string      `json:"check_protocol"`
	CheckTimeout   int         `json:"check_timeout"`
	CheckPort      int         `json:"check_port"`
	CheckHost      string      `json:"check_host"`
	CheckPath      string      `json:"check_path"`
	CheckNodeGroup string      `json:"check_node_group"`
	CheckAction    string      `json:"check_action"`
	BwLimit        string      `json:"bw_limit"`
	CreatedAt      time.Time   `json:"create_at" gorm:"column:create_at"`
	UpdatedAt      time.Time   `json:"update_at" gorm:"column:update_at"`
	SubIPs         []NodeSubIP `json:"sub_ips,omitempty" gorm:"-"`
}

func (Node) TableName() string {
	return "node"
}
