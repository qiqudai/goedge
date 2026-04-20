package main

import (
	"os"
	"strings"
	"testing"
)

func TestWriteProxyBlock_CustomHostHeaderNoDuplicate(t *testing.T) {
	domain := edgeDomain{
		Headers: map[string]string{
			"Host": "gf-oss-bucket.s3.ap-east-1.amazonaws.com",
		},
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if strings.Contains(out, "proxy_set_header Host $host;") {
		t.Fatalf("default host header should be skipped when custom Host is provided")
	}
	if count := strings.Count(out, "proxy_set_header Host "); count != 1 {
		t.Fatalf("expected exactly one Host header, got %d", count)
	}
	if !strings.Contains(out, "proxy_set_header Host \"gf-oss-bucket.s3.ap-east-1.amazonaws.com\";") {
		t.Fatalf("custom Host header missing from generated config")
	}
}

func TestWriteProxyBlock_DefaultXForwardedForNoAppend(t *testing.T) {
	// When no custom X-Forwarded-For is set, the default should use $remote_addr
	// (not $proxy_add_x_forwarded_for) to prevent duplicate headers when the
	// client already carries an X-Forwarded-For header. AWS S3 and other strict
	// origins reject requests with duplicate headers.
	domain := edgeDomain{}
	var b strings.Builder
	writeProxyBlock(&b, domain, false, nil, nil)
	out := b.String()
	if strings.Contains(out, "proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;") {
		t.Fatalf("default X-Forwarded-For must not use $proxy_add_x_forwarded_for to avoid duplicate headers at origin")
	}
	if !strings.Contains(out, "proxy_set_header X-Forwarded-For $remote_addr;") {
		t.Fatalf("default X-Forwarded-For should be set to $remote_addr")
	}
}

func TestWriteProxyBlock_CustomForwardedHeadersNoDuplicate(t *testing.T) {
	domain := edgeDomain{
		EnableWebsocket: true,
		Headers: map[string]string{
			"X-Real-IP":         "$http_x_real_ip",
			"x-forwarded-for":   "$http_x_forwarded_for",
			"X-Forwarded-Proto": "https",
			"Connection":        "upgrade",
			"Upgrade":           "$http_upgrade",
		},
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, false, nil, nil)
	out := b.String()

	if strings.Contains(out, "proxy_set_header X-Real-IP $remote_addr;") {
		t.Fatalf("default X-Real-IP should be skipped when custom header is provided")
	}
	if strings.Contains(out, "proxy_set_header X-Forwarded-For $remote_addr;") {
		t.Fatalf("default X-Forwarded-For should be skipped when custom header is provided")
	}
	if strings.Contains(out, "proxy_set_header X-Forwarded-Proto $scheme;") {
		t.Fatalf("default X-Forwarded-Proto should be skipped when custom header is provided")
	}
	if count := strings.Count(out, "proxy_set_header Connection "); count != 1 {
		t.Fatalf("expected exactly one Connection header, got %d", count)
	}
	if count := strings.Count(out, "proxy_set_header Upgrade "); count != 1 {
		t.Fatalf("expected exactly one Upgrade header, got %d", count)
	}
}

func TestWriteProxyBlock_HTTP3UnsupportedHidesAltSvc(t *testing.T) {
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
	scriptPath := workDir + "/nginx-fake.sh"
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

	domain := edgeDomain{HTTPSHTTP3: true}
	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()
	if !strings.Contains(out, "proxy_hide_header Alt-Svc;") {
		t.Fatalf("expected proxy_hide_header Alt-Svc when nginx http_v3 module is missing")
	}
}

func TestWriteProxyBlock_HTTP3SupportedStillHidesUpstreamAltSvc(t *testing.T) {
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
	scriptPath := workDir + "/nginx-fake.sh"
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-V\" ]; then\n" +
		"  echo 'configure arguments: --with-http_v2_module --with-http_v3_module' 1>&2\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake nginx script: %v", err)
	}
	NginxBinPath = scriptPath

	domain := edgeDomain{HTTPSHTTP3: true}
	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()
	if !strings.Contains(out, "proxy_hide_header Alt-Svc;") {
		t.Fatalf("expected proxy_hide_header Alt-Svc even when nginx http_v3 module is available")
	}
}
