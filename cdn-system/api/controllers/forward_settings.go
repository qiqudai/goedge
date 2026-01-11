package controllers

import (
	"cdn-api/models"
	"encoding/json"
	"strconv"
	"strings"
)

func applyOriginSettings(settings map[string]interface{}, updates map[string]interface{}) {
	origin, ok := settings["origin"].(map[string]interface{})
	if !ok {
		return
	}
	if v, ok := origin["balance_way"]; ok {
		if s := toString(v); s != "" {
			updates["balance_way"] = s
		}
	}
	if v, ok := origin["proxy_protocol"]; ok {
		updates["proxy_protocol"] = toBool(v)
	}
	if v, ok := origin["backsource_port"]; ok {
		if s := toString(v); s != "" {
			updates["backend_port"] = s
		}
	}
	if v, ok := origin["origins"]; ok {
		if encoded := encodeOriginsAny(v); encoded != "" {
			updates["backend"] = encoded
		}
	}
}

func extractBackendPort(origins []models.ForwardOrigin) string {
	if len(origins) == 0 {
		return ""
	}
	addr := strings.TrimSpace(origins[0].Address)
	if addr == "" {
		return ""
	}
	if strings.Count(addr, ":") == 1 {
		parts := strings.Split(addr, ":")
		if len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}
	if strings.Contains(addr, "]:") {
		parts := strings.Split(addr, "]:")
		if len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}
	return ""
}

func encodeOriginsAny(v interface{}) string {
	switch value := v.(type) {
	case []models.ForwardOrigin:
		return encodeOrigins(value)
	case []interface{}:
		origins := make([]models.ForwardOrigin, 0, len(value))
		for _, item := range value {
			if m, ok := item.(map[string]interface{}); ok {
				origins = append(origins, models.ForwardOrigin{
					Address: toString(m["address"]),
					Weight:  toInt(m["weight"], 1),
					Enable:  toBoolWithDefault(m["enable"], true),
				})
			}
		}
		return encodeOrigins(origins)
	default:
		return ""
	}
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func toBool(v interface{}) bool {
	return toBoolWithDefault(v, false)
}

func toBoolWithDefault(v interface{}, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			return def
		}
		return t == "1" || t == "true" || t == "yes" || t == "on"
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	default:
		return def
	}
}

func toInt(v interface{}, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return i
		}
	}
	return def
}

func encodeStringList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func encodeOrigins(items []models.ForwardOrigin) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}
