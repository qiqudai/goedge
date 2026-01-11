package models

// AgentPackageConfig maps to the Agent local config JSON structure.
// file: /etc/cdn/packages/{user_package_id}.json
type AgentPackageConfig struct {
	PackageID int32  `json:"package_id"`
	UID       int32  `json:"uid"`
	Version   int    `json:"version"`
	Status    string `json:"status"` // active, expired, deleted

	// Node allocation
	RegionID        int32 `json:"region_id"`
	NodeGroupID     int32 `json:"node_group_id"`
	BackupNodeGroup int32 `json:"backup_node_group"`
	EnableBackup    int   `json:"enable_backup_group"` // 0/1

	// CNAME Info
	// If mode=manual, user sets CNAME to our provided `hostname`
	// If mode=auto, we use API to set CNAME record `hostname -> origin` (not fully agent concern but kept for reference)
	Cname AgentCnameInfo `json:"cname"`

	// Limits
	Limits AgentLimits `json:"limits"`

	// Features
	Features AgentFeatures `json:"features"`

	// Validity
	Time AgentTime `json:"time"`
}

type AgentCnameInfo struct {
	Domain     string `json:"domain"`      // e.g. cdnfly.com
	Hostname   string `json:"hostname"`    // e.g. wefnt9k8
	Hostname2  string `json:"hostname2"`   // Optional secondary
	Mode       string `json:"mode"`        // auto, manual
	RecordID   string `json:"record_id"`   // DNS Record ID for updates
}

type AgentLimits struct {
	Traffic    int32  `json:"traffic"`    // GB
	Bandwidth  string `json:"bandwidth"`  // e.g. "100M"
	Connection int32  `json:"connection"` // Max concurrent
	Domain     int32  `json:"domain"`     // Max domains
}

type AgentFeatures struct {
	HTTPPort     int   `json:"http_port"`     // e.g. 80
	StreamPort   int   `json:"stream_port"`   // e.g. 443 (if generic stream is counted here, or separate?)
	Websocket    bool  `json:"websocket"`
	CustomCCRule bool  `json:"custom_cc_rule"`
}

type AgentTime struct {
	StartAt string `json:"start_at"` // "2025-01-01 00:00:00"
	EndAt   string `json:"end_at"`   // "2026-01-01 00:00:00"
}
