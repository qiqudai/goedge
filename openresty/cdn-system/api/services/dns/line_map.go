package dns

import "strings"

var providerLineMap = map[string]map[string]string{
	"aliyun": {
		"default":  "default",
		"telecom":  "telecom",
		"unicom":   "unicom",
		"mobile":   "mobile",
		"ctt":      "tieTong",
		"broadnet": "broadcast",
		"edu":      "edu",
		"cn":       "mainland",
		"global":   "oversea",
		"search":   "search",
	},
	"dnspod": {
		"default":  "默认",
		"telecom":  "电信",
		"unicom":   "联通",
		"mobile":   "移动",
		"china":    "境内",
		"cn":       "境内",
		"global":   "境外",
		"search":   "搜索引擎",
		"anhui":        "安徽",
		"beijing":      "北京",
		"chongqing":    "重庆",
		"fujian":       "福建",
		"gansu":        "甘肃",
		"guangdong":    "广东",
		"guangxi":      "广西",
		"guizhou":      "贵州",
		"hainan":       "海南",
		"hebei":        "河北",
		"heilongjiang": "黑龙江",
		"henan":        "河南",
		"hubei":        "湖北",
		"hunan":        "湖南",
		"jiangsu":      "江苏",
		"jiangxi":      "江西",
		"jilin":        "吉林",
		"liaoning":     "辽宁",
		"neimenggu":    "内蒙古",
		"ningxia":      "宁夏",
		"qinghai":      "青海",
		"shaanxi":      "陕西",
		"shandong":     "山东",
		"shanghai":     "上海",
		"shanxi":       "山西",
		"sichuan":      "四川",
		"tianjin":      "天津",
		"xizang":       "西藏",
		"xinjiang":     "新疆",
		"yunnan":       "云南",
		"zhejiang":     "浙江",
		"tie-tong":  "铁通",
		"ctt":       "铁通",
		"broadcast": "广电",
		"broadnet":  "广电",
		"edu":       "教育网",
	},
	"dnspod_intl": {
		"default":  "默认",
		"telecom":  "电信",
		"unicom":   "联通",
		"mobile":   "移动",
		"china":    "境内",
		"cn":       "境内",
		"global":   "境外",
		"search":   "搜索引擎",
		"anhui":        "安徽",
		"beijing":      "北京",
		"chongqing":    "重庆",
		"fujian":       "福建",
		"gansu":        "甘肃",
		"guangdong":    "广东",
		"guangxi":      "广西",
		"guizhou":      "贵州",
		"hainan":       "海南",
		"hebei":        "河北",
		"heilongjiang": "黑龙江",
		"henan":        "河南",
		"hubei":        "湖北",
		"hunan":        "湖南",
		"jiangsu":      "江苏",
		"jiangxi":      "江西",
		"jilin":        "吉林",
		"liaoning":     "辽宁",
		"neimenggu":    "内蒙古",
		"ningxia":      "宁夏",
		"qinghai":      "青海",
		"shaanxi":      "陕西",
		"shandong":     "山东",
		"shanghai":     "上海",
		"shanxi":       "山西",
		"sichuan":      "四川",
		"tianjin":      "天津",
		"xizang":       "西藏",
		"xinjiang":     "新疆",
		"yunnan":       "云南",
		"zhejiang":     "浙江",
		"tie-tong":  "铁通",
		"ctt":       "铁通",
		"broadcast": "广电",
		"broadnet":  "广电",
		"edu":       "教育网",
	},
	"dnsla": {
		"default":  "默认",
		"telecom":  "电信",
		"unicom":   "联通",
		"mobile":   "移动",
		"ctt":      "铁通",
		"broadnet": "广电",
		"edu":      "教育网",
		"cn":       "境内",
		"global":   "境外",
		"search":   "搜索引擎",
	},
	"huawei": {
		"default":  "默认",
		"telecom":  "电信",
		"unicom":   "联通",
		"mobile":   "移动",
		"ctt":      "铁通",
		"broadnet": "广电",
		"edu":      "教育网",
		"cn":       "境内",
		"global":   "境外",
		"search":   "搜索引擎",
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
	if strings.TrimSpace(customValue) != "" {
		return strings.TrimSpace(customValue)
	}
	return ""
}
