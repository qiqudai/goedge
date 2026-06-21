package i18n

import (
	"embed"
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed guard_default_strings.json
var guardDefaultsJSON []byte

//go:embed guard_templates/*.html
var guardTemplateFS embed.FS

var (
	guardDefaultsOnce  sync.Once
	guardDefaultMap    map[string]map[string]map[string]string
	guardTemplateMap   map[string]string
	guardDefaultsErr   error
	guardPageKeys      = []string{"click", "slide", "captcha", "delay_jump", "rotate"}
	guardTemplateFiles = map[string]string{
		"click":      "guard_templates/click.html",
		"slide":      "guard_templates/slide.html",
		"captcha":    "guard_templates/captcha.html",
		"delay_jump": "guard_templates/delay_jump.html",
		"rotate":     "guard_templates/rotate.html",
	}
)

type guardDefaultsFile struct {
	Locales []string                                `json:"locales"`
	Strings map[string]map[string]map[string]string `json:"strings"`
}

func loadGuardDefaults() {
	guardDefaultsOnce.Do(func() {
		var file guardDefaultsFile
		if err := json.Unmarshal(guardDefaultsJSON, &file); err != nil {
			guardDefaultsErr = err
			return
		}
		guardDefaultMap = file.Strings
		guardTemplateMap = map[string]string{}
		for key, path := range guardTemplateFiles {
			data, err := guardTemplateFS.ReadFile(path)
			if err != nil {
				guardDefaultsErr = err
				return
			}
			guardTemplateMap[key] = string(data)
		}
	})
}

// GuardPageKeys returns supported CC guard page keys stored in global config.
func GuardPageKeys() []string {
	out := make([]string, len(guardPageKeys))
	copy(out, guardPageKeys)
	return out
}

// GuardDefaultStrings returns preset CC guard strings keyed by page type and locale.
func GuardDefaultStrings() map[string]map[string]map[string]string {
	loadGuardDefaults()
	if guardDefaultsErr != nil || guardDefaultMap == nil {
		return map[string]map[string]map[string]string{}
	}
	return guardDefaultMap
}

// GuardDefaultTemplates returns preset CC guard HTML templates with {{placeholder}} tokens.
func GuardDefaultTemplates() map[string]string {
	loadGuardDefaults()
	if guardDefaultsErr != nil || guardTemplateMap == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(guardTemplateMap))
	for key, tpl := range guardTemplateMap {
		out[key] = tpl
	}
	return out
}
