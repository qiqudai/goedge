package services

import (
	"encoding/json"
	"sort"
	"strings"
)

func NormalizeSiteSettings(settings map[string]interface{}) map[string]interface{} {
	if settings == nil {
		return nil
	}
	if cacheCfg := getMap(settings, "cache"); cacheCfg != nil {
		if raw, ok := cacheCfg["rules"]; ok {
			cacheCfg["rules"] = normalizeCacheRulesRaw(raw)
		}
	}
	if raw, ok := settings["url_rewrites"]; ok {
		settings["url_rewrites"] = normalizeURLRewritesRaw(raw)
	}
	if adv := getMap(settings, "advanced"); adv != nil {
		if raw, ok := adv["url_redirects"]; ok {
			adv["url_redirects"] = normalizeURLRedirectsRaw(raw)
		}
		if raw, ok := adv["url_rewrites"]; ok {
			adv["url_rewrites"] = normalizeURLRewritesRaw(raw)
		}
		if raw, ok := adv["req_headers"]; ok {
			adv["req_headers"] = normalizeHeaderRulesRaw(raw)
		}
		if raw, ok := adv["res_headers"]; ok {
			adv["res_headers"] = normalizeHeaderRulesRaw(raw)
		}
	}
	return settings
}

func normalizeCacheRulesRaw(raw interface{}) []map[string]interface{} {
	items := normalizeMapSlice(raw)
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]map[string]interface{}, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		normalized, keep := normalizeCacheRuleItem(item, seen)
		if keep && normalized != nil {
			out = append(out, normalized)
		}
	}
	reverseMapSlice(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeCacheRuleItem(item map[string]interface{}, seen map[string]struct{}) (map[string]interface{}, bool) {
	if item == nil {
		return nil, false
	}
	ruleExpr := strings.TrimSpace(parseString(item["rule"]))
	uri := strings.TrimSpace(parseString(item["uri"]))
	prefix := strings.TrimSpace(parseString(item["prefix"]))
	ext := strings.TrimSpace(parseString(item["ext"]))
	ruleType := strings.ToLower(strings.TrimSpace(parseString(item["type"])))
	rawValue := parseString(item["value"])

	tryKeep := func(location string) bool {
		key := normalizeLocationKey(location)
		if key == "" {
			return false
		}
		if _, ok := seen[key]; ok {
			return false
		}
		seen[key] = struct{}{}
		return true
	}

	if ruleExpr != "" {
		location := normalizeRuleLocation(ruleExpr)
		if location != "" && tryKeep(location) {
			item["rule"] = ruleExpr
			return item, true
		}
		return nil, false
	}
	if uri != "" {
		path := normalizeCachePath(uri)
		location := ""
		if path != "" {
			location = "= " + path
		}
		if location != "" && tryKeep(location) {
			item["uri"] = path
			return item, true
		}
		return nil, false
	}
	if prefix != "" {
		path := normalizeCachePath(prefix)
		location := ""
		if path != "" {
			location = "^~ " + path
		}
		if location != "" && tryKeep(location) {
			item["prefix"] = path
			return item, true
		}
		return nil, false
	}
	if ext != "" {
		extValue := normalizeCacheExtValue(ext)
		location := ""
		if extValue != "" {
			location = "~* \\." + extValue + "$"
		}
		if location != "" && tryKeep(location) {
			item["ext"] = extValue
			return item, true
		}
		return nil, false
	}

	switch ruleType {
	case "all":
		if tryKeep("^~ /") {
			return item, true
		}
		return nil, false
	case "index":
		if tryKeep("= /") {
			return item, true
		}
		return nil, false
	case "dir", "path", "suffix":
		values := splitCacheRuleValues(rawValue)
		if len(values) == 0 {
			return nil, false
		}
		kept := make([]string, 0, len(values))
		for _, value := range values {
			switch ruleType {
			case "suffix":
				extValue := normalizeCacheExtValue(value)
				if extValue == "" {
					continue
				}
				location := "~* \\." + extValue + "$"
				if !tryKeep(location) {
					continue
				}
				kept = append(kept, extValue)
			case "dir":
				path := normalizeCachePath(value)
				if path == "" {
					continue
				}
				location := "^~ " + path
				if !tryKeep(location) {
					continue
				}
				kept = append(kept, path)
			default:
				path := normalizeCachePath(value)
				if path == "" {
					continue
				}
				location := "= " + path
				if !tryKeep(location) {
					continue
				}
				kept = append(kept, path)
			}
		}
		if len(kept) == 0 {
			return nil, false
		}
		item["value"] = strings.Join(kept, "|")
		return item, true
	default:
		return item, true
	}
}

func normalizeCacheExtValue(value string) string {
	raw := strings.TrimSpace(strings.ToLower(value))
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "*")
	raw = strings.TrimPrefix(raw, ".")
	return raw
}

func normalizeHeaderRulesRaw(raw interface{}) []map[string]interface{} {
	items := normalizeMapSlice(raw)
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]map[string]interface{}, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		name := strings.TrimSpace(parseString(item["name"]))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item["name"] = name
		out = append(out, item)
	}
	reverseMapSlice(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeURLRedirectsRaw(raw interface{}) []map[string]interface{} {
	items := normalizeMapSlice(raw)
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]map[string]interface{}, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		match := strings.TrimSpace(parseString(item["match"]))
		redirect := strings.TrimSpace(parseString(item["redirect"]))
		if match == "" || redirect == "" {
			continue
		}
		domain := strings.TrimSpace(parseString(item["domain"]))
		code := strings.TrimSpace(parseString(item["code"]))
		condKey := buildRedirectConditionKey(item["conditions"])
		key := strings.ToLower(domain) + "|" + match + "|" + redirect + "|" + code + "|" + condKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item["domain"] = domain
		item["match"] = match
		item["redirect"] = redirect
		item["code"] = code
		out = append(out, item)
	}
	reverseMapSlice(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeURLRewritesRaw(raw interface{}) []map[string]interface{} {
	items := normalizeMapSlice(raw)
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]map[string]interface{}, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		match := strings.TrimSpace(parseString(item["match"]))
		replace := strings.TrimSpace(parseString(item["replace"]))
		if replace == "" {
			replace = strings.TrimSpace(parseString(item["redirect"]))
		}
		if match == "" || replace == "" {
			continue
		}
		code := strings.TrimSpace(parseString(item["code"]))
		key := strings.ToLower(match) + "|" + replace + "|" + code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item["match"] = match
		item["replace"] = replace
		item["code"] = code
		out = append(out, item)
	}
	reverseMapSlice(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildRedirectConditionKey(raw interface{}) string {
	items := normalizeMapSlice(raw)
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(parseString(item["key"]))
		if key == "" {
			key = strings.TrimSpace(parseString(item["item"]))
		}
		value := strings.TrimSpace(parseString(item["value"]))
		if key == "" && value == "" {
			continue
		}
		parts = append(parts, strings.ToLower(key)+"="+value)
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

func normalizeMapSlice(raw interface{}) []map[string]interface{} {
	if raw == nil {
		return nil
	}
	switch list := raw.(type) {
	case []map[string]interface{}:
		if len(list) == 0 {
			return nil
		}
		return list
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		if b, err := json.Marshal(raw); err == nil {
			var parsed []map[string]interface{}
			if json.Unmarshal(b, &parsed) == nil && len(parsed) > 0 {
				return parsed
			}
		}
	}
	return nil
}

func reverseMapSlice(items []map[string]interface{}) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
