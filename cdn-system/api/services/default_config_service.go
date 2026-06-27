package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// LoadConfigMap loads config items by type and scope.
func LoadConfigMap(cfgType string, scopeName string, scopeID int64) (map[string]string, error) {
	items := []models.ConfigItem{}
	err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ?", cfgType, scopeName, scopeID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		if !item.Enable {
			continue
		}
		result[item.Name] = item.Value
	}
	return result, nil
}

// MergeConfigMap merges user defaults over global defaults.
func MergeConfigMap(global map[string]string, user map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range global {
		result[k] = v
	}
	for k, v := range user {
		result[k] = v
	}
	return result
}

func GetSiteDefaultMap(userID int64) (map[string]string, error) {
	return GetSiteDefaultMapWithGroup(userID, 0)
}

func GetSiteDefaultMapWithGroup(userID, groupID int64) (map[string]string, error) {
	global, err := LoadConfigMap("site_default_config", "global", 0)
	if err != nil {
		return nil, err
	}
	// Load Cert Defaults (Global)
	certGlobal, err := LoadConfigMap("cert_default_config", "global", 0)
	if err == nil {
		global = MergeConfigMap(global, certGlobal)
	}

	legacyUser, err := LoadConfigMap("site_default_config", "user", userID)
	if err != nil {
		return global, nil
	}
	userGlobal, err := LoadConfigMap("site_default_config", "global", userID)
	if err != nil {
		return MergeConfigMap(global, legacyUser), nil
	}
	if len(legacyUser) == 0 && len(userGlobal) == 0 {
		return global, nil
	}
	merged := MergeConfigMap(global, legacyUser)
	merged = MergeConfigMap(merged, userGlobal)
	if groupID != 0 {
		groupDefaults, err := LoadConfigMap("site_default_config", "group", groupID)
		if err == nil && len(groupDefaults) > 0 {
			merged = MergeConfigMap(merged, groupDefaults)
		}
	}
	return merged, nil
}

func GetSiteScopedDefaultMap(userID, groupID int64) map[string]string {
	scoped := map[string]string{}
	if userID != 0 {
		if legacyUser, err := LoadConfigMap("site_default_config", "user", userID); err == nil && len(legacyUser) > 0 {
			scoped = MergeConfigMap(scoped, legacyUser)
		}
		if userGlobal, err := LoadConfigMap("site_default_config", "global", userID); err == nil && len(userGlobal) > 0 {
			scoped = MergeConfigMap(scoped, userGlobal)
		}
	}
	if groupID != 0 {
		if groupDefaults, err := LoadConfigMap("site_default_config", "group", groupID); err == nil && len(groupDefaults) > 0 {
			scoped = MergeConfigMap(scoped, groupDefaults)
		}
	}
	if len(scoped) == 0 {
		return nil
	}
	return scoped
}

func GetStreamDefaultMap(userID int64) (map[string]string, error) {
	global, err := LoadConfigMap("stream_default_config", "global", 0)
	if err != nil {
		return nil, err
	}
	if forwardDefaults := loadForwardDefaultMap(); len(forwardDefaults) > 0 {
		global = MergeConfigMap(global, forwardDefaults)
	}
	user, err := LoadConfigMap("stream_default_config", "user", userID)
	if err != nil {
		return global, nil
	}
	if len(user) == 0 {
		return global, nil
	}
	return MergeConfigMap(global, user), nil
}

type forwardDefaultItem struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	Scope   string      `json:"scope"`
	GroupID int64       `json:"group_id"`
}

const forwardDefaultKey = "forward_default_settings"

