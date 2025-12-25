package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)

type ConfigService struct{}

// NewConfigService creates a new instance
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// GenerateConfigForNode constructs the configuration for a specific node
func (s *ConfigService) GenerateConfigForNode(nodeID string) (*models.EdgeConfig, error) {
	node, err := findNode(nodeID)
	if err != nil {
		return nil, err
	}

	payload := &models.EdgeConfig{
		Version:   0,
		NodeID:    strconv.FormatInt(node.ID, 10),
		Domains:   make([]models.EdgeDomain, 0),
		Upstreams: make([]models.EdgeUpstream, 0),
	}

	if globalCfg := loadGlobalConfig(); globalCfg != nil {
		payload.WAF = &globalCfg.WAF
	}
	if nginxCfg := loadNginxConfig(); nginxCfg != nil {
		payload.Nginx = nginxCfg
	}

	// 1. Find Node Groups this node belongs to
	var lines []models.Line
	if err := db.DB.Where("node_id = ?", node.ID).Find(&lines).Error; err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		payload.Version = hashConfigVersion(payload)
		return payload, nil
	}

	var groupIDs []int64
	for _, l := range lines {
		groupIDs = append(groupIDs, l.NodeGroupID)
	}

	// 2. Find Sites assigned to these Node Groups
	var sites []models.Site
	if err := db.DB.Where("node_group_id IN ?", groupIDs).Find(&sites).Error; err != nil {
		return nil, err
	}

	// Preload certs for HTTPS mapping
	var certs []models.Cert
	_ = db.DB.Where("enable = ?", true).Find(&certs).Error

	usedRuleIDs := make([]int64, 0)
	for _, site := range sites {
		if !site.Enable {
			continue
		}

		effectiveSite := cloneSiteForConfig(site)
		if defaults, err := GetSiteDefaultMap(site.UserID); err == nil {
			ApplySiteDefaults(effectiveSite, defaults)
		}

		cacheCfg := extractCacheConfig(effectiveSite.Settings)
		originProtocol, originHTTPPort, originHTTPSPort := extractOriginConfig(*effectiveSite)
		httpsCfg := extractHTTPSConfig(effectiveSite.Settings)
		advCfg := extractAdvancedConfig(effectiveSite.Settings)
		proxyTimeouts := extractProxyTimeouts(effectiveSite.Settings)

		upstreamKey := fmt.Sprintf("upstream_%d", effectiveSite.ID)
		targets := buildUpstreamTargets(effectiveSite.Backends, effectiveSite.BackendProtocol)
		if len(targets) > 0 {
			payload.Upstreams = append(payload.Upstreams, models.EdgeUpstream{
				ID:      upstreamKey,
				Targets: targets,
			})
		}

		policy := mapBalancePolicy(effectiveSite.BalanceWay)
		headers := buildHeaderMap(effectiveSite.Settings)
		responseHeaders := buildResponseHeaderMap(effectiveSite.Settings)
		hasHTTPS := len(effectiveSite.HttpsListen) > 0 || strings.TrimSpace(effectiveSite.HttpsListenRaw) != ""

		aclDefault, aclRules := buildACLForSite(*effectiveSite)
		if effectiveSite.CcDefaultRule > 0 {
			usedRuleIDs = append(usedRuleIDs, effectiveSite.CcDefaultRule)
		}
		for _, domain := range effectiveSite.Domains {
			domainConf := models.EdgeDomain{
				Name:              domain,
				UpstreamKey:       upstreamKey,
				LoadBalancePolicy: policy,
				Headers:           headers,
				ResponseHeaders:   responseHeaders,
				Status:            "active",
				ACLDefaultAction:  aclDefault,
				ACLRules:          aclRules,
				BlackIPs:          parseIPList(effectiveSite.BlackIPRaw),
				WhiteIPs:          parseIPList(effectiveSite.WhiteIPRaw),
				CCRuleID:          effectiveSite.CcDefaultRule,
				OriginProtocol:    originProtocol,
				OriginHTTPPort:    originHTTPPort,
				OriginHTTPSPort:   originHTTPSPort,
				Cache:             cacheCfg,
				HttpListen:        effectiveSite.HttpListen,
				HttpsListen:       effectiveSite.HttpsListen,
				HTTPSForce:        httpsCfg.force,
				HTTPSRedirectPort: httpsCfg.redirectPort,
				HTTPSHSTS:         httpsCfg.hsts,
				HTTPSHTTP2:        httpsCfg.http2,
				HTTPSSSLProtocols: httpsCfg.sslProtocols,
				HTTPSSSLCiphers:   httpsCfg.sslCiphers,
				HTTPSSSLPreferServerCiphers: httpsCfg.sslPreferServerCiphers,
				ProxyConnectTimeout: proxyTimeouts.connectTimeout,
				ProxyReadTimeout:    proxyTimeouts.readTimeout,
				ProxySendTimeout:    proxyTimeouts.sendTimeout,
				ProxyHTTPVersion:    advCfg.proxyHTTPVersion,
				ProxySSLProtocols:   advCfg.proxySSLProtocols,
				EnableGzip:          advCfg.gzip,
				GzipTypes:           advCfg.gzipTypes,
				EnableWebsocket:     advCfg.websocket,
				EnableRange:         advCfg.rangeEnabled,
				BodyLimit:           advCfg.bodyLimit,
				UpstreamKeepalive:   advCfg.keepalive,
				UpstreamKeepaliveConn: advCfg.keepaliveConn,
				UpstreamKeepaliveTimeout: advCfg.keepaliveTimeout,
			}
			if hasHTTPS {
				cert := findCertForDomain(domain, certs)
				if cert != nil {
					domainConf.SSLCertData = cert.Cert
					domainConf.SSLKeyData = cert.Key
				}
			}
			payload.Domains = append(payload.Domains, domainConf)
		}
	}

	payload.Streams = buildStreamsForNode(groupIDs)

	if len(usedRuleIDs) > 0 {
		ccRules, ccMatchers, ccFilters, err := loadCCData(usedRuleIDs)
		if err == nil {
			payload.CCRules = ccRules
			payload.CCMatchers = ccMatchers
			payload.CCFilters = ccFilters
		}
	}

	payload.Version = hashConfigVersion(payload)
	return payload, nil
}

