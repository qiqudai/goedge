package main

import (
	"encoding/json"
	"strings"
)

func parseEdgeConfigPayload(payload []byte) (edgeConfig, error) {
	var cfg edgeConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return cfg, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return cfg, err
	}
	if pagesRaw, ok := raw["error_pages"]; ok {
		cfg.ErrorPages = parseAgentErrorPagesRaw(pagesRaw)
	}
	if i18nRaw, ok := raw["error_page_i18n"]; ok {
		_ = json.Unmarshal(i18nRaw, &cfg.ErrorPageI18n)
	}
	cfg.ErrorPageI18n = normalizeAgentErrorPageI18n(cfg.ErrorPageI18n)
	return cfg, nil
}

func normalizeAgentErrorPageI18n(settings errorPageI18nSettings) errorPageI18nSettings {
	if strings.TrimSpace(settings.DefaultLang) == "" {
		settings.DefaultLang = "zh-CN"
	}
	settings.DefaultLang = normalizeAgentLocaleTag(settings.DefaultLang)
	mode := strings.ToLower(strings.TrimSpace(settings.LangMode))
	if mode == "fixed" {
		settings.LangMode = "fixed"
	} else {
		settings.LangMode = "browser"
	}
	langs := make([]string, 0, len(settings.EnabledLangs)+1)
	seen := map[string]struct{}{}
	add := func(lang string) {
		lang = normalizeAgentLocaleTag(lang)
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
		add(lang)
	}
	add(settings.DefaultLang)
	if len(langs) == 0 {
		langs = []string{"zh-CN", "en"}
		settings.DefaultLang = "zh-CN"
	}
	settings.EnabledLangs = langs
	return settings
}

func parseAgentErrorPagesRaw(raw json.RawMessage) map[string]errorPageDefinition {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var objectForm map[string]errorPageDefinition
	if err := json.Unmarshal(raw, &objectForm); err == nil && len(objectForm) > 0 {
		firstKey := ""
		for key := range objectForm {
			firstKey = key
			break
		}
		if firstKey != "" && strings.TrimSpace(objectForm[firstKey].Template) != "" {
			return objectForm
		}
	}
	var mixed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &mixed); err != nil {
		return nil
	}
	out := map[string]errorPageDefinition{}
	for key, itemRaw := range mixed {
		var asString string
		if err := json.Unmarshal(itemRaw, &asString); err == nil && strings.TrimSpace(asString) != "" {
			out[key] = errorPageDefinition{Template: asString}
			continue
		}
		var asDef errorPageDefinition
		if err := json.Unmarshal(itemRaw, &asDef); err == nil && strings.TrimSpace(asDef.Template) != "" {
			out[key] = asDef
		}
	}
	return out
}

func hasErrorPageDefinitions(pages map[string]errorPageDefinition) bool {
	for _, def := range pages {
		if strings.TrimSpace(def.Template) != "" {
			return true
		}
	}
	return false
}

func sortedAgentErrorPageKeys(pages map[string]errorPageDefinition) []string {
	if len(pages) == 0 {
		return nil
	}
	return sortedErrorPageKeys(pages)
}
