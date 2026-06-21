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
	Version            int64                      `json:"version"`
	NodeID             string                     `json:"node_id,omitempty"`
	NodeLevel          int                        `json:"node_level,omitempty"`
	NodeBandwidthLimit string                     `json:"node_bandwidth_limit,omitempty"`
	AntiBlocking       bool                       `json:"anti_blocking"`
	Domains            []EdgeDomain               `json:"domains"`
	Upstreams          []EdgeUpstream             `json:"upstreams"`
	WAF                *WAFConfig                 `json:"waf,omitempty"`
	Resources          *GlobalResourceConfig      `json:"resources,omitempty"`
	ErrorPageI18n      ErrorPageI18nSettings           `json:"error_page_i18n,omitempty"`
	ErrorPages         map[string]ErrorPageDefinition  `json:"error_pages,omitempty"`
	GuardPages         map[string]GuardPageDefinition  `json:"guard_pages,omitempty"`
	DefaultConfig      *DefaultSiteConfig         `json:"default_config,omitempty"`
	CCRules            map[int64][]EdgeCCRuleItem `json:"cc_rules,omitempty"`
	CCMatchers         map[int64]EdgeCCMatcher    `json:"cc_matchers,omitempty"`
	CCFilters          map[int64]EdgeCCFilter     `json:"cc_filters,omitempty"`
	Streams            []EdgeStream               `json:"streams,omitempty"`
	Nginx              *EdgeNginxConfig           `json:"nginx,omitempty"`
	FallbackCertData   string                     `json:"fallback_cert_data,omitempty"`
	FallbackKeyData    string                     `json:"fallback_key_data,omitempty"`
	IPUnblock          *EdgeIPUnblock             `json:"ip_unblock,omitempty"`
}

// EdgeIPUnblock carries IPs that should be removed from edge memory blacklist.
type EdgeIPUnblock struct {
	Rev int64    `json:"rev"`
	IPs []string `json:"ips,omitempty"`
}

