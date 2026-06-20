package main

import (
	fsutil "cdn-common/io"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// startConfigPull checks for config updates
func startConfigPull() {
	interval := CONFIG_PULL_INT
	if interval <= 0 {
		log.Printf("[Info] Config pull disabled")
		return
	}
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	initialDelay := configPullInitialDelay(interval)
	log.Printf("[Info] Config pull enabled interval=%s initial_delay=%s", interval, initialDelay)

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()
	for {
		<-timer.C
		if err := pullConfig(); err != nil {
			log.Printf("[Error] Config Pull Failed: %v", err)
		}
		timer.Reset(interval)
	}
}

func pullConfig() error {
	endpoint := API_BaseURL + "/api/v1/agent/config?node_id=" + url.QueryEscape(NodeID)
	if currentVersion := readLocalVersion(); currentVersion != 0 {
		endpoint += "&version=" + strconv.FormatInt(currentVersion, 10)
	}
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	body, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		return err
	}

	if status == http.StatusNotModified {
		debugLogInteraction("GET", req.URL.String(), status, nil, nil)
		return nil
	}

	if status == 200 {
		debugLogInteraction("GET", req.URL.String(), status, nil, body)
		if len(strings.TrimSpace(string(body))) == 0 {
			return nil
		}
		if _, err := applyConfigPayload(body); err != nil {
			return err
		}
		return nil
	}

	debugLogInteraction("GET", req.URL.String(), status, nil, nil)
	return fmt.Errorf("config pull status: %d", status)
}

func configPullInitialDelay(interval time.Duration) time.Duration {
	baseDelay := 5 * time.Second
	if interval <= baseDelay {
		return baseDelay
	}
	window := interval - baseDelay
	if window > 30*time.Second {
		window = 30 * time.Second
	}
	windowSeconds := int64(window / time.Second)
	if windowSeconds <= 0 {
		return baseDelay
	}
	nodeID, err := strconv.ParseInt(strings.TrimSpace(NodeID), 10, 64)
	if err != nil || nodeID < 0 {
		return baseDelay
	}
	return baseDelay + time.Duration(nodeID%windowSeconds)*time.Second
}

func applyConfigPayload(body []byte) (string, error) {
	return applyConfigPayloadWithOptions(body, false)
}

func applyConfigPayloadWithOptions(body []byte, forceReload bool) (string, error) {
	return applyConfigPayloadWithOptionsAndReload(body, forceReload, false)
}

func applyConfigPayloadWithOptionsAndReload(body []byte, forceReload bool, skipReload bool) (string, error) {
	if len(body) == 0 {
		return "", fmt.Errorf("empty config payload")
	}
	applyNodeRuntimeControls(body)
	newVersion := extractVersion(body)
	currentVersion := readLocalVersion()
	if !forceReload && newVersion != 0 && newVersion == currentVersion {
		log.Printf("[Info] Config unchanged (version=%d). Skipping reload.", currentVersion)
		return "skipped", nil
	}

	if err := writeConfigWithBackup(body); err != nil {
		log.Printf("[Error] Failed to write config file: %v", err)
		return "", err
	}

	if err := generateDynamicConfigs(body); err != nil {
		log.Printf("[Error] Failed to generate dynamic configs: %v", err)
		return "", err
	}

	syncRuntimeLuaAssets()

	if skipReload {
		if forceReload {
			log.Printf("[Info] Config Updated (version=%d, %d bytes, force). Reload skipped.", newVersion, len(body))
		} else {
			log.Printf("[Info] Config Updated (version=%d, %d bytes). Reload skipped.", newVersion, len(body))
		}
		return "ok", nil
	}

	if forceReload {
		log.Printf("[Info] Config Updated (version=%d, %d bytes, force). Reloading Nginx...", newVersion, len(body))
	} else {
		log.Printf("[Info] Config Updated (version=%d, %d bytes). Reloading Nginx...", newVersion, len(body))
	}
	if err := reloadNginxWithRollback(); err != nil {
		return "", err
	}
	return "ok", nil
}

func reloadNginxWithRollback() error {
	if err := executeReload(); err != nil {
		log.Printf("[Error] Reload Nginx Failed: %v", err)
		if restoreErr := restoreBackup(); restoreErr != nil {
			log.Printf("[Error] Failed to restore backup: %v", restoreErr)
			return fmt.Errorf("reload failed and restore failed: %v", restoreErr)
		}
		if retryErr := executeReload(); retryErr != nil {
			log.Printf("[Error] Reload after rollback failed: %v", retryErr)
			return fmt.Errorf("reload failed and rollback reload failed: %v", retryErr)
		}
		log.Println("[Warn] Rolled back to previous config")
		return err
	}
	return nil
}

func extractVersion(body []byte) int64 {
	var payload struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}
	return payload.Version
}

