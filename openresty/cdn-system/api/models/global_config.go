package models

type GlobalConfig struct {
	WAF           WAFConfig            `json:"waf"`
	Nginx         NginxConfig          `json:"nginx"`
	DefaultConfig DefaultSiteConfig    `json:"default_config"`
	ErrorPages    map[string]string    `json:"error_pages"` // code -> html
	Resources     GlobalResourceConfig `json:"resources"`
}

type WAFConfig struct {
	Enable bool `json:"enable"`

	// Block Actions
	DefaultBlockAction string `json:"default_block_action"` // ipset, disconnect, page
	AutoIPSetEnable    bool   `json:"auto_ipset_enable"`
	AutoIPSetThreshold int    `json:"auto_ipset_threshold"` // req/s

	// Block Page Limits
	BlockPageRateLimitEnable bool `json:"block_page_rate_limit_enable"`
	BlockPageRateLimit       int  `json:"block_page_rate_limit"` // count/60s
	BlockPageTrafficFree     bool `json:"block_page_traffic_free"`

	// Timeouts
	BlacklistTimeout     int `json:"blacklist_timeout"`      // seconds
	TempWhitelistTimeout int `json:"temp_whitelist_timeout"` // seconds

	// Temp Whitelist Triggers
	TempWhitelistLimitTotal int `json:"temp_whitelist_limit_total"` // 5s total
	TempWhitelistLimitURL   int `json:"temp_whitelist_limit_url"`   // 5s same url

	// IP Lists
	WhitelistIPs string `json:"whitelist_ips"`
	BlacklistIPs string `json:"blacklist_ips"`

	// Security Toggles
	PreventTLSHandshake bool `json:"prevent_tls_handshake"`
	BlockUnboundDomain  bool `json:"block_unbound_domain"`
	DisablePing         bool `json:"disable_ping"`

	// Default Page Protection
	DefaultPageProtection          string `json:"default_page_protection"`           // force, auto
	DefaultPageProtectionThreshold int    `json:"default_page_protection_threshold"` // req/s

	// System Keys & Logs
	SecretKey            string `json:"secret_key"`
	NodeLogCleanStrategy string `json:"node_log_clean_strategy"` // none, log_only, log_cache
	CCRuleAutoSwitch     bool   `json:"cc_rule_auto_switch"`

	// Anti-CC Page
	AntiCCImageSource    string `json:"anti_cc_image_source"` // system, custom
	AntiCCImageCustomURL string `json:"anti_cc_image_custom_url"`
	AntiCCType           string `json:"anti_cc_type"` // slide, click, 5s, rotate...
	AntiCCDebug          bool   `json:"anti_cc_debug"`

	// Complex Protection
	WellKnownProtectionThreshold   int            `json:"well_known_protection_threshold"`
	ResourceProtectionEnable       bool           `json:"resource_protection_enable"`
	ResourceProtectionThreshold    int            `json:"resource_protection_threshold"`
	ResourceProtectionBlockTimeout int            `json:"resource_protection_block_timeout"`
	ResourceProtectionRules        []ResourceRule `json:"resource_protection_rules"`

	// Legacy (Keep for compatibility if needed, or map to new)
	// Mode and CC struct from previous version can be deprecated or mapped
	Mode          string        `json:"mode"`   // ipset, page
	Policy        string        `json:"policy"` // loose, strict, log_only
	CC            CCConfig      `json:"cc"`
	AccessControl AccessControl `json:"access_control"`
	Syntactic     SyntacticWAF  `json:"syntactic"`
}

type ResourceRule struct {
	Duration    int `json:"duration"`
	MaxRequests int `json:"max_requests"`
}

type CCConfig struct {
	Enable        bool   `json:"enable"`
	Threshold     int    `json:"threshold"`      // req/sec
	Action        string `json:"action"`         // block, captcha, js_challenge, 5s_shield, slide
	BlockTimeout  int    `json:"block_timeout"`  // seconds
	EmergencyMode bool   `json:"emergency_mode"` // pure static mode
	SlideCount    int    `json:"slide_count"`    // for slide action
}

