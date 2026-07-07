package main

import (
	"cdn-api/config"
	"testing"
)

func TestResolveAllowedCORSOriginUsesConfiguredAllowlist(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://cccadmin.665305.cc, https://goai.665305.cc/")
	t.Setenv("APP_ALLOWED_ORIGINS", "")

	if got := resolveAllowedCORSOrigin("https://cccadmin.665305.cc"); got != "https://cccadmin.665305.cc" {
		t.Fatalf("expected configured origin, got %q", got)
	}
	if got := resolveAllowedCORSOrigin("https://evil.example"); got != "" {
		t.Fatalf("unexpected origin allowed: %q", got)
	}
}

func TestResolveAllowedCORSOriginFallsBackToAppAllowlist(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("APP_ALLOWED_ORIGINS", "https://admin.example.com")

	if got := resolveAllowedCORSOrigin("https://admin.example.com"); got != "https://admin.example.com" {
		t.Fatalf("expected APP_ALLOWED_ORIGINS origin, got %q", got)
	}
}

func TestResolveAllowedCORSOriginDeniesWhenAllowlistMissing(t *testing.T) {
	oldAllowedOrigins := config.App.CORSAllowedOrigins
	config.App.CORSAllowedOrigins = ""
	t.Cleanup(func() { config.App.CORSAllowedOrigins = oldAllowedOrigins })
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("APP_ALLOWED_ORIGINS", "")

	if got := resolveAllowedCORSOrigin("https://cccadmin.665305.cc"); got != "" {
		t.Fatalf("expected origin to be denied without allowlist, got %q", got)
	}
}

func TestResolveAllowedCORSOriginUsesConfigAllowlist(t *testing.T) {
	oldAllowedOrigins := config.App.CORSAllowedOrigins
	config.App.CORSAllowedOrigins = "https://config-admin.example.com"
	t.Cleanup(func() { config.App.CORSAllowedOrigins = oldAllowedOrigins })
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("APP_ALLOWED_ORIGINS", "")

	if got := resolveAllowedCORSOrigin("https://config-admin.example.com"); got != "https://config-admin.example.com" {
		t.Fatalf("expected config origin, got %q", got)
	}
}
