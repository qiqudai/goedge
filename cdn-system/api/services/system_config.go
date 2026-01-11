package services

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

const systemConfigTTL = 5 * time.Second

var systemConfigCache struct {
	mu        sync.RWMutex
	values    map[string]string
	updatedAt time.Time
}

// LoadSystemConfig returns cached system config items (type=system, scope=global).
func LoadSystemConfig() (map[string]string, error) {
	now := time.Now()
	systemConfigCache.mu.RLock()
	if systemConfigCache.values != nil && now.Sub(systemConfigCache.updatedAt) <= systemConfigTTL {
		cached := copyConfigMap(systemConfigCache.values)
		systemConfigCache.mu.RUnlock()
		return cached, nil
	}
	systemConfigCache.mu.RUnlock()

	values, err := LoadConfigMap("system", "global", 0)
	if err != nil {
		return nil, err
	}

	systemConfigCache.mu.Lock()
	systemConfigCache.values = values
	systemConfigCache.updatedAt = now
	systemConfigCache.mu.Unlock()

	return copyConfigMap(values), nil
}

func copyConfigMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func ParseBoolFlag(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func ParseMaintenance(raw string) (bool, string) {
	type maintainConfig struct {
		Enable int    `json:"enable"`
		Msg    string `json:"msg"`
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, ""
	}
	var cfg maintainConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return false, ""
	}
	return cfg.Enable == 1, strings.TrimSpace(cfg.Msg)
}

func SplitHostList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		host := NormalizeHost(part)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func NormalizeHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if idx := strings.Index(host, "]"); idx != -1 {
			return host[:idx+1]
		}
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return host
}

func ResolveLoginSessionTTL() time.Duration {
	cfg, err := LoadSystemConfig()
	if err != nil {
		return 24 * time.Hour
	}
	raw := strings.TrimSpace(cfg["login_session_valid_time"])
	if raw == "" {
		return 24 * time.Hour
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(seconds) * time.Second
}
