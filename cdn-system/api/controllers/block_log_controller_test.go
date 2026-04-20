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
