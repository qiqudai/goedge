package main

import (
	fsutil "cdn-common/io"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// startConfigPull checks for config updates
func startConfigPull() {
	// Pull every minute or use HTTP Long-Polling / Websocket in production
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		_ = pullConfig()
	}
}

func pullConfig() error {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/config?node_id="+NodeID, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	body, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		log.Printf("[Error] Config Pull Failed: %v", err)
		return err
	}

	if status == 200 {
		debugLogInteraction("GET", req.URL.String(), status, nil, body)

		newVersion := extractVersion(body)
		currentVersion := readLocalVersion()
		if newVersion != 0 && newVersion == currentVersion {
			log.Printf("[Info] Config unchanged (version=%d). Skipping reload.", currentVersion)
			return nil
		}

		if err := writeConfigWithBackup(body); err != nil {
			log.Printf("[Error] Failed to write config file: %v", err)
			return err
		}

		if err := generateDynamicConfigs(body); err != nil {
			log.Printf("[Error] Failed to generate dynamic configs: %v", err)
			return err
		}

		syncRuntimeLuaAssets()

		log.Printf("[Info] Config Updated (version=%d, %d bytes). Reloading Nginx...", newVersion, len(body))
		if err := executeReload(); err != nil {
			log.Printf("[Error] Reload Nginx Failed: %v", err)
			if restoreErr := restoreBackup(); restoreErr != nil {
				log.Printf("[Error] Failed to restore backup: %v", restoreErr)
				return fmt.Errorf("reload failed and restore failed: %v", restoreErr)
			}
			if retryErr := executeReload(); retryErr != nil {
				log.Printf("[Error] Reload after rollback failed: %v", retryErr)
				return fmt.Errorf("reload failed and rollback reload failed: %v", retryErr)
			} else {
				log.Println("[Warn] Rolled back to previous config")
			}
			return err
		}
		return nil
	}

	debugLogInteraction("GET", req.URL.String(), status, nil, nil)
	return fmt.Errorf("config pull status: %d", status)
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
	if WorkDir == "" {
		return
	}
	restoreDir("assets/lua", filepath.Join(WorkDir, "lua"))
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
}

type edgeStream struct {
	ID                  int64              `json:"id"`
	ListenPorts         []string           `json:"listen_ports"`
	Targets             []edgeStreamTarget `json:"targets"`
	BalanceWay          string             `json:"balance_way"`
	ProxyProtocol       bool               `json:"proxy_protocol"`
	ProxyConnectTimeout string             `json:"proxy_connect_timeout"`
	ProxyTimeout        string             `json:"proxy_timeout"`
	ConnLimit           int                `json:"conn_limit"`
}

type edgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

type edgeUpstream struct {
	ID      string               `json:"id"`
	Targets []edgeUpstreamTarget `json:"targets"`
}

type edgeCacheRule struct {
	Rule       string `json:"rule"`
	Ext        string `json:"ext"`
	URI        string `json:"uri"`
	Prefix     string `json:"prefix"`
	TTL        int    `json:"ttl"`
	Enable     *bool  `json:"enable"`
	NoCache    bool   `json:"no_cache"`
	ForceCache bool   `json:"force_cache"`
	Priority   int    `json:"priority"`
	IgnoreArgs bool   `json:"ignore_args"`
	CacheKey   string `json:"cache_key"`
}

type edgeCacheConfig struct {
	Enable     bool            `json:"enable"`
	DefaultTTL int             `json:"default_ttl"`
	Rules      []edgeCacheRule `json:"rules"`
}