func hashConfigVersion(cfg *models.EdgeConfig) int64 {
	clone := *cfg
	clone.Version = 0
	b, err := json.Marshal(clone)
	if err != nil {
		return time.Now().Unix()
	}
	h := fnv.New64a()
	_, _ = h.Write(b)
	return int64(h.Sum64())
}

func findNode(nodeID string) (*models.Node, error) {
	if nodeID == "" {
		return nil, errors.New("node_id is required")
	}

	var node models.Node
	if id, err := strconv.ParseInt(nodeID, 10, 64); err == nil {
		if err := db.DB.Where("id = ?", id).First(&node).Error; err == nil {
			return &node, nil
		}
	}

	if err := db.DB.Where("name = ?", nodeID).First(&node).Error; err != nil {
		return nil, errors.New("node not found")
	}
	return &node, nil
}

func loadGlobalConfig() *models.GlobalConfig {
	var sys models.SysConfig
	if err := db.DB.First(&sys, "`key` = ?", "global_config").Error; err != nil {
		return nil
	}
	if sys.Value == "" {
		return nil
	}
	var cfg models.GlobalConfig
	if err := json.Unmarshal([]byte(sys.Value), &cfg); err != nil {
		return nil
	}
	return &cfg
}

func buildACLForSite(site models.Site) (string, []models.EdgeACLRule) {
	if site.Settings == nil {
		return "", nil
	}
	access, ok := site.Settings["access"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	aclID := parseACLID(access["acl"])
	if aclID == 0 {
		return "", nil
	}
	var acl models.ACL
	if err := db.DB.Where("id = ?", aclID).First(&acl).Error; err != nil {
		return "", nil
	}
	defaultAction := strings.TrimSpace(acl.DefaultAction)
	if defaultAction == "" {
		defaultAction = "allow"
	}
	rules := parseACLRules(acl.Data)
	return defaultAction, rules
}

func parseACLRules(raw string) []models.EdgeACLRule {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []models.EdgeACLRule
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	// Try list of objects
	var generic []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil
	}
	for _, item := range generic {
		entry := models.EdgeACLRule{}
		if v, ok := item["ip"].(string); ok {
			entry.IP = v
		}
		if v, ok := item["action"].(string); ok {
			entry.Action = v
		}
		if entry.IP != "" {
			if entry.Action == "" {
				entry.Action = "allow"
			}
			items = append(items, entry)
		}
	}
	return items
}

