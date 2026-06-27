package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
)

var errorPageKeys = []string{
	"400", "403", "502", "504",
	"traffic_limit", "site_locked", "domain_invalid", "conn_limit", "timeout", "ip",
	"access_blocked",
}

var legacyErrorPageAliases = map[string]string{
	"p400":                "400",
	"p403":                "403",
	"p502":                "502",
	"p504":                "504",
	"p512":                "timeout",
	"p513":                "traffic_limit",
	"p514":                "site_locked",
	"p515":                "conn_limit",
	"access_ip_not_allow": "ip",
	"host_not_found":      "domain_invalid",
}

var placeholderPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

func DefaultErrorPageI18nSettings() models.ErrorPageI18nSettings {
	return models.ErrorPageI18nSettings{
		DefaultLang:  "zh-CN",
		LangMode:     "browser",
		EnabledLangs: i18n.ErrorPageDefaultLocales(),
	}
}

func defaultAccessBlockedTemplate() string {
	return `<!DOCTYPE html>
<html class="no-js" lang="en-US">
<head>
<title>{{title}}</title>
<meta charset="UTF-8" />
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta http-equiv="X-UA-Compatible" content="IE=Edge,chrome=1" />
<meta name="robots" content="noindex, nofollow" />
<meta name="viewport" content="width=device-width,initial-scale=1" />
<style>
*, body, html { margin: 0; padding: 0; box-sizing: border-box; }
body {
    background: #fff;
    color: #404040;
    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif;
    font-size: 16px;
    -webkit-font-smoothing: antialiased;
}
.wrap { max-width: 960px; margin: 0 auto; padding: 48px 24px 64px; }
.hero { margin-bottom: 32px; }
.hero h1 { font-size: 42px; font-weight: 400; line-height: 1.2; color: #313131; margin: 0 0 12px; }
.hero p { font-size: 22px; color: #666; margin: 0; }
.hero .host { color: #404040; font-weight: 500; word-break: break-all; }
.illustration { display: flex; justify-content: center; margin: 36px 0 48px; }
.browser {
    width: min(100%, 520px);
    border: 1px solid #d9d9d9;
    border-radius: 8px;
    background: #f5f5f5;
    overflow: hidden;
    box-shadow: 0 1px 2px rgba(0,0,0,.04);
}
.browser-bar { height: 28px; background: #ececec; border-bottom: 1px solid #ddd; }
.browser-body {
    min-height: 220px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fafafa;
}
.block-icon {
    width: 88px;
    height: 88px;
    border-radius: 50%;
    background: #b91c1c;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 48px;
    line-height: 1;
    font-weight: 300;
}
.columns { display: flex; gap: 48px; flex-wrap: wrap; }
.column { flex: 1 1 280px; min-width: 0; }
.column h2 { font-size: 22px; font-weight: 400; color: #313131; margin: 0 0 12px; }
.column p { font-size: 15px; line-height: 1.6; color: #666; margin: 0; }
.footer {
    margin-top: 40px;
    padding-top: 20px;
    border-top: 1px solid #e5e5e5;
    font-size: 13px;
    color: #888;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
@media (max-width: 768px) {
    .hero h1 { font-size: 30px; }
    .hero p { font-size: 18px; }
    .columns { flex-direction: column; gap: 28px; }
}
</style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <h1>{{heading}}</h1>
      <p>{{unable_access}} <span class="host">{host}</span></p>
    </div>
    <div class="illustration">
      <div class="browser" aria-hidden="true">
        <div class="browser-bar"></div>
        <div class="browser-body"><div class="block-icon">&times;</div></div>
      </div>
    </div>
    <div class="columns">
      <div class="column">
        <h2>{{what_happened}}</h2>
        <p>{{what_happened_desc}}</p>
      </div>
      <div class="column">
        <h2>{{what_can_i_do}}</h2>
        <p>{{what_can_i_do_desc}}</p>
      </div>
    </div>
    <div class="footer">{{client_ip_label}}: {client_ip} &bull; {{request_id_label}}: {request_id}</div>
  </div>
</body>
</html>`
}