type edgeDomain struct {
	Name              string            `json:"name"`
	UpstreamKey       string            `json:"upstream_key"`
	LoadBalancePolicy string            `json:"load_balance_policy"`
	Headers           map[string]string `json:"headers"`
	ResponseHeaders   map[string]string `json:"response_headers"`
	Status            string            `json:"status"`
	ConnLimit         int               `json:"conn_limit"`
	SSLCertData       string            `json:"ssl_cert_data"`
	SSLKeyData        string            `json:"ssl_key_data"`
	ACLDefaultAction  string            `json:"acl_default_action"`
	ACLRules          []struct {
		IP     string `json:"ip"`
		Action string `json:"action"`
	} `json:"acl_rules"`
	BlackIPs                    []string         `json:"black_ips"`
	WhiteIPs                    []string         `json:"white_ips"`
	CCRuleID                    int64            `json:"cc_rule_id"`
	OriginProtocol              string           `json:"origin_protocol"`
	OriginHTTPPort              string           `json:"origin_http_port"`
	OriginHTTPSPort             string           `json:"origin_https_port"`
	Cache                       *edgeCacheConfig `json:"cache"`
	HttpListen                  []string         `json:"http_listen"`
	HttpsListen                 []string         `json:"https_listen"`
	HTTPSForce                  bool             `json:"https_force"`
	HTTPSRedirectPort           string           `json:"https_redirect_port"`
	HTTPSHSTS                   bool             `json:"https_hsts"`
	HTTPSHTTP2                  bool             `json:"https_http2"`
	HTTPSSSLProtocols           string           `json:"https_ssl_protocols"`
	HTTPSSSLCiphers             string           `json:"https_ssl_ciphers"`
	HTTPSSSLPreferServerCiphers string           `json:"https_ssl_prefer_server_ciphers"`
	ProxyConnectTimeout         string           `json:"proxy_connect_timeout"`
	ProxyReadTimeout            string           `json:"proxy_read_timeout"`
	ProxySendTimeout            string           `json:"proxy_send_timeout"`
	ProxyHTTPVersion            string           `json:"proxy_http_version"`
	ProxySSLProtocols           string           `json:"proxy_ssl_protocols"`
	EnableGzip                  bool             `json:"enable_gzip"`
	GzipTypes                   string           `json:"gzip_types"`
	EnableWebsocket             bool             `json:"enable_websocket"`
	EnableRange                 bool             `json:"enable_range"`
	BodyLimit                   int64            `json:"body_limit"`
	LimitRate                   int64            `json:"limit_rate"`
	UpstreamKeepalive           bool             `json:"upstream_keepalive"`
	UpstreamKeepaliveConn       int              `json:"upstream_keepalive_conn"`
	UpstreamKeepaliveTimeout    int              `json:"upstream_keepalive_timeout"`
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
	Domains       []edgeDomain                `json:"domains"`
	Upstreams     []edgeUpstream              `json:"upstreams"`
	Streams       []edgeStream                `json:"streams"`
	Nginx         *edgeNginxConfig            `json:"nginx"`
	WAF           *edgeWAFConfig              `json:"waf,omitempty"`
	Resources     *edgeResources              `json:"resources,omitempty"`
	ErrorPages    map[string]string           `json:"error_pages,omitempty"`
	DefaultConfig *edgeDefaultConfig          `json:"default_config,omitempty"`
	CCRules       map[string][]edgeCCRuleItem `json:"cc_rules,omitempty"`
	CCMatchers    map[string]edgeCCMatcher    `json:"cc_matchers,omitempty"`
	CCFilters     map[string]edgeCCFilter     `json:"cc_filters,omitempty"`
}

