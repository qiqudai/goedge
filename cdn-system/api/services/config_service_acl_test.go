package services

import "testing"

func TestParseACLRulesWithConditions(t *testing.T) {
	raw := `{
		"rules": [
			{
				"conditions": [
					{"item": "uri", "operator": "eq", "value": "/api"},
					{"item": "ip", "operator": "eq", "value": "1.1.1.1"}
				],
				"action": "allow"
			},
			{
				"conditions": [{"item": "country", "operator": "eq", "value": "CN"}],
				"action": "deny",
				"deny_status": 403,
				"redirect_url": "https://blocked.example"
			}
		],
		"default_deny_status": 403,
		"default_redirect_url": "https://default.example"
	}`
	rules := parseACLRules(raw)
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
	if len(rules[0].Conditions) != 2 || rules[0].Conditions[0].Item != "uri" {
		t.Fatalf("first rule conditions = %+v", rules[0].Conditions)
	}
	if rules[1].Action != "deny" || rules[1].RedirectURL != "https://blocked.example" {
		t.Fatalf("second rule = %+v", rules[1])
	}
	status, redirect := parseACLDefaultDenyMeta(raw)
	if status != 403 || redirect != "https://default.example" {
		t.Fatalf("default deny meta = (%d, %q)", status, redirect)
	}
}

func TestParseACLRulesLegacyFlatIP(t *testing.T) {
	raw := `[{"ip":"8.8.8.8","action":"deny"}]`
	rules := parseACLRules(raw)
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0].Action != "deny" || len(rules[0].Conditions) != 1 || rules[0].Conditions[0].Value != "8.8.8.8" {
		t.Fatalf("rule = %+v", rules[0])
	}
}

func TestParseACLRulesLegacyAPIFormat(t *testing.T) {
	raw := `[{
		"acl_action": "allow",
		"acl_matcher": {
			"country_iso_code": {"operator": "=", "value": "CN"},
			"uri": {"operator": "contain", "value": "/api"}
		}
	}]`
	rules := parseACLRules(raw)
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0].Action != "allow" {
		t.Fatalf("action = %q, want allow", rules[0].Action)
	}
	if len(rules[0].Conditions) != 2 {
		t.Fatalf("conditions = %+v", rules[0].Conditions)
	}
	if rules[0].Conditions[0].Item != "country" || rules[0].Conditions[1].Operator != "contains" {
		t.Fatalf("normalized conditions = %+v", rules[0].Conditions)
	}
}

func TestNormalizeACLActionRejectAlias(t *testing.T) {
	if got := normalizeACLAction("reject"); got != "deny" {
		t.Fatalf("normalizeACLAction(reject) = %q, want deny", got)
	}
}

func TestResolveCCAutoSwitchRuleIDAliases(t *testing.T) {
	if got := resolveCCAutoSwitchRuleID("close"); got != 10002 {
		t.Fatalf("close = %d, want 10002", got)
	}
	if got := resolveCCAutoSwitchRuleID("lenient"); got != 6 {
		t.Fatalf("lenient = %d, want 6", got)
	}
}

func TestExtractCCAutoSwitchDisabled(t *testing.T) {
	settings := map[string]interface{}{
		"security": map[string]interface{}{
			"auto_switch": `{"enable":false,"qps":50,"rule":"strict"}`,
		},
	}
	if got := extractCCAutoSwitch(settings); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestExtractCCAutoSwitchEnabled(t *testing.T) {
	settings := map[string]interface{}{
		"security": map[string]interface{}{
			"auto_switch": map[string]interface{}{
				"enable": true,
				"qps":    50,
				"rule":   "captcha",
			},
		},
	}
	got := extractCCAutoSwitch(settings)
	if got == nil {
		t.Fatal("expected auto switch config")
	}
	if !got.Enable || got.QPS != 50 || got.RuleID != 4 {
		t.Fatalf("auto switch = %+v", got)
	}
}
