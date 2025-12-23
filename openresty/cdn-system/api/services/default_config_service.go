package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
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
	global, err := LoadConfigMap("site_default_config", "global", 0)
	if err != nil {
		return nil, err
	}
	user, err := LoadConfigMap("site_default_config", "user", userID)
	if err != nil {
		return global, nil
	}
	if len(user) == 0 {
		return global, nil
	}
	return MergeConfigMap(global, user), nil
}

func GetStreamDefaultMap(userID int64) (map[string]string, error) {
	global, err := LoadConfigMap("stream_default_config", "global", 0)
	if err != nil {
		return nil, err
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

func ApplySiteDefaults(site *models.Site, defaults map[string]string) {
	if site == nil || defaults == nil {
		return
	}

	if len(site.HttpListen) == 0 {
		if v := defaults["http_listen-port"]; v != "" {
			site.HttpListen = splitFields(v)
		}
	}
	if len(site.HttpsListen) == 0 {
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

	if site.Settings == nil {
		site.Settings = map[string]interface{}{}
	}

	httpsCfg := getSubMap(site.Settings, "https")
	setIfMissing(httpsCfg, "force", parseBool(defaults["https_listen-force_ssl_enable"]))
	setIfMissing(httpsCfg, "redirect_port", defaults["https_listen-port"])
	setIfMissing(httpsCfg, "hsts", parseBool(defaults["https_listen-hsts"]))
	setIfMissing(httpsCfg, "http2", parseBool(defaults["https_listen-http2"]))
	setIfMissing(httpsCfg, "ssl_protocols", defaults["https_listen-ssl_protocols"])
	setIfMissing(httpsCfg, "ssl_ciphers", defaults["https_listen-ssl_ciphers"])
	setIfMissing(httpsCfg, "ssl_prefer_server_ciphers", defaults["https_listen-ssl_prefer_server_ciphers"])

	backsourceCfg := getSubMap(site.Settings, "backsource")
	setIfMissing(backsourceCfg, "protocol", defaults["backend_protocol"])
	setIfMissing(backsourceCfg, "http_port", defaults["backend_http_port"])
	setIfMissing(backsourceCfg, "https_port", defaults["backend_https_port"])
	setIfMissing(backsourceCfg, "timeout", defaults["proxy_timeout"])
	setIfMissing(backsourceCfg, "connect_timeout", defaults["proxy_timeout"])

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

	advCfg := getSubMap(site.Settings, "advanced")
	setIfMissing(advCfg, "gzip", parseBool(defaults["gzip_enable"]))
	setIfMissing(advCfg, "gzip_types", defaults["gzip_types"])
	setIfMissing(advCfg, "websocket", parseBool(defaults["websocket_enable"]))
	setIfMissing(advCfg, "range", parseBool(defaults["range"]))
	setIfMissing(advCfg, "proxy_http_version", defaults["proxy_http_version"])
	setIfMissing(advCfg, "proxy_ssl_protocols", defaults["proxy_ssl_protocols"])
	setIfMissing(advCfg, "ups_keepalive", parseBool(defaults["ups_keepalive"]))
	setIfMissing(advCfg, "ups_keepalive_conn", parseInt64(defaults["ups_keepalive_conn"]))
	setIfMissing(advCfg, "ups_keepalive_timeout", parseInt64(defaults["ups_keepalive_timeout"]))
	setIfMissing(advCfg, "body_limit", parseInt64(defaults["post_size_limit"]))
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
		setIfMissing(originCfg, "proxy_protocol", parseBool(v))
	}
	if forward.BalanceWay == "" {
		forward.BalanceWay = defaults["balance_way"]
	}
	if forward.BackendPort == "" {
		if v := defaults["backsource_port"]; v != "" {
			forward.BackendPort = v
		}
	}
	if v := defaults["proxy_protocol"]; v != "" {
		forward.ProxyProtocol = parseBool(v)
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

func parseBool(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "1" || v == "true" || v == "on" || v == "yes"
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