type AccessControl struct {
	BlackIP      []string `json:"black_ip"`     // CIDR supported
	WhiteIP      []string `json:"white_ip"`     // CIDR supported
	BlackUA      []string `json:"black_ua"`     // Regex
	WhiteUA      []string `json:"white_ua"`     // Regex
	BlackURL     []string `json:"black_url"`    // Regex
	WhiteURL     []string `json:"white_url"`    // Regex
	RegionBlock  []string `json:"region_block"` // CN, US, etc.
	BlockEmptyUA bool     `json:"block_empty_ua"`
}

type SyntacticWAF struct {
	SQLInjection bool `json:"sql_injection"`
	XSS          bool `json:"xss"`
	Scanner      bool `json:"scanner"` // Block common scanners
}

type NginxConfig struct {
	WorkerProcesses       string `json:"worker_processes"`        // auto
	WorkerConnections     int    `json:"worker_connections"`      // 51200
	WorkerRlimitNofile    int    `json:"worker_rlimit_nofile"`    // 51200
	WorkerShutdownTimeout string `json:"worker_shutdown_timeout"` // 60s
	LogDirectory          string `json:"log_directory"`           // /usr/local/openresty/nginx/logs/

	// Keep existing useful ones
	KeepaliveTimeout int    `json:"keepalive_timeout"`
	Gzip             bool   `json:"gzip"`
	CustomSnippet    string `json:"custom_snippet"`
}

type DefaultSiteConfig struct {
	Website  SiteTemplate `json:"website"`
	API      SiteTemplate `json:"api"`
	Download SiteTemplate `json:"download"`
}

type SiteTemplate struct {
	CacheEnable bool `json:"cache_enable"`
	CacheTTL    int  `json:"cache_ttl"` // seconds
	Gzip        bool `json:"gzip"`
	WAFEnable   bool `json:"waf_enable"`
}

type GlobalResourceConfig struct {
	Website WebsiteResourceConfig `json:"website"`
	Forward ForwardResourceConfig `json:"forward"`
	Public  PublicResourceConfig  `json:"public"`
}

type WebsiteResourceConfig struct {
	MinLimit              int    `json:"min_limit"`                // 1000
	MaxLimitMultiplier    int    `json:"max_limit_multiplier"`     // 200
	MaxBlacklistIPs       int    `json:"max_blacklist_ips"`        // 50
	MaxWhitelistIPs       int    `json:"max_whitelist_ips"`        // 50
	DailyURLPurgeLimit    int    `json:"daily_url_purge_limit"`    // 2000
	DailyDirPurgeLimit    int    `json:"daily_dir_purge_limit"`    // 500
	DailyPreloadLimit     int    `json:"daily_preload_limit"`      // 2000
	DailyUnlockIPLimit    int    `json:"daily_unlock_ip_limit"`    // 1000
	UnlockIPBatchLimit    int    `json:"unlock_ip_batch_limit"`    // 50
	MaxCCRulesPerGroup    int    `json:"max_cc_rules_per_group"`   // 5
	MaxACLRules           int    `json:"max_acl_rules"`            // 5
	DailyLogDownloadLimit int    `json:"daily_log_download_limit"` // 10
	LogStorageDir         string `json:"log_storage_dir"`          // /data/download-temp/
	LogStorageHours       int    `json:"log_storage_hours"`        // 12
	MaxDomainsPerSite     int    `json:"max_domains_per_site"`     // 100
	DefaultListen80       bool   `json:"default_listen_80"`        // true
}

type ForwardResourceConfig struct {
	DisabledPorts      string `json:"disabled_ports"`       // 80 443
	MinLimit           int    `json:"min_limit"`            // 1000
	MaxLimitMultiplier int    `json:"max_limit_multiplier"` // 200
	MaxACLRules        int    `json:"max_acl_rules"`        // 10
}

type PublicResourceConfig struct {
	DisabledCustomPorts string `json:"disabled_custom_ports"` // 22
	AllowedCustomPorts  string `json:"allowed_custom_ports"`  // 1-65535
}
