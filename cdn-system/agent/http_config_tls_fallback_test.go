package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepareHTTPConfigTLSTestEnv(t *testing.T) string {
	t.Helper()
	prevWorkDir := WorkDir
	WorkDir = t.TempDir()
	t.Cleanup(func() {
		WorkDir = prevWorkDir
	})

	root := runtimeRoot()
	if err := os.MkdirAll(filepath.Join(root, "cert"), 0o755); err != nil {
		t.Fatalf("mkdir cert failed: %v", err)
	}
	return root
}

func TestSiteTLSPaths_UsesFallbackWhenConfiguredPairMissing(t *testing.T) {
	root := prepareHTTPConfigTLSTestEnv(t)
	fallbackCert := filepath.Join(root, "cert", "fallback.pem")
	fallbackKey := filepath.Join(root, "cert", "fallback.key")
	if err := os.WriteFile(fallbackCert, []byte("cert"), 0o644); err != nil {
		t.Fatalf("write fallback cert failed: %v", err)
	}
	if err := os.WriteFile(fallbackKey, []byte("key"), 0o600); err != nil {
		t.Fatalf("write fallback key failed: %v", err)
	}

	domain := edgeDomain{
		Name:        "missing.example.com",
		SSLCertPath: filepath.Join(root, "cert", "missing.pem"),
		SSLKeyPath:  filepath.Join(root, "cert", "missing.key"),
	}
	gotCert, gotKey := siteTLSPaths(domain)

	if gotCert != filepath.ToSlash(fallbackCert) {
		t.Fatalf("expected fallback cert path, got %s", gotCert)
	}
	if gotKey != filepath.ToSlash(fallbackKey) {
		t.Fatalf("expected fallback key path, got %s", gotKey)
	}
}

func TestSiteTLSPaths_UsesConfiguredPairWhenFilesExist(t *testing.T) {
	root := prepareHTTPConfigTLSTestEnv(t)
	certPath := filepath.Join(root, "cert", "site.pem")
	keyPath := filepath.Join(root, "cert", "site.key")
	if err := os.WriteFile(certPath, []byte("cert"), 0o644); err != nil {
		t.Fatalf("write cert failed: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatalf("write key failed: %v", err)
	}

	domain := edgeDomain{
		Name:        "ok.example.com",
		SSLCertPath: certPath,
		SSLKeyPath:  keyPath,
	}
	gotCert, gotKey := siteTLSPaths(domain)
	if gotCert != filepath.ToSlash(certPath) {
		t.Fatalf("expected configured cert path, got %s", gotCert)
	}
	if gotKey != filepath.ToSlash(keyPath) {
		t.Fatalf("expected configured key path, got %s", gotKey)
	}
}

func TestEnsureFallbackCertReady_GeneratesFallbackPair(t *testing.T) {
	root := prepareHTTPConfigTLSTestEnv(t)
	certPath := filepath.Join(root, "cert", "fallback.pem")
	keyPath := filepath.Join(root, "cert", "fallback.key")
	_ = os.Remove(certPath)
	_ = os.Remove(keyPath)

	ensureFallbackCertReady()

	if !fileExists(certPath) {
		t.Fatalf("fallback cert was not generated")
	}
	if !fileExists(keyPath) {
		t.Fatalf("fallback key was not generated")
	}
}

func TestWriteHTTPServer_UsesFallbackPairWhenConfiguredPathInvalid(t *testing.T) {
	root := prepareHTTPConfigTLSTestEnv(t)
	fallbackCert := filepath.Join(root, "cert", "fallback.pem")
	fallbackKey := filepath.Join(root, "cert", "fallback.key")
	if err := os.WriteFile(fallbackCert, []byte("cert"), 0o644); err != nil {
		t.Fatalf("write fallback cert failed: %v", err)
	}
	if err := os.WriteFile(fallbackKey, []byte("key"), 0o600); err != nil {
		t.Fatalf("write fallback key failed: %v", err)
	}

	domain := edgeDomain{
		Name:        "invalid.example.com",
		SSLCertPath: filepath.Join(root, "cert", "not-exist.pem"),
		SSLKeyPath:  filepath.Join(root, "cert", "not-exist.key"),
	}
	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, errorPageContext{}, 0, false)
	out := b.String()
	if !strings.Contains(out, "ssl_certificate "+filepath.ToSlash(fallbackCert)+";") {
		t.Fatalf("expected fallback ssl_certificate directive, got: %s", out)
	}
	if !strings.Contains(out, "ssl_certificate_key "+filepath.ToSlash(fallbackKey)+";") {
		t.Fatalf("expected fallback ssl_certificate_key directive, got: %s", out)
	}
}

