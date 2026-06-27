package controllers

import "testing"

func TestIsUnlimitedLimitText(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "", want: true},
		{in: "0", want: true},
		{in: "不限", want: true},
		{in: " unlimited ", want: true},
		{in: "100", want: false},
		{in: "13", want: false},
	}
	for _, tt := range tests {
		if got := isUnlimitedLimitText(tt.in); got != tt.want {
			t.Fatalf("isUnlimitedLimitText(%q) = %v want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseLimitInt(t *testing.T) {
	payload := map[string]interface{}{
		"traffic":    "不限",
		"connection": "100",
		"domain":     "",
	}
	if v, ok := parseLimitInt(payload, "traffic"); !ok || v != 0 {
		t.Fatalf("parseLimitInt(traffic) = (%d,%v) want (0,true)", v, ok)
	}
	if v, ok := parseLimitInt(payload, "connection"); !ok || v != 100 {
		t.Fatalf("parseLimitInt(connection) = (%d,%v) want (100,true)", v, ok)
	}
	if v, ok := parseLimitInt(payload, "domain"); !ok || v != 0 {
		t.Fatalf("parseLimitInt(domain) = (%d,%v) want (0,true)", v, ok)
	}
	if _, ok := parseLimitInt(payload, "missing"); ok {
		t.Fatal("parseLimitInt(missing) should be false")
	}
}

func TestParseLimitString(t *testing.T) {
	payload := map[string]interface{}{
		"bandwidth": "100",
		"empty":     "不限",
	}
	if v, ok := parseLimitString(payload, "bandwidth"); !ok || v != "100" {
		t.Fatalf("parseLimitString(bandwidth) = (%q,%v) want (100,true)", v, ok)
	}
	if v, ok := parseLimitString(payload, "empty"); !ok || v != "" {
		t.Fatalf("parseLimitString(empty) = (%q,%v) want (\"\",true)", v, ok)
	}
}

func TestUpdateUserPlanPayloadFields(t *testing.T) {
	payload := map[string]interface{}{
		"traffic":          "200",
		"bandwidth":        "500",
		"connection":       "300",
		"domain":           "10",
		"http_port":        "5",
		"stream_port":      "2",
		"main_domain_limit": "8",
		"custom_cc_rule":   true,
		"websocket":        false,
		"price_monthly":    99,
		"price_quarterly":  288,
		"price_yearly":     999,
	}
	updates := map[string]interface{}{}
	if v, ok := parseLimitInt(payload, "traffic"); ok {
		updates["traffic"] = v
	}
	if v, ok := parseLimitString(payload, "bandwidth"); ok {
		updates["bandwidth"] = v
	}
	if v, ok := parseLimitInt(payload, "connection"); ok {
		updates["connection"] = v
	}
	if v, ok := parseLimitInt(payload, "domain"); ok {
		updates["domain"] = v
	}
	if v, ok := parseLimitInt(payload, "http_port"); ok {
		updates["http_port"] = v
	}
	if v, ok := parseLimitInt(payload, "stream_port"); ok {
		updates["stream_port"] = v
	}
	if hasKey(payload, "main_domain_limit") {
		updates["main_domain_limit"] = getInt64(payload, "main_domain_limit")
	}
	if hasKey(payload, "custom_cc_rule") {
		updates["custom_cc_rule"] = getBool(payload, "custom_cc_rule")
	}
	if hasKey(payload, "websocket") {
		updates["websocket"] = getBool(payload, "websocket")
	}
	if hasKey(payload, "price_monthly") {
		updates["month_price"] = getInt64(payload, "price_monthly")
	}
	if hasKey(payload, "price_quarterly") {
		updates["quarter_price"] = getInt64(payload, "price_quarterly")
	}
	if hasKey(payload, "price_yearly") {
		updates["year_price"] = getInt64(payload, "price_yearly")
	}

	want := map[string]interface{}{
		"traffic":           int32(200),
		"bandwidth":         "500",
		"connection":        int32(300),
		"domain":            int32(10),
		"http_port":         int32(5),
		"stream_port":       int32(2),
		"main_domain_limit": int64(8),
		"custom_cc_rule":    true,
		"websocket":         false,
		"month_price":       int64(99),
		"quarter_price":     int64(288),
		"year_price":        int64(999),
	}
	for key, wantVal := range want {
		gotVal, ok := updates[key]
		if !ok {
			t.Fatalf("missing update key %q", key)
		}
		if gotVal != wantVal {
			t.Fatalf("updates[%q] = %v (%T) want %v (%T)", key, gotVal, gotVal, wantVal, wantVal)
		}
	}
}
