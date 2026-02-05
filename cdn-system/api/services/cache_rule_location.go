package services

import (
	"cdn-api/models"
	"strings"
)

func normalizeRuleLocation(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	if strings.HasPrefix(rule, "=") || strings.HasPrefix(rule, "^~") || strings.HasPrefix(rule, "~") {
		return rule
	}
	if strings.HasPrefix(rule, "/") {
		return "^~ " + rule
	}
	if strings.HasPrefix(rule, ".") {
		return "~* \\" + rule + "$"
	}
	return "~* " + rule
}

func normalizeLocationKey(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	parts := strings.Fields(location)
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "=":
		if len(parts) < 2 {
			return "exact"
		}
		return "exact " + strings.Join(parts[1:], " ")
	case "^~":
		if len(parts) < 2 {
			return "prefix"
		}
		return "prefix " + strings.Join(parts[1:], " ")
	default:
		if strings.HasPrefix(parts[0], "~") {
			if len(parts) < 2 {
				return "regex " + parts[0]
			}
			return "regex " + parts[0] + " " + strings.Join(parts[1:], " ")
		}
	}
	return "prefix " + strings.Join(parts, " ")
}

func cacheRuleLocation(rule models.EdgeCacheRule) string {
	if rule.Rule != "" {
		return normalizeRuleLocation(rule.Rule)
	}
	if rule.URI != "" {
		uri := strings.TrimSpace(rule.URI)
		if !strings.HasPrefix(uri, "/") {
			return ""
		}
		return "= " + uri
	}
	if rule.Prefix != "" {
		prefix := strings.TrimSpace(rule.Prefix)
		if !strings.HasPrefix(prefix, "/") {
			return ""
		}
		return "^~ " + prefix
	}
	if rule.Ext != "" {
		ext := strings.TrimSpace(rule.Ext)
		ext = strings.TrimPrefix(ext, "*")
		ext = strings.TrimPrefix(ext, ".")
		if ext == "" {
			return ""
		}
		return "~* \\." + ext + "$"
	}
	return ""
}

func cacheRuleLocationKey(rule models.EdgeCacheRule) string {
	location := cacheRuleLocation(rule)
	if location == "" {
		return ""
	}
	return normalizeLocationKey(location)
}

func dedupeEdgeCacheRules(rules []models.EdgeCacheRule) []models.EdgeCacheRule {
	if len(rules) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]models.EdgeCacheRule, 0, len(rules))
	for _, rule := range rules {
		key := cacheRuleLocationKey(rule)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, rule)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
