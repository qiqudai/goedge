package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHTTPServer_HTTP3EnabledAddsQuicListen(t *testing.T) {
	prevNginxBin := NginxBinPath
	t.Cleanup(func() {
		NginxBinPath = prevNginxBin
		resetHTTP3SupportCacheForTests()
	})
	NginxBinPath = ""
	resetHTTP3SupportCacheForTests()

	domain := edgeDomain{
		Name:       "example.com",
		HTTPSHTTP3: true,
		IPv6Enable: true,
	}

	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if !strings.Contains(out, "listen 443 ssl;") {
		t.Fatalf("expected tls listen directive")
	}
	if !strings.Contains(out, "listen 443 quic reuseport;") {
		t.Fatalf("expected quic listen directive when https_http3 is enabled")
	}
	if !strings.Contains(out, "listen [::]:443 quic reuseport;") {
		t.Fatalf("expected ipv6 quic listen directive when https_http3 is enabled")
	}
	if !strings.Contains(out, "add_header Alt-Svc 'h3=\\\":443\\\"; ma=86400' always;") {
		t.Fatalf("expected Alt-Svc header when https_http3 is enabled")
	}
}

func TestWriteHTTPServer_HTTP3DisabledSkipsQuicListen(t *testing.T) {
	prevNginxBin := NginxBinPath
	t.Cleanup(func() {
		NginxBinPath = prevNginxBin
		resetHTTP3SupportCacheForTests()
	})
	NginxBinPath = ""
	resetHTTP3SupportCacheForTests()

	domain := edgeDomain{
		Name:       "example.com",
		HTTPSHTTP3: false,
	}

	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if strings.Contains(out, " quic reuseport;") {
		t.Fatalf("unexpected quic listen directive when https_http3 is disabled")
	}
	if strings.Contains(out, "add_header Alt-Svc") {
		t.Fatalf("unexpected Alt-Svc advertisement when https_http3 is disabled")
	}
}

func TestWriteHTTPServer_HTTP3EnabledButNginxNoV3SkipsQuic(t *testing.T) {
	prevNginxBin := NginxBinPath
	prevRoot := WorkDir
	t.Cleanup(func() {
		NginxBinPath = prevNginxBin
		WorkDir = prevRoot
		resetHTTP3SupportCacheForTests()
	})
	resetHTTP3SupportCacheForTests()

	workDir := t.TempDir()
	WorkDir = workDir
	scriptPath := filepath.Join(workDir, "nginx-fake.sh")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-V\" ]; then\n" +
		"  echo 'configure arguments: --with-http_v2_module' 1>&2\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake nginx script: %v", err)
	}
	NginxBinPath = scriptPath

	domain := edgeDomain{
		Name:       "example.com",
		HTTPSHTTP3: true,
	}
	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if strings.Contains(out, " quic reuseport;") {
		t.Fatalf("unexpected quic listen directive when nginx has no http_v3 module")
	}
	if strings.Contains(out, "add_header Alt-Svc") {
		t.Fatalf("unexpected Alt-Svc advertisement when nginx has no http_v3 module")
	}
}

func TestWriteHTTPServer_HTTP3RaisesLowConnLimit(t *testing.T) {
	prevNginxBin := NginxBinPath
	t.Cleanup(func() {
		NginxBinPath = prevNginxBin
		resetHTTP3SupportCacheForTests()
	})
	NginxBinPath = ""
	resetHTTP3SupportCacheForTests()

	domain := edgeDomain{
		Name:       "example.com",
		HTTPSHTTP3: true,
		ConnLimit:  1,
	}

	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if !strings.Contains(out, "limit_conn addr_conn 10;") {
		t.Fatalf("expected low conn_limit to be raised for http3 domains")
	}
	if strings.Contains(out, "limit_conn addr_conn 1;") {
		t.Fatalf("unexpected raw conn_limit for http3 domain")
	}
}

func TestWriteHTTPServer_HTTP2KeepsConfiguredConnLimit(t *testing.T) {
	prevNginxBin := NginxBinPath
	t.Cleanup(func() {
		NginxBinPath = prevNginxBin
		resetHTTP3SupportCacheForTests()
	})
	NginxBinPath = ""
	resetHTTP3SupportCacheForTests()

	domain := edgeDomain{
		Name:       "example.com",
		HTTPSHTTP3: false,
		ConnLimit:  1,
	}

	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if !strings.Contains(out, "limit_conn addr_conn 1;") {
		t.Fatalf("expected non-http3 domain to keep configured conn_limit")
	}
}
