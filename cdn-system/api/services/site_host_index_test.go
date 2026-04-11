package services

import (
	"strings"
	"testing"
)

func TestHostFilterSQLCondition_UsesCanonicalSiteExpr(t *testing.T) {
	filter := HostFilter{
		Exact:     []string{"api.example.com"},
		Wildcards: []string{"example.org"},
	}

	cond, args := filter.SQLCondition()
	if !strings.Contains(cond, "if(site_name !=") {
		t.Fatalf("expected canonical site expression in SQL condition, got %q", cond)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d (%v)", len(args), args)
	}
	if args[0] != "api.example.com" || args[1] != "%example.org" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestHostFilterHTTPCondition_UsesCanonicalSiteExpr(t *testing.T) {
	filter := HostFilter{
		Exact: []string{"api.example.com"},
	}

	cond := filter.HTTPCondition()
	if !strings.Contains(cond, "if(site_name !=") {
		t.Fatalf("expected canonical site expression in HTTP condition, got %q", cond)
	}
	if !strings.Contains(cond, "'api.example.com'") {
		t.Fatalf("expected quoted exact host in HTTP condition, got %q", cond)
	}
}
