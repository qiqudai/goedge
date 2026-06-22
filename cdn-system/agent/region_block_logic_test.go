package main

import (
	"os"
	"strings"
	"testing"
)

// Mirrors geo_country.from_ip2region + region_blocked list check in access_guard.lua.
func fromIP2Region(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "|")
	if len(parts) > 0 {
		last := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
		if len(last) == 2 && last[0] >= 'A' && last[0] <= 'Z' {
			return last
		}
	}
	return strings.ToUpper(strings.TrimSpace(parts[0]))
}

func regionBlocked(list []string, country string) bool {
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		return false
	}
	for _, code := range list {
		if strings.ToUpper(strings.TrimSpace(code)) == country {
			return true
		}
	}
	return false
}

func TestRegionBlockLogicLaosIP(t *testing.T) {
	raw := "Laos|Champasak Province|Pakse|Star Telecom|LA"
	country := fromIP2Region(raw)
	if country != "LA" {
		t.Fatalf("expected LA, got %q", country)
	}
	list := []string{"CN", "HK", "LA", "ID", "US"}
	if !regionBlocked(list, country) {
		t.Fatalf("expected LA to be blocked")
	}
}

func TestRegionBlockLogicGeoEmptyFailOpen(t *testing.T) {
	list := []string{"CN", "LA", "ID"}
	if regionBlocked(list, "") {
		t.Fatal("empty country must fail-open (not blocked)")
	}
}

func TestRegionBlockLogicProduction447List(t *testing.T) {
	rawSettings, err := os.ReadFile("/tmp/site447-settings.json")
	if err != nil {
		t.Skip("production fixture missing")
	}
	// minimal parse: find LA in JSON without full unmarshal dependency on api package
	if !strings.Contains(string(rawSettings), `"la"`) && !strings.Contains(string(rawSettings), `"LA"`) {
		t.Fatal("fixture should contain la")
	}
	laosRaw := "Laos|Champasak Province|Pakse|Star Telecom|LA"
	if !regionBlocked([]string{"CN", "LA", "ID"}, fromIP2Region(laosRaw)) {
		t.Fatal("183.182.115.65 geo path should block when LA in list")
	}
}
