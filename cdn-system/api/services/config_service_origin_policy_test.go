package services

import (
	"testing"

	"cdn-api/models"
)

func TestExtractAdvancedConfigOriginHTTPPolicyDefaults(t *testing.T) {
	cfg := extractAdvancedConfig(nil)
	if cfg.originHTTPVersionPolicy != "auto" {
		t.Fatalf("policy = %q, want auto", cfg.originHTTPVersionPolicy)
	}
	if !cfg.originAutoDowngrade {
		t.Fatalf("auto downgrade should default to true")
	}
	if cfg.keepaliveConn != 64 || cfg.keepaliveTimeout != 60 {
		t.Fatalf("keepalive defaults = %d/%d, want 64/60", cfg.keepaliveConn, cfg.keepaliveTimeout)
	}
}

func TestExtractAdvancedConfigOriginHTTPPolicyLegacyKeepalive(t *testing.T) {
	cfg := extractAdvancedConfig(map[string]interface{}{
		"advanced": map[string]interface{}{
			"ups_keepalive": true,
		},
	})
	if cfg.originHTTPVersionPolicy != "auto" {
		t.Fatalf("policy = %q, want auto", cfg.originHTTPVersionPolicy)
	}
	if !cfg.keepalive {
		t.Fatalf("auto policy should enable upstream keepalive")
	}
}

func TestExtractAdvancedConfigOriginHTTPPolicyExplicit(t *testing.T) {
	for _, policy := range []string{"compat", "http11"} {
		cfg := extractAdvancedConfig(map[string]interface{}{
			"advanced": map[string]interface{}{
				"origin_http_version_policy": policy,
			},
		})
		if cfg.originHTTPVersionPolicy != policy {
			t.Fatalf("policy = %q, want %q", cfg.originHTTPVersionPolicy, policy)
		}
	}
}

func TestExtractOriginTLSConfig_NormalizesHostHeaderSentinels(t *testing.T) {
	site := models.Site{
		Domains: []string{"main.example.com", "alt.example.com"},
		Settings: map[string]interface{}{
			"origin": map[string]interface{}{
				"host_header": "follow",
			},
		},
	}

	host, sni, _ := extractOriginTLSConfig(site)
	if host != "" {
		t.Fatalf("host = %q, want empty for follow", host)
	}
	if sni != "" {
		t.Fatalf("sni = %q, want empty when host is empty", sni)
	}

	site.Settings["origin"] = map[string]interface{}{"host_header": "domain"}
	host, sni, _ = extractOriginTLSConfig(site)
	if host != "main.example.com" {
		t.Fatalf("host = %q, want first domain", host)
	}
	if sni != "main.example.com" {
		t.Fatalf("sni = %q, want first domain", sni)
	}
}

func TestBuildHeaderMap_NormalizesLegacyHostSentinels(t *testing.T) {
	site := models.Site{
		Domains: []string{"main.example.com"},
		Settings: map[string]interface{}{
			"headers": map[string]interface{}{
				"Host": "follow",
			},
		},
	}
	headers := buildHeaderMap(site)
	if headers["Host"] != "$host" {
		t.Fatalf("host header = %q, want $host", headers["Host"])
	}

	site.Settings["headers"] = map[string]interface{}{"Host": "domain"}
	headers = buildHeaderMap(site)
	if headers["Host"] != "main.example.com" {
		t.Fatalf("host header = %q, want main.example.com", headers["Host"])
	}
}
