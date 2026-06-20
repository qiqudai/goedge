package services

import "testing"

func TestParseEdgeCCRuleItem(t *testing.T) {
	item := map[string]interface{}{
		"matcher_id": float64(12),
		"filter1_id": float64(34),
		"filter2_id": float64(56),
		"action":     "block",
		"mode":       "stop",
		"is_on":      false,
	}
	got := parseEdgeCCRuleItem(item)
	if got.MatcherID != 12 {
		t.Fatalf("MatcherID = %d, want 12", got.MatcherID)
	}
	if got.FilterID != 34 {
		t.Fatalf("FilterID = %d, want 34 from filter1_id", got.FilterID)
	}
	if got.Filter2ID != 56 {
		t.Fatalf("Filter2ID = %d, want 56 from filter2_id", got.Filter2ID)
	}
	if got.Action != "block" {
		t.Fatalf("Action = %q, want block", got.Action)
	}
	if got.Mode != "stop" {
		t.Fatalf("Mode = %q, want stop", got.Mode)
	}
	if got.Enabled {
		t.Fatalf("Enabled = true, want false")
	}
}

func TestParseEdgeCCRuleItemLegacyFields(t *testing.T) {
	item := map[string]interface{}{
		"matcher": float64(3),
		"filter1": float64(8),
		"filter2": float64(9),
		"state":   true,
	}
	got := parseEdgeCCRuleItem(item)
	if got.MatcherID != 3 || got.FilterID != 8 || got.Filter2ID != 9 || !got.Enabled {
		t.Fatalf("legacy parse failed: %+v", got)
	}
}

func TestExtractCustomCCRules(t *testing.T) {
	settings := map[string]interface{}{
		"security": map[string]interface{}{
			"custom_rules": []interface{}{
				map[string]interface{}{
					"action": "allow",
					"on":     true,
					"matchers": []interface{}{
						map[string]interface{}{"key": "uri", "operator": "eq", "value": "/api"},
					},
				},
			},
		},
	}
	rules := extractCustomCCRules(settings)
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0]["action"] != "allow" {
		t.Fatalf("action = %v, want allow", rules[0]["action"])
	}
	if rules[0]["on"] != true {
		t.Fatalf("on = %v, want true", rules[0]["on"])
	}
}

func TestExtractCustomCCRulesPreservesOffSwitch(t *testing.T) {
	settings := map[string]interface{}{
		"security": map[string]interface{}{
			"custom_rules": []interface{}{
				map[string]interface{}{
					"action": "allow",
					"on":     false,
					"matchers": []interface{}{
						map[string]interface{}{"key": "uri", "operator": "eq", "value": "/api"},
					},
				},
			},
		},
	}
	rules := extractCustomCCRules(settings)
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0]["on"] != false {
		t.Fatalf("on = %v, want false", rules[0]["on"])
	}
}
