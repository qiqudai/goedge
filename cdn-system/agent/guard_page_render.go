package main

import (
	"path/filepath"

	fsutil "cdn-common/io"
)

type guardPageDefinition struct {
	Template string                       `json:"template"`
	Strings  map[string]map[string]string `json:"strings"`
}

var guardPageTemplateFiles = map[string]string{
	"click":      "click.html",
	"slide":      "slide.html",
	"captcha":    "captcha.html",
	"delay_jump": "delay_jump.html",
	"rotate":     "rotate.html",
}

var guardPageKeys = []string{"click", "slide", "captcha", "delay_jump", "rotate"}

func persistGuardPages(pages map[string]guardPageDefinition) error {
	if len(pages) == 0 {
		return nil
	}
	rootDir := runtimeRoot()
	guardDir := filepath.Join(rootDir, "conf", "guard")
	if err := fsutil.EnsureDir(guardDir); err != nil {
		return err
	}
	for _, key := range guardPageKeys {
		page, ok := pages[key]
		if !ok {
			continue
		}
		template := page.Template
		if template == "" {
			continue
		}
		fileName, ok := guardPageTemplateFiles[key]
		if !ok {
			fileName = key + ".html"
		}
		path := filepath.Join(guardDir, fileName)
		if err := fsutil.WriteFileAtomic(path, []byte(template), 0o644); err != nil {
			return err
		}
	}
	stringsRoot := map[string]map[string]map[string]string{}
	for _, key := range guardPageKeys {
		page, ok := pages[key]
		if !ok || len(page.Strings) == 0 {
			continue
		}
		stringsRoot[key] = page.Strings
	}
	i18nPath := filepath.Join(rootDir, "conf", "guard_i18n.json")
	return fsutil.WriteJSONAtomic(i18nPath, map[string]interface{}{
		"strings": stringsRoot,
	}, true)
}
