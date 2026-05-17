package controllers

import "testing"

func TestFormatBlockLocation(t *testing.T) {
	tests := []struct {
		name     string
		country  string
		province string
		want     string
	}{
		{name: "both empty", country: "", province: "", want: "-"},
		{name: "country only", country: "中国", province: "", want: "中国"},
		{name: "province only", country: "", province: "广东省", want: "广东省"},
		{name: "both present", country: "中国", province: "广东省", want: "中国-广东省"},
		{name: "dash placeholders", country: "-", province: "-", want: "-"},
	}

	for _, tt := range tests {
		if got := formatBlockLocation(tt.country, tt.province); got != tt.want {
			t.Fatalf("%s: formatBlockLocation(%q, %q) = %q, want %q", tt.name, tt.country, tt.province, got, tt.want)
		}
	}
}

func TestBlockFilterLabel(t *testing.T) {
	tests := []struct {
		name   string
		status int
		source string
		want   string
	}{
		{name: "explicit anti cc", status: 418, source: "anti_cc", want: "Anti-CC | HTTP_418"},
		{name: "explicit waf", status: 403, source: "waf", want: "WAF | HTTP_403"},
		{name: "explicit ip block", status: 403, source: "ip_block", want: "IP黑名单 | HTTP_403"},
		{name: "explicit origin", status: 403, source: "origin", want: "源站返回 | HTTP_403"},
		{name: "fallback 418", status: 418, source: "", want: "CC防护 | HTTP_418"},
		{name: "fallback 429", status: 429, source: "", want: "频控拦截 | HTTP_429"},
		{name: "fallback 403", status: 403, source: "", want: "访问控制 | HTTP_403"},
		{name: "fallback unknown", status: 499, source: "", want: "HTTP_499 | HTTP_499"},
		{name: "structured nginx default", status: 418, source: "type=local_protection;module=nginx.default_server;rule=unbound_domain;rule_id=0;condition=direct_ip_or_unbound_host", want: "本地防护 | 模块:nginx.default_server | 规则:unbound_domain | 条件:direct_ip_or_unbound_host | HTTP_418"},
		{name: "structured lua waf", status: 418, source: "type=waf;module=lua.waf;rule=scanner;rule_id=0;condition=user_agent", want: "WAF | 模块:lua.waf | 规则:scanner | 条件:user_agent | HTTP_418"},
	}

	for _, tt := range tests {
		if got := blockFilterLabel(tt.status, tt.source); got != tt.want {
			t.Fatalf("%s: blockFilterLabel(%d, %q) = %q, want %q", tt.name, tt.status, tt.source, got, tt.want)
		}
	}
}
