package controllers

import (
	"testing"
)

func TestNormalizeForwardListenEndpoint(t *testing.T) {
	tests := []struct {
		in       string
		wantHost string
		wantPort int
		wantOK   bool
	}{
		{in: "8080", wantHost: "*", wantPort: 8080, wantOK: true},
		{in: ":8080", wantHost: "*", wantPort: 8080, wantOK: true},
		{in: "0.0.0.0:8080", wantHost: "*", wantPort: 8080, wantOK: true},
		{in: "192.168.1.1:9000", wantHost: "192.168.1.1", wantPort: 9000, wantOK: true},
		{in: "[::]:443", wantHost: "*", wantPort: 443, wantOK: true},
		{in: "88/tcp", wantHost: "*", wantPort: 88, wantOK: true},
		{in: "99/udp", wantHost: "*", wantPort: 99, wantOK: true},
		{in: "", wantOK: false},
		{in: "abc", wantOK: false},
	}
	for _, tt := range tests {
		host, port, ok := normalizeForwardListenEndpoint(tt.in)
		if ok != tt.wantOK {
			t.Fatalf("normalizeForwardListenEndpoint(%q) ok=%v want %v", tt.in, ok, tt.wantOK)
		}
		if !tt.wantOK {
			continue
		}
		if host != tt.wantHost || port != tt.wantPort {
			t.Fatalf("normalizeForwardListenEndpoint(%q) = (%q,%d) want (%q,%d)", tt.in, host, port, tt.wantHost, tt.wantPort)
		}
	}
}

func TestForwardListenConflict(t *testing.T) {
	if !forwardListenConflict("*", 88, "*", 88) {
		t.Fatal("88 and 88/tcp should conflict on wildcard host")
	}
	host, port, ok := normalizeForwardListenEndpoint("99/udp")
	if !ok || port != 99 || host != "*" {
		t.Fatalf("normalizeForwardListenEndpoint(99/udp) = (%q,%d,%v)", host, port, ok)
	}
	if !forwardListenConflict("*", 8080, "0.0.0.0", 8080) {
		t.Fatal("wildcard host should conflict on same port")
	}
	if forwardListenConflict("10.0.0.1", 8080, "10.0.0.2", 8080) {
		t.Fatal("different hosts on same port should not conflict")
	}
	if !forwardListenConflict("10.0.0.1", 8080, "10.0.0.1", 8080) {
		t.Fatal("same host and port should conflict")
	}
	if forwardListenConflict("*", 8080, "10.0.0.1", 8081) {
		t.Fatal("different ports should not conflict")
	}
}

func TestParseForwardBatchLine(t *testing.T) {
	listen, origins, err := parseForwardBatchLine("88 99/udp|1.2.3.4:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listen) != 2 || listen[0] != "88" || listen[1] != "99/udp" {
		t.Fatalf("listen ports = %#v", listen)
	}
	if len(origins) != 1 || origins[0].Address != "1.2.3.4:8080" {
		t.Fatalf("origins = %#v", origins)
	}

	_, _, err = parseForwardBatchLine("invalid")
	if err == nil {
		t.Fatal("expected error for invalid batch line")
	}
}

func TestParseOrigins(t *testing.T) {
	origins := parseOrigins("1.1.1.1:99 8.8.8.8:53")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0].Address != "1.1.1.1:99" || !origins[0].Enable || origins[0].Weight != 1 {
		t.Fatalf("first origin = %#v", origins[0])
	}
}

func TestExtractBackendPort(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{addr: "1.1.1.1:8080", want: "8080"},
		{addr: "[2001:db8::1]:443", want: "443"},
		{addr: "example.com", want: ""},
	}
	for _, tt := range tests {
		got := extractBackendPort(parseOrigins(tt.addr))
		if got != tt.want {
			t.Fatalf("extractBackendPort(%q) = %q want %q", tt.addr, got, tt.want)
		}
	}
}

