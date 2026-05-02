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

func TestWriteProxyBlock_OriginHostHeaderSetsHostAndDefaultSNI(t *testing.T) {
	domain := edgeDomain{
		OriginProtocol:   "https",
		OriginHostHeader: "origin.example.com",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_set_header Host \"origin.example.com\";") {
		t.Fatalf("expected origin Host header, got: %s", out)
	}
	if !strings.Contains(out, "proxy_ssl_name origin.example.com;") {
		t.Fatalf("expected origin Host header to be default SNI, got: %s", out)
	}
}

func TestWriteProxyBlock_OriginHostHeaderFollowFallsBackToHostVariable(t *testing.T) {
	domain := edgeDomain{
		OriginHostHeader: "follow",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_set_header Host $host;") {
		t.Fatalf("expected follow sentinel to resolve to $host, got: %s", out)
	}
}

func TestWriteProxyBlock_OriginHostHeaderDomainUsesFirstServerName(t *testing.T) {
	domain := edgeDomain{
		Name:             "*.fxhj.app, fxhj.app",
		OriginHostHeader: "domain",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_set_header Host \"*.fxhj.app\";") {
		t.Fatalf("expected domain sentinel to resolve to first server name, got: %s", out)
	}
}

func TestWriteProxyBlock_ExplicitOriginSNIOverridesHostHeader(t *testing.T) {
	domain := edgeDomain{
		OriginProtocol:   "https",
		OriginHostHeader: "origin-host.example.com",
		OriginSNI:        "origin-sni.example.com",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_set_header Host \"origin-host.example.com\";") {
		t.Fatalf("expected origin Host header, got: %s", out)
	}
	if !strings.Contains(out, "proxy_ssl_name origin-sni.example.com;") {
		t.Fatalf("expected explicit SNI to override host header, got: %s", out)
	}
}

func TestWriteProxyBlock_HTTPOriginSkipsProxySSL(t *testing.T) {
	domain := edgeDomain{
		OriginProtocol: "http",
		OriginSNI:      "origin.example.com",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, false, nil, nil)
	out := b.String()

	if strings.Contains(out, "proxy_ssl_server_name") || strings.Contains(out, "proxy_ssl_name") {
		t.Fatalf("http origin must not emit proxy ssl directives, got: %s", out)
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
	if !strings.Contains(out, "proxy_set_header X-Forwarded-Host $host;") {
		t.Fatalf("default X-Forwarded-Host should be set to $host")
	}
	if !strings.Contains(out, "proxy_set_header X-Forwarded-Port $server_port;") {
		t.Fatalf("default X-Forwarded-Port should be set to $server_port")
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

func TestWriteProxyBlock_WebsiteCompatibilityHeaders(t *testing.T) {
	domain := edgeDomain{
		SiteType: "website",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, false, nil, nil)
	out := b.String()

	for _, want := range []string{
		"proxy_set_header User-Agent $http_user_agent;",
		"proxy_set_header Accept $http_accept;",
		"proxy_set_header Accept-Language $http_accept_language;",
		"proxy_set_header Accept-Encoding $http_accept_encoding;",
		"proxy_set_header Referer $http_referer;",
		"proxy_set_header Cache-Control $http_cache_control;",
		"proxy_set_header Upgrade-Insecure-Requests $http_upgrade_insecure_requests;",
		"proxy_set_header Sec-Fetch-Site $http_sec_fetch_site;",
		"proxy_set_header Sec-Fetch-Mode $http_sec_fetch_mode;",
		"proxy_set_header Sec-Fetch-User $http_sec_fetch_user;",
		"proxy_set_header Sec-Fetch-Dest $http_sec_fetch_dest;",
		"proxy_set_header Sec-CH-UA $http_sec_ch_ua;",
		"proxy_set_header Sec-CH-UA-Mobile $http_sec_ch_ua_mobile;",
		"proxy_set_header Sec-CH-UA-Platform $http_sec_ch_ua_platform;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing compatibility header %q\n%s", want, out)
		}
	}
}

func TestWriteProxyBlock_NonWebsiteSkipsCompatibilityHeaders(t *testing.T) {
	domain := edgeDomain{
		SiteType: "api",
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, false, nil, nil)
	out := b.String()

	for _, want := range []string{
		"proxy_set_header User-Agent $http_user_agent;",
		"proxy_set_header Accept $http_accept;",
		"proxy_set_header Accept-Language $http_accept_language;",
		"proxy_set_header Accept-Encoding $http_accept_encoding;",
		"proxy_set_header Referer $http_referer;",
		"proxy_set_header Cache-Control $http_cache_control;",
		"proxy_set_header Upgrade-Insecure-Requests $http_upgrade_insecure_requests;",
		"proxy_set_header Sec-Fetch-Site $http_sec_fetch_site;",
		"proxy_set_header Sec-Fetch-Mode $http_sec_fetch_mode;",
		"proxy_set_header Sec-Fetch-User $http_sec_fetch_user;",
		"proxy_set_header Sec-Fetch-Dest $http_sec_fetch_dest;",
		"proxy_set_header Sec-CH-UA $http_sec_ch_ua;",
		"proxy_set_header Sec-CH-UA-Mobile $http_sec_ch_ua_mobile;",
		"proxy_set_header Sec-CH-UA-Platform $http_sec_ch_ua_platform;",
	} {
		if strings.Contains(out, want) {
			t.Fatalf("did not expect compatibility header %q for API site\n%s", want, out)
		}
	}
}

func TestWriteProxyBlock_NonWebsocketHidesProblematicHopByHopResponseHeaders(t *testing.T) {
	domain := edgeDomain{}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	for _, want := range []string{
		"proxy_hide_header Upgrade;",
		"proxy_hide_header Connection;",
		"proxy_hide_header Keep-Alive;",
		"proxy_hide_header Proxy-Connection;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected non-websocket proxy to hide %q", want)
		}
	}
}