type edgeWAFConfig struct {
	Enable bool `json:"enable"`
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

func generateDynamicConfigs(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	var cfg edgeConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return err
	}
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
	if err := persistErrorPages(cfg.ErrorPages); err != nil {
		return err
	}
	if err := persistCCRules(cfg.CCRules, cfg.CCMatchers, cfg.CCFilters); err != nil {
		return err
	}
	if err := persistDefaultConfig(cfg.DefaultConfig); err != nil {
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
	if err := writeHTTPGlobalConfig(cfg.Nginx); err != nil {
		return err
	}
	return writeStreamGlobalConfig(cfg.Nginx)
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
}

func persistResources(resources *edgeResources) error {
	if resources == nil {
		return nil
	}
	path := filepath.Join(WorkDir, "conf", "resources.json")
	if err := fsutil.WriteJSONAtomic(path, resources, true); err != nil {
		return err
	}
	setLocalResources(resources)
	return nil
}

func persistErrorPages(pages map[string]string) error {
	if len(pages) == 0 {
		return nil
	}
	path := filepath.Join(WorkDir, "conf", "error_pages.json")
	if err := fsutil.WriteJSONAtomic(path, pages, true); err != nil {
		return err
	}
	setLocalErrorPages(pages)
	return nil
}

func persistCCRules(rules map[string][]edgeCCRuleItem, matchers map[string]edgeCCMatcher, filters map[string]edgeCCFilter) error {
	rulesPath := filepath.Join(WorkDir, "conf", "cc_rules.json")
	matchersPath := filepath.Join(WorkDir, "conf", "cc_matchers.json")
	filtersPath := filepath.Join(WorkDir, "conf", "cc_filters.json")

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
	path := filepath.Join(WorkDir, "conf", "default_config.json")
	if err := fsutil.WriteJSONAtomic(path, cfg, true); err != nil {
		return err
	}
	setLocalDefaultConfig(cfg)
	return nil
}

func setLocalResources(resources *edgeResources) {
	if resources == nil {
		return
	}
	localConfigMu.Lock()
	LocalResources = resources
	localConfigMu.Unlock()
}

func setLocalErrorPages(pages map[string]string) {
	if len(pages) == 0 {
		return
	}
	copyPages := make(map[string]string, len(pages))
	for key, value := range pages {
		copyPages[key] = value
	}
	localConfigMu.Lock()
	LocalErrorPages = copyPages
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

func loadPersistedConfigs() {
	if WorkDir == "" {
		return
	}

	var resources edgeResources
	if err := fsutil.ReadJSONFile(filepath.Join(WorkDir, "conf", "resources.json"), &resources); err == nil {
		setLocalResources(&resources)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load resources.json failed: %v", err)
	}

	var pages map[string]string
	if err := fsutil.ReadJSONFile(filepath.Join(WorkDir, "conf", "error_pages.json"), &pages); err == nil {
		setLocalErrorPages(pages)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load error_pages.json failed: %v", err)
	}

	var defCfg edgeDefaultConfig
	if err := fsutil.ReadJSONFile(filepath.Join(WorkDir, "conf", "default_config.json"), &defCfg); err == nil {
		setLocalDefaultConfig(&defCfg)
	} else if !os.IsNotExist(err) {
		log.Printf("[Warn] Load default_config.json failed: %v", err)
	}

	ccRules, err := readCCRules(filepath.Join(WorkDir, "conf", "cc_rules.json"))
	if err == nil {
		ccMatchers, matchErr := readCCMatchers(filepath.Join(WorkDir, "conf", "cc_matchers.json"))
		ccFilters, filterErr := readCCFilters(filepath.Join(WorkDir, "conf", "cc_filters.json"))
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

func writeStreamConfig(streams []edgeStream) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "stream.conf")
	if len(streams) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}

	var b strings.Builder
	for _, stream := range streams {
		if len(stream.ListenPorts) == 0 || len(stream.Targets) == 0 {
			continue
		}
		upstreamName := fmt.Sprintf("stream_up_%d", stream.ID)
		b.WriteString("upstream " + upstreamName + " {\n")
		switch strings.ToLower(stream.BalanceWay) {
		case "ip_hash":
			b.WriteString("    hash $remote_addr consistent;\n")
		case "least_conn":
			b.WriteString("    least_conn;\n")
		}
		for _, target := range stream.Targets {
			if !target.Enable || target.Addr == "" {
				continue
			}
			if target.Weight > 0 {
				b.WriteString(fmt.Sprintf("    server %s weight=%d;\n", target.Addr, target.Weight))
			} else {
				b.WriteString(fmt.Sprintf("    server %s;\n", target.Addr))
			}
		}
		b.WriteString("}\n")

		for _, port := range stream.ListenPorts {
			port = strings.TrimSpace(port)
			if port == "" {
				continue
			}
			b.WriteString("server {\n")
			if stream.ProxyProtocol {
				b.WriteString("    listen " + port + " proxy_protocol;\n")
			} else {
				b.WriteString("    listen " + port + ";\n")
			}
			b.WriteString("    proxy_pass " + upstreamName + ";\n")
			if stream.ProxyConnectTimeout != "" {
				b.WriteString("    proxy_connect_timeout " + stream.ProxyConnectTimeout + ";\n")
			} else {
				b.WriteString("    proxy_connect_timeout 10s;\n")
			}
			if stream.ProxyTimeout != "" {
				b.WriteString("    proxy_timeout " + stream.ProxyTimeout + ";\n")
			} else {
				b.WriteString("    proxy_timeout 60s;\n")
			}
			if stream.ConnLimit > 0 {
				b.WriteString(fmt.Sprintf("    limit_conn stream_conn %d;\n", stream.ConnLimit))
			}
			b.WriteString("}\n")
		}
	}

	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeHTTPConfig(cfg edgeConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "http.conf")
	if len(cfg.Domains) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}

	errorPageDir := filepath.Join(WorkDir, "conf", "error_pages")
	if absDir, err := filepath.Abs(errorPageDir); err == nil {
		errorPageDir = absDir
	}
	errorPages := normalizeErrorPages(cfg.ErrorPages)
	if len(errorPages) > 0 {
		if err := writeErrorPageFiles(errorPageDir, errorPages); err != nil {
			return err
		}
	}

	defaultListen80 := true
	if cfg.Resources != nil {
		defaultListen80 = cfg.Resources.Website.DefaultListen80
	}

	upstreamKeepalive := map[string]edgeDomain{}
	for _, domain := range cfg.Domains {
		if domain.UpstreamKey != "" && domain.UpstreamKeepalive {
			upstreamKeepalive[domain.UpstreamKey] = domain
		}
	}

	var b strings.Builder
	for _, upstream := range cfg.Upstreams {
		if upstream.ID == "" || len(upstream.Targets) == 0 {
			continue
		}
		b.WriteString("upstream " + upstream.ID + " {\n")
		for _, target := range upstream.Targets {
			if target.Addr == "" {
				continue
			}
			if target.Weight > 0 {
				b.WriteString(fmt.Sprintf("    server %s weight=%d;\n", target.Addr, target.Weight))
			} else {
				b.WriteString(fmt.Sprintf("    server %s;\n", target.Addr))
			}
		}
		if keep, ok := upstreamKeepalive[upstream.ID]; ok {
			conn := keep.UpstreamKeepaliveConn
			if conn <= 0 {
				conn = 32
			}
			b.WriteString(fmt.Sprintf("    keepalive %d;\n", conn))
		}
		b.WriteString("}\n")
	}

	for _, domain := range cfg.Domains {
		if domain.Name == "" || domain.UpstreamKey == "" {
			continue
		}
		writeDomainServers(&b, domain, errorPages, errorPageDir, defaultListen80)
	}

	if shouldBindDefaultHTTP(cfg.Domains, defaultListen80) {
		b.WriteString("server {\n")
		b.WriteString("    listen 80 default_server;\n")
		b.WriteString("    server_name _;\n")
		writeErrorPageDirectives(&b, errorPages, errorPageDir)
		b.WriteString("    location / {\n")
		b.WriteString("        return 404;\n")
		b.WriteString("    }\n")
		b.WriteString("}\n")
	}

	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func shouldBindDefaultHTTP(domains []edgeDomain, defaultListen80 bool) bool {
	if defaultListen80 {
		return true
	}
	for _, domain := range domains {
		if len(domain.HttpListen) > 0 {
			return true
		}
	}
	return false
}

func writeDomainServers(b *strings.Builder, domain edgeDomain, errorPages map[string]string, errorPageDir string, defaultListen80 bool) {
	httpPorts := domain.HttpListen
	if len(httpPorts) == 0 && defaultListen80 {
		httpPorts = []string{"80"}
	}
	httpsPorts := domain.HttpsListen

	blockedCode := blockedStatusCode(domain, errorPages)
	if blockedCode > 0 {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, blockedCode)
		}
		for _, port := range httpsPorts {
			writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, blockedCode)
		}
		return
	}

	if domain.HTTPSForce && len(httpsPorts) > 0 {
		writeHTTPSRedirectServer(b, domain, httpPorts, httpsPorts, errorPages, errorPageDir)
	} else {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false, errorPages, errorPageDir, 0)
		}
	}

	for _, port := range httpsPorts {
		writeHTTPServer(b, domain, port, true, errorPages, errorPageDir, 0)
	}
}