func parseACLID(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return id
		}
	}
	return 0
}

func buildUpstreamTargets(backends []string, protocol string) []models.EdgeUpstreamTarget {
	var targets []models.EdgeUpstreamTarget
	for _, backend := range backends {
		if strings.TrimSpace(backend) == "" {
			continue
		}
		// Basic implementation
		targets = append(targets, models.EdgeUpstreamTarget{
			Addr:   backend,
			Weight: 10,
		})
	}
	return targets
}

func mapBalancePolicy(way string) string {
	switch way {
	case "ip_hash":
		return "ip_hash"
	case "random":
		return "random"
	default:
		return "round_robin"
	}
}

func buildHeaderMap(settings map[string]interface{}) map[string]string {
	if settings == nil {
		return nil
	}
	result := make(map[string]string)
	headers, ok := settings["headers"].(map[string]interface{})
	if ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if ok {
		if list, ok := advanced["origin_headers"].([]interface{}); ok {
			for _, item := range list {
				if m, ok := item.(map[string]interface{}); ok {
					name := parseString(m["name"])
					value := parseString(m["value"])
					if name != "" && value != "" {
						result[name] = value
					}
				}
			}
		}
	}
	return result
}

func buildResponseHeaderMap(settings map[string]interface{}) map[string]string {
	if settings == nil {
		return nil
	}
	advanced, ok := settings["advanced"].(map[string]interface{})
	if !ok {
		return nil
	}
	list, ok := advanced["cdn_headers"].([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			name := parseString(m["name"])
			value := parseString(m["value"])
			if name != "" && value != "" {
				result[name] = value
			}
		}
	}
	return result
}

func findCertForDomain(domain string, certs []models.Cert) *models.Cert {
	for _, cert := range certs {
		if cert.Domain == domain {
			return &cert
		}
	}
	return nil
}

func parseIPList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed []string
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return normalizeIPList(parsed)
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r == ',' || r == ';' || r == ' '
	})
	return normalizeIPList(fields)
}

func normalizeIPList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		ip := strings.TrimSpace(item)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type httpsConfig struct {
	force                 bool
	redirectPort          string
	hsts                  bool
	http2                 bool
	sslProtocols          string
	sslCiphers            string
	sslPreferServerCiphers string
}

type advancedConfig struct {
	gzip             bool
	gzipTypes        string
	websocket        bool
	rangeEnabled     bool
	proxyHTTPVersion string
	proxySSLProtocols string
	bodyLimit        int64
	keepalive        bool
	keepaliveConn    int
	keepaliveTimeout int
}

type proxyTimeoutConfig struct {
	connectTimeout string
	readTimeout    string
	sendTimeout    string
}

func cloneSiteForConfig(site models.Site) *models.Site {
	cloned := site
	if site.Settings != nil {
		var copyMap map[string]interface{}
		if b, err := json.Marshal(site.Settings); err == nil {
			_ = json.Unmarshal(b, &copyMap)
		}
		cloned.Settings = copyMap
	}
	return &cloned
}

