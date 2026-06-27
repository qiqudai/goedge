package controllers

import (
	"cdn-api/services"
	"testing"
)

func TestBlockBatchEntryValidation(t *testing.T) {
	valid := []string{
		"1.2.3.4",
		"127.*.*.*",
		"10.0.0.0/24",
		"192.168.*",
	}
	for _, entry := range valid {
		if !services.IsValidIPBlacklistEntry(entry) {
			t.Fatalf("expected valid entry: %q", entry)
		}
	}
	invalid := []string{"", "abc", "999.*.*.*", "1.2.3.4.5"}
	for _, entry := range invalid {
		if services.IsValidIPBlacklistEntry(entry) {
			t.Fatalf("expected invalid entry: %q", entry)
		}
	}
}

func TestBlockBatchLineParsing(t *testing.T) {
	lines := services.ParseIPBlacklistLines("1.2.3.4\n127.*.*.*\n\n10.0.0.0/24\n1.2.3.4")
	if len(lines) != 3 {
		t.Fatalf("expected 3 unique lines, got %#v", lines)
	}
}
