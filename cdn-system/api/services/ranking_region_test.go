package services

import (
	"errors"
	"testing"
)

func TestBuildRegionRankingFromIP_Country(t *testing.T) {
	items := []RankItem{
		{Item: "1.1.1.1", RequestCount: 10, OutBytes: 100, OriginBytes: 50},
		{Item: "1.1.1.2", RequestCount: 5, OutBytes: 60, OriginBytes: 20},
		{Item: "2.2.2.2", RequestCount: 7, OutBytes: 70, OriginBytes: 30},
	}
	lookup := func(ip string) (string, string) {
		switch ip {
		case "1.1.1.1", "1.1.1.2":
			return "CN", "Guangdong"
		case "2.2.2.2":
			return "US", "California"
		default:
			return "", ""
		}
	}
	list := buildRegionRankingFromIP(items, "country", "", 10, lookup)
	if len(list) != 2 {
		t.Fatalf("expected 2 country items, got %d", len(list))
	}
	if list[0].Item != "CN" || list[0].RequestCount != 15 {
		t.Fatalf("unexpected top country: %+v", list[0])
	}
	if list[1].Item != "US" || list[1].RequestCount != 7 {
		t.Fatalf("unexpected second country: %+v", list[1])
	}
}

func TestBuildRegionRankingFromIP_ProvinceFallbackCountry(t *testing.T) {
	items := []RankItem{
		{Item: "3.3.3.3", RequestCount: 4, OutBytes: 40, OriginBytes: 10},
	}
	lookup := func(ip string) (string, string) {
		return "JP", ""
	}
	list := buildRegionRankingFromIP(items, "province", "", 10, lookup)
	if len(list) != 1 {
		t.Fatalf("expected 1 province item, got %d", len(list))
	}
	if list[0].Item != "JP" {
		t.Fatalf("expected province fallback to country JP, got %q", list[0].Item)
	}
}

func TestBuildRegionRankingFromIP_KeywordAndLimit(t *testing.T) {
	items := []RankItem{
		{Item: "1.1.1.1", RequestCount: 10, OutBytes: 100},
		{Item: "2.2.2.2", RequestCount: 9, OutBytes: 90},
		{Item: "3.3.3.3", RequestCount: 8, OutBytes: 80},
	}
	lookup := func(ip string) (string, string) {
		switch ip {
		case "1.1.1.1":
			return "CN", "Beijing"
		case "2.2.2.2":
			return "US", "New York"
		default:
			return "JP", "Tokyo"
		}
	}
	filtered := buildRegionRankingFromIP(items, "country", "U", 10, lookup)
	if len(filtered) != 1 || filtered[0].Item != "US" {
		t.Fatalf("unexpected keyword filtered list: %+v", filtered)
	}
	limited := buildRegionRankingFromIP(items, "country", "", 2, lookup)
	if len(limited) != 2 {
		t.Fatalf("expected limit=2, got %d", len(limited))
	}
}

func TestResolveRegionIPSampleLimit(t *testing.T) {
	if v := resolveRegionIPSampleLimit(0); v != 5000 {
		t.Fatalf("expected default 5000, got %d", v)
	}
	if v := resolveRegionIPSampleLimit(1); v != 5000 {
		t.Fatalf("expected min clamp 5000, got %d", v)
	}
	if v := resolveRegionIPSampleLimit(1000); v != 50000 {
		t.Fatalf("expected max clamp 50000, got %d", v)
	}
}

func TestIsUnknownIdentifierError(t *testing.T) {
	if !isUnknownIdentifierError(errors.New("Code: 47, Unknown identifier: remote_addr")) {
		t.Fatalf("expected true for unknown identifier")
	}
	if isUnknownIdentifierError(errors.New("timeout exceeded")) {
		t.Fatalf("expected false for unrelated error")
	}
}