func writeHTTPSRedirectServer(b *strings.Builder, domain edgeDomain, httpPorts []string, httpsPorts []string, errorPages map[string]string, errorPageDir string) {
	redirectPort := domain.HTTPSRedirectPort
	if redirectPort == "" {
		redirectPort = "443"
	}
	for _, port := range httpPorts {
		if strings.TrimSpace(port) == "" {
			continue
		}
		b.WriteString("server {\n")
		b.WriteString("    listen " + port + ";\n")
		b.WriteString("    server_name " + domain.Name + ";\n")
		writeErrorPageDirectives(b, errorPages, errorPageDir)
		b.WriteString("    return 301 https://$host:" + redirectPort + "$request_uri;\n")
		b.WriteString("}\n")
	}
}

func writeHTTPServer(b *strings.Builder, domain edgeDomain, port string, tls bool, errorPages map[string]string, errorPageDir string, blockedCode int) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	if tls {
		if domain.HTTPSHTTP2 {
			b.WriteString("    listen " + port + " ssl http2;\n")
		} else {
			b.WriteString("    listen " + port + " ssl;\n")
		}
		b.WriteString("    ssl_certificate cert/fallback.pem;\n")
		b.WriteString("    ssl_certificate_key cert/fallback.key;\n")
		b.WriteString("    ssl_certificate_by_lua_block {\n")
		b.WriteString("        local ssl_mgr = require \"lua.ssl_manager\"\n")
		b.WriteString("        ssl_mgr.set_certificate()\n")
		b.WriteString("    }\n")
		if domain.HTTPSSSLProtocols != "" {
			b.WriteString("    ssl_protocols " + domain.HTTPSSSLProtocols + ";\n")
		}
		if domain.HTTPSSSLCiphers != "" {
			b.WriteString("    ssl_ciphers " + domain.HTTPSSSLCiphers + ";\n")
		}
		if domain.HTTPSSSLPreferServerCiphers != "" {
			b.WriteString("    ssl_prefer_server_ciphers " + domain.HTTPSSSLPreferServerCiphers + ";\n")
		}
		if domain.HTTPSHSTS {
			b.WriteString("    add_header Strict-Transport-Security \"max-age=31536000\" always;\n")
		}
	} else {
		b.WriteString("    listen " + port + ";\n")
	}
	b.WriteString("    server_name " + domain.Name + ";\n")
	if blockedCode > 0 {
		writeErrorPageDirectives(b, errorPages, errorPageDir)
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        return %d;\n", blockedCode))
		b.WriteString("    }\n")
		b.WriteString("}\n")
		return
	}
	if domain.BodyLimit > 0 {
		b.WriteString(fmt.Sprintf("    client_max_body_size %dm;\n", domain.BodyLimit))
	}
	if domain.EnableGzip {
		b.WriteString("    gzip on;\n")
		if domain.GzipTypes != "" {
			b.WriteString("    gzip_types " + domain.GzipTypes + ";\n")
		}
	}
	if domain.LimitRate > 0 {
		b.WriteString(fmt.Sprintf("    limit_rate %d;\n", domain.LimitRate))
	}
	if domain.ConnLimit > 0 {
		b.WriteString(fmt.Sprintf("    limit_conn addr_conn %d;\n", domain.ConnLimit))
	}

	if _, ok := errorPages["conn_limit"]; ok {
		b.WriteString("    limit_conn_status 429;\n")
	}

	writeErrorPageDirectives(b, errorPages, errorPageDir)

	b.WriteString("    set $cc_rule_id " + fmt.Sprintf("%d", domain.CCRuleID) + ";\n")

	writeCacheLocations(b, domain, tls)

	b.WriteString("}\n")
}