func defaultAccessBlockedStrings() map[string]map[string]string {
	type localeText struct {
		title, heading, unableAccess, whatHappened, whatHappenedDesc, whatCanIDo, whatCanIDoDesc, clientIPLabel, requestIDLabel string
	}
	byLocale := map[string]localeText{
		"zh-CN": {
			title: "访问已被拦截", heading: "抱歉，您已被禁止访问",
			unableAccess: "您无法访问", whatHappened: "为什么会被拦截？",
			whatHappenedDesc: "该网站启用了安全防护服务，用于拦截异常或受限来源访问。当前请求触发了 IP 黑名单或地区访问限制。",
			whatCanIDo: "如何解决？", whatCanIDoDesc: "请联系网站管理员说明情况，并提供您的 IP 地址与请求 ID。",
			clientIPLabel: "您的 IP", requestIDLabel: "请求 ID",
		},
		"zh-TW": {
			title: "存取已被攔截", heading: "抱歉，您已被禁止存取",
			unableAccess: "您無法存取", whatHappened: "為什麼會被攔截？",
			whatHappenedDesc: "此網站啟用了安全防護服務，用於攔截異常或受限來源存取。目前請求觸發了 IP 黑名單或地區存取限制。",
			whatCanIDo: "如何解決？", whatCanIDoDesc: "請聯繫網站管理員說明情況，並提供您的 IP 位址與請求 ID。",
			clientIPLabel: "您的 IP", requestIDLabel: "請求 ID",
		},
		"en": {
			title: "Access Blocked", heading: "Sorry, you have been blocked",
			unableAccess: "You are unable to access", whatHappened: "Why have I been blocked?",
			whatHappenedDesc: "This website is using a security service to protect itself from online attacks. Your request matched an IP blacklist or regional access restriction.",
			whatCanIDo: "What can I do to resolve this?", whatCanIDoDesc: "Email the site owner and include what you were doing when this page appeared, your IP address, and the request ID below.",
			clientIPLabel: "Your IP", requestIDLabel: "Request ID",
		},
		"fr": {
			title: "Accès bloqué", heading: "Désolé, vous avez été bloqué",
			unableAccess: "Vous ne pouvez pas accéder à", whatHappened: "Pourquoi ai-je été bloqué ?",
			whatHappenedDesc: "Ce site utilise un service de sécurité pour se protéger. Votre requête correspond à une liste noire IP ou à une restriction régionale.",
			whatCanIDo: "Que puis-je faire ?", whatCanIDoDesc: "Contactez le propriétaire du site en indiquant votre IP et l'identifiant de requête ci-dessous.",
			clientIPLabel: "Votre IP", requestIDLabel: "ID de requête",
		},
	}
	fallback := byLocale["en"]
	out := make(map[string]map[string]string, len(i18n.ErrorPageDefaultLocales()))
	for _, locale := range i18n.ErrorPageDefaultLocales() {
		text, ok := byLocale[locale]
		if !ok {
			text = fallback
		}
		out[locale] = map[string]string{
			"title":              text.title,
			"heading":            text.heading,
			"unable_access":      text.unableAccess,
			"what_happened":      text.whatHappened,
			"what_happened_desc": text.whatHappenedDesc,
			"what_can_i_do":      text.whatCanIDo,
			"what_can_i_do_desc": text.whatCanIDoDesc,
			"client_ip_label":    text.clientIPLabel,
			"request_id_label":   text.requestIDLabel,
		}
	}
	return out
}

