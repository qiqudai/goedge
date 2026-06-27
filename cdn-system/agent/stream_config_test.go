package main

import (
	"strings"
	"testing"
)

func TestParseStreamListenEntry(t *testing.T) {
	tests := []struct {
		raw         string
		defaultProto string
		wantPort    string
		wantProto   string
		wantOK      bool
	}{
		{raw: "8080", defaultProto: "tcp", wantPort: "8080", wantProto: "tcp", wantOK: true},
		{raw: "99/udp", defaultProto: "tcp", wantPort: "99", wantProto: "udp", wantOK: true},
		{raw: "88/tcp", defaultProto: "udp", wantPort: "88", wantProto: "tcp", wantOK: true},
		{raw: "invalid", defaultProto: "tcp", wantOK: false},
	}
	for _, tt := range tests {
		entry, ok := parseStreamListenEntry(tt.raw, tt.defaultProto)
		if ok != tt.wantOK {
			t.Fatalf("parseStreamListenEntry(%q) ok=%v want %v", tt.raw, ok, tt.wantOK)
		}
		if !tt.wantOK {
			continue
		}
		if entry.Port != tt.wantPort || entry.Protocol != tt.wantProto {
			t.Fatalf("parseStreamListenEntry(%q) = (%s,%s) want (%s,%s)", tt.raw, entry.Port, entry.Protocol, tt.wantPort, tt.wantProto)
		}
	}
}

func TestRenderStreamConfigTCP(t *testing.T) {
	cfg := renderStreamConfig([]edgeStream{
		{
			ID:             1,
			ListenPorts:    []string{"8080"},
			ListenProtocol: "tcp",
			BalanceWay:     "ip_hash",
			ProxyProtocol:  true,
			ConnLimit:      100,
			Targets: []edgeStreamTarget{
				{Addr: "10.0.0.1:9000", Weight: 2, Enable: true},
			},
		},
	}, streamStatusSnapshot{})
	if cfg == "" {
		t.Fatal("expected non-empty stream config")
	}
	if !strings.Contains(cfg, "hash $remote_addr consistent;") {
		t.Fatalf("expected ip_hash, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "listen 8080 proxy_protocol;") {
		t.Fatalf("expected tcp proxy_protocol listen, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "server 10.0.0.1:9000 weight=2;") {
		t.Fatalf("expected weighted upstream target, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "limit_conn stream_conn 100;") {
		t.Fatalf("expected conn limit, got:\n%s", cfg)
	}
}

func TestRenderStreamConfigUDPNoProxyProtocol(t *testing.T) {
	cfg := renderStreamConfig([]edgeStream{
		{
			ID:             2,
			ListenPorts:    []string{"53/udp"},
			ListenProtocol: "udp",
			ProxyProtocol:  true,
			Targets: []edgeStreamTarget{
				{Addr: "8.8.8.8:53", Enable: true},
			},
		},
	}, streamStatusSnapshot{})
	if !strings.Contains(cfg, "listen 53 udp;") {
		t.Fatalf("expected udp listen, got:\n%s", cfg)
	}
	if strings.Contains(cfg, "proxy_protocol") {
		t.Fatalf("udp listen must not use proxy_protocol, got:\n%s", cfg)
	}
}

func TestRenderStreamConfigLeastConnAndL2ListenPort(t *testing.T) {
	cfg := renderStreamConfig([]edgeStream{
		{
			ID:             3,
			ListenPorts:    []string{"88", "99/udp"},
			ListenProtocol: "tcp",
			BalanceWay:     "least_conn",
			UseListenPort:  true,
			Targets: []edgeStreamTarget{
				{Addr: "10.0.0.2", NodeID: 10, Enable: true},
				{Addr: "1.1.1.1:8080", Enable: true, Backup: true},
			},
		},
	}, streamStatusSnapshot{L2: map[int64]bool{10: true}})
	if !strings.Contains(cfg, "least_conn;") {
		t.Fatalf("expected least_conn, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "upstream stream_up_3_88") {
		t.Fatalf("expected per-port upstream for L2 mode, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "upstream stream_up_3_99_udp") {
		t.Fatalf("expected udp upstream suffix, got:\n%s", cfg)
	}
	if !strings.Contains(cfg, "server 10.0.0.2:88;") {
		t.Fatalf("expected L2 target with listen port appended, got:\n%s", cfg)
	}
}

func TestSelectStreamTargetsL2Failover(t *testing.T) {
	stream := edgeStream{
		UseListenPort: true,
		Targets: []edgeStreamTarget{
			{Addr: "10.0.0.1", NodeID: 1, Enable: true},
			{Addr: "1.1.1.1:8080", Enable: true, Backup: true},
		},
	}
	healthy := selectStreamTargets(stream, streamStatusSnapshot{L2: map[int64]bool{1: true}})
	if len(healthy) != 2 || !healthy[1].Backup {
		t.Fatalf("healthy L2 should keep backup origin, got %#v", healthy)
	}

	fallback := selectStreamTargets(stream, streamStatusSnapshot{L2: map[int64]bool{1: false}})
	if len(fallback) != 1 || fallback[0].Backup {
		t.Fatalf("offline L2 should fallback to origin only without backup flag, got %#v", fallback)
	}
}

func TestFilterStreamPortsDisabled(t *testing.T) {
	resources := &edgeResources{
		Forward: edgeForwardResources{DisabledPorts: "80,443"},
	}
	filtered := filterStreamPorts([]string{"80", "8080", "443/udp"}, resources)
	if len(filtered) != 1 || filtered[0] != "8080" {
		t.Fatalf("filterStreamPorts = %#v", filtered)
	}
}
