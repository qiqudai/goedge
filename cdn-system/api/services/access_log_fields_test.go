package services

import (
	"strings"
	"testing"
)

func TestAccessLogSiteExpr_FallsBackToHostForDefaultServer(t *testing.T) {
	expr := AccessLogSiteExpr()
	if !strings.Contains(expr, "lower(site_name) NOT LIKE '_:%'") {
		t.Fatalf("expected default server names to fall back to host, got %q", expr)
	}
	if !strings.Contains(expr, "replaceRegexpAll") {
		t.Fatalf("expected site expr to normalize host/site values, got %q", expr)
	}
}

func TestAccessLogRealSiteTrafficCondition_AllowsDefaultServerHostFallback(t *testing.T) {
	cond := AccessLogRealSiteTrafficCondition()
	if !strings.Contains(cond, "match(") {
		t.Fatalf("expected IP regex filtering in condition, got %q", cond)
	}
	if !strings.Contains(cond, "NOT IN ('127.0.0.1', 'localhost')") {
		t.Fatalf("expected localhost hosts to stay excluded, got %q", cond)
	}
	if strings.Contains(cond, "NOT LIKE '%.%.%.%'") {
		t.Fatalf("expected 3-dot hostname filter to be removed, got %q", cond)
	}
}

func TestAccessLogNormalizeHostExpr_StripsPort(t *testing.T) {
	expr := AccessLogNormalizeHostExpr("site_name")
	if !strings.Contains(expr, ":[0-9]+$") {
		t.Fatalf("expected host normalization to strip ports, got %q", expr)
	}
}

func TestAccessLogRefererExpr_GroupsByHost(t *testing.T) {
	expr := AccessLogRefererExpr()
	if !strings.Contains(expr, "extract(http_referer") {
		t.Fatalf("expected referer host extraction in expression, got %q", expr)
	}
	if !strings.Contains(expr, "http_referer = '-'") {
		t.Fatalf("expected dash fallback for empty referer, got %q", expr)
	}
}
