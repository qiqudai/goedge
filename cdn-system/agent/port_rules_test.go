package main

import "testing"

func TestFilterCustomPortsDisabled5000(t *testing.T) {
	filtered := filterCustomPorts([]string{"5000", "8080"}, "1-65535", "22 5000")
	if len(filtered) != 1 || filtered[0] != "8080" {
		t.Fatalf("filterCustomPorts = %#v, want [8080]", filtered)
	}
}

func TestIsPortAllowedPublicPolicy(t *testing.T) {
	if isPortAllowed(5000, "1-65535", "22 5000") {
		t.Fatal("port 5000 should be disabled")
	}
	if !isPortAllowed(8080, "1-65535", "22 5000") {
		t.Fatal("port 8080 should be allowed")
	}
}