type EdgeDomain struct {
	Name                           string                   `json:"name"`
	SiteType                       string                   `json:"site_type,omitempty"`
	UpstreamKey                    string                   `json:"upstream_key"`
	L2UpstreamKey                  string                   `json:"l2_upstream_key,omitempty"`
	UseL2                          bool                     `json:"use_l2,omitempty"`
	L2HTTPPort                     string                   `json:"l2_http_port,omitempty"`
	L2HTTPSPort                    string                   `json:"l2_https_port,omitempty"`
	LoadBalancePolicy              string                   `json:"load_balance_policy,omitempty"` // round_robin, random, ip_hash
	Headers                        map[string]string        `json:"headers,omitempty"`
	ResponseHeaders                map[string]string        `json:"response_headers,omitempty"`
	Hotlink                        *EdgeHotlinkConfig       `json:"hotlink,omitempty"`
	CORS                           *EdgeCorsConfig          `json:"cors,omitempty"`
	Cookie                         *EdgeCookieConfig        `json:"cookie,omitempty"`
	BlockTransparentProxy          bool                     `json:"block_transparent_proxy,omitempty"`
	CrawlerAction                  string                   `json:"crawler_action,omitempty"`
	GuardPassTTL                   int                      `json:"guard_pass_ttl,omitempty"`
	GuardBlockTTL                  int                      `json:"guard_block_ttl,omitempty"`
	URLRedirects                   []map[string]interface{} `json:"url_redirects,omitempty"`
	URLRewrites                    []map[string]interface{} `json:"url_rewrites,omitempty"`
	OriginConditions               []map[string]interface{} `json:"origin_conditions,omitempty"`
	Status                         string                   `json:"status,omitempty"` // active, suspended
	ConnLimit                      int                      `json:"conn_limit,omitempty"`
	SSLCertData                    string                   `json:"ssl_cert_data,omitempty"`
	SSLKeyData                     string                   `json:"ssl_key_data,omitempty"`
	SSLCertPath                    string                   `json:"ssl_cert_path,omitempty"`
	SSLKeyPath                     string                   `json:"ssl_key_path,omitempty"`
	WAFEnable                      *bool                    `json:"waf_enable,omitempty"`
	ACLDefaultAction               string                   `json:"acl_default_action,omitempty"`
	ACLDefaultDenyStatus           int                      `json:"acl_default_deny_status,omitempty"`
	ACLDefaultRedirectURL          string                   `json:"acl_default_redirect_url,omitempty"`
	ACLRules                       []EdgeACLRule            `json:"acl_rules,omitempty"`
	BlackIPs                       []string                 `json:"black_ips,omitempty"`
	WhiteIPs                       []string                 `json:"white_ips,omitempty"`
	RegionBlock                    []string                 `json:"region_block,omitempty"`
	CCRuleID                       int64                    `json:"cc_rule_id,omitempty"`
	CCAutoSwitch                   *EdgeCCAutoSwitch        `json:"cc_auto_switch,omitempty"`
	CustomCCRules                  []map[string]interface{} `json:"custom_cc_rules,omitempty"`
	OriginProtocol                 string                   `json:"origin_protocol,omitempty"`
	OriginHTTPPort                 string                   `json:"origin_http_port,omitempty"`
	OriginHTTPSPort                string                   `json:"origin_https_port,omitempty"`
	OriginHostHeader               string                   `json:"origin_host_header,omitempty"`
	OriginSNI                      string                   `json:"origin_sni,omitempty"`
	OriginVerifyTLS                bool                     `json:"origin_verify_tls,omitempty"`
	Cache                          *EdgeCacheConfig         `json:"cache,omitempty"`
	HttpListen                     []string                 `json:"http_listen,omitempty"`
	HttpsListen                    []string                 `json:"https_listen,omitempty"`
	HTTPSForce                     bool                     `json:"https_force,omitempty"`
	HTTPSRedirectPort              string                   `json:"https_redirect_port,omitempty"`
	HTTPSHSTS                      bool                     `json:"https_hsts,omitempty"`
	HTTPSHTTP2                     bool                     `json:"https_http2,omitempty"`
	HTTPSOCSP                      bool                     `json:"https_ocsp,omitempty"`
	HTTPSHTTP3                     bool                     `json:"https_http3,omitempty"`
	HTTPSSSLProtocols              string                   `json:"https_ssl_protocols,omitempty"`
	HTTPSSSLCiphers                string                   `json:"https_ssl_ciphers,omitempty"`
	HTTPSSSLPreferServerCiphers    string                   `json:"https_ssl_prefer_server_ciphers,omitempty"`
	ProxyConnectTimeout            string                   `json:"proxy_connect_timeout,omitempty"`
	ProxyReadTimeout               string                   `json:"proxy_read_timeout,omitempty"`
	ProxySendTimeout               string                   `json:"proxy_send_timeout,omitempty"`
	ProxyHTTPVersion               string                   `json:"proxy_http_version,omitempty"`
	OriginHTTPVersionPolicy        string                   `json:"origin_http_version_policy,omitempty"`
	OriginAutoDowngrade            bool                     `json:"origin_auto_downgrade,omitempty"`
	OriginDowngradeThreshold       int                      `json:"origin_downgrade_threshold,omitempty"`
	OriginDowngradeWindowSeconds   int                      `json:"origin_downgrade_window_seconds,omitempty"`
	OriginDowngradeCooldownSeconds int                      `json:"origin_downgrade_cooldown_seconds,omitempty"`
	ProxySSLProtocols              string                   `json:"proxy_ssl_protocols,omitempty"`
	EnableGzip                     bool                     `json:"enable_gzip,omitempty"`
	GzipTypes                      string                   `json:"gzip_types,omitempty"`
	EnableWebsocket                bool                     `json:"enable_websocket,omitempty"`
	EnableRange                    bool                     `json:"enable_range,omitempty"`
	BodyLimit                      int64                    `json:"body_limit,omitempty"`
	LogRequestHeader               bool                     `json:"log_request_header,omitempty"`
	LogResponseHeader              bool                     `json:"log_response_header,omitempty"`
	LogRequestBody                 bool                     `json:"log_request_body,omitempty"`
	LogRequestBodySizeLimit        int                      `json:"log_request_body_size_limit,omitempty"`
	OriginCert                     bool                     `json:"origin_cert,omitempty"`
	RealtimeIdentify               bool                     `json:"realtime_identify,omitempty"`
	RealtimeSend                   bool                     `json:"realtime_send"`
	RealtimeReturn                 bool                     `json:"realtime_return"`
	DefaultSite                    bool                     `json:"default_site,omitempty"`
	IPv6Enable                     bool                     `json:"ipv6_enable,omitempty"`
	LimitRate                      int64                    `json:"limit_rate,omitempty"`
	UpstreamKeepalive              bool                     `json:"upstream_keepalive,omitempty"`
	UpstreamKeepaliveConn          int                      `json:"upstream_keepalive_conn,omitempty"`
	UpstreamKeepaliveTimeout       int                      `json:"upstream_keepalive_timeout,omitempty"`
	ErrorPageLang                  string                   `json:"error_page_lang,omitempty"`
}

type EdgeHotlinkConfig struct {
	Enable     bool     `json:"enable"`
	Scope      string   `json:"scope,omitempty"`
	Value      string   `json:"value,omitempty"`
	AllowEmpty bool     `json:"allow_empty,omitempty"`
	Domains    []string `json:"domains,omitempty"`
}

