package services

import "testing"

func TestSplitDNSZoneAndRecord(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantZone   string
		wantRecord string
	}{
		{name: "normal subdomain", input: "www.example.com", wantZone: "example.com", wantRecord: "www"},
		{name: "apex", input: "example.com", wantZone: "example.com", wantRecord: "@"},
		{name: "china public suffix", input: "www.example.com.cn", wantZone: "example.com.cn", wantRecord: "www"},
		{name: "uk public suffix", input: "a.b.foo.co.uk", wantZone: "foo.co.uk", wantRecord: "a.b"},
		{name: "wildcard", input: "*.example.com", wantZone: "example.com", wantRecord: "*"},
		{name: "url input", input: "https://cdn.example.com/path", wantZone: "example.com", wantRecord: "cdn"},
		{name: "ip rejected", input: "192.0.2.1", wantZone: "", wantRecord: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotZone, gotRecord := SplitDNSZoneAndRecord(tt.input)
			if gotZone != tt.wantZone || gotRecord != tt.wantRecord {
				t.Fatalf("SplitDNSZoneAndRecord(%q) = (%q, %q), want (%q, %q)", tt.input, gotZone, gotRecord, tt.wantZone, tt.wantRecord)
			}
		})
	}
}
