package models

import "time"

type Plan struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"` // Description
	Group     string    `json:"group"` // Default, etc.
	Region    string    `json:"region"` // Default, etc.
	LineGroup string    `json:"line_group"`
	BackupGroup string  `json:"backup_group"`

	// Resource Limits (0 or -1 for unlimited usually, but using string or struct might be better for "Unlimited"/"Disabled")
	// Using int for limits, -1 or 0 for unlimited depending on logic.
    // Based on screenshot, some are switches.
    
    // Limits
	TrafficLimit    string `json:"traffic_limit"` // e.g., "100G", "Unlimited"
	BandwidthLimit  string `json:"bandwidth_limit"`
	ConnectionLimit string `json:"connection_limit"`
    L4PortLimit     string `json:"l4_port_limit"`
    DomainLimit     string `json:"domain_limit"`
    MainDomainLimit string `json:"main_domain_limit"`
    
    // Switches / Features
    NonStandardPortCount string `json:"non_standard_port_count"` // "Unlimited" or number
    CustomCCRules        bool   `json:"custom_cc_rules"`
    Websocket            bool   `json:"websocket"`
    HTTP3                bool   `json:"http3"`
    L2Origin             bool   `json:"l2_origin"`
    CCProtection         string `json:"cc_protection"` // e.g. "Support"
    DDOSProtection       string `json:"ddos_protection"` // e.g. "100G"

    // Pricing
    PriceMonthly   float64 `json:"price_monthly"`
    PriceQuarterly float64 `json:"price_quarterly"`
    PriceYearly    float64 `json:"price_yearly"`

    // CNAME
    CNAMEHostname string `json:"cname_hostname"` // e.g. "Leave empty for primary domain"
    CNAMEDomain   string `json:"cname_domain"`   // e.g. "cdnfly.com"
    CNAMEMode     string `json:"cname_mode"`     // e.g. "Generate by Site"

    // Purchase Limits
    SingleUserLimit int    `json:"single_user_limit"`
    Validity        string `json:"validity"` // Date range or "Unlimited"
    RealNameAuth    bool   `json:"real_name_auth"`
    RenewalDelay    int    `json:"renewal_delay"` // Days before expiry

    // Other
    AssignedUser  string `json:"assigned_user"` // User IDs
    SortOrder     int    `json:"sort_order"`
    Status        bool   `json:"status"` // Enabled/Disabled
    SourceIPLimit string `json:"source_ip_limit"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserPlan struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    UserID      uint      `json:"user_id"` // Link to User
    PlanID      uint      `json:"plan_id"`
    PlanName    string    `json:"plan_name"` // Snapshot name
    
    // Expiry
    ExpireAt    time.Time `json:"expire_at"`
    
    Status      string    `json:"status"` // Active, Expired
    
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
