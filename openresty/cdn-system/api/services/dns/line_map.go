package dns

import "strings"

var providerLineMap = map[string]map[string]string{
	"aliyun": {
		"default": "default",
		"telecom": "telecom",
		"unicom":  "unicom",
		"mobile":  "mobile",
		"ctt":     "tieTong",
		"broadnet": "broadcast",
		"edu":     "edu",
		"cn":      "mainland",
		"global":  "oversea",
		"search":  "search",
	},
	"dnspod": {
		"default": "默认",
		"telecom": "电信",
		"unicom":  "联通",
		"mobile":  "移动",
		"ctt":     "铁通",
		"broadnet": "广电",
		"edu":     "教育网",
		"cn":      "境内",
		"global":  "境外",
		"search":  "搜索引擎",
	},
	"dnsla": {
		"default": "默认",
		"telecom": "电信",
		"unicom":  "联通",
		"mobile":  "移动",
		"ctt":     "铁通",
		"broadnet": "广电",
		"edu":     "教育网",
		"cn":      "境内",
		"global":  "境外",
		"search":  "搜索引擎",
	},
	"huawei": {
		"default": "默认",
		"telecom": "电信",
		"unicom":  "联通",
		"mobile":  "移动",
		"ctt":     "铁通",
		"broadnet": "广电",
		"edu":     "教育网",
		"cn":      "境内",
		"global":  "境外",
		"search":  "搜索引擎",
	},
}

// ResolveLineValue returns the vendor-specific line value.
// For custom line, pass the vendor line name via customValue.
func ResolveLineValue(providerType, lineID, customValue string) string {
	p := strings.ToLower(strings.TrimSpace(providerType))
	l := strings.ToLower(strings.TrimSpace(lineID))
	if l == "custom" {
		return strings.TrimSpace(customValue)
	}
	if m, ok := providerLineMap[p]; ok {
		if v, exists := m[l]; exists {
			return v
		}
	}
	return ""
}
