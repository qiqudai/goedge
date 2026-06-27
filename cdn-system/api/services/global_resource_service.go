package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func defaultWebsiteResources() models.WebsiteResourceConfig {
	return models.WebsiteResourceConfig{
		MinLimit:              1000,
		MaxLimitMultiplier:    200,
		MaxBlacklistIPs:       50,
		MaxWhitelistIPs:       50,
		MaxWAFPatternIPs:      100,
		DailyURLPurgeLimit:    2000,
		DailyDirPurgeLimit:    500,
		DailyPreloadLimit:     2000,
		PreloadTimeout:        120,
		DailyUnlockIPLimit:    1000,
		UnlockIPBatchLimit:    50,
		MaxCCRulesPerGroup:    5,
		MaxACLRules:           5,
		DailyLogDownloadLimit: 10,
		LogStorageDir:         "/data/download-temp/",
		LogStorageHours:       12,
		MaxDomainsPerSite:     100,
		DefaultListen80:       true,
	}
}

func defaultForwardResources() models.ForwardResourceConfig {
	return models.ForwardResourceConfig{
		DisabledPorts:      "80 443",
		MinLimit:           1000,
		MaxLimitMultiplier: 200,
		MaxACLRules:        10,
	}
}

func defaultPublicResources() models.PublicResourceConfig {
	return models.PublicResourceConfig{
		DisabledCustomPorts: "22 5000",
		AllowedCustomPorts:  "1-65535",
	}
}

func mergeWebsiteResources(base models.WebsiteResourceConfig) models.WebsiteResourceConfig {
	def := defaultWebsiteResources()
	if base.MinLimit <= 0 {
		base.MinLimit = def.MinLimit
	}
	if base.MaxLimitMultiplier <= 0 {
		base.MaxLimitMultiplier = def.MaxLimitMultiplier
	}
	if base.MaxBlacklistIPs <= 0 {
		base.MaxBlacklistIPs = def.MaxBlacklistIPs
	}
	if base.MaxWhitelistIPs <= 0 {
		base.MaxWhitelistIPs = def.MaxWhitelistIPs
	}
	if base.MaxWAFPatternIPs <= 0 {
		base.MaxWAFPatternIPs = def.MaxWAFPatternIPs
	}
	if base.DailyURLPurgeLimit <= 0 {
		base.DailyURLPurgeLimit = def.DailyURLPurgeLimit
	}
	if base.DailyDirPurgeLimit <= 0 {
		base.DailyDirPurgeLimit = def.DailyDirPurgeLimit
	}
	if base.DailyPreloadLimit <= 0 {
		base.DailyPreloadLimit = def.DailyPreloadLimit
	}
	if base.PreloadTimeout <= 0 {
		base.PreloadTimeout = def.PreloadTimeout
	}
	if base.DailyUnlockIPLimit <= 0 {
		base.DailyUnlockIPLimit = def.DailyUnlockIPLimit
	}
	if base.UnlockIPBatchLimit <= 0 {
		base.UnlockIPBatchLimit = def.UnlockIPBatchLimit
	}
	if base.MaxCCRulesPerGroup <= 0 {
		base.MaxCCRulesPerGroup = def.MaxCCRulesPerGroup
	}
	if base.MaxACLRules <= 0 {
		base.MaxACLRules = def.MaxACLRules
	}
	if base.DailyLogDownloadLimit <= 0 {
		base.DailyLogDownloadLimit = def.DailyLogDownloadLimit
	}
	if strings.TrimSpace(base.LogStorageDir) == "" {
		base.LogStorageDir = def.LogStorageDir
	}
	if base.LogStorageHours <= 0 {
		base.LogStorageHours = def.LogStorageHours
	}
	if base.MaxDomainsPerSite <= 0 {
		base.MaxDomainsPerSite = def.MaxDomainsPerSite
	}
	return base
}

