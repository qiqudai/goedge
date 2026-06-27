package services

import (
	"cdn-api/models"
	"testing"
)

func TestIsValidIPBlacklistEntry(t *testing.T) {
	tests := []struct {
		entry string
		want  bool
	}{
		{entry: "1.2.3.4", want: true},
		{entry: "127.0.0.1", want: true},
		{entry: "127.*.*.*", want: true},
		{entry: "192.168.*", want: true},
		{entry: "10.0.0.0/24", want: true},
		{entry: "2001:db8::1", want: true},
		{entry: "", want: false},
		{entry: "abc", want: false},
		{entry: "999.*.*.*", want: false},
		{entry: "127.*.1", want: true},
	}
	for _, tt := range tests {
		if got := IsValidIPBlacklistEntry(tt.entry); got != tt.want {
			t.Fatalf("IsValidIPBlacklistEntry(%q) = %v want %v", tt.entry, got, tt.want)
		}
	}
}

func TestParseIPBlacklistLines(t *testing.T) {
	got := ParseIPBlacklistLines("1.2.3.4\n127.*.*.*\n\n1.2.3.4\n10.0.0.0/24")
	want := []string{"1.2.3.4", "127.*.*.*", "10.0.0.0/24"}
	if len(got) != len(want) {
		t.Fatalf("ParseIPBlacklistLines = %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %q want %q", i, got[i], want[i])
		}
	}
}

func TestAnalyzeIPListCountsExactAndPatternEntries(t *testing.T) {
	got := AnalyzeIPList("1.2.3.4\n2001:db8::1\n10.0.0.0/24\n127.*.*.*\ninvalid", 10)
	if got.Total != 5 {
		t.Fatalf("Total = %d, want 5", got.Total)
	}
	if got.Exact != 2 {
		t.Fatalf("Exact = %d, want 2", got.Exact)
	}
	if got.Pattern != 2 {
		t.Fatalf("Pattern = %d, want 2", got.Pattern)
	}
	if got.Invalid != 1 {
		t.Fatalf("Invalid = %d, want 1", got.Invalid)
	}
}

func TestValidateWAFIPListsRejectsInvalidEntries(t *testing.T) {
	err := ValidateWAFIPLists(models.WAFConfig{BlacklistIPs: "1.2.3.4\nnot-an-ip"}, 10)
	if err == nil {
		t.Fatal("expected invalid WAF blacklist entry error")
	}
}

func TestValidateWAFIPListsRejectsPatternLimit(t *testing.T) {
	err := ValidateWAFIPLists(models.WAFConfig{BlacklistIPs: "10.0.0.0/24\n127.*.*.*"}, 1)
	if err == nil {
		t.Fatal("expected WAF pattern limit error")
	}
}

func TestValidateWAFIPListsAllowsManyExactIPs(t *testing.T) {
	err := ValidateWAFIPLists(models.WAFConfig{BlacklistIPs: "1.1.1.1\n2.2.2.2\n3.3.3.3"}, 1)
	if err != nil {
		t.Fatalf("ValidateWAFIPLists exact IPs error = %v", err)
	}
}
