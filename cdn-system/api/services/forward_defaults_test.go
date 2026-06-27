package services

import (
	"cdn-api/models"
	"testing"
)

func TestApplyForwardDefaults(t *testing.T) {
	forward := &models.Forward{
		ListenPorts: []string{"8080"},
		Origins: []models.ForwardOrigin{
			{Address: "1.1.1.1:9000", Weight: 1, Enable: true},
		},
	}
	defaults := map[string]string{
		"listen_protocol": "udp",
		"balance_way":     "ip_hash",
		"proxy_protocol":  "true",
		"backsource_port": "8080",
	}
	ApplyForwardDefaults(forward, defaults)

	if forward.Settings["listen_protocol"] != "udp" {
		t.Fatalf("listen_protocol = %#v", forward.Settings["listen_protocol"])
	}
	if forward.BalanceWay != "ip_hash" {
		t.Fatalf("balance_way = %q", forward.BalanceWay)
	}
	if !forward.ProxyProtocol {
		t.Fatal("proxy_protocol should be true")
	}
	if forward.BackendPort != "8080" {
		t.Fatalf("backend_port = %q", forward.BackendPort)
	}

	originCfg := forward.Settings["origin"].(map[string]interface{})
	if originCfg["balance_way"] != "ip_hash" {
		t.Fatalf("origin balance_way = %#v", originCfg["balance_way"])
	}
	if originCfg["proxy_protocol"] != true {
		t.Fatalf("origin proxy_protocol = %#v", originCfg["proxy_protocol"])
	}
}

func TestApplyForwardDefaultsDoesNotOverrideExisting(t *testing.T) {
	forward := &models.Forward{
		BalanceWay:    "least_conn",
		BackendPort:   "9000",
		ProxyProtocol: false,
		Settings: map[string]interface{}{
			"listen_protocol": "tcp",
			"origin": map[string]interface{}{
				"proxy_protocol": false,
			},
		},
	}
	ApplyForwardDefaults(forward, map[string]string{
		"listen_protocol": "udp",
		"balance_way":     "ip_hash",
		"proxy_protocol":  "true",
		"backsource_port": "8080",
	})
	if forward.BalanceWay != "least_conn" {
		t.Fatalf("balance_way should not be overridden, got %q", forward.BalanceWay)
	}
	if forward.BackendPort != "9000" {
		t.Fatalf("backend_port should not be overridden, got %q", forward.BackendPort)
	}
	if forward.ProxyProtocol {
		t.Fatal("proxy_protocol should not be overridden")
	}
	if forward.Settings["listen_protocol"] != "tcp" {
		t.Fatalf("listen_protocol should not be overridden, got %#v", forward.Settings["listen_protocol"])
	}
}
