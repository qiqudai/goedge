package services

import "testing"

func TestDiagnoseAccessLogSlowReason(t *testing.T) {
	tests := []struct {
		name string
		in   DiagnoseInput
		want string
	}{
		{
			name: "hit fast",
			in:   DiagnoseInput{RequestTime: 0.2, UpstreamResponseTime: 0, UpstreamConnectTime: 0, UpstreamCacheStatus: "HIT", Status: 200},
			want: "正常命中",
		},
		{
			name: "miss origin slow",
			in:   DiagnoseInput{RequestTime: 2.2, UpstreamResponseTime: 2, UpstreamCacheStatus: "MISS", Status: 200},
			want: "缓存未命中回源慢",
		},
		{
			name: "connect slow",
			in:   DiagnoseInput{RequestTime: 1.1, UpstreamConnectTime: 0.8, UpstreamResponseTime: 0.9, Status: 200},
			want: "回源建连慢",
		},
		{
			name: "header slow",
			in:   DiagnoseInput{RequestTime: 1.7, UpstreamHeaderTime: 1.5, UpstreamResponseTime: 0.8, Status: 200},
			want: "源站首包慢",
		},
		{
			name: "response slow",
			in:   DiagnoseInput{RequestTime: 2.1, UpstreamResponseTime: 2, Status: 200},
			want: "源站响应慢",
		},
		{
			name: "tls client side slow",
			in:   DiagnoseInput{RequestTime: 1.1, UpstreamResponseTime: 0.3, Scheme: "https", Status: 200},
			want: "客户端链路或 TLS 握手慢",
		},
		{
			name: "server error",
			in:   DiagnoseInput{RequestTime: 1.1, UpstreamResponseTime: 0.8, Status: 502},
			want: "源站或节点错误",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := DiagnoseAccessLogSlowReason(tt.in)
			if got != tt.want {
				t.Fatalf("reason = %q, want %q", got, tt.want)
			}
		})
	}
}
