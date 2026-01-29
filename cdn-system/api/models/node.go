package models

import "time"

type NodeSubIP struct {
	ID int64  `json:"id"`
	IP string `json:"ip"`
}

// Node maps to the `node` table.
type Node struct {
	ID             int64  `json:"id" gorm:"primaryKey"`
	PID            int64  `json:"pid" gorm:"column:pid"`
	GroupID        int64  `json:"group_id" gorm:"column:group_id;->"`
	RegionID       *int64 `json:"region_id"`
	Name           string `json:"name"`
	Remark         string `json:"remark" gorm:"column:des"`
	IP             string `json:"ip" gorm:"index"`
	Token          string `json:"token" gorm:"column:token;size:255"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	HttpProxy      string `json:"http_proxy"`
	IsMgmt         bool   `json:"is_mgmt"`
	Enable         bool   `json:"enable"`
	DisableBy      string `json:"disable_by"`
	ConfigTask     string `json:"config_task"`
	RegionName     string `json:"region_name" gorm:"-"` // Added for list view
	CheckOn        bool   `json:"check_on"`
	CheckProtocol  string `json:"check_protocol"`
	CheckTimeout   int    `json:"check_timeout"`
	CheckPort      int    `json:"check_port"`
	CheckHost      string `json:"check_host"`
	CheckPath      string `json:"check_path"`
	CheckNodeGroup string `json:"check_node_group"`
	CheckAction    string `json:"check_action"`
	BwLimit        string `json:"bw_limit"`
	Online         bool   `json:"online" gorm:"-"`
	LineCount      int64  `json:"line_count" gorm:"-"`
	// New fields for Node Settings
	Level                int        `json:"type" gorm:"column:level;default:1"` // 1: L1, 2: L2
	Sort                 int        `json:"sort_order" gorm:"column:sort;default:0"`
	CacheDir             string     `json:"cache_dir" gorm:"column:cache_dir"`
	MaxCacheSize         int        `json:"cache_limit" gorm:"column:max_cache_size"`
	LogDir               string     `json:"log_dir" gorm:"column:log_dir"`
	SSHHost              string     `json:"ssh_host" gorm:"column:ssh_host"`
	SSHPort              int        `json:"ssh_port" gorm:"column:ssh_port"`
	SSHUser              string     `json:"ssh_user" gorm:"column:ssh_user"`
	SSHAuthType          string     `json:"ssh_auth_type" gorm:"column:ssh_auth_type"`
	SSHPassword          string     `json:"-" gorm:"column:ssh_password"`
	SSHKey               string     `json:"-" gorm:"column:ssh_key;type:longtext"`
	WorkDir              string     `json:"work_dir" gorm:"column:work_dir"`
	AutoInstall          bool       `json:"auto_install" gorm:"column:auto_install"`
	InstallStatus        string     `json:"install_status" gorm:"column:install_status"`
	InstallError         string     `json:"install_error" gorm:"column:install_error;type:text"`
	InstallAt            *time.Time `json:"install_at" gorm:"column:install_at"`
	InstallStage         string     `json:"install_stage" gorm:"-"`
	InstallProgress      int        `json:"install_progress" gorm:"-"`
	InstallProgressBytes int64      `json:"install_progress_bytes" gorm:"-"`
	InstallProgressTotal int64      `json:"install_progress_total" gorm:"-"`

	CreatedAt time.Time   `json:"create_at" gorm:"column:create_at"`
	UpdatedAt time.Time   `json:"update_at" gorm:"column:update_at"`
	SubIPs    []NodeSubIP `json:"sub_ips,omitempty" gorm:"-"`
}

func (Node) TableName() string {
	return "node"
}