func TestParseForwardKeyword(t *testing.T) {
	tests := []struct {
		in          string
		wantPort    int
		wantProto   string
	}{
		{in: "", wantPort: 0, wantProto: ""},
		{in: "8080", wantPort: 8080, wantProto: ""},
		{in: "8080/tcp", wantPort: 8080, wantProto: "TCP"},
		{in: "99/udp", wantPort: 99, wantProto: "UDP"},
		{in: "tcp", wantPort: 0, wantProto: "TCP"},
		{in: "udp", wantPort: 0, wantProto: "UDP"},
	}
	for _, tt := range tests {
		port, proto := parseForwardKeyword(tt.in)
		if port != tt.wantPort || proto != tt.wantProto {
			t.Fatalf("parseForwardKeyword(%q) = (%d,%q) want (%d,%q)", tt.in, port, proto, tt.wantPort, tt.wantProto)
		}
	}
}

func TestParseForwardListenPort(t *testing.T) {
	tests := []struct {
		in        string
		wantPort  int
		wantProto string
	}{
		{in: "88", wantPort: 88, wantProto: ""},
		{in: "99/udp", wantPort: 99, wantProto: "UDP"},
		{in: "8080/tcp", wantPort: 8080, wantProto: "TCP"},
		{in: ":443", wantPort: 443, wantProto: ""},
	}
	for _, tt := range tests {
		port, proto := parseForwardListenPort(tt.in)
		if port != tt.wantPort || proto != tt.wantProto {
			t.Fatalf("parseForwardListenPort(%q) = (%d,%q) want (%d,%q)", tt.in, port, proto, tt.wantPort, tt.wantProto)
		}
	}
}

func TestPortAllowedAndFilterByProtocol(t *testing.T) {
	portMap := map[int]map[string]bool{
		8080: {"TCP": true},
		53:   {"UDP": true},
		9000: {"TCP": true, "UDP": true},
	}
	if !portAllowed(portMap, 8080, "TCP") {
		t.Fatal("8080/tcp should be allowed")
	}
	if portAllowed(portMap, 8080, "UDP") {
		t.Fatal("8080/udp should not be allowed")
	}
	if portAllowed(portMap, 9999, "") {
		t.Fatal("unknown port should not be allowed")
	}
	filtered := filterPortsByProtocol(portMap, "UDP")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 udp ports, got %v", filtered)
	}
}

func TestParseForwardDefaultValue(t *testing.T) {
	if v := parseForwardDefaultValue("proxy_protocol", "true"); v != true {
		t.Fatalf("proxy_protocol true = %#v", v)
	}
	if v := parseForwardDefaultValue("listen_protocol", "udp"); v != "udp" {
		t.Fatalf("listen_protocol = %#v", v)
	}
	if v := parseForwardDefaultValue("balance_way", "ip_hash"); v != "ip_hash" {
		t.Fatalf("balance_way = %#v", v)
	}
}

func TestEncodeForwardDefaultValue(t *testing.T) {
	if got := encodeForwardDefaultValue("proxy_protocol", true); got != "true" {
		t.Fatalf("encode bool true = %q", got)
	}
	if got := encodeForwardDefaultValue("balance_way", "least_conn"); got != "least_conn" {
		t.Fatalf("encode string = %q", got)
	}
}

func TestApplyOriginSettings(t *testing.T) {
	updates := map[string]interface{}{}
	settings := map[string]interface{}{
		"origin": map[string]interface{}{
			"balance_way":     "ip_hash",
			"proxy_protocol":  true,
			"backsource_port": "8080",
			"origins": []interface{}{
				map[string]interface{}{"address": "1.1.1.1:8080", "weight": 2, "enable": true},
			},
		},
	}
	applyOriginSettings(settings, updates)
	if updates["balance_way"] != "ip_hash" {
		t.Fatalf("balance_way = %#v", updates["balance_way"])
	}
	if updates["proxy_protocol"] != true {
		t.Fatalf("proxy_protocol = %#v", updates["proxy_protocol"])
	}
	if updates["backend_port"] != "8080" {
		t.Fatalf("backend_port = %#v", updates["backend_port"])
	}
	if updates["backend"] == "" {
		t.Fatal("backend should be encoded")
	}
}