func defaultErrorPageTemplate(statusCode string, showIP bool) string {
	ipBlock := ""
	if showIP {
		ipBlock = `         <span class="inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed">{{client_ip_label}}: {client_ip} &bull;</span>
         <span class="inline-block md:block heading-ray-id font-mono text-15 lg:text-sm lg:leading-relaxed">{{node_ip_label}}: {node_ip}</span>
`
	}
	return `<!DOCTYPE html>
<html class="no-js" lang="en-US">
<head>
<title>{{title}}</title>
<meta charset="UTF-8" />
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta http-equiv="X-UA-Compatible" content="IE=Edge,chrome=1" />
<meta name="robots" content="noindex, nofollow" />
<meta name="viewport" content="width=device-width,initial-scale=1" />
<style>
*, body, html { margin: 0; padding: 0; }
body, html {
    --text-opacity: 1;
    color: #404040;
    color: rgba(64,64,64,var(--text-opacity));
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    font-family: system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif;
    font-size: 16px;
}
* { box-sizing: border-box; }
.p-0 { padding: 0; }
.w-240 { width: 60rem; }
.antialiased { -webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale; }
.pt-10 { padding-top: 2.5rem; }
.mb-15 { margin-bottom: 3.75rem; }
.mx-auto { margin-left: auto; margin-right: auto; }
.text-black-dark { color: #404040; }
.mr-2 { margin-right: .5rem; }
.leading-tight { line-height: 1.25; }
.text-60 { font-size: 60px; }
.font-light { font-weight: 300; }
.inline-block { display: inline-block; }
.text-15 { font-size: 15px; }
.font-mono { font-family: monaco,courier,monospace; }
.text-gray-600 { color: #999; }
.leading-1\.3 { line-height: 1.3; }
.text-3xl { font-size: 1.875rem; }
.mb-8 { margin-bottom: 2rem; }
.w-1\/2 { width: 50%; }
.mt-6 { margin-top: 1.5rem; }
.mb-4 { margin-bottom: 1rem; }
.font-normal { font-weight: 400; }
#what-happened-section p { font-size: 15px; line-height: 1.5; }
</style>
</head>
<body>
  <div id="cf-wrapper">
    <div id="cf-error-details" class="p-0">
      <header class="mx-auto pt-10 lg:pt-6 lg:px-8 w-240 lg:w-full mb-15 antialiased">
         <h1 class="inline-block md:block mr-2 md:mb-2 font-light text-60 md:text-3xl text-black-dark leading-tight">
           <span>{{error_label}}</span>
           <span>` + statusCode + `</span>
         </h1>
` + ipBlock + `        <h2 class="text-gray-600 leading-1.3 text-3xl lg:text-2xl font-light">{{subtitle}}</h2>
      </header>
      <section class="w-240 lg:w-full mx-auto mb-8 lg:px-8">
          <div id="what-happened-section" class="w-1/2 md:w-full">
            <h2 class="text-3xl leading-tight font-normal mb-4 text-black-dark antialiased">{{what_happened}}</h2>
            <p>{{what_happened_desc}}</p>
          </div>
          <div id="resolution-copy-section" class="w-1/2 mt-6 text-15 leading-normal">
            <h2 class="text-3xl leading-tight font-normal mb-4 text-black-dark antialiased">{{what_can_i_do}}</h2>
            <p>{{what_can_i_do_desc}}</p>
          </div>
      </section>
    </div>
  </div>
</body>
</html>`
}

func defaultErrorPageStrings() map[string]map[string]map[string]string {
	return i18n.ErrorPageDefaultStrings()
}

func errorPageStatusCode(key string) string {
	switch key {
	case "traffic_limit":
		return "509"
	case "site_locked":
		return "451"
	case "domain_invalid":
		return "530"
	case "conn_limit":
		return "515"
	case "timeout":
		return "512"
	case "ip":
		return "1003"
	case "access_blocked":
		return "419"
	default:
		return key
	}
}

func errorPageShowsIP(key string) bool {
	return key != "ip" && key != "access_blocked"
}

func DefaultErrorPageDefinitions() map[string]models.ErrorPageDefinition {
	defaults := defaultErrorPageStrings()
	out := make(map[string]models.ErrorPageDefinition, len(errorPageKeys))
	for _, key := range errorPageKeys {
		stringsByLang := defaults[key]
		if stringsByLang == nil {
			stringsByLang = map[string]map[string]string{}
		}
		if key == "access_blocked" {
			if len(stringsByLang) == 0 {
				stringsByLang = defaultAccessBlockedStrings()
			}
			out[key] = models.ErrorPageDefinition{
				Template: defaultAccessBlockedTemplate(),
				Strings:  stringsByLang,
			}
			continue
		}
		out[key] = models.ErrorPageDefinition{
			Template: defaultErrorPageTemplate(errorPageStatusCode(key), errorPageShowsIP(key)),
			Strings:  stringsByLang,
		}
	}
	return out
}

func NormalizeGlobalConfigErrorPages(cfg *models.GlobalConfig) {
	if cfg == nil {
		return
	}
	cfg.ErrorPageI18n = normalizeErrorPageI18nSettings(cfg.ErrorPageI18n)
	cfg.ErrorPages = normalizeErrorPageDefinitions(cfg.ErrorPages, cfg.ErrorPageI18n.DefaultLang)
}

