package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type errorPageI18nSettings struct {
	DefaultLang  string   `json:"default_lang"`
	LangMode     string   `json:"lang_mode"`
	EnabledLangs []string `json:"enabled_langs"`
}

type errorPageDefinition struct {
	Template string                       `json:"template"`
	Strings  map[string]map[string]string `json:"strings"`
}

type errorPageBundle struct {
	I18n  errorPageI18nSettings           `json:"error_page_i18n"`
	Pages map[string]errorPageDefinition  `json:"error_pages"`
}

var agentPlaceholderPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

func normalizeAgentLocaleTag(lang string) string {
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

func renderAgentErrorPage(template string, localeStrings map[string]string) string {
	if strings.TrimSpace(template) == "" {
		return ""
	}
	return agentPlaceholderPattern.ReplaceAllStringFunc(template, func(token string) string {
		match := agentPlaceholderPattern.FindStringSubmatch(token)
		if len(match) < 2 {
			return token
		}
		if val, ok := localeStrings[match[1]]; ok {
			return val
		}
		return token
	})
}

func resolveAgentErrorPageStrings(def errorPageDefinition, lang, defaultLang string, enabledLangs []string) map[string]string {
	lang = normalizeAgentLocaleTag(lang)
	defaultLang = normalizeAgentLocaleTag(defaultLang)
	candidates := []string{lang}
	if lang != "" {
		if base := strings.Split(lang, "-")[0]; base != lang {
			candidates = append(candidates, base)
		}
	}
	candidates = append(candidates, defaultLang)
	for _, enabled := range enabledLangs {
		candidates = append(candidates, normalizeAgentLocaleTag(enabled))
	}
	for _, candidate := range candidates {
		if candidate == "" || def.Strings == nil {
			continue
		}
		if values, ok := def.Strings[candidate]; ok && len(values) > 0 {
			return values
		}
		if base := strings.Split(candidate, "-")[0]; base != candidate {
			if values, ok := def.Strings[base]; ok && len(values) > 0 {
				return values
			}
		}
	}
	for _, values := range def.Strings {
		if len(values) > 0 {
			return values
		}
	}
	return map[string]string{}
}

func renderAllAgentErrorPages(pages map[string]errorPageDefinition, i18n errorPageI18nSettings) map[string]map[string]string {
	out := map[string]map[string]string{}
	if len(pages) == 0 {
		return out
	}
	defaultLang := normalizeAgentLocaleTag(i18n.DefaultLang)
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}
	langs := make([]string, 0, len(i18n.EnabledLangs))
	seen := map[string]struct{}{}
	for _, lang := range i18n.EnabledLangs {
		lang = normalizeAgentLocaleTag(lang)
		if lang == "" {
			continue
		}
		if _, ok := seen[lang]; ok {
			continue
		}
		seen[lang] = struct{}{}
		langs = append(langs, lang)
	}
	if len(langs) == 0 {
		langs = []string{defaultLang}
	}
	keys := sortedErrorPageKeys(pages)
	for _, code := range keys {
		def := pages[code]
		out[code] = map[string]string{}
		for _, lang := range langs {
			values := resolveAgentErrorPageStrings(def, lang, defaultLang, langs)
			html := renderAgentErrorPage(def.Template, values)
			if strings.TrimSpace(html) != "" {
				out[code][lang] = html
			}
		}
	}
	return out
}

func sortedErrorPageKeys(pages map[string]errorPageDefinition) []string {
	keys := make([]string, 0, len(pages))
	for key := range pages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeRenderedErrorPageFiles(dir string, rendered map[string]map[string]string) error {
	if len(rendered) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Remove legacy flat html files in root dir.
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".html") {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	for code, byLang := range rendered {
		for lang, html := range byLang {
			langDir := filepath.Join(dir, lang)
			if err := os.MkdirAll(langDir, 0o755); err != nil {
				return err
			}
			target := filepath.Join(langDir, code+".html")
			if err := os.WriteFile(target, []byte(html), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