func extractHTTPSConfig(settings map[string]interface{}) httpsConfig {
	cfg := httpsConfig{}
	httpsCfg := getMap(settings, "https")
	if httpsCfg == nil {
		return cfg
	}
	cfg.force = parseBoolValue(httpsCfg["force"], false)
	cfg.redirectPort = parseString(httpsCfg["redirect_port"])
	cfg.hsts = parseBoolValue(httpsCfg["hsts"], false)
	cfg.http2 = parseBoolValue(httpsCfg["http2"], false)
	cfg.sslProtocols = parseString(httpsCfg["ssl_protocols"])
	cfg.sslCiphers = parseString(httpsCfg["ssl_ciphers"])
	cfg.sslPreferServerCiphers = parseString(httpsCfg["ssl_prefer_server_ciphers"])
	return cfg
}

func extractAdvancedConfig(settings map[string]interface{}) advancedConfig {
	cfg := advancedConfig{}
	adv := getMap(settings, "advanced")
	if adv == nil {
		return cfg
	}
	cfg.gzip = parseBoolValue(adv["gzip"], false)
	cfg.gzipTypes = parseString(adv["gzip_types"])
	cfg.websocket = parseBoolValue(adv["websocket"], false)
	cfg.rangeEnabled = parseBoolValue(adv["range"], false)
	cfg.proxyHTTPVersion = parseString(adv["proxy_http_version"])
	cfg.proxySSLProtocols = parseString(adv["proxy_ssl_protocols"])
	cfg.bodyLimit = int64(parseIntValue(adv["body_limit"], 0))
	cfg.keepalive = parseBoolValue(adv["ups_keepalive"], false)
	cfg.keepaliveConn = parseIntValue(adv["ups_keepalive_conn"], 0)
	cfg.keepaliveTimeout = parseIntValue(adv["ups_keepalive_timeout"], 0)
	return cfg
}

func extractProxyTimeouts(settings map[string]interface{}) proxyTimeoutConfig {
	cfg := proxyTimeoutConfig{}
	backsource := getMap(settings, "backsource")
	if backsource == nil {
		return cfg
	}
	cfg.connectTimeout = parseString(backsource["connect_timeout"])
	if cfg.connectTimeout == "" {
		cfg.connectTimeout = parseString(backsource["timeout"])
	}
	cfg.readTimeout = parseString(backsource["timeout"])
	cfg.sendTimeout = parseString(backsource["timeout"])
	return cfg
}

func loadNginxConfig() *models.EdgeNginxConfig {
	values, err := LoadConfigMap("nginx_config", "global", 0)
	if err != nil {
		return nil
	}
	raw := strings.TrimSpace(values["nginx-config-file"])
	if raw == "" {
		return nil
	}
	var cfg models.EdgeNginxConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	return &cfg
}

func extractCacheConfig(settings map[string]interface{}) *models.EdgeCacheConfig {
	if settings == nil {
		return nil
	}
	cacheCfg := getMap(settings, "cache")
	if cacheCfg == nil {
		return nil
	}
	enable := parseBoolValue(cacheCfg["enable"], true)
	defaultTTL := parseIntValue(cacheCfg["ttl"], 0)
	rules := parseCacheRulesFromSettings(cacheCfg["rules"])
	return &models.EdgeCacheConfig{
		Enable:     enable,
		DefaultTTL: defaultTTL,
		Rules:      rules,
	}
}

func parseCacheRulesFromSettings(raw interface{}) []models.EdgeCacheRule {
	if raw == nil {
		return nil
	}
	rules := make([]models.EdgeCacheRule, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				rules = append(rules, mapToCacheRule(m))
			}
		}
	case []map[string]interface{}:
		for _, item := range v {
			rules = append(rules, mapToCacheRule(item))
		}
	default:
		if b, err := json.Marshal(raw); err == nil {
			var list []map[string]interface{}
			if err := json.Unmarshal(b, &list); err == nil {
				for _, item := range list {
					rules = append(rules, mapToCacheRule(item))
				}
			}
		}
	}
	if len(rules) == 0 {
		return nil
	}
	return rules
}

