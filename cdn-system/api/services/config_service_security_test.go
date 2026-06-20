package services

import "testing"

func TestBuildSearchEngineOriginConditionRequiresSpiderIPAllowlist(t *testing.T) {
	cond := buildSearchEngineOriginConditionWithAllowlist(map[string]interface{}{
		"search_engine_origin":    true,
		"search_engine_origin_ip": "10.0.0.10",
	}, "")
	if cond != nil {
		t.Fatalf("search engine origin must not fall back to spoofable user-agent matching: %#v", cond)
	}
}

func TestSiteConfigGroupMatchesIncludesEnabledBackupOnly(t *testing.T) {
	groups := int64Set([]int64{6})
	if !siteConfigGroupMatches(10, 6, true, groups) {
		t.Fatalf("expected enabled backup group to match")
	}
	if siteConfigGroupMatches(10, 6, false, groups) {
		t.Fatalf("disabled backup group must not match")
	}
	if !siteConfigGroupMatches(6, 0, false, groups) {
		t.Fatalf("primary group should match")
	}
}

func TestBuildResponseHeaderMapSkipsOriginLeakHeaders(t *testing.T) {
	got := buildResponseHeaderMap(map[string]interface{}{
		"advanced": map[string]interface{}{
			"cdn_headers": []interface{}{
				map[string]interface{}{"name": "X-Origin-IP", "value": "10.0.0.10"},
				map[string]interface{}{"name": "X-Safe", "value": "ok"},
			},
		},
	})
	if _, ok := got["X-Origin-IP"]; ok {
		t.Fatalf("origin leak response header must be filtered: %#v", got)
	}
	if got["X-Safe"] != "ok" {
		t.Fatalf("safe response header should be preserved: %#v", got)
	}
}