func normalizeErrorPageI18nSettings(settings models.ErrorPageI18nSettings) models.ErrorPageI18nSettings {
	defaults := DefaultErrorPageI18nSettings()
	if strings.TrimSpace(settings.DefaultLang) == "" {
		settings.DefaultLang = defaults.DefaultLang
	}
	settings.DefaultLang = normalizeLocaleTag(settings.DefaultLang)
	mode := strings.ToLower(strings.TrimSpace(settings.LangMode))
	if mode != "fixed" {
		settings.LangMode = "browser"
	} else {
		settings.LangMode = "fixed"
	}
	langs := make([]string, 0, len(settings.EnabledLangs)+2)
	seen := map[string]struct{}{}
	addLang := func(lang string) {
		lang = normalizeLocaleTag(lang)
		if lang == "" {
			return
		}
		if _, ok := seen[lang]; ok {
			return
		}
		seen[lang] = struct{}{}
		langs = append(langs, lang)
	}
	for _, lang := range settings.EnabledLangs {
		addLang(lang)
	}
	addLang(settings.DefaultLang)
	if len(langs) == 0 {
		langs = append(langs, defaults.EnabledLangs...)
	} else if len(langs) == 2 {
		hasZhCN := false
		hasEn := false
		for _, lang := range langs {
			if lang == "zh-CN" {
				hasZhCN = true
			}
			if lang == "en" {
				hasEn = true
			}
		}
		if hasZhCN && hasEn {
			langs = append([]string(nil), defaults.EnabledLangs...)
		}
	}
	settings.EnabledLangs = langs
	return settings
}

func normalizeLocaleTag(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" || lang == "browser" || lang == "inherit" {
		return lang
	}
	lang = strings.ReplaceAll(lang, "_", "-")
	parts := strings.Split(lang, "-")
	if len(parts) == 0 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 2 {
			parts[i] = strings.ToUpper(parts[i])
		} else {
			parts[i] = strings.ToLower(parts[i])
		}
	}
	return strings.Join(parts, "-")
}

func normalizeErrorPageDefinitions(pages map[string]models.ErrorPageDefinition, defaultLang string) map[string]models.ErrorPageDefinition {
	defaults := DefaultErrorPageDefinitions()
	out := make(map[string]models.ErrorPageDefinition, len(errorPageKeys))
	for _, key := range errorPageKeys {
		def := defaults[key]
		if pages != nil {
			if raw, ok := pages[key]; ok {
				def = mergeErrorPageDefinition(def, raw)
			}
		}
		def = ensureErrorPageStrings(def, defaultLang, defaults[key].Strings)
		out[key] = def
	}
	return out
}

func mergeErrorPageDefinition(base, incoming models.ErrorPageDefinition) models.ErrorPageDefinition {
	if strings.TrimSpace(incoming.Template) != "" {
		base.Template = incoming.Template
	}
	if incoming.Strings == nil {
		return base
	}
	if base.Strings == nil {
		base.Strings = map[string]map[string]string{}
	}
	for lang, values := range incoming.Strings {
		if values == nil {
			continue
		}
		if base.Strings[lang] == nil {
			base.Strings[lang] = map[string]string{}
		}
		for key, value := range values {
			if strings.TrimSpace(value) != "" {
				base.Strings[lang][key] = value
			}
		}
	}
	return base
}

func ensureErrorPageStrings(def models.ErrorPageDefinition, defaultLang string, fallback map[string]map[string]string) models.ErrorPageDefinition {
	if def.Strings == nil {
		def.Strings = map[string]map[string]string{}
	}
	if fallback != nil {
		for lang, values := range fallback {
			if def.Strings[lang] == nil {
				def.Strings[lang] = map[string]string{}
			}
			for key, value := range values {
				if strings.TrimSpace(def.Strings[lang][key]) == "" {
					def.Strings[lang][key] = value
				}
			}
		}
	}
	if strings.TrimSpace(def.Template) == "" {
		return def
	}
	required := extractTemplateKeys(def.Template)
	if len(required) == 0 {
		return def
	}
	if strings.TrimSpace(defaultLang) == "" {
		defaultLang = "zh-CN"
	}
	if def.Strings[defaultLang] == nil {
		def.Strings[defaultLang] = map[string]string{}
	}
	for _, key := range required {
		if strings.TrimSpace(def.Strings[defaultLang][key]) != "" {
			continue
		}
		for _, langValues := range def.Strings {
			if strings.TrimSpace(langValues[key]) != "" {
				def.Strings[defaultLang][key] = langValues[key]
				break
			}
		}
	}
	return def
}

