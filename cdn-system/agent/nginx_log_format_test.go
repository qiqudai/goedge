package main

import (
	"os"
	"strings"
	"testing"
)

func TestNginxAccessLogContainsSlowDiagnosisTimingFields(t *testing.T) {
	data, err := os.ReadFile("assets/conf/nginx.conf")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"upstream_connect_time",
		"upstream_header_time",
		"upstream_response_time",
		"upstream_cache_status",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("nginx access log missing %q", want)
		}
	}
}
