package main

import (
	"strings"
	"testing"
)

func TestGuardPassCookieDoesNotRequireLocalState(t *testing.T) {
	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua pass cookie", guardLua,
		"return true",
		"local s = store()",
		"s:set(pass_state_key",
		"parts[3] ~= (host or \"\") or parts[4] ~= (ip or \"\")",
		"parts[7] ~= browser_id or parts[8] ~= fingerprint or parts[9] ~= ua_sig",
	)
	if strings.Contains(guardLua, "s:get(pass_state_key") {
		t.Fatalf("guard pass cookies must not require local shared-dict state; cross-node validation would fail")
	}
}

func TestGuardChallengeStateIsSignedAndStateless(t *testing.T) {
	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua challenge state", guardLua,
		"COOKIE_GUARD_STATE",
		"sign_state_payload",
		"load_state_cookie(host, ip, filter_type, filter_id)",
		"load_state_cookie_by_nonce",
		"save_state_cookie(st)",
		"clear_state_cookie()",
		"constant_time_eq(sig, sign_state_payload(payload))",
		"validate_state(st, host, ip, filter_type, filter_id)",
		"http_only = true",
	)
}

func TestGuardSecretUsesPlatformKey(t *testing.T) {
	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua secret", guardLua,
		"waf.secret_key",
		"guard:secret",
	)
	for _, forbidden := range []string{
		"FIXED_GUARD_SECRET",
		"cdn-agent-guard-v4-fixed",
	} {
		if strings.Contains(guardLua, forbidden) {
			t.Fatalf("guard.lua must not use hardcoded public guard secret, found %q", forbidden)
		}
	}
}

func TestGuardPassCookieIsBoundToClientAndRequestScope(t *testing.T) {
	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua pass binding", guardLua,
		"parts[3] ~= (host or \"\") or parts[4] ~= (ip or \"\")",
		"parts[5] ~= normalize_type(filter_type)",
		"parts[6] ~= tostring(filter_id or 0)",
		"parts[7] ~= browser_id or parts[8] ~= fingerprint or parts[9] ~= ua_sig",
		"valid_browser_id(browser_id)",
		"valid_fingerprint(fingerprint)",
		"constant_time_eq(parts[11], expected)",
	)
}

func TestGuardCaptchaBypassHardening(t *testing.T) {
	guardLua := readAsset(t, "assets/lua/guard.lua")
	assertContains(t, "guard.lua captcha hardening", guardLua,
		"type(move) ~= \"table\" or #move < 3",
		"last_ts-first_ts < 500",
		"return false, \"slide distance\"",
		"local expected = slider - btn",
		"end_x - start_x < expected - 3",
		"slide non-monotonic",
		"click too fast",
		"auto verify too fast",
		"secret() .. \"|\" .. tostring(nonce8 or \"\")",
		"local answer = tonumber(st.rotate.answer) or 0",
		"abs_diff_mod360(deg, answer) <= ROTATE_TOLERANCE_DEG",
	)
	for _, forbidden := range []string{
		"tonumber(st.rotate.degree)",
		"abs_diff_mod360(deg, degree)",
	} {
		if strings.Contains(guardLua, forbidden) {
			t.Fatalf("guard.lua rotate captcha must not accept the image degree as a valid answer, found %q", forbidden)
		}
	}
}
