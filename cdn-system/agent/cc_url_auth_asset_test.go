package main

import "testing"

func TestCCURLAuthMatchesCDNFlyFormula(t *testing.T) {
	ccLua := readAsset(t, "assets/lua/cc.lua")
	assertContains(t, "cc.lua url auth", ccLua,
		"normalize_auth_uri",
		"return string.lower(path)",
		"sign:match(\"^([0-9]+)%-([^%-]+)%-([^%-]+)%-([0-9a-fA-F]+)$\")",
		"path .. \"-\" .. ts .. \"-\" .. rand .. \"-\" .. uid .. \"-\" .. key",
		"raw = raw .. \"-\" .. tostring(ip or \"\")",
		"key .. path .. ts",
		"raw = raw .. tostring(ip or \"\")",
	)
}
