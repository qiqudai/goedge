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

type streamListenEntry struct {
	Raw         string
	ListenValue string
	Port        string
	Protocol    string
}

func filterStreamPorts(ports []string, resources *edgeResources) []string {
	if len(ports) == 0 {
		return ports
	}
	if resources == nil {
		return ports
	}
	disabled := strings.TrimSpace(resources.Forward.DisabledPorts)
	if disabled == "" {
		return ports
	}
	out := make([]string, 0, len(ports))
	for _, portRaw := range ports {
		port, ok := parseListenPort(portRaw)
		if !ok {
			out = append(out, portRaw)
			continue
		}
		if isPortAllowed(port, "", disabled) {
			out = append(out, portRaw)
		}
	}
	return out
}

func normalizeStreamListenProtocol(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "udp" {
		return "udp"
	}
	return "tcp"
}

func parseStreamListenEntry(raw string, defaultProto string) (streamListenEntry, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return streamListenEntry{}, false
	}
	proto := normalizeStreamListenProtocol(defaultProto)
	listenValue := raw
	if idx := strings.LastIndex(raw, "/"); idx != -1 {
		suffix := strings.ToLower(strings.TrimSpace(raw[idx+1:]))
		if suffix == "udp" || suffix == "tcp" {
			proto = suffix
			listenValue = strings.TrimSpace(raw[:idx])
		}
	}
	if listenValue == "" {
		return streamListenEntry{}, false
	}
	port, ok := parseListenPort(listenValue)
	if !ok {
		return streamListenEntry{}, false
	}
	return streamListenEntry{
		Raw:         raw,
		ListenValue: listenValue,
		Port:        strconv.Itoa(port),
		Protocol:    proto,
	}, true
}

