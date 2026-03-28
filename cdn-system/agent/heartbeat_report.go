package main

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	heartbeatReportMu       sync.Mutex
	heartbeatReportCache    map[string]interface{}
	heartbeatReportCacheAt  time.Time
	heartbeatReportCacheTTL = 10 * time.Second
)

func buildReportedConfig() map[string]interface{} {
	heartbeatReportMu.Lock()
	defer heartbeatReportMu.Unlock()

	now := time.Now()
	if heartbeatReportCache != nil && now.Sub(heartbeatReportCacheAt) < heartbeatReportCacheTTL {
		return heartbeatReportCache
	}

	report := map[string]interface{}{
		"anti_blocking":  AutoDisableFirewall,
		"config_version": readLocalVersion(),
		"nginx_running":  isManagedNginxRunning(),
	}

	if cfgHash := readConfigHash(); cfgHash != "" {
		report["config_hash"] = cfgHash
	}

	if LocalNginxConfig != nil && LocalNginxConfig.HTTP != nil {
		httpCfg := LocalNginxConfig.HTTP
		if v := strings.TrimSpace(toString(httpCfg["proxy_cache_valid_statuses"])); v != "" {
			report["cache_valid_statuses"] = v
		}
		if v := strings.TrimSpace(toString(httpCfg["cache_404_revalidate_enable"])); v != "" {
			report["cache_404_revalidate_enable"] = v
		}
		if v := strings.TrimSpace(toString(httpCfg["cache_404_revalidate_after"])); v != "" {
			report["cache_404_revalidate_after"] = v
		}
		if v := strings.TrimSpace(toString(httpCfg["cache_404_probe_interval"])); v != "" {
			report["cache_404_probe_interval"] = v
		}
		if v := strings.TrimSpace(toString(httpCfg["cache_404_probe_timeout_ms"])); v != "" {
			report["cache_404_probe_timeout_ms"] = v
		}
	}

	heartbeatReportCache = report
	heartbeatReportCacheAt = now
	return report
}

func readConfigHash() string {
	if strings.TrimSpace(CONFIG_PATH) == "" {
		return ""
	}
	data, err := os.ReadFile(CONFIG_PATH)
	if err != nil || len(data) == 0 {
		return ""
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}
