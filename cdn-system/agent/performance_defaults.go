package main

import (
	"runtime"
	"strings"
)

func resolveWorkerProcesses(cfg *edgeNginxConfig) string {
	if cfg != nil {
		if v := strings.TrimSpace(cfg.WorkerProcesses); v != "" {
			return v
		}
	}
	return "auto"
}

func resolveWorkerConnections(cfg *edgeNginxConfig) int {
	if cfg != nil && cfg.WorkerConnections > 0 {
		return cfg.WorkerConnections
	}
	cpu := runtime.NumCPU()
	switch {
	case cpu <= 2:
		return 8192
	case cpu <= 4:
		return 16384
	case cpu <= 8:
		return 32768
	default:
		return 65535
	}
}

func resolveWorkerRlimitNofile(cfg *edgeNginxConfig, workerConnections int) int {
	if cfg != nil && cfg.WorkerRlimitNofile > 0 {
		return cfg.WorkerRlimitNofile
	}
	rlimit := workerConnections * 4
	if rlimit < 65535 {
		rlimit = 65535
	}
	if rlimit > 1048576 {
		rlimit = 1048576
	}
	return rlimit
}

func resolveWorkerShutdownTimeout(cfg *edgeNginxConfig) string {
	if cfg != nil {
		if v := strings.TrimSpace(cfg.WorkerShutdownTimeout); v != "" {
			return v
		}
	}
	return "30s"
}

func hasHTTPDirective(cfg *edgeNginxConfig, key string) bool {
	if cfg == nil || cfg.HTTP == nil {
		return false
	}
	_, ok := cfg.HTTP[key]
	return ok
}

func writeHTTPPerformanceDefaults(b *strings.Builder, cfg *edgeNginxConfig, cacheEnabled bool) {
	if !hasHTTPDirective(cfg, "keepalive_timeout") {
		b.WriteString("keepalive_timeout 30s;\n")
	}
	if !hasHTTPDirective(cfg, "keepalive_requests") {
		b.WriteString("keepalive_requests 10000;\n")
	}
	if !hasHTTPDirective(cfg, "reset_timedout_connection") {
		b.WriteString("reset_timedout_connection on;\n")
	}
	if !hasHTTPDirective(cfg, "sendfile_max_chunk") {
		b.WriteString("sendfile_max_chunk 1m;\n")
	}
	if !hasHTTPDirective(cfg, "client_header_timeout") {
		b.WriteString("client_header_timeout 15s;\n")
	}
	if !hasHTTPDirective(cfg, "client_body_timeout") {
		b.WriteString("client_body_timeout 15s;\n")
	}
	if !hasHTTPDirective(cfg, "open_file_cache") {
		b.WriteString("open_file_cache max=200000 inactive=30s;\n")
	}
	if !hasHTTPDirective(cfg, "open_file_cache_valid") {
		b.WriteString("open_file_cache_valid 60s;\n")
	}
	if !hasHTTPDirective(cfg, "open_file_cache_min_uses") {
		b.WriteString("open_file_cache_min_uses 2;\n")
	}
	if !hasHTTPDirective(cfg, "open_file_cache_errors") {
		b.WriteString("open_file_cache_errors on;\n")
	}
	if cacheEnabled {
		if !hasHTTPDirective(cfg, "proxy_cache_methods") {
			b.WriteString("proxy_cache_methods GET HEAD;\n")
		}
		if !hasHTTPDirective(cfg, "proxy_cache_revalidate") {
			b.WriteString("proxy_cache_revalidate on;\n")
		}
	}
}
