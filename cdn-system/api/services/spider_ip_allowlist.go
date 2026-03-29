package services

import (
	"cdn-api/config"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const spiderIPAllowlistFile = "spider_ip_allowlist.json"

type spiderAllowlist struct {
	exact    map[string]struct{}
	prefixes []string
	cidrs    []*net.IPNet
}

func (allowlist *spiderAllowlist) match(ip string) bool {
	if allowlist == nil || ip == "" {
		return false
	}
	if _, ok := allowlist.exact[ip]; ok {
		return true
	}
	if len(allowlist.cidrs) > 0 {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			for _, cidr := range allowlist.cidrs {
				if cidr.Contains(parsed) {
					return true
				}
			}
		}
	}
	for _, prefix := range allowlist.prefixes {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

var (
	emptySpiderAllowlist = &spiderAllowlist{
		exact: map[string]struct{}{},
	}
	spiderAllowlistCache struct {
		mu        sync.RWMutex
		allowlist *spiderAllowlist
		mtime     time.Time
		path      string
	}
)

func IsSpiderIP(raw string) bool {
	ip := normalizeIPv4(raw)
	if ip == "" {
		return false
	}
	return loadSpiderAllowlist().match(ip)
}

func loadSpiderAllowlist() *spiderAllowlist {
	path := spiderAllowlistPath()
	stat, err := os.Stat(path)

	spiderAllowlistCache.mu.RLock()
	cached := spiderAllowlistCache.allowlist
	cachedMtime := spiderAllowlistCache.mtime
	cachedPath := spiderAllowlistCache.path
	spiderAllowlistCache.mu.RUnlock()

	if err != nil {
		if cached != nil {
			return cached
		}
		return emptySpiderAllowlist
	}
	if cached != nil && cachedPath == path && stat.ModTime().Equal(cachedMtime) {
		return cached
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if cached != nil {
			return cached
		}
		return emptySpiderAllowlist
	}
	allowlist := parseSpiderAllowlist(data)

	spiderAllowlistCache.mu.Lock()
	spiderAllowlistCache.allowlist = allowlist
	spiderAllowlistCache.mtime = stat.ModTime()
	spiderAllowlistCache.path = path
	spiderAllowlistCache.mu.Unlock()

	return allowlist
}

func spiderAllowlistPath() string {
	dir := config.ConfigDir()
	if dir == "" || dir == "." {
		return spiderIPAllowlistFile
	}
	return filepath.Join(dir, spiderIPAllowlistFile)
}

func parseSpiderAllowlist(data []byte) *spiderAllowlist {
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return emptySpiderAllowlist
	}
	exact := make(map[string]struct{})
	prefixSet := make(map[string]struct{})
	var cidrs []*net.IPNet
	for _, entries := range raw {
		for _, entry := range entries {
			token := strings.TrimSpace(entry)
			if token == "" {
				continue
			}
			if strings.Contains(token, "/") {
				if _, cidr, err := net.ParseCIDR(token); err == nil && cidr != nil {
					cidrs = append(cidrs, cidr)
					continue
				}
			}
			parts := strings.Split(token, ".")
			if len(parts) == 4 {
				if ip := normalizeIPv4(token); ip != "" {
					exact[ip] = struct{}{}
				}
				continue
			}
			if len(parts) == 3 {
				if prefix, ok := normalizeIPv4Prefix(parts); ok {
					prefixSet[prefix] = struct{}{}
				}
			}
		}
	}
	prefixes := make([]string, 0, len(prefixSet))
	for prefix := range prefixSet {
		prefixes = append(prefixes, prefix)
	}
	return &spiderAllowlist{
		exact:    exact,
		prefixes: prefixes,
		cidrs:    cidrs,
	}
}

func normalizeIPv4(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	return ip.String()
}

func normalizeIPv4Prefix(parts []string) (string, bool) {
	if len(parts) != 3 {
		return "", false
	}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", false
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 || value > 255 {
			return "", false
		}
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ".") + ".", true
}
