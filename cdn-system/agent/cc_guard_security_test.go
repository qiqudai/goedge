package main

import (
	"os"
	"strings"
	"testing"
)

func readAsset(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, name, content string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing %q", name, needle)
		}
	}
}

func assertFilesEqual(t *testing.T, left, right string) {
	t.Helper()
	leftData, err := os.ReadFile(left)
	if err != nil {
		t.Fatalf("read %s: %v", left, err)
	}
	rightData, err := os.ReadFile(right)
	if err != nil {
		t.Fatalf("read %s: %v", right, err)
	}
	if string(leftData) != string(rightData) {
		t.Fatalf("%s and %s must stay synchronized", left, right)
	}
}

func TestCCGuardSecurityAssets(t *testing.T) {
	assertFilesEqual(t, "assets/lua/guard.lua", "edge-node/lua/guard.lua")
	assertFilesEqual(t, "assets/lua/cc.lua", "edge-node/lua/cc.lua")
	assertFilesEqual(t, "assets/lua/acl.lua", "edge-node/lua/acl.lua")
	assertFilesEqual(t, "assets/lua/ip_block.lua", "edge-node/lua/ip_block.lua")
	assertFilesEqual(t, "assets/lua/access_guard.lua", "edge-node/lua/access_guard.lua")
	assertFilesEqual(t, "assets/lua/cc_matcher.lua", "edge-node/lua/cc_matcher.lua")
	assertFilesEqual(t, "assets/lua/cc_stats.lua", "edge-node/lua/cc_stats.lua")
	assertFilesEqual(t, "assets/conf/guard/fingerprint.js", "edge-node/conf/guard/fingerprint.js")

	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua", guardLua,
		"__cdn_guard_bid",
		"__cdn_guard_fp",
		"__cdn_guard_state",
		"waf.secret_key",
		"guard:secret",
		"guard:pass:",
		"load_state_cookie",
		"save_state_cookie",
		"parts[1] ~= \"v3\"",
		"verify_pass_cookie_value",
		"is_common_non_browser_request",
		"browser_verify_auto",
		"click_filter",
		"slide_filter",
		"captcha_filter",
		"delay_jump_filter",
		"rotate_filter",
		"302_challenge",
	)
	if strings.Contains(guardLua, "parts[1] == \"v1\"") || strings.Contains(guardLua, "parts[1] ~= \"v2\"") {
		t.Fatalf("guard.lua must not accept legacy stateless guard pass cookies")
	}

	ccLua := readAsset(t, "assets/lua/cc.lua")
	assertContains(t, "cc.lua", ccLua,
		"is_common_non_browser_request",
		"block_non_browser",
		"custom_cc_rules",
		"cc_matcher",
		"url_auth",
		"rule_should_stop",
		"finalize_allow",
		"breakMatch",
		"probe_filter",
		"resolve_effective_rule_id",
		"cc_auto_switch",
		"enforce_action",
		"blacklist_on_trigger",
		"cc_ipset",
		"cc_exit",
	)

	aclLua := readAsset(t, "assets/lua/acl.lua")
	assertContains(t, "acl.lua", aclLua,
		"cc_matcher.match_data",
		"acl_default_action",
		"acl_default_deny_status",
		"acl_default_redirect_url",
	)

	accessGuard := readAsset(t, "assets/lua/access_guard.lua")
	assertContains(t, "access_guard.lua", accessGuard,
		"acl.check(domain_conf",
		"cc.check(domain_conf",
	)

	ccMatcher := readAsset(t, "assets/lua/cc_matcher.lua")
	assertContains(t, "cc_matcher.lua", ccMatcher,
		"request_uri",
		"uri_no_args",
		"ua_count",
		"404_count",
		"ip_range",
		"accept_language",
		"header_accept_language",
	)

	fpScript := readAsset(t, "assets/conf/guard/fingerprint.js")
	assertContains(t, "fingerprint.js", fpScript,
		"buildFingerprint",
		"__cdn_guard_bid",
		"__cdn_guard_fp",
	)

	for _, page := range []string{
		"browser_verify_auto.html",
		"delay_jump.html",
		"captcha.html",
		"click.html",
		"slide.html",
		"rotate.html",
	} {
		content := readAsset(t, "assets/conf/guard/"+page)
		assertContains(t, page, content, "/_guard/fingerprint.js")
		assertFilesEqual(t, "assets/conf/guard/"+page, "edge-node/conf/guard/"+page)
	}
}
