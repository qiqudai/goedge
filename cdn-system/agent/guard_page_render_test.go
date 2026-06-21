package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepareGuardPageTestEnv(t *testing.T) string {
	t.Helper()
	prevWorkDir := WorkDir
	WorkDir = t.TempDir()
	t.Cleanup(func() {
		WorkDir = prevWorkDir
	})
	return runtimeRoot()
}

func TestPersistGuardPagesWritesTemplateAndI18n(t *testing.T) {
	root := prepareGuardPageTestEnv(t)
	pages := map[string]guardPageDefinition{
		"slide": {
			Template: "<html>{{slide_hint}}</html>",
			Strings: map[string]map[string]string{
				"zh-CN": {"slide_hint": "向右滑动验证"},
				"en":    {"slide_hint": "Slide to verify"},
			},
		},
	}
	if err := persistGuardPages(pages); err != nil {
		t.Fatalf("persistGuardPages: %v", err)
	}

	slidePath := filepath.Join(root, "conf", "guard", "slide.html")
	slideData, err := os.ReadFile(slidePath)
	if err != nil {
		t.Fatalf("read slide template: %v", err)
	}
	if string(slideData) != pages["slide"].Template {
		t.Fatalf("unexpected slide template: %q", string(slideData))
	}

	i18nPath := filepath.Join(root, "conf", "guard_i18n.json")
	raw, err := os.ReadFile(i18nPath)
	if err != nil {
		t.Fatalf("read guard_i18n.json: %v", err)
	}
	var payload struct {
		Strings map[string]map[string]map[string]string `json:"strings"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode guard_i18n.json: %v", err)
	}
	if payload.Strings["slide"]["en"]["slide_hint"] != "Slide to verify" {
		t.Fatalf("unexpected guard i18n payload: %+v", payload.Strings["slide"])
	}
}

func TestPersistGuardPagesUsesPlaceholderTemplate(t *testing.T) {
	pages := map[string]guardPageDefinition{
		"click": {
			Template: "<title>{{page_title}}</title><p>{{traffic_notice}}</p>",
			Strings: map[string]map[string]string{
				"zh-CN": {
					"page_title":     "CC LOCK",
					"traffic_notice": "网站当前访问量较大",
				},
			},
		},
	}
	template := pages["click"].Template
	if !strings.Contains(template, "{{page_title}}") || !strings.Contains(template, "{{traffic_notice}}") {
		t.Fatalf("expected placeholder template, got %q", template)
	}
}
