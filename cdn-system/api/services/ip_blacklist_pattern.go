package services

import (
	"cdn-api/models"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type IPListStats struct {
	Total        int `json:"total"`
	Exact        int `json:"exact"`
	Pattern      int `json:"pattern"`
	Invalid      int `json:"invalid"`
	Overbroad    int `json:"overbroad"`
	PatternLimit int `json:"pattern_limit"`
}

// IsValidIPBlacklistEntry accepts single IPv4/IPv6, CIDR, or wildcard patterns like 127.*.*.*
func IsValidIPBlacklistEntry(raw string) bool {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return false
	}
	if net.ParseIP(entry) != nil {
		return true
	}
	if _, _, err := net.ParseCIDR(entry); err == nil {
		return true
	}
	if strings.Contains(entry, "*") {
		return isValidIPWildcardPattern(entry)
	}
	return false
}

func IsPatternIPEntry(raw string) bool {
	entry := strings.TrimSpace(raw)
	return strings.Contains(entry, "/") || strings.Contains(entry, "*")
}

func IsOverbroadIPPattern(raw string) bool {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return false
	}
	if strings.Contains(entry, "*") {
		parts := strings.Split(entry, ".")
		fixed := 0
		for _, part := range parts {
			if strings.TrimSpace(part) != "*" {
				fixed++
			}
		}
		return fixed == 0
	}
	if strings.Contains(entry, "/") {
		ip, network, err := net.ParseCIDR(entry)
		if err != nil || ip == nil || network == nil {
			return false
		}
		ones, bits := network.Mask.Size()
		if bits == 32 {
			return ones <= 8
		}
		if bits == 128 {
			return ones <= 32
		}
	}
	return false
}

func AnalyzeIPList(text string, patternLimit int) IPListStats {
	entries := ParseIPBlacklistLines(text)
	stats := IPListStats{
		Total:        len(entries),
		PatternLimit: patternLimit,
	}
	for _, entry := range entries {
		if !IsValidIPBlacklistEntry(entry) {
			stats.Invalid++
			continue
		}
		if IsPatternIPEntry(entry) {
			stats.Pattern++
			if IsOverbroadIPPattern(entry) {
				stats.Overbroad++
			}
			continue
		}
		stats.Exact++
	}
	return stats
}

func ValidateWAFIPLists(waf models.WAFConfig, patternLimit int) error {
	if patternLimit <= 0 {
		patternLimit = defaultWebsiteResources().MaxWAFPatternIPs
	}
	lists := []struct {
		name string
		text string
	}{
		{name: "WAF whitelist", text: waf.WhitelistIPs},
		{name: "WAF blacklist", text: waf.BlacklistIPs},
	}
	for _, item := range lists {
		stats := AnalyzeIPList(item.text, patternLimit)
		if stats.Invalid > 0 {
			return fmt.Errorf("%s contains %d invalid IP entries", item.name, stats.Invalid)
		}
		if stats.Pattern > patternLimit {
			return fmt.Errorf("%s pattern IP count %d exceeds limit %d", item.name, stats.Pattern, patternLimit)
		}
	}
	return nil
}

func isValidIPWildcardPattern(pattern string) bool {
	parts := strings.Split(pattern, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		if part == "*" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

func NormalizeIPBlacklistEntry(raw string) string {
	return strings.TrimSpace(raw)
}

func ParseIPBlacklistLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		entry := NormalizeIPBlacklistEntry(line)
		if entry == "" {
			continue
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return out
}