func normalizeErrorPages(pages map[string]string) map[string]string {
	if len(pages) == 0 {
		return nil
	}
	out := make(map[string]string, len(pages))
	for code, content := range pages {
		key := strings.TrimSpace(code)
		val := strings.TrimSpace(content)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeErrorPageFiles(dir string, pages map[string]string) error {
	if len(pages) == 0 {
		return nil
	}
	if err := fsutil.EnsureDir(dir); err != nil {
		return err
	}
	for code, content := range pages {
		filename := filepath.Join(dir, code+".html")
		if err := fsutil.WriteFileAtomic(filename, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func isNumericStatus(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func writeErrorPageDirectives(b *strings.Builder, pages map[string]string, dir string) {
	if len(pages) == 0 {
		return
	}
	for key := range pages {
		status := errorPageStatusForKey(key)
		if status == 0 {
			continue
		}
		fileName := key + ".html"
		uri := "/__cdn_error/" + fileName
		filePath := filepath.ToSlash(filepath.Join(dir, fileName))
		b.WriteString(fmt.Sprintf("    error_page %d %s;\n", status, uri))
		b.WriteString("    location = " + uri + " {\n")
		b.WriteString("        internal;\n")
		b.WriteString("        default_type text/html;\n")
		b.WriteString("        alias " + filePath + ";\n")
		b.WriteString("    }\n")
	}
}

func errorPageStatusForKey(key string) int {
	if isNumericStatus(key) {
		if v, err := strconv.Atoi(key); err == nil {
			return v
		}
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "traffic_limit":
		return 509
	case "site_locked":
		return 451
	case "domain_invalid":
		return 404
	case "conn_limit":
		return 429
	case "timeout":
		return 410
	case "ip":
		return 418
	default:
		return 0
	}
}

func blockedStatusCode(domain edgeDomain, pages map[string]string) int {
	status := strings.ToLower(strings.TrimSpace(domain.Status))
	var key string
	switch status {
	case "locked":
		key = "site_locked"
	case "expired":
		key = "timeout"
	case "traffic_limit":
		key = "traffic_limit"
	case "conn_limit":
		key = "conn_limit"
	default:
		return 0
	}
	if _, ok := pages[key]; !ok {
		return 0
	}
	return errorPageStatusForKey(key)
}

func writeCacheLocations(b *strings.Builder, domain edgeDomain, tls bool) {
	writeAcmeLocation(b)
	cacheCfg := domain.Cache
	rules := make([]edgeCacheRule, 0)
	if cacheCfg != nil && len(cacheCfg.Rules) > 0 {
		rules = append(rules, cacheCfg.Rules...)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	for _, rule := range rules {
		location := buildRuleLocation(rule)
		if location == "" {
			continue
		}
		b.WriteString("    location " + location + " {\n")
		writeProxyBlock(b, domain, tls, cacheCfg, &rule)
		b.WriteString("    }\n")
	}

	b.WriteString("    location / {\n")
	writeProxyBlock(b, domain, tls, cacheCfg, nil)
	b.WriteString("    }\n")
}

func writeAcmeLocation(b *strings.Builder) {
	acmeRoot := filepath.ToSlash(filepath.Join(WorkDir, "cert", "acme"))
	apiBase := strings.TrimRight(strings.TrimSpace(API_BaseURL), "/")
	if apiBase == "" {
		return
	}
	b.WriteString("    location ^~ /.well-known/acme-challenge/ {\n")
	b.WriteString("        root " + acmeRoot + ";\n")
	b.WriteString("        default_type text/plain;\n")
	b.WriteString("        try_files $uri @acme_master;\n")
	b.WriteString("    }\n")
	b.WriteString("    location @acme_master {\n")
	b.WriteString("        proxy_pass " + apiBase + ";\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("    }\n")
}

func buildRuleLocation(rule edgeCacheRule) string {
	if rule.Rule != "" {
		return normalizeRuleLocation(rule.Rule)
	}
	if rule.URI != "" {
		return "= " + rule.URI
	}
	if rule.Prefix != "" {
		return "^~ " + rule.Prefix
	}
	if rule.Ext != "" {
		ext := rule.Ext
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		return "~* \\" + ext + "$"
	}
	return ""
}

func normalizeRuleLocation(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	if strings.HasPrefix(rule, "=") {
		return rule
	}
	if strings.HasPrefix(rule, "^~") || strings.HasPrefix(rule, "~") {
		return rule
	}
	if strings.HasPrefix(rule, "/") {
		return "^~ " + rule
	}
	if strings.HasPrefix(rule, ".") {
		return "~* \\" + rule + "$"
	}
	return "~* " + rule
}

func writeProxyBlock(b *strings.Builder, domain edgeDomain, tls bool, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	b.WriteString("        limit_req zone=cc_limit burst=20 nodelay;\n")
	b.WriteString("        limit_conn addr_conn 50;\n")
	b.WriteString("        access_by_lua_file lua/access_guard.lua;\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	if domain.EnableWebsocket {
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection $connection_upgrade;\n")
	} else if domain.ProxyHTTPVersion != "" {
		b.WriteString("        proxy_http_version " + domain.ProxyHTTPVersion + ";\n")
	}
	if domain.ProxyConnectTimeout != "" {
		b.WriteString("        proxy_connect_timeout " + domain.ProxyConnectTimeout + ";\n")
	}
	if domain.ProxyReadTimeout != "" {
		b.WriteString("        proxy_read_timeout " + domain.ProxyReadTimeout + ";\n")
	}
	if domain.ProxySendTimeout != "" {
		b.WriteString("        proxy_send_timeout " + domain.ProxySendTimeout + ";\n")
	}
	if domain.EnableRange {
		b.WriteString("        proxy_force_ranges on;\n")
	}
	for k, v := range domain.Headers {
		if k == "" || v == "" {
			continue
		}
		b.WriteString("        proxy_set_header " + k + " " + v + ";\n")
	}
	for k, v := range domain.ResponseHeaders {
		if k == "" || v == "" {
			continue
		}
		b.WriteString("        add_header " + k + " " + v + " always;\n")
	}

	scheme := "http"
	if strings.ToLower(domain.OriginProtocol) == "https" {
		scheme = "https"
	}
	b.WriteString("        proxy_pass " + scheme + "://" + domain.UpstreamKey + ";\n")
	if scheme == "https" {
		b.WriteString("        proxy_ssl_server_name on;\n")
		if domain.ProxySSLProtocols != "" {
			b.WriteString("        proxy_ssl_protocols " + domain.ProxySSLProtocols + ";\n")
		}
	}

	applyCacheDirectives(b, cacheCfg, rule)
}

func applyCacheDirectives(b *strings.Builder, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	if cacheCfg == nil || !cacheCfg.Enable {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	enabled := true
	if rule != nil && rule.Enable != nil && !*rule.Enable {
		enabled = false
	}
	if rule != nil && rule.NoCache {
		enabled = false
	}
	if !enabled {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	b.WriteString("        proxy_cache my_cache;\n")
	b.WriteString("        proxy_cache_lock on;\n")
	b.WriteString("        proxy_cache_lock_timeout 5s;\n")
	b.WriteString("        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;\n")
	b.WriteString("        proxy_cache_background_update on;\n")
	if rule != nil && rule.ForceCache {
		b.WriteString("        proxy_ignore_headers Cache-Control Expires;\n")
	}
	ttl := cacheCfg.DefaultTTL
	if rule != nil && rule.TTL > 0 {
		ttl = rule.TTL
	}
	if ttl > 0 {
		b.WriteString(fmt.Sprintf("        proxy_cache_valid 200 302 %ds;\n", ttl))
	}
	cacheKey := ""
	if rule != nil && rule.CacheKey != "" {
		cacheKey = rule.CacheKey
	} else if rule != nil && rule.IgnoreArgs {
		cacheKey = "$host$uri"
	} else {
		cacheKey = "$host$uri$is_args$args"
	}
	b.WriteString("        proxy_cache_key " + cacheKey + ";\n")
}

func writeHTTPGlobalConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "http_global.conf")
	var b strings.Builder
	cacheDir := filepath.ToSlash(filepath.Join(WorkDir, "cache"))
	cacheMaxSize := ""
	cacheZoneSize := ""
	if cfg != nil && cfg.HTTP != nil {
		cacheDir = fallbackString(toString(cfg.HTTP["proxy_cache_dir"]), cacheDir)
		cacheMaxSize = toString(cfg.HTTP["proxy_cache_max_size"])
		cacheZoneSize = toString(cfg.HTTP["proxy_cache_keys_zone_size"])
	}
	zoneSize := "50m"
	if cacheZoneSize != "" {
		zoneSize = cacheZoneSize
	}
	cacheLine := "proxy_cache_path " + cacheDir + " levels=1:2 keys_zone=my_cache:" + zoneSize + " inactive=24h use_temp_path=off"
	if cacheMaxSize != "" {
		cacheLine = cacheLine + " max_size=" + cacheMaxSize
	}
	b.WriteString(cacheLine + ";\n")

	if cfg != nil && cfg.HTTP != nil {
		writeHTTPDirectives(&b, cfg.HTTP)
		if v := toString(cfg.HTTP["proxy_cache_methods"]); v != "" {
			b.WriteString("proxy_cache_methods " + v + ";\n")
		}
		if v := toString(cfg.HTTP["custom_snippet"]); v != "" {
			if !strings.HasSuffix(v, "\n") {
				v += "\n"
			}
			b.WriteString(v)
		}
	}
	if cfg != nil {
		if v := strings.TrimSpace(cfg.Resolver); v != "" {
			b.WriteString("resolver " + v + ";\n")
		}
		if v := strings.TrimSpace(cfg.ResolverTimeout); v != "" {
			b.WriteString("resolver_timeout " + v + ";\n")
		}
		if logs := strings.TrimSpace(cfg.LogsDir); logs != "" {
			logs = strings.TrimRight(logs, "/")
			b.WriteString("access_log " + logs + "/access.json json_analytics;\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeStreamGlobalConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "stream_global.conf")
	var b strings.Builder
	if cfg != nil && cfg.Stream != nil {
		if v := toString(cfg.Stream["proxy_connect_timeout"]); v != "" {
			b.WriteString("proxy_connect_timeout " + v + ";\n")
		}
		if v := toString(cfg.Stream["proxy_timeout"]); v != "" {
			b.WriteString("proxy_timeout " + v + ";\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeMainConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "main.conf")
	var b strings.Builder
	if cfg == nil {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}
	if v := strings.TrimSpace(cfg.WorkerProcesses); v != "" {
		b.WriteString("worker_processes " + v + ";\n")
	}
	if cfg.WorkerRlimitNofile > 0 {
		b.WriteString(fmt.Sprintf("worker_rlimit_nofile %d;\n", cfg.WorkerRlimitNofile))
	}
	if v := strings.TrimSpace(cfg.WorkerShutdownTimeout); v != "" {
		b.WriteString("worker_shutdown_timeout " + v + ";\n")
	}
	if logs := strings.TrimSpace(cfg.LogsDir); logs != "" {
		logs = strings.TrimRight(logs, "/")
		b.WriteString("error_log " + logs + "/error.log warn;\n")
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeEventsConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "events.conf")
	var b strings.Builder
	if cfg != nil && cfg.WorkerConnections > 0 {
		b.WriteString(fmt.Sprintf("worker_connections %d;\n", cfg.WorkerConnections))
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
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
		"client_max_body_size":        "client_max_body_size",
		"large_client_header_buffers": "large_client_header_buffers",
		"gzip":                        "gzip",
		"keepalive_timeout":           "keepalive_timeout",
		"keepalive_requests":          "keepalive_requests",
		"gzip_comp_level":             "gzip_comp_level",
		"gzip_http_version":           "gzip_http_version",
		"gzip_min_length":             "gzip_min_length",
		"gzip_vary":                   "gzip_vary",
		"server_tokens":               "server_tokens",
		"log_not_found":               "log_not_found",
		"default_type":                "default_type",
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
