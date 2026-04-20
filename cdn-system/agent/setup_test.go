package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchNginxConfigPathsReplacesCacheAndGeoPaths(t *testing.T) {
	rootDir := filepath.Join(string(filepath.Separator), "tmp", "edge-node")
	content := strings.Join([]string{
		"proxy_cache_path /var/cache/nginx;",
		`file = "/opt/cdn-agent/data/ip2region.xdb"`,
		`file = "/www/server/go_project/openresty/cdn-system/agent/edge-node/data/ip2region.xdb"`,
	}, "\n")

	patched := patchNginxConfigPaths(content, rootDir)

	if strings.Contains(patched, "/var/cache/nginx") {
		t.Fatalf("cache path placeholder was not replaced: %s", patched)
	}
	if strings.Contains(patched, "/opt/cdn-agent/data/ip2region.xdb") {
		t.Fatalf("agent geo placeholder was not replaced: %s", patched)
	}
	if strings.Contains(patched, "/www/server/go_project/openresty/cdn-system/agent/edge-node/data/ip2region.xdb") {
		t.Fatalf("legacy geo placeholder was not replaced: %s", patched)
	}

	wantGeo := filepath.ToSlash(filepath.Join(rootDir, "data", "ip2region.xdb"))
	if !strings.Contains(patched, wantGeo) {
		t.Fatalf("patched content missing geo path %q: %s", wantGeo, patched)
	}
}