func MergeWebsiteResourcesForConfig(base models.WebsiteResourceConfig) models.WebsiteResourceConfig {
	return mergeWebsiteResources(base)
}

// GetGlobalResources returns merged global resource limits (global_config with defaults).
func GetGlobalResources() models.GlobalResourceConfig {
	cfg := LoadGlobalConfigNormalized()
	if cfg == nil || db.DB == nil {
		return models.GlobalResourceConfig{
			Website: defaultWebsiteResources(),
			Forward: defaultForwardResources(),
			Public:  defaultPublicResources(),
		}
	}
	out := models.GlobalResourceConfig{
		Website: mergeWebsiteResources(cfg.Resources.Website),
		Forward: cfg.Resources.Forward,
		Public:  cfg.Resources.Public,
	}
	if strings.TrimSpace(out.Forward.DisabledPorts) == "" {
		out.Forward = defaultForwardResources()
	} else {
		def := defaultForwardResources()
		if out.Forward.MinLimit <= 0 {
			out.Forward.MinLimit = def.MinLimit
		}
		if out.Forward.MaxLimitMultiplier <= 0 {
			out.Forward.MaxLimitMultiplier = def.MaxLimitMultiplier
		}
		if out.Forward.MaxACLRules <= 0 {
			out.Forward.MaxACLRules = def.MaxACLRules
		}
	}
	if strings.TrimSpace(out.Public.DisabledCustomPorts) == "" && strings.TrimSpace(out.Public.AllowedCustomPorts) == "" {
		out.Public = defaultPublicResources()
	} else {
		def := defaultPublicResources()
		if strings.TrimSpace(out.Public.DisabledCustomPorts) == "" {
			out.Public.DisabledCustomPorts = def.DisabledCustomPorts
		}
		if strings.TrimSpace(out.Public.AllowedCustomPorts) == "" {
			out.Public.AllowedCustomPorts = def.AllowedCustomPorts
		}
	}
	return out
}

type legacyConfigEntry struct {
	Name      string
	Value     string
	Type      string
	ScopeName string
	ScopeID   int
}

