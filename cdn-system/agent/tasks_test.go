package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestBuildIssueCAFallbackList_OnlyUsesAutomaticFallbacksThatCanRegisterWithoutEAB(t *testing.T) {
	cases := []struct {
		name    string
		primary string
		want    []string
	}{
		{name: "letsencrypt", primary: "letsencrypt", want: []string{"letsencrypt"}},
		{name: "zerossl", primary: "zerossl", want: []string{"zerossl", "letsencrypt"}},
		{name: "google", primary: "google", want: []string{"google", "letsencrypt"}},
		{name: "buypass legacy", primary: "buypass", want: []string{"buypass", "letsencrypt"}},
		{name: "unknown", primary: "unknown", want: []string{"letsencrypt"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildIssueCAFallbackList(tc.primary); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("fallback list = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNormalizeAgentPackageConfigMarksPastEndAtExpired(t *testing.T) {
	raw := []byte(`{"package_id":7,"version":58,"status":"active","time":{"end_at":"2026-05-28 07:15:12"}}`)
	now := time.Date(2026, 5, 28, 8, 0, 0, 0, time.Local)

	var parsed AgentPackageConfig
	normalized, changed, err := normalizeAgentPackageConfig(7, raw, &parsed, now)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if !changed {
		t.Fatalf("expected package config to change")
	}
	if parsed.Status != "expired" {
		t.Fatalf("parsed status = %q, want expired", parsed.Status)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(normalized, &out); err != nil {
		t.Fatalf("normalized json invalid: %v", err)
	}
	if out["status"] != "expired" {
		t.Fatalf("normalized status = %v, want expired", out["status"])
	}
}

func TestNormalizeAgentPackageConfigLeavesFutureEndAtActive(t *testing.T) {
	raw := []byte(`{"package_id":7,"version":58,"status":"active","time":{"end_at":"2026-05-28 09:15:12"}}`)
	now := time.Date(2026, 5, 28, 8, 0, 0, 0, time.Local)

	var parsed AgentPackageConfig
	_, changed, err := normalizeAgentPackageConfig(7, raw, &parsed, now)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if changed {
		t.Fatalf("future package should not be changed")
	}
	if parsed.Status != "active" {
		t.Fatalf("parsed status = %q, want active", parsed.Status)
	}
}
