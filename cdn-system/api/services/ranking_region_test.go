package services

import (
	"strings"
	"testing"
)

func TestRegionRankingExprForType_Country(t *testing.T) {
	expr, ok := regionRankingExprForType("country")
	if !ok {
		t.Fatalf("expected country expr")
	}
	if !strings.Contains(expr, "client_country") {
		t.Fatalf("expected client_country in expr, got %q", expr)
	}
	if strings.Contains(expr, "remote_addr") {
		t.Fatalf("country ranking should no longer depend on remote_addr, got %q", expr)
	}
}

func TestRegionRankingExprForType_ProvinceFallsBackToCountry(t *testing.T) {
	expr, ok := regionRankingExprForType("province")
	if !ok {
		t.Fatalf("expected province expr")
	}
	if !strings.Contains(expr, "client_province") || !strings.Contains(expr, "client_country") {
		t.Fatalf("expected province expr to fall back to client_country, got %q", expr)
	}
}

func TestRegionRankingExprForType_Unknown(t *testing.T) {
	if _, ok := regionRankingExprForType("city"); ok {
		t.Fatalf("unexpected expr for unsupported region type")
	}
}
