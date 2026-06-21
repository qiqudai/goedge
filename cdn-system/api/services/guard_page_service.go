package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"cdn-api/models"
	"cdn-common/i18n"
)

var guardPageKeys = i18n.GuardPageKeys()

var antiCCTypeToPageKey = map[string]string{
	"slide":        "slide",
	"slide_simple": "slide",
	"captcha":      "captcha",
	"click":        "click",
	"click_simple": "click",
	"5s":           "delay_jump",
	"rotate":       "rotate",
}

var guardPageTemplateFiles = map[string]string{
	"click":      "click.html",
	"slide":      "slide.html",
	"captcha":    "captcha.html",
	"delay_jump": "delay_jump.html",
	"rotate":     "rotate.html",
}

func GuardPageKeyForAntiCCType(antiCCType string) string {
	key := strings.TrimSpace(antiCCType)
	if mapped, ok := antiCCTypeToPageKey[key]; ok {
		return mapped
	}
	return "click"
}

func DefaultGuardPageDefinitions() map[string]models.GuardPageDefinition {
	defaultStrings := i18n.GuardDefaultStrings()
	defaultTemplates := i18n.GuardDefaultTemplates()
	enabledLangs := DefaultErrorPageI18nSettings().EnabledLangs
	pages := make(map[string]models.GuardPageDefinition, len(guardPageKeys))
	for _, key := range guardPageKeys {
		pages[key] = models.GuardPageDefinition{
			Template: defaultTemplates[key],
			Strings:  fillMissingGuardPageStrings(key, nil, enabledLangs, defaultStrings),
		}
	}
	return pages
}

func fillMissingGuardPageStrings(
	pageKey string,
	strings map[string]map[string]string,
	langs []string,
	defaults map[string]map[string]map[string]string,
) map[string]map[string]string {
	next := map[string]map[string]string{}
	for lang, values := range strings {
		next[lang] = copyStringMap(values)
	}
	pageDefaults := defaults[pageKey]
	for _, lang := range langs {
		next[lang] = mergeMissingStringValues(next[lang], pageDefaults[lang])
	}
	for lang, values := range pageDefaults {
		next[lang] = mergeMissingStringValues(next[lang], values)
	}
	return next
}

func mergeMissingStringValues(target map[string]string, fallback map[string]string) map[string]string {
	out := copyStringMap(target)
	for key, value := range fallback {
		if strings.TrimSpace(out[key]) == "" {
			out[key] = value
		}
	}
	return out
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func NormalizeGlobalConfigGuardPages(cfg *models.GlobalConfig) {
	if cfg == nil {
		return
	}
	NormalizeGlobalConfigErrorPages(cfg)
	enabledLangs := cfg.ErrorPageI18n.EnabledLangs
	if len(enabledLangs) == 0 {
		enabledLangs = DefaultErrorPageI18nSettings().EnabledLangs
	}
	defaults := DefaultGuardPageDefinitions()
	if cfg.GuardPages == nil {
		cfg.GuardPages = map[string]models.GuardPageDefinition{}
	}
	for _, key := range guardPageKeys {
		existing := cfg.GuardPages[key]
		def := defaults[key]
		if strings.TrimSpace(existing.Template) == "" {
			existing.Template = def.Template
		}
		existing.Strings = fillMissingGuardPageStrings(key, existing.Strings, enabledLangs, i18n.GuardDefaultStrings())
		cfg.GuardPages[key] = existing
	}
	migrateLegacyAntiCCPageCustom(cfg)
}

func migrateLegacyAntiCCPageCustom(cfg *models.GlobalConfig) {
	custom := strings.TrimSpace(cfg.WAF.AntiCCPageCustom)
	if custom == "" {
		return
	}
	pageKey := GuardPageKeyForAntiCCType(cfg.WAF.AntiCCType)
	page := cfg.GuardPages[pageKey]
	if strings.TrimSpace(page.Template) != "" && page.Template != i18n.GuardDefaultTemplates()[pageKey] {
		return
	}
	page.Template = custom
	cfg.GuardPages[pageKey] = page
	cfg.WAF.AntiCCPageCustom = ""
}

func ValidateGuardPageConfig(cfg *models.GlobalConfig) error {
	if cfg == nil {
		return nil
	}
	NormalizeGlobalConfigGuardPages(cfg)
	for _, key := range guardPageKeys {
		page := cfg.GuardPages[key]
		if strings.TrimSpace(page.Template) == "" {
			return fmt.Errorf("guard page %s template is required", key)
		}
		keys := extractTemplateKeys(page.Template)
		if len(keys) == 0 {
			continue
		}
		defaultLang := cfg.ErrorPageI18n.DefaultLang
		if defaultLang == "" {
			defaultLang = "zh-CN"
		}
		stringsForLang := page.Strings[defaultLang]
		if stringsForLang == nil {
			stringsForLang = map[string]string{}
		}
		for _, placeholder := range keys {
			if placeholder == "html_lang" {
				continue
			}
			if strings.TrimSpace(stringsForLang[placeholder]) == "" {
				return fmt.Errorf("guard page %s missing default language string for {{%s}}", key, placeholder)
			}
		}
	}
	return nil
}

func GuardPageTemplateFileName(pageKey string) string {
	if file, ok := guardPageTemplateFiles[pageKey]; ok {
		return file
	}
	return pageKey + ".html"
}

func BuildGuardI18nPayload(pages map[string]models.GuardPageDefinition) map[string]interface{} {
	strings := map[string]map[string]map[string]string{}
	for _, key := range guardPageKeys {
		page := pages[key]
		if len(page.Strings) == 0 {
			continue
		}
		strings[key] = page.Strings
	}
	return map[string]interface{}{
		"strings": strings,
	}
}

func parseGuardPagesFromRaw(raw json.RawMessage) map[string]models.GuardPageDefinition {
	if len(raw) == 0 {
		return nil
	}
	var pages map[string]models.GuardPageDefinition
	if err := json.Unmarshal(raw, &pages); err != nil {
		return nil
	}
	return pages
}