func TestWriteDefaultServer_AllowsHTTP01ChallengeBeforeBlockingUnboundHost(t *testing.T) {
	root := prepareHTTPConfigTLSTestEnv(t)
	prevAPI := API_BaseURL
	API_BaseURL = "http://127.0.0.1:8080"
	t.Cleanup(func() {
		API_BaseURL = prevAPI
	})

	var b strings.Builder
	writeDefaultServer(&b, "80", false, errorPageContext{}, 418, false)
	out := b.String()
	acmeIndex := strings.Index(out, "location ^~ /.well-known/acme-challenge/")
	blockIndex := strings.Index(out, "guard.enforce(418)")

	if acmeIndex == -1 {
		t.Fatalf("expected default server to include ACME challenge location, got: %s", out)
	}
	if blockIndex == -1 {
		t.Fatalf("expected default server block response, got: %s", out)
	}
	if !strings.Contains(out, "X-Block-Source 'type=local_protection;module=nginx.default_server;rule=unbound_domain;rule_id=0;condition=direct_ip_or_unbound_host' always;") {
		t.Fatalf("expected default server to tag the block source, got: %s", out)
	}
	if !strings.Contains(out, "error_page 404 = @acme_master;") || !strings.Contains(out, "proxy_pass http://127.0.0.1:8080;") {
		t.Fatalf("expected default server to proxy missing ACME token to master, got: %s", out)
	}
	if acmeIndex > blockIndex {
		t.Fatalf("expected ACME challenge location before blocking location, got: %s", out)
	}
	if !strings.Contains(out, "alias "+filepath.ToSlash(filepath.Join(root, "cert", "acme", ".well-known", "acme-challenge"))+"/;") {
		t.Fatalf("expected ACME webroot in default server, got: %s", out)
	}
}

func TestWriteAcmeLocation_ServesLocalChallengeDirectoryWithoutMasterProxy(t *testing.T) {
	prepareHTTPConfigTLSTestEnv(t)
	prevAPI := API_BaseURL
	API_BaseURL = ""
	t.Cleanup(func() {
		API_BaseURL = prevAPI
	})

	var b strings.Builder
	writeAcmeLocation(&b)
	out := b.String()
	if !strings.Contains(out, "alias ") {
		t.Fatalf("expected local ACME alias, got: %s", out)
	}
	if strings.Contains(out, "location @acme_master") {
		t.Fatalf("did not expect master proxy location without API base, got: %s", out)
	}
}

func TestAcmeMasterProxyBaseSanitizesAPIBase(t *testing.T) {
	prevAPI := API_BaseURL
	t.Cleanup(func() {
		API_BaseURL = prevAPI
	})

	API_BaseURL = "https://api.example.com/base?x=1"
	if got := acmeMasterProxyBase(); got != "https://api.example.com" {
		t.Fatalf("acmeMasterProxyBase() = %q", got)
	}

	API_BaseURL = "javascript:alert(1)"
	if got := acmeMasterProxyBase(); got != "" {
		t.Fatalf("expected unsafe scheme to be rejected, got %q", got)
	}

	API_BaseURL = "http://127.0.0.1:8080;\nproxy_pass http://evil"
	if got := acmeMasterProxyBase(); got != "" {
		t.Fatalf("expected injected API base to be rejected, got %q", got)
	}
}
