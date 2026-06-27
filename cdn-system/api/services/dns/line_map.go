package dns

import (
	"cdn-common/i18n"
	"strings"
	"sync"
)

var (
	providerLineMap  map[string]map[string]string
	providerLineOnce sync.Once
)

func initProviderLineMap() {
	providerLineMap = map[string]map[string]string{
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
		"dnspod":      buildChinaLineMap(),
		"dnspod_intl": buildDNSPodIntlLineMap(),
		"dnsla":       buildCommonLineMap(),
		"huawei":      buildCommonLineMap(),
	}
}

func buildCommonLineMap() map[string]string {
	return map[string]string{
		"default":  i18n.T("dns.line_default"),
		"telecom":  i18n.T("dns.line_telecom"),
		"unicom":   i18n.T("dns.line_unicom"),
		"mobile":   i18n.T("dns.line_mobile"),
		"ctt":      i18n.T("dns.line_ctt"),
		"broadnet": i18n.T("dns.line_broadnet"),
		"edu":      i18n.T("dns.line_edu"),
		"cn":       i18n.T("dns.line_cn"),
		"global":   i18n.T("dns.line_global"),
		"search":   i18n.T("dns.line_search"),
	}
}

func buildChinaLineMap() map[string]string {
	return map[string]string{
		"default":      i18n.T("dns.line_default"),
		"telecom":      i18n.T("dns.line_telecom"),
		"unicom":       i18n.T("dns.line_unicom"),
		"mobile":       i18n.T("dns.line_mobile"),
		"china":        i18n.T("dns.line_cn"),
		"cn":           i18n.T("dns.line_cn"),
		"global":       i18n.T("dns.line_global"),
		"search":       i18n.T("dns.line_search"),
		"anhui":        i18n.T("dns.region_anhui"),
		"beijing":      i18n.T("dns.region_beijing"),
		"chongqing":    i18n.T("dns.region_chongqing"),
		"fujian":       i18n.T("dns.region_fujian"),
		"gansu":        i18n.T("dns.region_gansu"),
		"guangdong":    i18n.T("dns.region_guangdong"),
		"guangxi":      i18n.T("dns.region_guangxi"),
		"guizhou":      i18n.T("dns.region_guizhou"),
		"hainan":       i18n.T("dns.region_hainan"),
		"hebei":        i18n.T("dns.region_hebei"),
		"heilongjiang": i18n.T("dns.region_heilongjiang"),
		"henan":        i18n.T("dns.region_henan"),
		"hubei":        i18n.T("dns.region_hubei"),
		"hunan":        i18n.T("dns.region_hunan"),
		"jiangsu":      i18n.T("dns.region_jiangsu"),
		"jiangxi":      i18n.T("dns.region_jiangxi"),
		"jilin":        i18n.T("dns.region_jilin"),
		"liaoning":     i18n.T("dns.region_liaoning"),
		"neimenggu":    i18n.T("dns.region_neimenggu"),
		"ningxia":      i18n.T("dns.region_ningxia"),
		"qinghai":      i18n.T("dns.region_qinghai"),
		"shaanxi":      i18n.T("dns.region_shaanxi"),
		"shandong":     i18n.T("dns.region_shandong"),
		"shanghai":     i18n.T("dns.region_shanghai"),
		"shanxi":       i18n.T("dns.region_shanxi"),
		"sichuan":      i18n.T("dns.region_sichuan"),
		"tianjin":      i18n.T("dns.region_tianjin"),
		"xizang":       i18n.T("dns.region_xizang"),
		"xinjiang":     i18n.T("dns.region_xinjiang"),
		"yunnan":       i18n.T("dns.region_yunnan"),
		"zhejiang":     i18n.T("dns.region_zhejiang"),
		"tie-tong":     i18n.T("dns.line_ctt"),
		"ctt":          i18n.T("dns.line_ctt"),
		"broadcast":    i18n.T("dns.line_broadnet"),
		"broadnet":     i18n.T("dns.line_broadnet"),
		"edu":          i18n.T("dns.line_edu"),
	}
}

func buildDNSPodIntlLineMap() map[string]string {
	return map[string]string{
		"default":   "Default",
		"telecom":   "Telecom",
		"unicom":    "Unicom",
		"mobile":    "Mobile",
		"china":     "China",
		"cn":        "China",
		"global":    "Oversea",
		"search":    "Search",
		"tie-tong":  "TieTong",
		"ctt":       "TieTong",
		"broadcast": "Broadcast",
		"broadnet":  "Broadcast",
		"edu":       "Edu",
	}
}

// ResolveLineValue returns the vendor-specific line value.
// For custom line, pass the vendor line name via customValue.
func ResolveLineValue(providerType, lineID, customValue string) string {
	providerLineOnce.Do(initProviderLineMap)
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