func TestWriteProxyBlock_WebsocketKeepsHopByHopResponseHeadersVisible(t *testing.T) {
	domain := edgeDomain{
		EnableWebsocket: true,
	}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	for _, unexpected := range []string{
		"proxy_hide_header Upgrade;",
		"proxy_hide_header Connection;",
		"proxy_hide_header Keep-Alive;",
		"proxy_hide_header Proxy-Connection;",
	} {
		if strings.Contains(out, unexpected) {
			t.Fatalf("did not expect websocket proxy to hide %q", unexpected)
		}
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

func TestWriteProxyBlock_TLSRewritesAbsoluteHTTPRedirectsToHTTPS(t *testing.T) {
	domain := edgeDomain{}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_redirect ~^http://([^/]+)(/.*)?$ https://$1$2;") {
		t.Fatalf("expected TLS proxy to rewrite absolute http redirects to https")
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

func TestWriteHTTPServer_HSTSDoesNotDuplicateCustomResponseHeader(t *testing.T) {
	domain := edgeDomain{
		Name:        "example.com",
		HttpsListen: []string{"443"},
		HTTPSHSTS:   true,
		ResponseHeaders: map[string]string{
			"Strict-Transport-Security": "max-age=1",
		},
	}

	var b strings.Builder
	writeHTTPServer(&b, domain, "443", true, nil, "", 0, false)
	out := b.String()

	if count := strings.Count(strings.ToLower(out), "add_header strict-transport-security "); count != 1 {
		t.Fatalf("expected exactly one Strict-Transport-Security header, got %d\n%s", count, out)
	}
	if !strings.Contains(out, "add_header Strict-Transport-Security \"max-age=1\" always;") {
		t.Fatalf("expected custom Strict-Transport-Security header to be preserved")
	}
	if strings.Contains(out, "add_header Strict-Transport-Security \"max-age=31536000\" always;") {
		t.Fatalf("default Strict-Transport-Security header should be skipped when custom header is present")
	}
}

func TestWriteProxyBlock_HSTSHidesOriginHeader(t *testing.T) {
	domain := edgeDomain{HTTPSHSTS: true}

	var b strings.Builder
	writeProxyBlock(&b, domain, true, nil, nil)
	out := b.String()

	if !strings.Contains(out, "proxy_hide_header Strict-Transport-Security;") {
		t.Fatalf("expected origin Strict-Transport-Security header to be hidden when CDN HSTS is enabled")
	}
}