func mapToCacheRule(raw map[string]interface{}) models.EdgeCacheRule {
	return models.EdgeCacheRule{
		Rule:   parseString(raw["rule"]),
		Ext:    parseString(raw["ext"]),
		URI:    parseString(raw["uri"]),
		Prefix: parseString(raw["prefix"]),
		TTL:    parseIntValue(raw["ttl"], 0),
		Enable: parseBoolPtr(raw["enable"]),
		NoCache: parseBoolValue(raw["no_cache"], false),
		ForceCache: parseBoolValue(raw["force_cache"], false),
		Priority: parseIntValue(raw["priority"], 0),
		IgnoreArgs: parseBoolValue(raw["ignore_args"], false),
		CacheKey: parseString(raw["cache_key"]),
	}
}

func extractOriginConfig(site models.Site) (string, string, string) {
	protocol := strings.TrimSpace(site.BackendProtocol)
	httpPort := ""
	httpsPort := ""
	if site.Settings != nil {
		backsourceCfg := getMap(site.Settings, "backsource")
		if backsourceCfg != nil {
			if protocol == "" {
				protocol = parseString(backsourceCfg["protocol"])
			}
			httpPort = parseString(backsourceCfg["http_port"])
			httpsPort = parseString(backsourceCfg["https_port"])
		}
	}
	return protocol, httpPort, httpsPort
}

func buildStreamsForNode(groupIDs []int64) []models.EdgeStream {
	if len(groupIDs) == 0 {
		return nil
	}
	var forwards []models.Forward
	if err := db.DB.Where("node_group_id IN ? AND enable = ?", groupIDs, true).Find(&forwards).Error; err != nil {
		return nil
	}
	if len(forwards) == 0 {
		return nil
	}
	streams := make([]models.EdgeStream, 0, len(forwards))
	for _, forward := range forwards {
		effectiveForward := forward
		if defaults, err := GetStreamDefaultMap(forward.UserID); err == nil {
			ApplyForwardDefaults(&effectiveForward, defaults)
		}
		streamSettings := effectiveForward.Settings
		originCfg := getMap(streamSettings, "origin")

		connectTimeout := ""
		proxyTimeout := ""
		connLimit := 0
		if originCfg != nil {
			connectTimeout = parseString(originCfg["connect_timeout"])
			proxyTimeout = parseString(originCfg["proxy_timeout"])
			connLimit = parseIntValue(originCfg["conn_limit"], 0)
		}
		if connectTimeout == "" {
			connectTimeout = "10s"
		}
		if proxyTimeout == "" {
			proxyTimeout = "60s"
		}

		targets := make([]models.EdgeStreamTarget, 0, len(forward.Origins))
		for _, origin := range effectiveForward.Origins {
			if !origin.Enable {
				continue
			}
			targets = append(targets, models.EdgeStreamTarget{
				Addr:   origin.Address,
				Weight: origin.Weight,
				Enable: origin.Enable,
			})
		}
		streams = append(streams, models.EdgeStream{
			ID:            effectiveForward.ID,
			ListenPorts:   effectiveForward.ListenPorts,
			Targets:       targets,
			BalanceWay:    strings.TrimSpace(effectiveForward.BalanceWay),
			ProxyProtocol: effectiveForward.ProxyProtocol,
			ProxyConnectTimeout: connectTimeout,
			ProxyTimeout:        proxyTimeout,
			ConnLimit:           connLimit,
		})
	}
	return streams
}