type EdgeCorsConfig struct {
	Enable           bool   `json:"enable"`
	AllowOrigin      string `json:"allow_origin,omitempty"`
	AllowMethods     string `json:"allow_methods,omitempty"`
	AllowHeaders     string `json:"allow_headers,omitempty"`
	ExposeHeaders    string `json:"expose_headers,omitempty"`
	AllowCredentials bool   `json:"allow_credentials,omitempty"`
	MaxAge           string `json:"max_age,omitempty"`
}

type EdgeCookieConfig struct {
	Enable bool   `json:"enable"`
	Domain string `json:"domain,omitempty"`
}

type EdgeUpstream struct {
	ID      string               `json:"id"`
	Targets []EdgeUpstreamTarget `json:"targets"`
}

type EdgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	NodeID int64  `json:"node_id,omitempty"`
}

type EdgeACLCondition struct {
	Item     string `json:"item"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type EdgeACLRule struct {
	Conditions  []EdgeACLCondition `json:"conditions,omitempty"`
	Action      string             `json:"action"`
	DenyStatus  int                `json:"deny_status,omitempty"`
	RedirectURL string             `json:"redirect_url,omitempty"`
	IP          string             `json:"ip,omitempty"`
}

type EdgeCCAutoSwitch struct {
	Enable bool  `json:"enable"`
	QPS    int   `json:"qps"`
	RuleID int64 `json:"rule_id"`
}

type EdgeCCRuleItem struct {
	MatcherID int64  `json:"matcher_id,omitempty"`
	FilterID  int64  `json:"filter_id,omitempty"`
	Filter2ID int64  `json:"filter2_id,omitempty"`
	Action    string `json:"action,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Enabled   bool   `json:"enabled"`
}

type EdgeCCMatcher struct {
	ID   int64  `json:"id"`
	Data string `json:"data"`
}

type EdgeCCFilter struct {
	ID           int64  `json:"id"`
	Type         string `json:"type"`
	WithinSecond int    `json:"within_second"`
	MaxReq       int    `json:"max_req"`
	MaxReqPerURI int    `json:"max_req_per_uri"`
	Extra        string `json:"extra,omitempty"`
}

type EdgeCacheRule struct {
	Rule           string                   `json:"rule,omitempty"`
	Ext            string                   `json:"ext,omitempty"`
	URI            string                   `json:"uri,omitempty"`
	Prefix         string                   `json:"prefix,omitempty"`
	TTL            int                      `json:"ttl,omitempty"`
	Enable         *bool                    `json:"enable,omitempty"`
	NoCache        bool                     `json:"no_cache,omitempty"`
	ForceCache     bool                     `json:"force_cache,omitempty"`
	EnableRange    bool                     `json:"enable_range,omitempty"`
	IgnoreVary     bool                     `json:"ignore_vary,omitempty"`
	SkipConditions []EdgeCacheSkipCondition `json:"skip_conditions,omitempty"`
	Priority       int                      `json:"priority,omitempty"`
	IgnoreArgs     bool                     `json:"ignore_args,omitempty"`
	CacheKey       string                   `json:"cache_key,omitempty"`
}

type EdgeCacheSkipCondition struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type EdgeCacheConfig struct {
	Enable     bool            `json:"enable"`
	DefaultTTL int             `json:"default_ttl,omitempty"`
	Rules      []EdgeCacheRule `json:"rules,omitempty"`
}

type EdgeStream struct {
	ID                  int64              `json:"id"`
	ListenPorts         []string           `json:"listen_ports"`
	ListenProtocol      string             `json:"listen_protocol,omitempty"`
	Targets             []EdgeStreamTarget `json:"targets"`
	UseListenPort       bool               `json:"use_listen_port,omitempty"`
	BalanceWay          string             `json:"balance_way,omitempty"`
	ProxyProtocol       bool               `json:"proxy_protocol,omitempty"`
	ProxyConnectTimeout string             `json:"proxy_connect_timeout,omitempty"`
	ProxyTimeout        string             `json:"proxy_timeout,omitempty"`
	ConnLimit           int                `json:"conn_limit,omitempty"`
}

type EdgeStreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	Enable bool   `json:"enable"`
	NodeID int64  `json:"node_id,omitempty"`
	Backup bool   `json:"backup,omitempty"`
}

type EdgeNginxConfig struct {
	LogsDir               string                 `json:"logs_dir,omitempty"`
	WorkerProcesses       string                 `json:"worker_processes,omitempty"`
	WorkerConnections     int                    `json:"worker_connections,omitempty"`
	WorkerRlimitNofile    int                    `json:"worker_rlimit_nofile,omitempty"`
	WorkerShutdownTimeout string                 `json:"worker_shutdown_timeout,omitempty"`
	Resolver              string                 `json:"resolver,omitempty"`
	ResolverTimeout       string                 `json:"resolver_timeout,omitempty"`
	HTTP                  map[string]interface{} `json:"http,omitempty"`
	Stream                map[string]interface{} `json:"stream,omitempty"`
}
