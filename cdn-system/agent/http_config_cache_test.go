package main

import (
	"strings"
	"testing"
)

func collectLocationTokens(conf string) []string {
	lines := strings.Split(conf, "\n")
	out := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "location ") || !strings.HasSuffix(trimmed, "{") {
			continue
		}
		token := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "location "), "{"))
		if token == "" {
			continue
		}
		out = append(out, token)
	}
	return out
}

func countToken(tokens []string, token string) int {
	count := 0
	for _, item := range tokens {
		if item == token {
			count++
		}
	}
	return count
}

func TestWriteCacheLocations_DedupAllPath(t *testing.T) {
	prevWorkDir := WorkDir
	prevAPI := API_BaseURL
	t.Cleanup(func() {
		WorkDir = prevWorkDir
		API_BaseURL = prevAPI
	})

	WorkDir = t.TempDir()
	API_BaseURL = ""

	domain := edgeDomain{
		Name: "example.com",
		Cache: &edgeCacheConfig{
			Enable:     true,
			DefaultTTL: 60,
			Rules: []edgeCacheRule{
				{Rule: "/", TTL: 60, Priority: 10},
			},
		},
	}

	var b strings.Builder
	writeCacheLocations(&b, domain, false)
	out := b.String()
	tokens := collectLocationTokens(out)

	if got := countToken(tokens, "^~ /"); got != 1 {
		t.Fatalf("expected single location ^~ /, got %d", got)
	}
	if got := countToken(tokens, "/"); got != 0 {
		t.Fatalf("expected default location / skipped when ^~ / exists")
	}
}

func TestWriteCacheLocations_DedupPrefixAndExt(t *testing.T) {
	prevWorkDir := WorkDir
	prevAPI := API_BaseURL
	t.Cleanup(func() {
		WorkDir = prevWorkDir
		API_BaseURL = prevAPI
	})

	WorkDir = t.TempDir()
	API_BaseURL = ""

	domain := edgeDomain{
		Name: "example.com",
		Cache: &edgeCacheConfig{
			Enable:     true,
			DefaultTTL: 60,
			Rules: []edgeCacheRule{
				{Prefix: "/images/", TTL: 60, Priority: 10},
				{Prefix: "/images/", TTL: 30, Priority: 5},
				{Ext: "jpg", TTL: 60, Priority: 8},
				{Ext: ".jpg", TTL: 30, Priority: 2},
			},
		},
	}

	var b strings.Builder
	writeCacheLocations(&b, domain, false)
	out := b.String()
	tokens := collectLocationTokens(out)

	if got := countToken(tokens, "^~ /images/"); got != 1 {
		t.Fatalf("expected single location ^~ /images/, got %d", got)
	}
	if got := countToken(tokens, "~* \\.jpg$"); got != 1 {
		t.Fatalf("expected single location ~* \\.jpg$, got %d", got)
	}
	if got := countToken(tokens, "/"); got != 1 {
		t.Fatalf("expected default location / once, got %d", got)
	}
}

func TestWriteCacheLocations_SkipInvalidRule(t *testing.T) {
	prevWorkDir := WorkDir
	prevAPI := API_BaseURL
	t.Cleanup(func() {
		WorkDir = prevWorkDir
		API_BaseURL = prevAPI
	})

	WorkDir = t.TempDir()
	API_BaseURL = ""

	domain := edgeDomain{
		Name: "example.com",
		Cache: &edgeCacheConfig{
			Enable:     true,
			DefaultTTL: 60,
			Rules: []edgeCacheRule{
				{URI: "no-slash", TTL: 60, Priority: 10},
			},
		},
	}

	var b strings.Builder
	writeCacheLocations(&b, domain, false)
	out := b.String()
	tokens := collectLocationTokens(out)

	if got := countToken(tokens, "= no-slash"); got != 0 {
		t.Fatalf("expected invalid rule skipped")
	}
	if got := countToken(tokens, "/"); got != 1 {
		t.Fatalf("expected default location / once, got %d", got)
	}
}

func TestApplyCacheDirectives_UsesConfiguredStatusCodesAndNoCacheFlag(t *testing.T) {
	prevNginxCfg := LocalNginxConfig
	t.Cleanup(func() {
		LocalNginxConfig = prevNginxCfg
	})
	LocalNginxConfig = &edgeNginxConfig{
		HTTP: map[string]interface{}{
			"proxy_cache_valid_statuses": "200 206 301 302",
		},
	}

	var b strings.Builder
	applyCacheDirectives(&b, &edgeCacheConfig{Enable: true, DefaultTTL: 60}, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_cache_valid 200 206 301 302 60s;") {
		t.Fatalf("configured status codes were not used in proxy_cache_valid")
	}
	if !strings.Contains(out, "proxy_no_cache $cache_bypass $cdn_no_cache_status;") {
		t.Fatalf("proxy_no_cache must include $cdn_no_cache_status")
	}
}