func websiteLegacyResourceEntries(res models.GlobalResourceConfig) []legacyConfigEntry {
	w := res.Website
	listen80 := 0
	if w.DefaultListen80 {
		listen80 = 1
	}
	return []legacyConfigEntry{
		{Name: "related-config-min-limit", Value: strconv.Itoa(w.MinLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "related-config-max-times-limit", Value: strconv.Itoa(w.MaxLimitMultiplier), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "black-ip-limit", Value: strconv.Itoa(w.MaxBlacklistIPs), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "white-ip-limit", Value: strconv.Itoa(w.MaxWhitelistIPs), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "waf-pattern-ip-limit", Value: strconv.Itoa(w.MaxWAFPatternIPs), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "max-domain-persite-limit", Value: strconv.Itoa(w.MaxDomainsPerSite), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "listen-default-http-80", Value: strconv.Itoa(listen80), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "clean_url", Value: strconv.Itoa(w.DailyURLPurgeLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "clean_dir", Value: strconv.Itoa(w.DailyDirPurgeLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "pre_cache_url", Value: strconv.Itoa(w.DailyPreloadLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "pre_cache_timeout", Value: strconv.Itoa(w.PreloadTimeout), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "ip-unlock-max-limit", Value: strconv.Itoa(w.DailyUnlockIPLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "ip-unlock-max-per-limit", Value: strconv.Itoa(w.UnlockIPBatchLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "cc-rule-max-limit", Value: strconv.Itoa(w.MaxCCRulesPerGroup), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "acl-max-limit", Value: strconv.Itoa(w.MaxACLRules), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "download-access-log-limit", Value: strconv.Itoa(w.DailyLogDownloadLimit), Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "download-access-log-tmp-dir", Value: w.LogStorageDir, Type: "site", ScopeName: "global", ScopeID: 0},
		{Name: "download-access-log-retain", Value: strconv.Itoa(w.LogStorageHours), Type: "site", ScopeName: "global", ScopeID: 0},
	}
}

func forwardLegacyResourceEntries(res models.GlobalResourceConfig) []legacyConfigEntry {
	f := res.Forward
	return []legacyConfigEntry{
		{Name: "custom-port-not-allow", Value: f.DisabledPorts, Type: "stream", ScopeName: "global", ScopeID: 0},
		{Name: "related-config-min-limit", Value: strconv.Itoa(f.MinLimit), Type: "stream", ScopeName: "global", ScopeID: 0},
		{Name: "related-config-max-times-limit", Value: strconv.Itoa(f.MaxLimitMultiplier), Type: "stream", ScopeName: "global", ScopeID: 0},
		{Name: "acl-max-limit", Value: strconv.Itoa(f.MaxACLRules), Type: "stream", ScopeName: "global", ScopeID: 0},
	}
}

func publicLegacyResourceEntries(res models.GlobalResourceConfig) []legacyConfigEntry {
	p := res.Public
	return []legacyConfigEntry{
		{Name: "custom-port-not-allow", Value: p.DisabledCustomPorts, Type: "site_stream", ScopeName: "global", ScopeID: 0},
		{Name: "custom-port-allow", Value: p.AllowedCustomPorts, Type: "site_stream", ScopeName: "global", ScopeID: 0},
	}
}

// SyncLegacyResourceConfigs mirrors global_config.resources into legacy config rows.
func SyncLegacyResourceConfigs(cfg *models.GlobalConfig) error {
	if cfg == nil {
		return nil
	}
	merged := models.GlobalResourceConfig{
		Website: mergeWebsiteResources(cfg.Resources.Website),
		Forward: cfg.Resources.Forward,
		Public:  cfg.Resources.Public,
	}
	if strings.TrimSpace(merged.Forward.DisabledPorts) == "" {
		merged.Forward = defaultForwardResources()
	}
	if strings.TrimSpace(merged.Public.DisabledCustomPorts) == "" && strings.TrimSpace(merged.Public.AllowedCustomPorts) == "" {
		merged.Public = defaultPublicResources()
	} else if strings.TrimSpace(merged.Public.AllowedCustomPorts) == "" {
		merged.Public.AllowedCustomPorts = defaultPublicResources().AllowedCustomPorts
	}

	entries := append(append(websiteLegacyResourceEntries(merged), forwardLegacyResourceEntries(merged)...), publicLegacyResourceEntries(merged)...)
	now := time.Now()
	for _, entry := range entries {
		if err := upsertLegacyConfig(entry, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertLegacyConfig(entry legacyConfigEntry, now time.Time) error {
	var cfg models.SysConfig
	err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?",
		entry.Name, entry.Type, entry.ScopeName, entry.ScopeID).First(&cfg).Error
	if err != nil {
		cfg = models.SysConfig{
			Name:      entry.Name,
			Value:     entry.Value,
			Type:      entry.Type,
			ScopeName: entry.ScopeName,
			ScopeID:   entry.ScopeID,
			Enable:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return db.DB.Create(&cfg).Error
	}
	return db.DB.Model(&models.SysConfig{}).Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?",
		entry.Name, entry.Type, entry.ScopeName, entry.ScopeID).Updates(map[string]interface{}{
		"value":     entry.Value,
		"update_at": now,
	}).Error
}

func loadLegacyIntConfig(name, configType string) (int, bool) {
	var cfg models.SysConfig
	if err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?",
		name, configType, "global", 0).First(&cfg).Error; err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(cfg.Value))
	if err != nil || val <= 0 {
		return 0, false
	}
	return val, true
}

// LoadPurgeLimits reads purge/preheat limits from global_config (legacy config fallback).
func LoadPurgeLimits() (urlLimit, dirLimit, preheatLimit int) {
	res := GetGlobalResources()
	urlLimit = res.Website.DailyURLPurgeLimit
	dirLimit = res.Website.DailyDirPurgeLimit
	preheatLimit = res.Website.DailyPreloadLimit

	if urlLimit <= 0 {
		if v, ok := loadLegacyIntConfig("clean_url", "site"); ok {
			urlLimit = v
		}
	}
	if dirLimit <= 0 {
		if v, ok := loadLegacyIntConfig("clean_dir", "site"); ok {
			dirLimit = v
		}
	}
	if preheatLimit <= 0 {
		if v, ok := loadLegacyIntConfig("pre_cache_url", "site"); ok {
			preheatLimit = v
		}
	}
	def := defaultWebsiteResources()
	if urlLimit <= 0 {
		urlLimit = def.DailyURLPurgeLimit
	}
	if dirLimit <= 0 {
		dirLimit = def.DailyDirPurgeLimit
	}
	if preheatLimit <= 0 {
		preheatLimit = def.DailyPreloadLimit
	}
	return urlLimit, dirLimit, preheatLimit
}

// CheckSiteDomainsPerSiteLimit validates domain count for a single site.
func CheckSiteDomainsPerSiteLimit(domains []string) error {
	limit := GetGlobalResources().Website.MaxDomainsPerSite
	if limit <= 0 {
		return nil
	}
	normalized := normalizeDomainList(domains)
	if len(normalized) > limit {
		return fmt.Errorf(i18n.T("Site domain count exceeds limit: %d"), limit)
	}
	return nil
}

func normalizeDomainList(domains []string) []string {
	if len(domains) == 0 {
		return nil
	}
	out := make([]string, 0, len(domains))
	seen := map[string]struct{}{}
	for _, d := range domains {
		key := strings.ToLower(strings.TrimSpace(d))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// CheckSiteIPListLimit validates blacklist/whitelist size against global limits.
func CheckSiteIPListLimit(listType string, count int) error {
	if count <= 0 {
		return nil
	}
	res := GetGlobalResources().Website
	var limit int
	switch strings.ToLower(strings.TrimSpace(listType)) {
	case "blacklist", "black":
		limit = res.MaxBlacklistIPs
	case "whitelist", "white":
		limit = res.MaxWhitelistIPs
	default:
		return nil
	}
	if limit <= 0 || count <= limit {
		return nil
	}
	switch listType {
	case "whitelist", "white":
		return fmt.Errorf(i18n.T("Whitelist IP count exceeds limit: %d"), limit)
	default:
		return fmt.Errorf(i18n.T("Blacklist IP count exceeds limit: %d"), limit)
	}
}

// TrimSiteIPList trims IP lists to the configured global limit (edge-side safety).
func TrimSiteIPList(listType string, items []string) []string {
	if len(items) == 0 {
		return items
	}
	res := GetGlobalResources().Website
	var limit int
	switch strings.ToLower(strings.TrimSpace(listType)) {
	case "blacklist", "black":
		limit = res.MaxBlacklistIPs
	case "whitelist", "white":
		limit = res.MaxWhitelistIPs
	default:
		return items
	}
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return append([]string(nil), items[:limit]...)
}

// GlobalConfigResourcesFromJSON unmarshals resources and merges defaults.
func GlobalConfigResourcesFromJSON(raw json.RawMessage) models.GlobalResourceConfig {
	if len(raw) == 0 {
		return models.GlobalResourceConfig{
			Website: defaultWebsiteResources(),
			Forward: defaultForwardResources(),
			Public:  defaultPublicResources(),
		}
	}
	var res models.GlobalResourceConfig
	if err := json.Unmarshal(raw, &res); err != nil {
		return GetGlobalResources()
	}
	return models.GlobalResourceConfig{
		Website: mergeWebsiteResources(res.Website),
		Forward: res.Forward,
		Public:  res.Public,
	}
}

var ErrResourceLimitExceeded = errors.New("resource limit exceeded")