func loadForwardDefaultMap() map[string]string {
	var cfg models.SysConfig
	if err := db.DB.Where("name = ? AND type = ?", forwardDefaultKey, "system").First(&cfg).Error; err != nil {
		return nil
	}
	if strings.TrimSpace(cfg.Value) == "" {
		return nil
	}
	var items []forwardDefaultItem
	if err := json.Unmarshal([]byte(cfg.Value), &items); err != nil {
		return nil
	}
	out := map[string]string{}
	for _, item := range items {
		if item.Key == "" {
			continue
		}
		if scope := strings.ToLower(strings.TrimSpace(item.Scope)); scope != "" && scope != "global" {
			continue
		}
		switch v := item.Value.(type) {
		case bool:
			out[item.Key] = strconv.FormatBool(v)
		case float64:
			out[item.Key] = strconv.FormatInt(int64(v), 10)
		case string:
			out[item.Key] = v
		default:
			if b, err := json.Marshal(v); err == nil {
				out[item.Key] = string(b)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ApplySiteDefaults(site *models.Site, defaults map[string]string) {
	if site == nil || defaults == nil {
		return
	}

	if site.Settings == nil {
		site.Settings = map[string]interface{}{}
	}
	httpEnable := true
	if raw, ok := site.Settings["http_enable"]; ok {
		httpEnable = parseBoolValue(raw, true)
	}
	if httpEnable {
		if len(site.HttpListen) == 0 || (len(site.HttpListen) == 1 && site.HttpListen[0] == "80") {
			if v := defaults["http_listen-port"]; v != "" {
				site.HttpListen = splitFields(v)
			}
		}
	}
	httpsCfg := getSubMap(site.Settings, "https")
	setIfMissing(httpsCfg, "enable", false)
	if len(site.HttpsListen) == 0 && parseBoolValue(httpsCfg["enable"], false) {
		if v := defaults["https_listen-port"]; v != "" {
			site.HttpsListen = splitFields(v)
		}
	}
	if site.BalanceWay == "" {
		if v := defaults["balance_way"]; v != "" {
			site.BalanceWay = v
		}
	}
	if site.BackendProtocol == "" {
		if v := defaults["backend_protocol"]; v != "" {
			site.BackendProtocol = v
		}
	}
	if site.CcDefaultRule == 0 {
		if v := defaults["cc_default_rule"]; v != "" {
			site.CcDefaultRule = parseInt64(v)
		}
	}
	if site.DNSProviderID == 0 {
		if v := defaults["dns_provider_id"]; v != "" {
			site.DNSProviderID = parseInt64(v)
		}
	}
	if site.BlackIPRaw == "" {
		if v := defaults["black_ip"]; v != "" {
			site.BlackIPRaw = v
		}
	}
	if site.WhiteIPRaw == "" {
		if v := defaults["white_ip"]; v != "" {
			site.WhiteIPRaw = v
		}
	}

	httpsCfg = getSubMap(site.Settings, "https")
	setIfMissing(httpsCfg, "force", parseBool(defaults["https_listen-force_ssl_enable"], false))
	setIfMissing(httpsCfg, "redirect_port", defaults["https_listen-port"])
	setIfMissing(httpsCfg, "hsts", parseBool(defaults["https_listen-hsts"], false))
	setIfMissing(httpsCfg, "http2", parseBool(defaults["https_listen-http2"], false))
	setIfMissing(httpsCfg, "http3", parseBool(defaults["https_listen-http3"], false))
	setIfMissing(httpsCfg, "ocsp_stapling", parseBool(defaults["https_listen-ocsp_stapling"], false))
	setIfMissing(httpsCfg, "ssl_protocols", defaults["https_listen-ssl_protocols"])
	setIfMissing(httpsCfg, "ssl_ciphers", defaults["https_listen-ssl_ciphers"])
	setIfMissing(httpsCfg, "ssl_prefer_server_ciphers", defaults["https_listen-ssl_prefer_server_ciphers"])

	// Apply Cert defaults
	certCfg := getSubMap(site.Settings, "cert")
	setIfMissing(certCfg, "type", defaults["cert_default_type"])
	setIfMissing(certCfg, "dnsapi_type", defaults["cert_default_dnsapi_type"])
	// We might store data json string directly or parse it
	if v := defaults["cert_default_dnsapi_data"]; v != "" {
		var dataMap map[string]interface{}
		if json.Unmarshal([]byte(v), &dataMap) == nil {
			setIfMissing(certCfg, "dnsapi_data", dataMap)
		}
	}

	backsourceCfg := getSubMap(site.Settings, "backsource")
	applyLegacyBacksourceSettings(site, backsourceCfg)
	setIfMissing(backsourceCfg, "protocol", defaults["backend_protocol"])
	setIfMissing(backsourceCfg, "http_port", defaults["backend_http_port"])
	setIfMissing(backsourceCfg, "https_port", defaults["backend_https_port"])
	setIfMissing(backsourceCfg, "timeout", defaults["proxy_timeout"])
	setIfMissing(backsourceCfg, "connect_timeout", defaults["connect_timeout"])

	cacheCfg := getSubMap(site.Settings, "cache")
	if _, ok := cacheCfg["enable"]; !ok {
		raw := strings.TrimSpace(defaults["proxy_cache"])
		cacheCfg["enable"] = raw != "" && raw != "[]"
	}
	if _, ok := cacheCfg["rules"]; !ok {
		cacheCfg["rules"] = parseCacheRules(defaults["proxy_cache"])
	}

	securityCfg := getSubMap(site.Settings, "security")
	setIfMissing(securityCfg, "default_rule", site.CcDefaultRule)
	if v := defaults["security_bot"]; v != "" {
		setIfMissing(securityCfg, "crawlers_action", v)
	}
	if v := defaults["black_ip"]; v != "" {
		setIfMissing(securityCfg, "blacklist", splitFields(v))
	}
	if v := defaults["white_ip"]; v != "" {
		setIfMissing(securityCfg, "whitelist", splitFields(v))
	}
	if v := defaults["security_black_time"]; v != "" {
		setIfMissing(securityCfg, "ip_black_timeout", parseInt64(v))
		setIfMissing(securityCfg, "black_time_mode", "custom")
		setIfMissing(securityCfg, "black_time_custom", parseInt64(v))
	}
	if v := defaults["security_white_time"]; v != "" {
		setIfMissing(securityCfg, "ip_white_timeout", parseInt64(v))
		setIfMissing(securityCfg, "white_time_mode", "custom")
		setIfMissing(securityCfg, "white_time_custom", parseInt64(v))
	}
	if v := defaults["security_shield_proxy"]; v != "" {
		setIfMissing(securityCfg, "block_transparent_proxy", parseBool(v, false))
	}
	if v := defaults["block_region"]; v != "" {
		if site.BlockRegionRaw == "" {
			site.BlockRegionRaw = v
		}
		if _, ok := securityCfg["region_block"]; !ok {
			if v == "none" {
				securityCfg["region_block"] = []string{}
			} else {
				securityCfg["region_block"] = splitCommaList(v)
			}
		}
	}

	advCfg := getSubMap(site.Settings, "advanced")
	applyLegacyAdvancedSettings(site.Settings, advCfg)
	setIfMissing(advCfg, "gzip", parseBool(defaults["gzip_enable"], false))
	setIfMissing(advCfg, "gzip_types", defaults["gzip_types"])
	setIfMissing(advCfg, "websocket", parseBool(defaults["websocket_enable"], false))
	setIfMissing(advCfg, "ipv6", parseBool(defaults["ipv6_enable"], false))
	setIfMissing(advCfg, "range", parseBool(defaults["range"], false))
	setIfMissing(advCfg, "proxy_http_version", defaults["proxy_http_version"])
	setIfMissing(advCfg, "proxy_ssl_protocols", defaults["proxy_ssl_protocols"])
	setIfMissing(advCfg, "ups_keepalive", parseBool(defaults["ups_keepalive"], false))
	setIfMissing(advCfg, "ups_keepalive_conn", parseInt64(defaults["ups_keepalive_conn"]))
	setIfMissing(advCfg, "ups_keepalive_timeout", parseInt64(defaults["ups_keepalive_timeout"]))
	if v := defaults["post_size_limit"]; v != "" {
		setIfMissing(advCfg, "body_limit", parseInt64(v))
		setIfMissing(advCfg, "body_limit_unit", "kb")
	}
	setIfMissing(advCfg, "log_request_header", parseBool(defaults["log_request_header"], false))
	setIfMissing(advCfg, "log_response_header", parseBool(defaults["log_response_header"], false))
	setIfMissing(advCfg, "log_request_body", parseBool(defaults["log_request_body"], false))
	setIfMissing(advCfg, "realtime_send", parseBool(defaults["realtime_send"], false))
	setIfMissing(advCfg, "realtime_return", parseBool(defaults["realtime_return"], false))
	if v := defaults["origin_headers"]; v != "" {
		setIfMissing(advCfg, "origin_headers", parseHeaderList(v))
	}
}

func ApplySiteDefaultsScopedOverrides(site *models.Site, defaults map[string]string) {
	if site == nil || defaults == nil {
		return
	}
	if site.Settings == nil {
		site.Settings = map[string]interface{}{}
	}
	if v := defaults["gzip_enable"]; v != "" {
		advCfg := getSubMap(site.Settings, "advanced")
		advCfg["gzip"] = parseBool(v, false)
	}
	if v := defaults["https_listen-ssl_ciphers"]; v != "" {
		httpsCfg := getSubMap(site.Settings, "https")
		httpsCfg["ssl_ciphers"] = v
	}
	if v := defaults["proxy_cache"]; v != "" {
		cacheCfg := getSubMap(site.Settings, "cache")
		raw := strings.TrimSpace(v)
		cacheCfg["enable"] = raw != "" && raw != "[]"
		cacheCfg["rules"] = parseCacheRules(v)
	}
}

func ApplySiteTemplateDefaults(site *models.Site, template models.SiteTemplate) {
	if site == nil {
		return
	}
	if site.Settings == nil {
		site.Settings = map[string]interface{}{}
	}

	cacheCfg := getSubMap(site.Settings, "cache")
	if _, ok := cacheCfg["enable"]; !ok {
		cacheCfg["enable"] = template.CacheEnable
	}
	if template.CacheTTL > 0 {
		if _, ok := cacheCfg["ttl"]; !ok {
			cacheCfg["ttl"] = template.CacheTTL
		}
	}

	advCfg := getSubMap(site.Settings, "advanced")
	if _, ok := advCfg["gzip"]; !ok {
		advCfg["gzip"] = template.Gzip
	}

	httpsCfg := getSubMap(site.Settings, "https")
	if template.SSLCiphers != "" {
		setIfMissing(httpsCfg, "ssl_ciphers", template.SSLCiphers)
	}

	securityCfg := getSubMap(site.Settings, "security")
	if _, ok := securityCfg["waf_enable"]; !ok {
		securityCfg["waf_enable"] = template.WAFEnable
	}
}

func ApplySiteTemplateDefaultsByType(site *models.Site, defaults *models.DefaultSiteConfig) {
	if site == nil || defaults == nil {
		return
	}
	siteType := ""
	if site.Settings != nil {
		siteType = strings.ToLower(parseStringValue(site.Settings["site_type"]))
	}
	switch siteType {
	case "api":
		ApplySiteTemplateDefaults(site, defaults.API)
	case "download":
		ApplySiteTemplateDefaults(site, defaults.Download)
	default:
		ApplySiteTemplateDefaults(site, defaults.Website)
	}
}

func ApplyForwardDefaults(forward *models.Forward, defaults map[string]string) {
	if forward == nil || defaults == nil {
		return
	}
	if forward.Settings == nil {
		forward.Settings = map[string]interface{}{}
	}

	if _, ok := forward.Settings["listen_protocol"]; !ok {
		if v := defaults["listen_protocol"]; v != "" {
			forward.Settings["listen_protocol"] = v
		}
	}

	originCfg := getSubMap(forward.Settings, "origin")
	setIfMissing(originCfg, "balance_way", defaults["balance_way"])
	if v := defaults["proxy_protocol"]; v != "" {
		setIfMissing(originCfg, "proxy_protocol", parseBool(v, false))
	}
	if forward.BalanceWay == "" {
		forward.BalanceWay = defaults["balance_way"]
	}
	if forward.BackendPort == "" {
		if v := defaults["backsource_port"]; v != "" {
			forward.BackendPort = v
		}
	}
	if v, ok := originCfg["proxy_protocol"]; ok {
		forward.ProxyProtocol = parseBool(v, false)
	} else if v := defaults["proxy_protocol"]; v != "" {
		forward.ProxyProtocol = parseBool(v, false)
	}
}

func getSubMap(root map[string]interface{}, key string) map[string]interface{} {
	if val, ok := root[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	sub := map[string]interface{}{}
	root[key] = sub
	return sub
}

func setIfMissing(target map[string]interface{}, key string, value interface{}) {
	if value == nil {
		return
	}
	if _, ok := target[key]; !ok {
		target[key] = value
	}
}

func setIfMissingNonEmpty(target map[string]interface{}, key string, value interface{}) {
	if value == nil {
		return
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return
		}
	case []byte:
		if strings.TrimSpace(string(v)) == "" {
			return
		}
	}
	setIfMissing(target, key, value)
}

func parseStringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func applyLegacyBacksourceSettings(site *models.Site, backsourceCfg map[string]interface{}) {
	if site == nil || site.Settings == nil || backsourceCfg == nil {
		return
	}
	settings := site.Settings
	setIfMissingNonEmpty(backsourceCfg, "http_port", settings["origin_http_port"])
	setIfMissingNonEmpty(backsourceCfg, "https_port", settings["origin_https_port"])
	setIfMissingNonEmpty(backsourceCfg, "timeout", settings["origin_timeout"])
	if site.BackendProtocol != "" {
		setIfMissingNonEmpty(backsourceCfg, "protocol", site.BackendProtocol)
	}
	if originCfg, ok := settings["origin"].(map[string]interface{}); ok {
		setIfMissingNonEmpty(backsourceCfg, "connect_timeout", originCfg["connTimeout"])
		setIfMissingNonEmpty(backsourceCfg, "connect_timeout", originCfg["connect_timeout"])
	}

	rawHost := parseStringValue(settings["origin_host"])
	if rawHost == "" {
		if originCfg, ok := settings["origin"].(map[string]interface{}); ok {
			rawHost = parseStringValue(originCfg["host"])
		}
	}
	switch strings.ToLower(rawHost) {
	case "":
		return
	case "follow", "domain":
		setIfMissingNonEmpty(backsourceCfg, "host_mode", rawHost)
	default:
		setIfMissingNonEmpty(backsourceCfg, "host_mode", "custom")
		setIfMissingNonEmpty(backsourceCfg, "host_custom", rawHost)
	}
}

func applyLegacyAdvancedSettings(settings map[string]interface{}, advCfg map[string]interface{}) {
	if settings == nil || advCfg == nil {
		return
	}
	setIfMissing(advCfg, "gzip", settings["gzip"])
	setIfMissing(advCfg, "websocket", settings["websocket"])
	setIfMissing(advCfg, "body_limit", settings["upload_limit"])
	if _, ok := settings["upload_limit"]; ok {
		setIfMissing(advCfg, "body_limit_unit", "mb")
	}
	setIfMissing(advCfg, "log_request_header", settings["log_request_header"])
	setIfMissing(advCfg, "log_response_header", settings["log_response_header"])
	setIfMissing(advCfg, "log_request_body", settings["log_request_body"])
	setIfMissing(advCfg, "log_request_body_size_limit", settings["log_request_body_size_limit"])
	setIfMissing(advCfg, "origin_cert", settings["origin_cert"])
	setIfMissing(advCfg, "realtime_identify", settings["realtime_identify"])
	setIfMissing(advCfg, "realtime_send", settings["realtime_send"])
	setIfMissing(advCfg, "default_site", settings["default_site"])
	setIfMissing(advCfg, "l2_config", settings["l2_config"])
	if _, ok := advCfg["origin_headers"]; !ok {
		setIfMissing(advCfg, "origin_headers", settings["req_headers"])
	}
	if _, ok := advCfg["cdn_headers"]; !ok {
		setIfMissing(advCfg, "cdn_headers", settings["res_headers"])
	}
	if _, ok := advCfg["url_redirects"]; !ok {
		setIfMissing(advCfg, "url_redirects", settings["url_redirects"])
	}
	if _, ok := advCfg["url_redirects"]; !ok {
		setIfMissing(advCfg, "url_redirects", settings["url_rewrites"])
	}
}

func parseInt64(value string) int64 {
	i, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return i
}

func splitFields(input string) []string {
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")
	parts := strings.Fields(input)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseCacheRules(raw string) []map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []map[string]interface{}{}
	}
	var rules []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rules); err == nil {
		return rules
	}
	return []map[string]interface{}{}
}

func parseHeaderList(raw string) []map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []map[string]string{}
	}
	var headers []map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err == nil {
		return headers
	}
	return []map[string]string{}
}

func splitCommaList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "none" {
			out = append(out, p)
		}
	}
	return out
}
