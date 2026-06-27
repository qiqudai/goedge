package services

import (
	"cdn-api/models"
	"testing"
)

func TestMergeWebsiteResourcesDefaults(t *testing.T) {
	got := mergeWebsiteResources(models.WebsiteResourceConfig{})
	if got.MaxBlacklistIPs != 50 {
		t.Fatalf("MaxBlacklistIPs = %d, want 50", got.MaxBlacklistIPs)
	}
	if got.MaxWAFPatternIPs != 100 {
		t.Fatalf("MaxWAFPatternIPs = %d, want 100", got.MaxWAFPatternIPs)
	}
	if got.PreloadTimeout != 120 {
		t.Fatalf("PreloadTimeout = %d, want 120", got.PreloadTimeout)
	}
	if got.MaxDomainsPerSite != 100 {
		t.Fatalf("MaxDomainsPerSite = %d, want 100", got.MaxDomainsPerSite)
	}
}

func TestDefaultPublicResourcesDisabledPorts(t *testing.T) {
	pub := defaultPublicResources()
	if pub.DisabledCustomPorts != "22 5000" {
		t.Fatalf("DisabledCustomPorts = %q, want %q", pub.DisabledCustomPorts, "22 5000")
	}
}

func TestTrimSiteIPList(t *testing.T) {
	items := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	trimmed := TrimSiteIPList("blacklist", items)
	if len(trimmed) != 3 {
		t.Fatalf("expected 3 items with default limit 50, got %d", len(trimmed))
	}
}

func TestNormalizeDomainListDedup(t *testing.T) {
	got := normalizeDomainList([]string{"A.EXAMPLE.com", "a.example.com", "b.example.com"})
	if len(got) != 2 {
		t.Fatalf("normalizeDomainList len = %d, want 2", len(got))
	}
}
