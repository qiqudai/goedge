package services

import (
	"encoding/json"
	"strings"
	"testing"

	"cdn-api/models"
)

func TestParseErrorPagesFromRawLegacyString(t *testing.T) {
	raw := json.RawMessage(`{"403":"<html><title>请求被禁止访问</title><body>test</body></html>"}`)
	pages := ParseErrorPagesFromRaw(raw)
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	def := pages["403"]
	if def.Template == "" {
		t.Fatalf("expected migrated template")
	}
	if def.Strings["zh-CN"]["title"] != "请求被禁止访问" {
		t.Fatalf("unexpected title: %q", def.Strings["zh-CN"]["title"])
	}
}

func TestRenderErrorPage(t *testing.T) {
	html := RenderErrorPage("<title>{{title}}</title>", map[string]string{"title": "Access Denied"})
	if html != "<title>Access Denied</title>" {
		t.Fatalf("unexpected render result: %s", html)
	}
}

func TestValidateErrorPageConfig(t *testing.T) {
	cfg := &models.GlobalConfig{
		ErrorPageI18n: DefaultErrorPageI18nSettings(),
		ErrorPages:    DefaultErrorPageDefinitions(),
	}
	if err := ValidateErrorPageConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestDefaultAccessBlockedPage(t *testing.T) {
	defs := DefaultErrorPageDefinitions()
	def, ok := defs["access_blocked"]
	if !ok {
		t.Fatal("missing access_blocked definition")
	}
	if !strings.Contains(def.Template, "{{heading}}") {
		t.Fatalf("template missing heading placeholder: %s", def.Template)
	}
	if errorPageStatusCode("access_blocked") != "419" {
		t.Fatalf("unexpected status code: %s", errorPageStatusCode("access_blocked"))
	}
	html := RenderErrorPage(def.Template, def.Strings["en"])
	if !strings.Contains(html, "Sorry, you have been blocked") {
		t.Fatalf("unexpected render: %s", html)
	}
}

func TestExtractErrorPageLang(t *testing.T) {
	settings := map[string]interface{}{
		"error_page_lang": "zh-CN",
	}
	if got := extractErrorPageLang(settings); got != "zh-CN" {
		t.Fatalf("expected zh-CN, got %q", got)
	}
	if got := extractErrorPageLang(map[string]interface{}{"error_page_lang": "browser"}); got != "browser" {
		t.Fatalf("expected browser, got %q", got)
	}
}
