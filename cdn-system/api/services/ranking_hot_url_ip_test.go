package services

import (
	"strings"
	"testing"
)

func TestHotURLPairCondition_DedupesSiteURIPairs(t *testing.T) {
	condition, args := hotURLPairCondition("site_expr", "uri_expr", []HotURLIPItem{
		{Site: "example.com", URI: "/a"},
		{Site: "example.com", URI: "/a"},
		{Site: "example.com", URI: "/b"},
		{Site: "", URI: "/ignored"},
	})
	if strings.Count(condition, "site_expr = ? AND uri_expr = ?") != 2 {
		t.Fatalf("expected two pair predicates, got %q", condition)
	}
	if len(args) != 4 {
		t.Fatalf("expected four args, got %d: %#v", len(args), args)
	}
	if args[0] != "example.com" || args[1] != "/a" || args[2] != "example.com" || args[3] != "/b" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