func applyNodeRuntimeControls(body []byte) {
	var payload struct {
		AntiBlocking       *bool  `json:"anti_blocking"`
		NodeBandwidthLimit string `json:"node_bandwidth_limit"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}
	if payload.AntiBlocking != nil {
		AutoDisableFirewall = *payload.AntiBlocking
		applyAntiBlockingPreference(*payload.AntiBlocking, "config_sync")
	}
	if err := applyNodeBandwidthLimit(strings.TrimSpace(payload.NodeBandwidthLimit)); err != nil {
		log.Printf("[Warn] apply node bandwidth limit failed: %v", err)
	}
}

func applyNodeBandwidthLimit(raw string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	iface, err := detectPrimaryInterface()
	if err != nil {
		return err
	}
	kbit, limited := parseBandwidthLimitKbit(raw)
	if !limited {
		return clearInterfaceBandwidthLimit(iface)
	}
	return setInterfaceBandwidthLimit(iface, kbit)
}

func detectPrimaryInterface() (string, error) {
	candidates := [][]string{
		{"route", "get", "1.1.1.1"},
		{"route", "show", "default"},
	}
	devRegex := regexp.MustCompile(`\bdev\s+([a-zA-Z0-9_.:-]+)\b`)
	for _, args := range candidates {
		out, err := exec.Command("ip", args...).CombinedOutput()
		if err != nil {
			continue
		}
		match := devRegex.FindStringSubmatch(string(out))
		if len(match) >= 2 && strings.TrimSpace(match[1]) != "" {
			return strings.TrimSpace(match[1]), nil
		}
	}
	return "", fmt.Errorf("cannot detect primary network interface")
}

func parseBandwidthLimitKbit(raw string) (int64, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimSuffix(value, "bps")
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || value == "unlimited" || value == "unlimit" {
		return 0, false
	}
	numPart := strings.TrimRight(value, "kmg")
	unitPart := strings.TrimSpace(strings.TrimPrefix(value, numPart))
	numPart = strings.TrimSpace(numPart)
	if numPart == "" {
		return 0, false
	}
	num, err := strconv.ParseFloat(numPart, 64)
	if err != nil || num <= 0 {
		return 0, false
	}
	multiplier := 1.0
	switch unitPart {
	case "g":
		multiplier = 1000 * 1000
	case "m":
		multiplier = 1000
	case "k", "":
		multiplier = 1
	default:
		return 0, false
	}
	kbit := int64(num * multiplier)
	if kbit <= 0 {
		return 0, false
	}
	return kbit, true
}

func clearInterfaceBandwidthLimit(iface string) error {
	commands := [][]string{
		{"qdisc", "del", "dev", iface, "root"},
		{"qdisc", "del", "dev", iface, "ingress"},
	}
	var firstErr error
	for _, args := range commands {
		out, err := exec.Command("tc", args...).CombinedOutput()
		if err != nil {
			text := strings.ToLower(string(out))
			if strings.Contains(text, "noqueue") || strings.Contains(text, "no such file") || strings.Contains(text, "cannot find") || strings.Contains(text, "invalid handle") {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("tc %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
			}
		}
	}
	return firstErr
}

func setInterfaceBandwidthLimit(iface string, kbit int64) error {
	_ = clearInterfaceBandwidthLimit(iface)
	burst := strconv.FormatInt(maxInt64(kbit/20, 64), 10) + "kbit"
	latency := "50ms"
	egress := []string{"qdisc", "replace", "dev", iface, "root", "tbf", "rate", strconv.FormatInt(kbit, 10) + "kbit", "burst", burst, "latency", latency}
	if out, err := exec.Command("tc", egress...).CombinedOutput(); err != nil {
		return fmt.Errorf("tc %s: %v (%s)", strings.Join(egress, " "), err, strings.TrimSpace(string(out)))
	}
	ingressQdisc := []string{"qdisc", "replace", "dev", iface, "handle", "ffff:", "ingress"}
	if out, err := exec.Command("tc", ingressQdisc...).CombinedOutput(); err != nil {
		return fmt.Errorf("tc %s: %v (%s)", strings.Join(ingressQdisc, " "), err, strings.TrimSpace(string(out)))
	}
	police := []string{"filter", "replace", "dev", iface, "parent", "ffff:", "protocol", "all", "u32", "match", "u32", "0", "0", "police", "rate", strconv.FormatInt(kbit, 10) + "kbit", "burst", burst, "drop", "flowid", ":1"}
	if out, err := exec.Command("tc", police...).CombinedOutput(); err != nil {
		return fmt.Errorf("tc %s: %v (%s)", strings.Join(police, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func readLocalVersion() int64 {
	if CONFIG_PATH == "" {
		return 0
	}
	data, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		return 0
	}
	return extractVersion(data)
}

func writeConfigWithBackup(body []byte) error {
	if CONFIG_PATH == "" {
		return os.ErrNotExist
	}
	// Backup
	if current, err := ioutil.ReadFile(CONFIG_PATH); err == nil {
		_ = ioutil.WriteFile(CONFIG_BAK, current, 0644)
	}
	return fsutil.WriteFileAtomic(CONFIG_PATH, body, 0o644)
}

func syncRuntimeLuaAssets() {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return
	}
	restoreDirIfMissing("assets/lua", filepath.Join(rootDir, "lua"))
}

func restoreBackup() error {
	if CONFIG_PATH == "" {
		return os.ErrNotExist
	}
	data, err := ioutil.ReadFile(CONFIG_BAK)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(CONFIG_PATH, data, 0644); err != nil {
		return err
	}
	return nil
}

type edgeStreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	Enable bool   `json:"enable"`
	Backup bool   `json:"backup"`
	NodeID int64  `json:"node_id,omitempty"`
}

type edgeStream struct {
	ID                  int64              `json:"id"`
	ListenPorts         []string           `json:"listen_ports"`
	ListenProtocol      string             `json:"listen_protocol"`
	Targets             []edgeStreamTarget `json:"targets"`
	UseListenPort       bool               `json:"use_listen_port"`
	BalanceWay          string             `json:"balance_way"`
	ProxyProtocol       bool               `json:"proxy_protocol"`
	ProxyConnectTimeout string             `json:"proxy_connect_timeout"`
	ProxyTimeout        string             `json:"proxy_timeout"`
	ConnLimit           int                `json:"conn_limit"`
}

type edgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	NodeID int64  `json:"node_id"`
}

type edgeUpstream struct {
	ID      string               `json:"id"`
	Targets []edgeUpstreamTarget `json:"targets"`
}

type edgeCacheRule struct {
	Rule           string               `json:"rule"`
	Ext            string               `json:"ext"`
	URI            string               `json:"uri"`
	Prefix         string               `json:"prefix"`
	TTL            int                  `json:"ttl"`
	Enable         *bool                `json:"enable"`
	NoCache        bool                 `json:"no_cache"`
	ForceCache     bool                 `json:"force_cache"`
	EnableRange    bool                 `json:"enable_range"`
	IgnoreVary     bool                 `json:"ignore_vary"`
	SkipConditions []edgeCacheCondition `json:"skip_conditions"`
	Priority       int                  `json:"priority"`
	IgnoreArgs     bool                 `json:"ignore_args"`
	CacheKey       string               `json:"cache_key"`
}

type edgeCacheCondition struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type edgeCacheConfig struct {
	Enable     bool            `json:"enable"`
	DefaultTTL int             `json:"default_ttl"`
	Rules      []edgeCacheRule `json:"rules"`
}

type edgeHotlinkConfig struct {
	Enable     bool     `json:"enable"`
	Scope      string   `json:"scope"`
	Value      string   `json:"value"`
	AllowEmpty bool     `json:"allow_empty"`
	Domains    []string `json:"domains"`
}

type edgeCorsConfig struct {
	Enable           bool   `json:"enable"`
	AllowOrigin      string `json:"allow_origin"`
	AllowMethods     string `json:"allow_methods"`
	AllowHeaders     string `json:"allow_headers"`
	ExposeHeaders    string `json:"expose_headers"`
	AllowCredentials bool   `json:"allow_credentials"`
	MaxAge           string `json:"max_age"`
}

type edgeCookieConfig struct {
	Enable bool   `json:"enable"`
	Domain string `json:"domain"`
}

type edgeDomain struct {
	Name                  string                   `json:"name"`
	SiteType              string                   `json:"site_type"`
	UpstreamKey           string                   `json:"upstream_key"`
	L2UpstreamKey         string                   `json:"l2_upstream_key"`
	UseL2                 bool                     `json:"use_l2"`
	L2HTTPPort            string                   `json:"l2_http_port"`
	L2HTTPSPort           string                   `json:"l2_https_port"`
	LoadBalancePolicy     string                   `json:"load_balance_policy"`
	Headers               map[string]string        `json:"headers"`
	ResponseHeaders       map[string]string        `json:"response_headers"`
	Hotlink               *edgeHotlinkConfig       `json:"hotlink"`
	Cors                  *edgeCorsConfig          `json:"cors"`
	Cookie                *edgeCookieConfig        `json:"cookie"`
	BlockTransparentProxy bool                     `json:"block_transparent_proxy"`
	CrawlerAction         string                   `json:"crawler_action"`
	GuardPassTTL          int                      `json:"guard_pass_ttl"`
	GuardBlockTTL         int                      `json:"guard_block_ttl"`
	URLRedirects          []map[string]interface{} `json:"url_redirects"`
	URLRewrites           []map[string]interface{} `json:"url_rewrites"`
	OriginConditions      []map[string]interface{} `json:"origin_conditions"`
	Status                string                   `json:"status"`
	ConnLimit             int                      `json:"conn_limit"`
	SSLCertData           string                   `json:"ssl_cert_data"`
	SSLKeyData            string                   `json:"ssl_key_data"`
	SSLCertPath           string                   `json:"ssl_cert_path"`
	SSLKeyPath            string                   `json:"ssl_key_path"`
	WAFEnable             *bool                    `json:"waf_enable"`
	ACLDefaultAction      string `json:"acl_default_action"`
	ACLDefaultDenyStatus  int    `json:"acl_default_deny_status"`
	ACLDefaultRedirectURL string `json:"acl_default_redirect_url"`
	ACLRules              []struct {
		Conditions  []struct {
			Item     string `json:"item"`
			Operator string `json:"operator"`
			Value    string `json:"value"`
		} `json:"conditions,omitempty"`
		Action      string `json:"action"`
		DenyStatus  int    `json:"deny_status,omitempty"`
		RedirectURL string `json:"redirect_url,omitempty"`
		IP          string `json:"ip,omitempty"`
	} `json:"acl_rules"`
	BlackIPs                       []string         `json:"black_ips"`
	WhiteIPs                       []string         `json:"white_ips"`
	CCRuleID                       int64                    `json:"cc_rule_id"`
	CCAutoSwitch                   *struct {
		Enable bool  `json:"enable"`
		QPS    int   `json:"qps"`
		RuleID int64 `json:"rule_id"`
	} `json:"cc_auto_switch,omitempty"`
	CustomCCRules                  []map[string]interface{} `json:"custom_cc_rules,omitempty"`
	OriginProtocol                 string           `json:"origin_protocol"`
	OriginHTTPPort                 string           `json:"origin_http_port"`
	OriginHTTPSPort                string           `json:"origin_https_port"`
	OriginHostHeader               string           `json:"origin_host_header"`
	OriginSNI                      string           `json:"origin_sni"`
	OriginVerifyTLS                bool             `json:"origin_verify_tls"`
	Cache                          *edgeCacheConfig `json:"cache"`
	HttpListen                     []string         `json:"http_listen"`
	HttpsListen                    []string         `json:"https_listen"`
	HTTPSForce                     bool             `json:"https_force"`
	HTTPSRedirectPort              string           `json:"https_redirect_port"`
	HTTPSHSTS                      bool             `json:"https_hsts"`
	HTTPSHTTP2                     bool             `json:"https_http2"`
	HTTPSOCSP                      bool             `json:"https_ocsp"`
	HTTPSHTTP3                     bool             `json:"https_http3"`
	HTTPSSSLProtocols              string           `json:"https_ssl_protocols"`
	HTTPSSSLCiphers                string           `json:"https_ssl_ciphers"`
	HTTPSSSLPreferServerCiphers    string           `json:"https_ssl_prefer_server_ciphers"`
	ProxyConnectTimeout            string           `json:"proxy_connect_timeout"`
	ProxyReadTimeout               string           `json:"proxy_read_timeout"`
	ProxySendTimeout               string           `json:"proxy_send_timeout"`
	ProxyHTTPVersion               string           `json:"proxy_http_version"`
	OriginHTTPVersionPolicy        string           `json:"origin_http_version_policy"`
	OriginAutoDowngrade            bool             `json:"origin_auto_downgrade"`
	OriginDowngradeThreshold       int              `json:"origin_downgrade_threshold"`
	OriginDowngradeWindowSeconds   int              `json:"origin_downgrade_window_seconds"`
	OriginDowngradeCooldownSeconds int              `json:"origin_downgrade_cooldown_seconds"`
	ProxySSLProtocols              string           `json:"proxy_ssl_protocols"`
	EnableGzip                     bool             `json:"enable_gzip"`
	GzipTypes                      string           `json:"gzip_types"`
	EnableWebsocket                bool             `json:"enable_websocket"`
	EnableRange                    bool             `json:"enable_range"`
	BodyLimit                      int64            `json:"body_limit"`
	LogRequestHeader               bool             `json:"log_request_header"`
	LogResponseHeader              bool             `json:"log_response_header"`
	LogRequestBody                 bool             `json:"log_request_body"`
	LogRequestBodySizeLimit        int              `json:"log_request_body_size_limit"`
	OriginCert                     bool             `json:"origin_cert"`
	RealtimeIdentify               bool             `json:"realtime_identify"`
	RealtimeSend                   bool             `json:"realtime_send"`
	RealtimeReturn                 bool             `json:"realtime_return"`
	DefaultSite                    bool             `json:"default_site"`
	IPv6Enable                     bool             `json:"ipv6_enable"`
	LimitRate                      int64            `json:"limit_rate"`
	UpstreamKeepalive              bool             `json:"upstream_keepalive"`
	UpstreamKeepaliveConn          int              `json:"upstream_keepalive_conn"`
	UpstreamKeepaliveTimeout       int              `json:"upstream_keepalive_timeout"`
	ErrorPageLang                  string           `json:"error_page_lang"`
}

type edgeNginxConfig struct {
	LogsDir               string                 `json:"logs_dir"`
	WorkerProcesses       string                 `json:"worker_processes"`
	WorkerConnections     int                    `json:"worker_connections"`
	WorkerRlimitNofile    int                    `json:"worker_rlimit_nofile"`
	WorkerShutdownTimeout string                 `json:"worker_shutdown_timeout"`
	Resolver              string                 `json:"resolver"`
	ResolverTimeout       string                 `json:"resolver_timeout"`
	HTTP                  map[string]interface{} `json:"http"`
	Stream                map[string]interface{} `json:"stream"`
}

type edgeConfig struct {
	Domains            []edgeDomain                `json:"domains"`
	Upstreams          []edgeUpstream              `json:"upstreams"`
	Streams            []edgeStream                `json:"streams"`
	NodeBandwidthLimit string                      `json:"node_bandwidth_limit"`
	NodeLevel          int                         `json:"node_level"`
	Nginx              *edgeNginxConfig            `json:"nginx"`
	FallbackCertData   string                      `json:"fallback_cert_data"`
	FallbackKeyData    string                      `json:"fallback_key_data"`
	WAF                *edgeWAFConfig              `json:"waf,omitempty"`
	Resources          *edgeResources              `json:"resources,omitempty"`
	ErrorPageI18n      errorPageI18nSettings           `json:"error_page_i18n"`
	ErrorPages         map[string]errorPageDefinition  `json:"error_pages"`
	DefaultConfig      *edgeDefaultConfig          `json:"default_config,omitempty"`
	CCRules            map[string][]edgeCCRuleItem `json:"cc_rules,omitempty"`
	CCMatchers         map[string]edgeCCMatcher    `json:"cc_matchers,omitempty"`
	CCFilters          map[string]edgeCCFilter     `json:"cc_filters,omitempty"`
	IPUnblock          *struct {
		Rev int64    `json:"rev"`
		IPs []string `json:"ips,omitempty"`
	} `json:"ip_unblock,omitempty"`
}

type edgeWAFConfig struct {
	Enable             bool   `json:"enable"`
	BlockUnboundDomain bool   `json:"block_unbound_domain"`
	DefaultBlockAction string `json:"default_block_action"`
	AutoIPSetEnable    bool   `json:"auto_ipset_enable"`
	AutoIPSetThreshold int    `json:"auto_ipset_threshold"`

	BlockPageRateLimitEnable bool `json:"block_page_rate_limit_enable"`
	BlockPageRateLimit       int  `json:"block_page_rate_limit"`
	BlockPageTrafficFree     bool `json:"block_page_traffic_free"`

	BlacklistTimeout        int `json:"blacklist_timeout"`
	TempWhitelistTimeout    int `json:"temp_whitelist_timeout"`
	TempWhitelistLimitTotal int `json:"temp_whitelist_limit_total"`
	TempWhitelistLimitURL   int `json:"temp_whitelist_limit_url"`

	PreventTLSHandshake bool `json:"prevent_tls_handshake"`
	DisablePing         bool `json:"disable_ping"`

	DefaultPageProtection          string `json:"default_page_protection"`
	DefaultPageProtectionThreshold int    `json:"default_page_protection_threshold"`

	SecretKey            string `json:"secret_key"`
	NodeLogCleanStrategy string `json:"node_log_clean_strategy"`
	CCRuleAutoSwitch     bool   `json:"cc_rule_auto_switch"`

	AntiCCImageSource    string `json:"anti_cc_image_source"`
	AntiCCImageCustomURL string `json:"anti_cc_image_custom_url"`
	AntiCCType           string `json:"anti_cc_type"`
	AntiCCDebug          bool   `json:"anti_cc_debug"`

	WellKnownProtectionThreshold   int                `json:"well_known_protection_threshold"`
	ResourceProtectionEnable       bool               `json:"resource_protection_enable"`
	ResourceProtectionThreshold    int                `json:"resource_protection_threshold"`
	ResourceProtectionBlockTimeout int                `json:"resource_protection_block_timeout"`
	ResourceProtectionRules        []edgeResourceRule `json:"resource_protection_rules"`
}

type edgeResourceRule struct {
	Duration    int `json:"duration"`
	MaxRequests int `json:"max_requests"`
}

type edgeResources struct {
	Website edgeWebsiteResources `json:"website"`
	Forward edgeForwardResources `json:"forward"`
	Public  edgePublicResources  `json:"public"`
}

type edgeWebsiteResources struct {
	DefaultListen80 bool   `json:"default_listen_80"`
	LogStorageDir   string `json:"log_storage_dir"`
	LogStorageHours int    `json:"log_storage_hours"`
}

type edgeForwardResources struct {
	DisabledPorts string `json:"disabled_ports"`
}

type edgePublicResources struct {
	DisabledCustomPorts string `json:"disabled_custom_ports"`
	AllowedCustomPorts  string `json:"allowed_custom_ports"`
}

type edgeDefaultConfig struct {
	Website  edgeSiteTemplate `json:"website"`
	API      edgeSiteTemplate `json:"api"`
	Download edgeSiteTemplate `json:"download"`
}

type edgeSiteTemplate struct {
	CacheEnable bool   `json:"cache_enable"`
	CacheTTL    int    `json:"cache_ttl"`
	Gzip        bool   `json:"gzip"`
	WAFEnable   bool   `json:"waf_enable"`
	SSLCiphers  string `json:"ssl_ciphers"`
}

type edgeCCRuleItem struct {
	MatcherID int64  `json:"matcher_id,omitempty"`
	FilterID  int64  `json:"filter_id,omitempty"`
	Action    string `json:"action,omitempty"`
	Enabled   bool   `json:"enabled"`
}

type edgeCCMatcher struct {
	ID   int64  `json:"id"`
	Data string `json:"data"`
}

type edgeCCFilter struct {
	ID           int64  `json:"id"`
	Type         string `json:"type"`
	WithinSecond int    `json:"within_second"`
	MaxReq       int    `json:"max_req"`
	MaxReqPerURI int    `json:"max_req_per_uri"`
	Extra        string `json:"extra,omitempty"`
}

func normalizeEdgeDomainName(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return strings.TrimSpace(host)
}

func normalizeEdgeDomains(domains []edgeDomain) []edgeDomain {
	if len(domains) == 0 {
		return nil
	}
	out := make([]edgeDomain, 0, len(domains))
	seen := map[string]struct{}{}
	for _, domain := range domains {
		host := normalizeEdgeDomainName(domain.Name)
		if host == "" {
			continue
		}
		domain.Name = host
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, domain)
	}
	return out
}

func generateDynamicConfigs(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	cfg, err := parseEdgeConfigPayload(payload)
	if err != nil {
		return err
	}
	cfg.Domains = normalizeEdgeDomains(cfg.Domains)
	setLocalNginxConfig(cfg.Nginx)
	if fallback := parseResourcesFallback(payload); fallback != nil {
		if cfg.Resources == nil {
			cfg.Resources = fallback
		} else {
			mergeResources(cfg.Resources, fallback)
		}
	}
	if err := persistResources(cfg.Resources); err != nil {
		return err
	}
	if err := persistErrorPages(cfg.ErrorPageI18n, cfg.ErrorPages); err != nil {
		return err
	}
	if err := persistCCRules(cfg.CCRules, cfg.CCMatchers, cfg.CCFilters); err != nil {
		return err
	}
	if err := persistDefaultConfig(cfg.DefaultConfig); err != nil {
		return err
	}
	setLocalWAFConfig(cfg.WAF)
	if err := persistFallbackCert(cfg.FallbackCertData, cfg.FallbackKeyData); err != nil {
		return err
	}
	if err := persistDomainCertificates(cfg.Domains); err != nil {
		return err
	}
	if err := writeHTTPConfig(cfg); err != nil {
		return err
	}
	if err := writeStreamConfig(cfg.Streams); err != nil {
		return err
	}
	if err := writeMainConfig(cfg.Nginx); err != nil {
		return err
	}
	if err := writeEventsConfig(cfg.Nginx); err != nil {
		return err
	}
	cacheEnabled := hasAnyCacheEnabled(cfg.Domains, cfg.DefaultConfig)
	if err := writeHTTPGlobalConfig(cfg.Nginx, cacheEnabled); err != nil {
		return err
	}
	if err := writeStreamGlobalConfig(cfg.Nginx); err != nil {
		return err
	}
	setLocalNginxConfig(cfg.Nginx)
	return nil
}

func hasAnyCacheEnabled(domains []edgeDomain, defaults *edgeDefaultConfig) bool {
	if len(domains) == 0 {
		return false
	}
	effectiveDefaults := defaults
	if effectiveDefaults == nil {
		effectiveDefaults = LocalDefaultConf
	}
	for _, domain := range domains {
		item := domain
		applyDefaultConfigToDomain(&item, effectiveDefaults)
		if item.Cache != nil && item.Cache.Enable {
			return true
		}
	}
	return false
}

func parseResourcesFallback(payload []byte) *edgeResources {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	resRaw, ok := raw["resources"]
	if !ok || resRaw == nil {
		return nil
	}
	data, err := json.Marshal(resRaw)
	if err != nil {
		return nil
	}
	var res edgeResources
	if err := json.Unmarshal(data, &res); err != nil {
		return nil
	}
	return &res
}

func mergeResources(dst, src *edgeResources) {
	if dst == nil || src == nil {
		return
	}
	if dst.Website.LogStorageDir == "" && src.Website.LogStorageDir != "" {
		dst.Website.LogStorageDir = src.Website.LogStorageDir
	}
	if dst.Website.LogStorageHours == 0 && src.Website.LogStorageHours > 0 {
		dst.Website.LogStorageHours = src.Website.LogStorageHours
	}
	if dst.Forward.DisabledPorts == "" && src.Forward.DisabledPorts != "" {
		dst.Forward.DisabledPorts = src.Forward.DisabledPorts
	}
	if dst.Public.DisabledCustomPorts == "" && src.Public.DisabledCustomPorts != "" {
		dst.Public.DisabledCustomPorts = src.Public.DisabledCustomPorts
	}
	if dst.Public.AllowedCustomPorts == "" && src.Public.AllowedCustomPorts != "" {
		dst.Public.AllowedCustomPorts = src.Public.AllowedCustomPorts
	}
}

func persistResources(resources *edgeResources) error {
	if resources == nil {
		return nil
	}
	rootDir := runtimeRoot()
	path := filepath.Join(rootDir, "conf", "resources.json")
	if err := fsutil.WriteJSONAtomic(path, resources, true); err != nil {
		return err
	}
	setLocalResources(resources)
	// Apply log cleanup immediately when log retention settings change.
	cleanupStoredLogs()
	return nil
}

func persistErrorPages(i18n errorPageI18nSettings, pages map[string]errorPageDefinition) error {
	if len(pages) == 0 {
		return nil
	}
	rootDir := runtimeRoot()
	bundle := errorPageBundle{
		I18n:  i18n,
		Pages: pages,
	}
	path := filepath.Join(rootDir, "conf", "error_pages.json")
	if err := fsutil.WriteJSONAtomic(path, bundle, true); err != nil {
		return err
	}
	i18nPath := filepath.Join(rootDir, "conf", "error_page_i18n.json")
	if err := fsutil.WriteJSONAtomic(i18nPath, i18n, true); err != nil {
		return err
	}
	rendered := renderAllAgentErrorPages(pages, i18n)
	errorPageDir := filepath.Join(rootDir, "conf", "error_pages")
	if err := writeRenderedErrorPageFiles(errorPageDir, rendered); err != nil {
		return err
	}
	setLocalErrorPageBundle(&bundle)
	return nil
}

func persistCCRules(rules map[string][]edgeCCRuleItem, matchers map[string]edgeCCMatcher, filters map[string]edgeCCFilter) error {
	rootDir := runtimeRoot()
	rulesPath := filepath.Join(rootDir, "conf", "cc_rules.json")
	matchersPath := filepath.Join(rootDir, "conf", "cc_matchers.json")
	filtersPath := filepath.Join(rootDir, "conf", "cc_filters.json")

	if len(rules) > 0 {
		if err := fsutil.WriteJSONAtomic(rulesPath, rules, true); err != nil {
			return err
		}
	} else {
		_ = os.Remove(rulesPath)
	}

	if len(matchers) > 0 {
		if err := fsutil.WriteJSONAtomic(matchersPath, matchers, true); err != nil {
			return err
		}
	} else {
		_ = os.Remove(matchersPath)
	}

	if len(filters) > 0 {
		if err := fsutil.WriteJSONAtomic(filtersPath, filters, true); err != nil {
			return err
		}
	} else {
		_ = os.Remove(filtersPath)
	}

	localRules := parseCCRuleMap(rules)
	localMatchers := parseCCMatcherMap(matchers)
	localFilters := parseCCFilterMap(filters)
	setLocalCCRules(localRules, localMatchers, localFilters)
	return nil
}

func persistDefaultConfig(cfg *edgeDefaultConfig) error {
	if cfg == nil {
		return nil
	}
	rootDir := runtimeRoot()
	path := filepath.Join(rootDir, "conf", "default_config.json")
	if err := fsutil.WriteJSONAtomic(path, cfg, true); err != nil {
		return err
	}
	setLocalDefaultConfig(cfg)
	return nil
}

func persistFallbackCert(certData, keyData string) error {
	if strings.TrimSpace(certData) == "" || strings.TrimSpace(keyData) == "" {
		return nil
	}
	rootDir := runtimeRoot()
	certDir := filepath.Join(rootDir, "cert")
	if err := fsutil.EnsureDir(certDir); err != nil {
		return err
	}
	certPath := filepath.Join(certDir, "fallback.pem")
	keyPath := filepath.Join(certDir, "fallback.key")
	if err := fsutil.WriteFileAtomic(certPath, []byte(certData), 0o644); err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(keyPath, []byte(keyData), 0o600); err != nil {
		return err
	}
	return nil
}

func persistDomainCertificates(domains []edgeDomain) error {
	rootDir := runtimeRoot()
	certDir := filepath.Join(rootDir, "cert", "sites")
	if err := os.RemoveAll(certDir); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(certDir); err != nil {
		return err
	}
	for i := range domains {
		certData := strings.TrimSpace(domains[i].SSLCertData)
		keyData := strings.TrimSpace(domains[i].SSLKeyData)
		if certData == "" || keyData == "" {
			continue
		}
		baseName := sanitizeCertificateFileName(domains[i].Name)
		if baseName == "" {
			baseName = fmt.Sprintf("domain_%d", i+1)
		}
		certPath := filepath.Join(certDir, baseName+".pem")
		keyPath := filepath.Join(certDir, baseName+".key")
		if err := fsutil.WriteFileAtomic(certPath, []byte(certData), 0o644); err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomic(keyPath, []byte(keyData), 0o600); err != nil {
			return err
		}
		domains[i].SSLCertPath = filepath.ToSlash(certPath)
		domains[i].SSLKeyPath = filepath.ToSlash(keyPath)
	}
	return nil
}

func sanitizeCertificateFileName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		case r == '*':
			b.WriteString("_wildcard_")
		default:
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "._-")
}

func setLocalResources(resources *edgeResources) {
	if resources == nil {
		return
	}
	localConfigMu.Lock()
	LocalResources = resources
	localConfigMu.Unlock()
}

func setLocalErrorPageBundle(bundle *errorPageBundle) {
	if bundle == nil || len(bundle.Pages) == 0 {
		return
	}
	copyBundle := errorPageBundle{
		I18n:  bundle.I18n,
		Pages: make(map[string]errorPageDefinition, len(bundle.Pages)),
	}
	for key, def := range bundle.Pages {
		copyBundle.Pages[key] = def
	}
	localConfigMu.Lock()
	LocalErrorPageBundle = &copyBundle
	localConfigMu.Unlock()
}

func setLocalCCRules(rules map[int64][]edgeCCRuleItem, matchers map[int64]edgeCCMatcher, filters map[int64]edgeCCFilter) {
	copyRules := map[int64][]edgeCCRuleItem{}
	for key, list := range rules {
		items := make([]edgeCCRuleItem, 0, len(list))
		items = append(items, list...)
		copyRules[key] = items
	}
	copyMatchers := map[int64]edgeCCMatcher{}
	for key, matcher := range matchers {
		copyMatchers[key] = matcher
	}
	copyFilters := map[int64]edgeCCFilter{}
	for key, filter := range filters {
		copyFilters[key] = filter
	}
	localConfigMu.Lock()
	LocalCCRules = copyRules
	LocalCCMatchers = copyMatchers
	LocalCCFilters = copyFilters
	localConfigMu.Unlock()
}

func setLocalDefaultConfig(cfg *edgeDefaultConfig) {
	if cfg == nil {
		return
	}
	localConfigMu.Lock()
	LocalDefaultConf = cfg
	localConfigMu.Unlock()
}

func setLocalWAFConfig(cfg *edgeWAFConfig) {
	localConfigMu.Lock()
	LocalWAFConfig = cfg
	localConfigMu.Unlock()
}

func setLocalNginxConfig(cfg *edgeNginxConfig) {
	if cfg == nil {
		return
	}
	localConfigMu.Lock()
	LocalNginxConfig = cfg
	localConfigMu.Unlock()
}

func loadPersistedConfigs() {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return
	}

	var resources edgeResources
	if err := fsutil.ReadJSONFile(filepath.Join(rootDir, "conf", "resources.json"), &resources); err == nil {
		setLocalResources(&resources)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load resources.json failed: %v", err)
	}

	var bundle errorPageBundle
	if err := fsutil.ReadJSONFile(filepath.Join(rootDir, "conf", "error_pages.json"), &bundle); err == nil {
		setLocalErrorPageBundle(&bundle)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load error_pages.json failed: %v", err)
	}

	var defCfg edgeDefaultConfig
	if err := fsutil.ReadJSONFile(filepath.Join(rootDir, "conf", "default_config.json"), &defCfg); err == nil {
		setLocalDefaultConfig(&defCfg)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load default_config.json failed: %v", err)
	}

	ccRules, err := readCCRules(filepath.Join(rootDir, "conf", "cc_rules.json"))
	if err == nil {
		ccMatchers, matchErr := readCCMatchers(filepath.Join(rootDir, "conf", "cc_matchers.json"))
		ccFilters, filterErr := readCCFilters(filepath.Join(rootDir, "conf", "cc_filters.json"))
		if matchErr == nil && filterErr == nil {
			setLocalCCRules(ccRules, ccMatchers, ccFilters)
		} else {
			if matchErr != nil && !os.IsNotExist(matchErr) {
				log.Printf("[Warn] Load cc_matchers.json failed: %v", matchErr)
			}
			if filterErr != nil && !os.IsNotExist(filterErr) {
				log.Printf("[Warn] Load cc_filters.json failed: %v", filterErr)
			}
		}
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load cc_rules.json failed: %v", err)
	}

	loadPersistedNginxConfig()
}

func loadPersistedNginxConfig() {
	if CONFIG_PATH == "" {
		return
	}
	var cfg edgeConfig
	if err := fsutil.ReadJSONFile(CONFIG_PATH, &cfg); err == nil {
		setLocalNginxConfig(cfg.Nginx)
		setLocalWAFConfig(cfg.WAF)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load cdn_config.json failed: %v", err)
	}
}

func readCCRules(path string) (map[int64][]edgeCCRuleItem, error) {
	var raw map[string][]edgeCCRuleItem
	if err := fsutil.ReadJSONFile(path, &raw); err != nil {
		return nil, err
	}
	out := make(map[int64][]edgeCCRuleItem, len(raw))
	for key, items := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = items
		}
	}
	return out, nil
}

func parseCCRuleMap(raw map[string][]edgeCCRuleItem) map[int64][]edgeCCRuleItem {
	out := make(map[int64][]edgeCCRuleItem, len(raw))
	for key, items := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = items
		}
	}
	return out
}

func parseCCMatcherMap(raw map[string]edgeCCMatcher) map[int64]edgeCCMatcher {
	out := make(map[int64]edgeCCMatcher, len(raw))
	for key, item := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = item
		}
	}
	return out
}

func parseCCFilterMap(raw map[string]edgeCCFilter) map[int64]edgeCCFilter {
	out := make(map[int64]edgeCCFilter, len(raw))
	for key, item := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = item
		}
	}
	return out
}

func readCCMatchers(path string) (map[int64]edgeCCMatcher, error) {
	var raw map[string]edgeCCMatcher
	if err := fsutil.ReadJSONFile(path, &raw); err != nil {
		return nil, err
	}
	out := make(map[int64]edgeCCMatcher, len(raw))
	for key, item := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = item
		}
	}
	return out, nil
}

func readCCFilters(path string) (map[int64]edgeCCFilter, error) {
	var raw map[string]edgeCCFilter
	if err := fsutil.ReadJSONFile(path, &raw); err != nil {
		return nil, err
	}
	out := make(map[int64]edgeCCFilter, len(raw))
	for key, item := range raw {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = item
		}
	}
	return out, nil
}

func writeHTTPDirectives(b *strings.Builder, httpCfg map[string]interface{}) {
	if httpCfg == nil {
		return
	}
	directives := map[string]string{
		"proxy_request_buffering":     "proxy_request_buffering",
		"proxy_buffering":             "proxy_buffering",
		"proxy_http_version":          "proxy_http_version",
		"proxy_next_upstream":         "proxy_next_upstream",
		"proxy_max_temp_file_size":    "proxy_max_temp_file_size",
		"proxy_connect_timeout":       "proxy_connect_timeout",
		"proxy_send_timeout":          "proxy_send_timeout",
		"proxy_read_timeout":          "proxy_read_timeout",
		"proxy_cache_revalidate":      "proxy_cache_revalidate",
		"client_max_body_size":        "client_max_body_size",
		"large_client_header_buffers": "large_client_header_buffers",
		"gzip":                        "gzip",
		"keepalive_timeout":           "keepalive_timeout",
		"keepalive_requests":          "keepalive_requests",
		"reset_timedout_connection":   "reset_timedout_connection",
		"sendfile_max_chunk":          "sendfile_max_chunk",
		"client_header_timeout":       "client_header_timeout",
		"client_body_timeout":         "client_body_timeout",
		"gzip_comp_level":             "gzip_comp_level",
		"gzip_http_version":           "gzip_http_version",
		"gzip_min_length":             "gzip_min_length",
		"gzip_vary":                   "gzip_vary",
		"server_tokens":               "server_tokens",
		"log_not_found":               "log_not_found",
		"default_type":                "default_type",
		"open_file_cache":             "open_file_cache",
		"open_file_cache_valid":       "open_file_cache_valid",
		"open_file_cache_min_uses":    "open_file_cache_min_uses",
		"open_file_cache_errors":      "open_file_cache_errors",
	}
	for key, directive := range directives {
		if value, ok := httpCfg[key]; ok {
			if rendered := renderValue(value); rendered != "" {
				b.WriteString(directive + " " + rendered + ";\n")
			}
		}
	}
}

func renderValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case bool:
		if v {
			return "on"
		}
		return "off"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return toString(value)
	}
}
