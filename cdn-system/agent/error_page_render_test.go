package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAgentErrorPage(t *testing.T) {
	html := renderAgentErrorPage("<h1>{{title}}</h1>", map[string]string{"title": "Forbidden"})
	if html != "<h1>Forbidden</h1>" {
		t.Fatalf("unexpected html: %s", html)
	}
}

func TestRenderAllAgentErrorPagesFallback(t *testing.T) {
	pages := map[string]errorPageDefinition{
		"403": {
			Template: "<title>{{title}}</title>",
			Strings: map[string]map[string]string{
				"zh-CN": {"title": "禁止访问"},
			},
		},
	}
	i18n := errorPageI18nSettings{
		DefaultLang:  "zh-CN",
		LangMode:     "browser",
		EnabledLangs: []string{"zh-CN", "en"},
	}
	rendered := renderAllAgentErrorPages(pages, i18n)
	if rendered["403"]["zh-CN"] != "<title>禁止访问</title>" {
		t.Fatalf("unexpected zh-CN render: %s", rendered["403"]["zh-CN"])
	}
	if rendered["403"]["en"] != "<title>禁止访问</title>" {
		t.Fatalf("expected en fallback to zh-CN, got %s", rendered["403"]["en"])
	}
}

func TestWriteErrorPageDirectivesUsesLua(t *testing.T) {
	ctx := errorPageContext{
		pages: map[string]errorPageDefinition{
			"403": {Template: "<html>{{title}}</html>", Strings: map[string]map[string]string{"zh-CN": {"title": "x"}}},
		},
		i18n: normalizeAgentErrorPageI18n(errorPageI18nSettings{DefaultLang: "zh-CN", EnabledLangs: []string{"zh-CN"}}),
	}
	var b strings.Builder
	writeErrorPageDirectives(&b, ctx)
	out := b.String()
	if !strings.Contains(out, "error_page 403 /__cdn_error/403.html;") {
		t.Fatalf("missing error_page directive: %s", out)
	}
	if !strings.Contains(out, "lua.error_page_serve") {
		t.Fatalf("expected lua handler: %s", out)
	}
}

func TestWriteRenderedErrorPageFiles(t *testing.T) {
	dir := t.TempDir()
	rendered := map[string]map[string]string{
		"403": {
			"zh-CN": "<html>zh</html>",
			"en":    "<html>en</html>",
		},
	}
	if err := writeRenderedErrorPageFiles(dir, rendered); err != nil {
		t.Fatalf("write files failed: %v", err)
	}
	zhPath := filepath.Join(dir, "zh-CN", "403.html")
	data, err := os.ReadFile(zhPath)
	if err != nil {
		t.Fatalf("read zh file failed: %v", err)
	}
	if string(data) != "<html>zh</html>" {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}
