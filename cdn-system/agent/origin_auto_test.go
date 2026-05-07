package main

import (
	"strings"
	"testing"
)

func TestWriteProxyBlockOriginAutoPolicy(t *testing.T) {
	var b strings.Builder
	writeProxyBlock(&b, edgeDomain{
		OriginHTTPVersionPolicy:        "auto",
		OriginAutoDowngrade:            true,
		OriginDowngradeThreshold:       3,
		OriginDowngradeWindowSeconds:   60,
		OriginDowngradeCooldownSeconds: 600,
	}, false, nil, nil)
	out := b.String()
	for _, want := range []string{
		"set $origin_http_policy \"auto\";",
		"set $origin_auto_downgrade 1;",
		"proxy_http_version 1.1;",
		"proxy_set_header Connection $origin_connection;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("auto config missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteProxyBlockOriginHTTP11Policy(t *testing.T) {
	var b strings.Builder
	writeProxyBlock(&b, edgeDomain{OriginHTTPVersionPolicy: "http11"}, false, nil, nil)
	out := b.String()
	if !strings.Contains(out, "set $origin_http_policy \"http11\";") {
		t.Fatalf("http11 policy marker missing:\n%s", out)
	}
	if !strings.Contains(out, "proxy_http_version 1.1;") {
		t.Fatalf("http11 policy must use HTTP/1.1:\n%s", out)
	}
	if strings.Contains(out, "proxy_set_header Connection $origin_connection;") {
		t.Fatalf("http11 policy must not use auto downgrade connection variable:\n%s", out)
	}
}

func TestWriteProxyBlockOriginCompatPolicy(t *testing.T) {
	var b strings.Builder
	writeProxyBlock(&b, edgeDomain{OriginHTTPVersionPolicy: "compat"}, false, nil, nil)
	out := b.String()
	if !strings.Contains(out, "proxy_http_version 1.0;") {
		t.Fatalf("compat policy must use HTTP/1.0:\n%s", out)
	}
	if !strings.Contains(out, "proxy_set_header Connection close;") {
		t.Fatalf("compat policy must close origin connection:\n%s", out)
	}
	if shouldUseOriginKeepalive(edgeDomain{OriginHTTPVersionPolicy: "compat"}) {
		t.Fatalf("compat policy must not enable upstream keepalive")
	}
}

func TestWriteProxyBlockOriginWebsocketPreserved(t *testing.T) {
	var b strings.Builder
	writeProxyBlock(&b, edgeDomain{OriginHTTPVersionPolicy: "auto", EnableWebsocket: true}, false, nil, nil)
	out := b.String()
	if !strings.Contains(out, "proxy_set_header Upgrade $http_upgrade;") ||
		!strings.Contains(out, "proxy_set_header Connection $connection_upgrade;") {
		t.Fatalf("websocket upgrade headers not preserved:\n%s", out)
	}
}
