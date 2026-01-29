package main

import (
	fsutil "cdn-common/io"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

func loadL2StatusSnapshot() map[int64]bool {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return nil
	}
	path := filepath.Join(rootDir, "conf", "l2_status.json")
	var raw struct {
		Nodes map[string]bool `json:"nodes"`
	}
	if err := fsutil.ReadJSONFile(path, &raw); err != nil {
		return nil
	}
	if len(raw.Nodes) == 0 {
		return map[int64]bool{}
	}
	out := make(map[int64]bool, len(raw.Nodes))
	for key, val := range raw.Nodes {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			out[id] = val
		}
	}
	return out
}

func selectStreamTargets(stream edgeStream, l2Status map[int64]bool) []edgeStreamTarget {
	if !stream.UseListenPort || len(stream.Targets) == 0 {
		return stream.Targets
	}
	if l2Status == nil {
		return stream.Targets
	}
	l2Targets := make([]edgeStreamTarget, 0, len(stream.Targets))
	originTargets := make([]edgeStreamTarget, 0, len(stream.Targets))
	for _, target := range stream.Targets {
		if target.NodeID > 0 {
			l2Targets = append(l2Targets, target)
		} else {
			originTargets = append(originTargets, target)
		}
	}
	if len(l2Targets) == 0 {
		return stream.Targets
	}
	healthyL2 := make([]edgeStreamTarget, 0, len(l2Targets))
	for _, target := range l2Targets {
		if online, ok := l2Status[target.NodeID]; ok && online {
			healthyL2 = append(healthyL2, target)
		}
	}
	if len(healthyL2) > 0 {
		return append(healthyL2, originTargets...)
	}
	if len(originTargets) == 0 {
		return nil
	}
	for i := range originTargets {
		originTargets[i].Backup = false
	}
	return originTargets
}

func renderStreamConfig(streams []edgeStream, l2Status map[int64]bool) string {
	if len(streams) == 0 {
		return ""
	}
	var b strings.Builder
	for _, stream := range streams {
		targets := selectStreamTargets(stream, l2Status)
		if len(stream.ListenPorts) == 0 || len(targets) == 0 {
			continue
		}
		writeUpstream := func(name string, listenPort string) {
			b.WriteString("upstream " + name + " {\n")
			switch strings.ToLower(stream.BalanceWay) {
			case "ip_hash":
				b.WriteString("    hash $remote_addr consistent;\n")
			case "least_conn":
				b.WriteString("    least_conn;\n")
			}
			for _, target := range targets {
				if !target.Enable || target.Addr == "" {
					continue
				}
				addr := strings.TrimSpace(target.Addr)
				if stream.UseListenPort && listenPort != "" && !strings.Contains(addr, ":") {
					addr = addr + ":" + listenPort
				}
				if addr == "" {
					continue
				}
				params := ""
				if target.Weight > 0 {
					params = fmt.Sprintf(" weight=%d", target.Weight)
				}
				if target.Backup {
					params = params + " backup"
				}
				b.WriteString(fmt.Sprintf("    server %s%s;\n", addr, params))
			}
			b.WriteString("}\n")
		}
		writeServer := func(port, upstreamName string) {
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

		if stream.UseListenPort {
			for _, port := range stream.ListenPorts {
				port = strings.TrimSpace(port)
				if port == "" {
					continue
				}
				upstreamName := fmt.Sprintf("stream_up_%d_%s", stream.ID, sanitizeStreamUpstreamSuffix(port))
				writeUpstream(upstreamName, port)
				writeServer(port, upstreamName)
			}
			continue
		}

		upstreamName := fmt.Sprintf("stream_up_%d", stream.ID)
		writeUpstream(upstreamName, "")
		for _, port := range stream.ListenPorts {
			port = strings.TrimSpace(port)
			if port == "" {
				continue
			}
			writeServer(port, upstreamName)
		}
	}
	return b.String()
}

func writeStreamConfig(streams []edgeStream) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "stream.conf")
	if len(streams) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}
	content := renderStreamConfig(streams, loadL2StatusSnapshot())
	return ioutil.WriteFile(confPath, []byte(content), 0644)
}

func refreshStreamConfigForL2Status(snapshot map[string]bool) {
	rootDir := runtimeRoot()
	if rootDir == "" || CONFIG_PATH == "" {
		return
	}
	l2Status := map[int64]bool{}
	for key, val := range snapshot {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			l2Status[id] = val
		}
	}
	data, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		return
	}
	var cfg edgeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[Warn] L2 stream refresh skipped: %v", err)
		return
	}
	if len(cfg.Streams) == 0 {
		return
	}
	hasL2Streams := false
	for _, stream := range cfg.Streams {
		if stream.UseListenPort {
			hasL2Streams = true
			break
		}
	}
	if !hasL2Streams {
		return
	}
	content := renderStreamConfig(cfg.Streams, l2Status)
	confPath := filepath.Join(rootDir, "conf", "dynamic", "stream.conf")
	existing, err := ioutil.ReadFile(confPath)
	if err == nil && string(existing) == content {
		return
	}
	if err := ioutil.WriteFile(confPath, []byte(content), 0644); err != nil {
		log.Printf("[Warn] L2 stream refresh failed: %v", err)
		return
	}
	if err := reloadNginx(); err != nil {
		log.Printf("[Warn] L2 stream reload failed: %v", err)
	}
}

func writeStreamGlobalConfig(cfg *edgeNginxConfig) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "stream_global.conf")
	var b strings.Builder
	if cfg != nil && cfg.Stream != nil {
		if v := toString(cfg.Stream["proxy_connect_timeout"]); v != "" {
			b.WriteString("proxy_connect_timeout " + v + ";\n")
		}
		if v := toString(cfg.Stream["proxy_timeout"]); v != "" {
			b.WriteString("proxy_timeout " + v + ";\n")
		}
	}
	if cfg != nil {
		if v := strings.TrimSpace(cfg.Resolver); v != "" {
			b.WriteString("resolver " + v + ";\n")
		}
		if v := strings.TrimSpace(cfg.ResolverTimeout); v != "" {
			b.WriteString("resolver_timeout " + v + ";\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}