func extractTemplateKeys(template string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		key := match[1]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func ParseErrorPagesFromRaw(raw json.RawMessage) map[string]models.ErrorPageDefinition {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var objectForm map[string]models.ErrorPageDefinition
	if err := json.Unmarshal(raw, &objectForm); err == nil && len(objectForm) > 0 {
		first := ""
		for key := range objectForm {
			first = key
			break
		}
		if first != "" && strings.TrimSpace(objectForm[first].Template) != "" {
			return objectForm
		}
	}
	var mixed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &mixed); err != nil {
		return nil
	}
	out := make(map[string]models.ErrorPageDefinition)
	for key, itemRaw := range mixed {
		mappedKey := mapLegacyErrorPageKey(key)
		var asString string
		if err := json.Unmarshal(itemRaw, &asString); err == nil && strings.TrimSpace(asString) != "" {
			out[mappedKey] = migrateLegacyErrorPageHTML(mappedKey, asString)
			continue
		}
		var asDef models.ErrorPageDefinition
		if err := json.Unmarshal(itemRaw, &asDef); err == nil && strings.TrimSpace(asDef.Template) != "" {
			out[mappedKey] = asDef
		}
	}
	return out
}

func mapLegacyErrorPageKey(key string) string {
	key = strings.TrimSpace(key)
	if mapped, ok := legacyErrorPageAliases[key]; ok {
		return mapped
	}
	return key
}

func migrateLegacyErrorPageHTML(key, html string) models.ErrorPageDefinition {
	defaults := DefaultErrorPageDefinitions()
	def := defaults[key]
	if strings.TrimSpace(def.Template) == "" {
		def = defaults["403"]
	}
	zhStrings := map[string]string{}
	if def.Strings != nil && def.Strings["zh-CN"] != nil {
		for k, v := range def.Strings["zh-CN"] {
			zhStrings[k] = v
		}
	}
	if title := extractHTMLTagText(html, "title"); title != "" {
		zhStrings["title"] = title
	}
	if strings.Contains(html, "什么问题") {
		zhStrings["what_happened"] = "什么问题?"
	}
	if strings.Contains(html, "如何解决") {
		zhStrings["what_can_i_do"] = "如何解决?"
	}
	def.Strings = map[string]map[string]string{
		"zh-CN": zhStrings,
	}
	if def.Strings["en"] == nil && defaults[key].Strings != nil {
		def.Strings["en"] = defaults[key].Strings["en"]
	}
	return def
}

