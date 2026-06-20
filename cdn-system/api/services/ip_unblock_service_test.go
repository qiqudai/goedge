package services

import "testing"

func TestNormalizeUnblockIPs(t *testing.T) {
	got := normalizeUnblockIPs([]string{" 1.1.1.1 ", "1.1.1.1", "", "8.8.8.8"})
	if len(got) != 2 || got[0] != "1.1.1.1" || got[1] != "8.8.8.8" {
		t.Fatalf("normalizeUnblockIPs = %#v", got)
	}
}

func TestExtractGuardTTLsDefaults(t *testing.T) {
	pass, block := extractGuardTTLs(map[string]interface{}{
		"security": map[string]interface{}{},
	})
	if pass != 21600 || block != 3600 {
		t.Fatalf("defaults = (%d, %d)", pass, block)
	}
}

func TestExtractGuardTTLsCustom(t *testing.T) {
	pass, block := extractGuardTTLs(map[string]interface{}{
		"security": map[string]interface{}{
			"ip_white_timeout": 12,
			"ip_black_timeout": 12,
		},
	})
	if pass != 12 || block != 12 {
		t.Fatalf("custom = (%d, %d)", pass, block)
	}
}
