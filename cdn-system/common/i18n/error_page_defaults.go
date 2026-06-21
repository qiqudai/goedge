package i18n

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed error_page_defaults.json
var errorPageDefaultsJSON []byte

var (
	errorPageDefaultsOnce sync.Once
	errorPageLocales      []string
	errorPageDefaultMap   map[string]map[string]map[string]string
	errorPageDefaultsErr  error
)

type errorPageDefaultsFile struct {
	Locales []string                                  `json:"locales"`
	Strings map[string]map[string]map[string]string `json:"strings"`
}

func loadErrorPageDefaults() {
	errorPageDefaultsOnce.Do(func() {
		var file errorPageDefaultsFile
		if err := json.Unmarshal(errorPageDefaultsJSON, &file); err != nil {
			errorPageDefaultsErr = err
			return
		}
		errorPageLocales = append([]string(nil), file.Locales...)
		errorPageDefaultMap = file.Strings
	})
}

// ErrorPageDefaultLocales returns preset BCP47 locales for error pages.
func ErrorPageDefaultLocales() []string {
	loadErrorPageDefaults()
	if errorPageDefaultsErr != nil || len(errorPageLocales) == 0 {
		return []string{"zh-CN", "en"}
	}
	out := make([]string, len(errorPageLocales))
	copy(out, errorPageLocales)
	return out
}

// ErrorPageDefaultStrings returns preset strings keyed by page code and locale.
func ErrorPageDefaultStrings() map[string]map[string]map[string]string {
	loadErrorPageDefaults()
	if errorPageDefaultsErr != nil || errorPageDefaultMap == nil {
		return map[string]map[string]map[string]string{}
	}
	return errorPageDefaultMap
}
