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
	WAF       *WAFConfig       `json:"waf,omitempty"`
	CCRules   map[int64][]EdgeCCRuleItem `json:"cc_rules,omitempty"`
	CCMatchers map[int64]EdgeCCMatcher   `json:"cc_matchers,omitempty"`
	CCFilters  map[int64]EdgeCCFilter    `json:"cc_filters,omitempty"`
	Streams   []EdgeStream     `json:"streams,omitempty"`
	Nginx     *EdgeNginxConfig `json:"nginx,omitempty"`
}

type EdgeDomain struct {
	Name              string            `json:"name"`
	UpstreamKey       string            `json:"upstream_key"`
	LoadBalancePolicy string            `json:"load_balance_policy,omitempty"` // round_robin, random, ip_hash
	Headers           map[string]string `json:"headers,omitempty"`
	ResponseHeaders   map[string]string `json:"response_headers,omitempty"`
	Status            string            `json:"status,omitempty"` // active, suspended
	SSLCertData       string            `json:"ssl_cert_data,omitempty"`
	SSLKeyData        string            `json:"ssl_key_data,omitempty"`
	ACLDefaultAction  string            `json:"acl_default_action,omitempty"`
	ACLRules          []EdgeACLRule     `json:"acl_rules,omitempty"`
	BlackIPs          []string          `json:"black_ips,omitempty"`
	WhiteIPs          []string          `json:"white_ips,omitempty"`
	CCRuleID          int64             `json:"cc_rule_id,omitempty"`
	OriginProtocol    string            `json:"origin_protocol,omitempty"`
	OriginHTTPPort    string            `json:"origin_http_port,omitempty"`
	OriginHTTPSPort   string            `json:"origin_https_port,omitempty"`
	Cache             *EdgeCacheConfig  `json:"cache,omitempty"`
	HttpListen        []string          `json:"http_listen,omitempty"`
	HttpsListen       []string          `json:"https_listen,omitempty"`
	HTTPSForce        bool              `json:"https_force,omitempty"`
	HTTPSRedirectPort string            `json:"https_redirect_port,omitempty"`
	HTTPSHSTS         bool              `json:"https_hsts,omitempty"`
	HTTPSHTTP2        bool              `json:"https_http2,omitempty"`
	HTTPSSSLProtocols string            `json:"https_ssl_protocols,omitempty"`
	HTTPSSSLCiphers   string            `json:"https_ssl_ciphers,omitempty"`
	HTTPSSSLPreferServerCiphers string   `json:"https_ssl_prefer_server_ciphers,omitempty"`
	ProxyConnectTimeout string           `json:"proxy_connect_timeout,omitempty"`
	ProxyReadTimeout    string           `json:"proxy_read_timeout,omitempty"`
	ProxySendTimeout    string           `json:"proxy_send_timeout,omitempty"`
	ProxyHTTPVersion    string           `json:"proxy_http_version,omitempty"`
	ProxySSLProtocols   string           `json:"proxy_ssl_protocols,omitempty"`
	EnableGzip          bool             `json:"enable_gzip,omitempty"`
	GzipTypes           string           `json:"gzip_types,omitempty"`
	EnableWebsocket     bool             `json:"enable_websocket,omitempty"`
	EnableRange         bool             `json:"enable_range,omitempty"`
	BodyLimit           int64            `json:"body_limit,omitempty"`
	UpstreamKeepalive   bool             `json:"upstream_keepalive,omitempty"`
	UpstreamKeepaliveConn int            `json:"upstream_keepalive_conn,omitempty"`
	UpstreamKeepaliveTimeout int         `json:"upstream_keepalive_timeout,omitempty"`
}

type EdgeUpstream struct {
	ID      string               `json:"id"`
	Targets []EdgeUpstreamTarget `json:"targets"`
}

type EdgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

type EdgeACLRule struct {
	IP     string `json:"ip"`
	Action string `json:"action"`
}

type EdgeCCRuleItem struct {
	MatcherID int64  `json:"matcher_id,omitempty"`
	FilterID  int64  `json:"filter_id,omitempty"`
	Action    string `json:"action,omitempty"`
	Enabled   bool   `json:"enabled"`
}

type EdgeCCMatcher struct {
	ID   int64  `json:"id"`
	Data string `json:"data"`
}

type EdgeCCFilter struct {
	ID            int64  `json:"id"`
	Type          string `json:"type"`
	WithinSecond  int    `json:"within_second"`
	MaxReq        int    `json:"max_req"`
	MaxReqPerURI  int    `json:"max_req_per_uri"`
	Extra         string `json:"extra,omitempty"`
}

type EdgeCacheRule struct {
	Rule   string `json:"rule,omitempty"`
	Ext    string `json:"ext,omitempty"`
	URI    string `json:"uri,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	TTL    int    `json:"ttl,omitempty"`
	Enable *bool  `json:"enable,omitempty"`
	NoCache bool  `json:"no_cache,omitempty"`
	ForceCache bool `json:"force_cache,omitempty"`
	Priority int  `json:"priority,omitempty"`
	IgnoreArgs bool `json:"ignore_args,omitempty"`
	CacheKey string `json:"cache_key,omitempty"`
}

type EdgeCacheConfig struct {
	Enable     bool           `json:"enable"`
	DefaultTTL int            `json:"default_ttl,omitempty"`
	Rules      []EdgeCacheRule `json:"rules,omitempty"`
}

type EdgeStream struct {
	ID           int64               `json:"id"`
	ListenPorts  []string            `json:"listen_ports"`
	Targets      []EdgeStreamTarget  `json:"targets"`
	BalanceWay   string              `json:"balance_way,omitempty"`
	ProxyProtocol bool               `json:"proxy_protocol,omitempty"`
	ProxyConnectTimeout string       `json:"proxy_connect_timeout,omitempty"`
	ProxyTimeout        string       `json:"proxy_timeout,omitempty"`
	ConnLimit           int           `json:"conn_limit,omitempty"`
}

type EdgeStreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	Enable bool   `json:"enable"`
}

type EdgeNginxConfig struct {
	LogsDir              string                 `json:"logs_dir,omitempty"`
	WorkerProcesses      string                 `json:"worker_processes,omitempty"`
	WorkerConnections    int                    `json:"worker_connections,omitempty"`
	WorkerRlimitNofile   int                    `json:"worker_rlimit_nofile,omitempty"`
	WorkerShutdownTimeout string                `json:"worker_shutdown_timeout,omitempty"`
	Resolver             string                 `json:"resolver,omitempty"`
	ResolverTimeout      string                 `json:"resolver_timeout,omitempty"`
	HTTP                 map[string]interface{} `json:"http,omitempty"`
	Stream               map[string]interface{} `json:"stream,omitempty"`
}
