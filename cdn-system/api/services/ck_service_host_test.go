package services

import "testing"

func TestEffectiveAccessHostPrefersRawHostHeaders(t *testing.T) {
	tests := []struct {
		name string
		raw  rawAccessLog
		want string
	}{
		{
			name: "http_host with port wins over nginx host",
			raw:  rawAccessLog{Host: "_", HTTPHost: "Example.COM:443", SSLServerName: "sni.example.com"},
			want: "example.com",
		},
		{
			name: "legacy cdn request headers host wins when http_host missing",
			raw:  rawAccessLog{Host: "_", CDNReqHeaders: `{"user-agent":"curl","host":"real.example.com"}`, SSLServerName: "sni.example.com"},
			want: "real.example.com",
		},
		{
			name: "sni used for https default server when host missing",
			raw:  rawAccessLog{Host: "_", SSLServerName: "TLS.Example.COM."},
			want: "tls.example.com",
		},
		{
			name: "direct ipv4 host keeps ip without port",
			raw:  rawAccessLog{Host: "61.4.122.233", HTTPHost: "61.4.122.233:80"},
			want: "61.4.122.233",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveAccessHost(tt.raw); got != tt.want {
				t.Fatalf("effectiveAccessHost() = %q, want %q", got, tt.want)
			}
		})
	}
}