func extractHTMLTagText(html, tag string) string {
	re := regexp.MustCompile(`(?is)<` + tag + `[^>]*>(.*?)</` + tag + `>`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func RenderErrorPage(template string, localeStrings map[string]string) string {
	if strings.TrimSpace(template) == "" {
		return ""
	}
	return placeholderPattern.ReplaceAllStringFunc(template, func(token string) string {
		match := placeholderPattern.FindStringSubmatch(token)
		if len(match) < 2 {
			return token
		}
		if val, ok := localeStrings[match[1]]; ok {
			return val
		}
		return token
	})
}

func ResolveErrorPageStrings(def models.ErrorPageDefinition, lang, defaultLang string, enabledLangs []string) (map[string]string, string) {
	lang = normalizeLocaleTag(lang)
	defaultLang = normalizeLocaleTag(defaultLang)
	candidates := []string{lang}
	if lang != "" {
		if base := strings.Split(lang, "-")[0]; base != lang {
			candidates = append(candidates, base)
		}
	}
	candidates = append(candidates, defaultLang)
	for _, enabled := range enabledLangs {
		candidates = append(candidates, normalizeLocaleTag(enabled))
	}
	for _, candidate := range candidates {
		if candidate == "" || def.Strings == nil {
			continue
		}
		if values, ok := def.Strings[candidate]; ok && len(values) > 0 {
			return values, candidate
		}
		if base := strings.Split(candidate, "-")[0]; base != candidate {
			if values, ok := def.Strings[base]; ok && len(values) > 0 {
				return values, base
			}
		}
	}
	for candidate, values := range def.Strings {
		if len(values) > 0 {
			return values, candidate
		}
	}
	return map[string]string{}, ""
}

func RenderErrorPageDefinition(def models.ErrorPageDefinition, lang, defaultLang string, enabledLangs []string) string {
	values, _ := ResolveErrorPageStrings(def, lang, defaultLang, enabledLangs)
	return RenderErrorPage(def.Template, values)
}

func ValidateErrorPageConfig(cfg *models.GlobalConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	i18n := normalizeErrorPageI18nSettings(cfg.ErrorPageI18n)
	if len(i18n.EnabledLangs) == 0 {
		return fmt.Errorf("enabled_langs is required")
	}
	hasDefault := false
	for _, lang := range i18n.EnabledLangs {
		if lang == i18n.DefaultLang {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		return fmt.Errorf("default_lang must be included in enabled_langs")
	}
	pages := normalizeErrorPageDefinitions(cfg.ErrorPages, i18n.DefaultLang)
	for _, key := range errorPageKeys {
		def := pages[key]
		if strings.TrimSpace(def.Template) == "" {
			return fmt.Errorf("error page %s template is required", key)
		}
		required := extractTemplateKeys(def.Template)
		for _, lang := range i18n.EnabledLangs {
			values := def.Strings[lang]
			for _, reqKey := range required {
				if strings.TrimSpace(values[reqKey]) == "" {
					return fmt.Errorf("error page %s missing translation %s for %s", key, reqKey, lang)
				}
			}
		}
	}
	return nil
}

func LoadGlobalConfigNormalized() *models.GlobalConfig {
	cfg := loadGlobalConfigRaw()
	if cfg == nil {
		cfg = &models.GlobalConfig{}
	}
	NormalizeGlobalConfigErrorPages(cfg)
	NormalizeGlobalConfigGuardPages(cfg)
	return cfg
}

func loadGlobalConfigRaw() *models.GlobalConfig {
	if db.DB == nil {
		return nil
	}
	var sys models.SysConfig
	if err := db.DB.First(&sys, "name = ?", "global_config").Error; err != nil {
		return nil
	}
	if sys.Value == "" {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(sys.Value), &raw); err != nil {
		return nil
	}
	var cfg models.GlobalConfig
	if wafRaw, ok := raw["waf"]; ok {
		_ = json.Unmarshal(wafRaw, &cfg.WAF)
	}
	if nginxRaw, ok := raw["nginx"]; ok {
		_ = json.Unmarshal(nginxRaw, &cfg.Nginx)
	}
	if defaultRaw, ok := raw["default_config"]; ok {
		_ = json.Unmarshal(defaultRaw, &cfg.DefaultConfig)
	}
	if resourcesRaw, ok := raw["resources"]; ok {
		_ = json.Unmarshal(resourcesRaw, &cfg.Resources)
	}
	if i18nRaw, ok := raw["error_page_i18n"]; ok {
		_ = json.Unmarshal(i18nRaw, &cfg.ErrorPageI18n)
	}
	if pagesRaw, ok := raw["error_pages"]; ok {
		parsed := ParseErrorPagesFromRaw(pagesRaw)
		if len(parsed) > 0 {
			cfg.ErrorPages = parsed
		}
	}
	if guardRaw, ok := raw["guard_pages"]; ok {
		if parsed := parseGuardPagesFromRaw(guardRaw); len(parsed) > 0 {
			cfg.GuardPages = parsed
		}
	}
	if len(cfg.ErrorPages) == 0 {
		legacyPages := loadLegacyErrorPagesStringMap()
		if len(legacyPages) > 0 {
			cfg.ErrorPages = map[string]models.ErrorPageDefinition{}
			for key, html := range legacyPages {
				cfg.ErrorPages[mapLegacyErrorPageKey(key)] = migrateLegacyErrorPageHTML(mapLegacyErrorPageKey(key), html)
			}
		}
	}
	return &cfg
}

func loadLegacyErrorPagesStringMap() map[string]string {
	var cfgItem models.ConfigItem
	if err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "error-page", "error_page", "global", 0).
		First(&cfgItem).Error; err == nil && cfgItem.Value != "" {
		var pages map[string]string
		if json.Unmarshal([]byte(cfgItem.Value), &pages) == nil && len(pages) > 0 {
			return normalizeLegacyErrorPagesStringMap(pages)
		}
	}
	return nil
}

func normalizeLegacyErrorPagesStringMap(pages map[string]string) map[string]string {
	if len(pages) == 0 {
		return pages
	}
	normalized := make(map[string]string)
	copyIfPresent := func(key string) {
		if val, ok := pages[key]; ok && val != "" {
			normalized[key] = val
		}
	}
	for _, key := range errorPageKeys {
		copyIfPresent(key)
	}
	for legacy, mapped := range legacyErrorPageAliases {
		if _, ok := normalized[mapped]; ok {
			continue
		}
		if val, ok := pages[legacy]; ok && val != "" {
			normalized[mapped] = val
		}
	}
	return normalized
}
