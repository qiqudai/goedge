package services

import (
	"strings"
	"testing"

	"cdn-api/models"
)

func TestNormalizeGuardPagesUsesTemplatePlaceholders(t *testing.T) {
	cfg := &models.GlobalConfig{
		ErrorPageI18n: DefaultErrorPageI18nSettings(),
		WAF: models.WAFConfig{
			AntiCCType: "slide",
		},
	}
	NormalizeGlobalConfigGuardPages(cfg)
	page := cfg.GuardPages["slide"]
	if !strings.Contains(page.Template, "{{slide_hint}}") {
		t.Fatalf("expected slide template to contain {{slide_hint}}")
	}
	if page.Strings["zh-CN"]["slide_hint"] == "" {
		t.Fatalf("expected zh-CN slide_hint preset string")
	}
	if page.Strings["en"]["slide_hint"] == "" {
		t.Fatalf("expected en slide_hint preset string")
	}
}

func TestValidateGuardPageConfig(t *testing.T) {
	cfg := &models.GlobalConfig{
		ErrorPageI18n: DefaultErrorPageI18nSettings(),
	}
	NormalizeGlobalConfigGuardPages(cfg)
	if err := ValidateGuardPageConfig(cfg); err != nil {
		t.Fatalf("expected valid guard pages, got %v", err)
	}
}

func TestMigrateLegacyAntiCCPageCustom(t *testing.T) {
	cfg := &models.GlobalConfig{
		ErrorPageI18n: DefaultErrorPageI18nSettings(),
		WAF: models.WAFConfig{
			AntiCCType:       "click",
			AntiCCPageCustom: "<html>{{enter_site}}</html>",
		},
	}
	NormalizeGlobalConfigGuardPages(cfg)
	if cfg.GuardPages["click"].Template != "<html>{{enter_site}}</html>" {
		t.Fatalf("expected legacy custom page migrated to guard_pages.click")
	}
	if cfg.WAF.AntiCCPageCustom != "" {
		t.Fatalf("expected legacy anti_cc_page_custom cleared after migration")
	}
}

func TestGuardPageKeyForAntiCCType(t *testing.T) {
	cases := map[string]string{
		"5s":           "delay_jump",
		"slide_simple": "slide",
		"click_simple": "click",
		"unknown":      "click",
	}
	for input, want := range cases {
		if got := GuardPageKeyForAntiCCType(input); got != want {
			t.Fatalf("GuardPageKeyForAntiCCType(%q) = %q, want %q", input, got, want)
		}
	}
}