func normalizeStreamListenPorts(ports []string, defaultProto string) []streamListenEntry {
	if len(ports) == 0 {
		return nil
	}
	out := make([]streamListenEntry, 0, len(ports))
	for _, raw := range ports {
		entry, ok := parseStreamListenEntry(raw, defaultProto)
		if !ok {
			log.Printf("[Warn] Invalid stream listen port skipped: %s", raw)
			continue
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

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

func loadParentStatusSnapshot() (map[int64]bool, map[int64]bool) {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return nil, nil
	}
	path := filepath.Join(rootDir, "conf", "parent_status.json")
	var raw struct {
		L1 map[string]bool `json:"l1"`
		L2 map[string]bool `json:"l2"`
	}
	if err := fsutil.ReadJSONFile(path, &raw); err != nil {
		return map[int64]bool{}, map[int64]bool{}
	}
	parse := func(in map[string]bool) map[int64]bool {
		out := map[int64]bool{}
		for key, val := range in {
			if id, err := strconv.ParseInt(key, 10, 64); err == nil {
				out[id] = val
			}
		}
		return out
	}
	return parse(raw.L1), parse(raw.L2)
}

type streamStatusSnapshot struct {
	L2       map[int64]bool
	ParentL1 map[int64]bool
	ParentL2 map[int64]bool
}

func loadStreamStatusSnapshot() streamStatusSnapshot {
	l2 := loadL2StatusSnapshot()
	if l2 == nil {
		l2 = map[int64]bool{}
	}
	l1, l2p := loadParentStatusSnapshot()
	if l1 == nil {
		l1 = map[int64]bool{}
	}
	if l2p == nil {
		l2p = map[int64]bool{}
	}
	return streamStatusSnapshot{L2: l2, ParentL1: l1, ParentL2: l2p}
}

func selectStreamTargets(stream edgeStream, status streamStatusSnapshot) []edgeStreamTarget {
	if !stream.UseListenPort || len(stream.Targets) == 0 {
		return stream.Targets
	}
	mode := strings.ToLower(strings.TrimSpace(stream.ParentFetchMode))
	if mode == "l1" || mode == "l2" {
		return selectParentStreamTargets(stream, status, mode)
	}
	statusMap := status.L2
	if statusMap == nil {
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
		if online, ok := statusMap[target.NodeID]; ok && online {
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

func selectParentStreamTargets(stream edgeStream, status streamStatusSnapshot, mode string) []edgeStreamTarget {
	primary := make([]edgeStreamTarget, 0, len(stream.Targets))
	backupParents := make([]edgeStreamTarget, 0, len(stream.Targets))
	originTargets := make([]edgeStreamTarget, 0, len(stream.Targets))
	for _, target := range stream.Targets {
		if target.NodeID == 0 {
			originTargets = append(originTargets, target)
			continue
		}
		if target.Backup {
			backupParents = append(backupParents, target)
			continue
		}
		primary = append(primary, target)
	}
	filterByStatus := func(targets []edgeStreamTarget, statusMap map[int64]bool) []edgeStreamTarget {
		if statusMap == nil {
			return targets
		}
		out := make([]edgeStreamTarget, 0, len(targets))
		for _, target := range targets {
			if online, ok := statusMap[target.NodeID]; ok && online {
				out = append(out, target)
			}
		}
		return out
	}
	if mode == "l2" {
		healthy := filterByStatus(primary, status.ParentL2)
		if len(healthy) > 0 {
			return append(healthy, originTargets...)
		}
		if len(originTargets) == 0 {
			return nil
		}
		for i := range originTargets {
			originTargets[i].Backup = false
		}
		return originTargets
	}
	healthyPrimary := filterByStatus(primary, status.ParentL1)
	if len(healthyPrimary) > 0 {
		return append(healthyPrimary, append(backupParents, originTargets...)...)
	}
	healthyBackup := filterByStatus(backupParents, status.ParentL2)
	if len(healthyBackup) > 0 {
		return append(healthyBackup, originTargets...)
	}
	if len(originTargets) == 0 {
		return nil
	}
	for i := range originTargets {
		originTargets[i].Backup = false
	}
	return originTargets
}

func renderStreamConfig(streams []edgeStream, status streamStatusSnapshot) string {
	if len(streams) == 0 {
		return ""
	}
	var b strings.Builder
	for _, stream := range streams {
		listenPorts := filterStreamPorts(stream.ListenPorts, LocalResources)
		listenEntries := normalizeStreamListenPorts(listenPorts, stream.ListenProtocol)
		targets := selectStreamTargets(stream, status)
		if len(listenEntries) == 0 || len(targets) == 0 {
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
		writeServer := func(entry streamListenEntry, upstreamName string) {
			b.WriteString("server {\n")
			listenLine := entry.ListenValue
			if entry.Protocol == "udp" {
				listenLine += " udp"
			}
			// OpenResty stream does not allow "proxy_protocol" with UDP listens.
			if stream.ProxyProtocol && entry.Protocol != "udp" {
				listenLine += " proxy_protocol"
			}
			b.WriteString("    listen " + listenLine + ";\n")
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
			for _, entry := range listenEntries {
				upstreamName := fmt.Sprintf("stream_up_%d_%s", stream.ID, sanitizeStreamUpstreamSuffix(entry.Port))
				if entry.Protocol == "udp" {
					upstreamName = upstreamName + "_udp"
				}
				writeUpstream(upstreamName, entry.Port)
				writeServer(entry, upstreamName)
			}
			continue
		}

		upstreamName := fmt.Sprintf("stream_up_%d", stream.ID)
		writeUpstream(upstreamName, "")
		for _, entry := range listenEntries {
			writeServer(entry, upstreamName)
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
	content := renderStreamConfig(streams, loadStreamStatusSnapshot())
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
	content := renderStreamConfig(cfg.Streams, streamStatusSnapshot{L2: l2Status})
	confPath := filepath.Join(rootDir, "conf", "dynamic", "stream.conf")
	existing, err := ioutil.ReadFile(confPath)
	if err == nil && string(existing) == content {
		return
	}
	if err := ioutil.WriteFile(confPath, []byte(content), 0644); err != nil {
		log.Printf("[Warn] L2 stream refresh failed: %v", err)
		return
	}
	if err := reloadNginxWithRollback(); err != nil {
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
