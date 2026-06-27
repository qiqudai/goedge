package dns

import "testing"

func TestResolveLineValueDNSPodIntlDefault(t *testing.T) {
	got := ResolveLineValue("dnspod_intl", "default", "")
	if got != "Default" {
		t.Fatalf("ResolveLineValue(dnspod_intl, default) = %q, want Default", got)
	}
}

func TestResolveLineValueDNSPodIntlCustom(t *testing.T) {
	got := ResolveLineValue("dnspod_intl", "custom", "Default")
	if got != "Default" {
		t.Fatalf("ResolveLineValue(dnspod_intl, custom) = %q, want Default", got)
	}
}
