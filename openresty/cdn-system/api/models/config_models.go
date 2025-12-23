package models

// SiteConfig represents a CDN domain configuration (control-plane view)
type SiteConfig struct {
	ID        int64    `json:"id"`
	UserID    int64    `json:"user_id"`
	Domain    string   `json:"domain"` // e.g., "example.com"
	Origins   []Origin `json:"origins"`
	SSLEnable bool     `json:"ssl_enable"`
	CertID    int64    `json:"cert_id"`
}

// Origin represents a backend server
type Origin struct {
	ID       int64  `json:"id"`
	Addr     string `json:"addr"`     // IP or Domain
	Port     int    `json:"port"`     // 80 or 443
	Weight   int    `json:"weight"`   // Load balancing weight
	Protocol string `json:"protocol"` // http or https
}

// EdgeConfig is the final JSON sent to edge nodes.
// It matches the structure expected by cdn-edge-node/lua/config_loader.lua.
type EdgeConfig struct {
	Version   int64            `json:"version"`
	NodeID    string           `json:"node_id,omitempty"`
	Domains   []EdgeDomain     `json:"domains"`
	Upstreams []EdgeUpstream   `json:"upstreams"`
	Redis     *EdgeRedisConfig `json:"redis,omitempty"`
	WAF       *WAFConfig       `json:"waf,omitempty"`
}

type EdgeDomain struct {
	Name              string            `json:"name"`
	UpstreamKey       string            `json:"upstream_key"`
	LoadBalancePolicy string            `json:"load_balance_policy,omitempty"` // round_robin, random, ip_hash
	Headers           map[string]string `json:"headers,omitempty"`
	Status            string            `json:"status,omitempty"` // active, suspended
	SSLCertData       string            `json:"ssl_cert_data,omitempty"`
	SSLKeyData        string            `json:"ssl_key_data,omitempty"`
	ACLDefaultAction  string            `json:"acl_default_action,omitempty"`
	ACLRules          []EdgeACLRule     `json:"acl_rules,omitempty"`
}

type EdgeUpstream struct {
	ID      string               `json:"id"`
	Targets []EdgeUpstreamTarget `json:"targets"`
}

type EdgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

type EdgeRedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
}


type EdgeACLRule struct {
	IP     string `json:"ip"`
	Action string `json:"action"`
}