func getMap(root map[string]interface{}, key string) map[string]interface{} {
	if root == nil {
		return nil
	}
	if val, ok := root[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func parseIntValue(value interface{}, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return fallback
}

func parseBoolValue(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "true" || v == "1" || v == "yes" || v == "on" {
			return true
		}
		if v == "false" || v == "0" || v == "no" || v == "off" {
			return false
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return fallback
}

func parseBoolPtr(value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		return &v
	case string:
		if v == "" {
			return nil
		}
		parsed := parseBoolValue(v, false)
		return &parsed
	case float64:
		parsed := v != 0
		return &parsed
	case int:
		parsed := v != 0
		return &parsed
	case int64:
		parsed := v != 0
		return &parsed
	}
	return nil
}

func loadCCData(ruleIDs []int64) (map[int64][]models.EdgeCCRuleItem, map[int64]models.EdgeCCMatcher, map[int64]models.EdgeCCFilter, error) {
	uniqueRuleIDs := uniqueInt64(ruleIDs)
	if len(uniqueRuleIDs) == 0 {
		return nil, nil, nil, nil
	}

	var rules []models.CCRule
	if err := db.DB.Where("id IN ? AND enable = ?", uniqueRuleIDs, true).Find(&rules).Error; err != nil {
		return nil, nil, nil, err
	}

	ccRules := make(map[int64][]models.EdgeCCRuleItem)
	matcherIDs := make([]int64, 0)
	filterIDs := make([]int64, 0)

	for _, rule := range rules {
		items := parseCCRuleData(rule.Data)
		if len(items) == 0 {
			ccRules[rule.ID] = []models.EdgeCCRuleItem{}
			continue
		}
		for _, item := range items {
			entry := models.EdgeCCRuleItem{
				MatcherID: parseACLID(item["matcher"]),
				FilterID:  parseACLID(item["filter1"]),
				Action:    parseString(item["action"]),
				Enabled:   parseBool(item["state"], true),
			}
			if entry.MatcherID == 0 {
				entry.MatcherID = parseACLID(item["matcher_id"])
			}
			if entry.FilterID == 0 {
				entry.FilterID = parseACLID(item["filter_id"])
			}
			ccRules[rule.ID] = append(ccRules[rule.ID], entry)
			if entry.MatcherID > 0 {
				matcherIDs = append(matcherIDs, entry.MatcherID)
			}
			if entry.FilterID > 0 {
				filterIDs = append(filterIDs, entry.FilterID)
			}
		}
	}

	ccMatchers := make(map[int64]models.EdgeCCMatcher)
	if len(matcherIDs) > 0 {
		var matchers []models.CCMatch
		if err := db.DB.Where("id IN ? AND enable = ?", uniqueInt64(matcherIDs), true).Find(&matchers).Error; err != nil {
			return ccRules, ccMatchers, nil, err
		}
		for _, matcher := range matchers {
			ccMatchers[matcher.ID] = models.EdgeCCMatcher{
				ID:   matcher.ID,
				Data: matcher.Data,
			}
		}
	}

	ccFilters := make(map[int64]models.EdgeCCFilter)
	if len(filterIDs) > 0 {
		var filters []models.CCFilter
		if err := db.DB.Where("id IN ? AND enable = ?", uniqueInt64(filterIDs), true).Find(&filters).Error; err != nil {
			return ccRules, ccMatchers, nil, err
		}
		for _, filter := range filters {
			ccFilters[filter.ID] = models.EdgeCCFilter{
				ID:           filter.ID,
				Type:         filter.Type,
				WithinSecond: filter.WithinSecond,
				MaxReq:       filter.MaxReq,
				MaxReqPerURI: filter.MaxReqPerUri,
				Extra:        filter.Extra,
			}
		}
	}

	return ccRules, ccMatchers, ccFilters, nil
}

func parseCCRuleData(raw string) []map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return list
	}
	var wrapper struct {
		Rules []map[string]interface{} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil {
		return wrapper.Rules
	}
	return nil
}

func uniqueInt64(input []int64) []int64 {
	if len(input) == 0 {
		return nil
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(input))
	for _, id := range input {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func parseString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "true" || v == "1" || v == "yes" {
			return true
		}
		if v == "false" || v == "0" || v == "no" {
			return false
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	}
	return fallback
}
