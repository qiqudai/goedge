package main

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setCacheTestEnv(t *testing.T, cacheDir string) {
	t.Helper()
	prevWorkDir := WorkDir
	prevNginx := LocalNginxConfig
	WorkDir = t.TempDir()
	LocalNginxConfig = &edgeNginxConfig{
		HTTP: map[string]interface{}{
			"proxy_cache_dir": cacheDir,
		},
	}
	t.Cleanup(func() {
		WorkDir = prevWorkDir
		LocalNginxConfig = prevNginx
	})
}

func writeCacheFileByKey(t *testing.T, cacheDir, key string, withKeyHeader bool) string {
	t.Helper()
	sum := md5.Sum([]byte(key))
	hash := fmt.Sprintf("%x", sum)
	path := filepath.Join(cacheDir, hash[0:1], hash[1:3], hash)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir cache path failed: %v", err)
	}
	content := "dummy-cache"
	if withKeyHeader {
		content = "META\nKEY: " + key + "\nBODY\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cache file failed: %v", err)
	}
	return path
}

func TestBuildPurgeCacheKeys_CoversRawAndNormalizedQuery(t *testing.T) {
	u, err := url.Parse("https://Example.COM/static/a.png?b=2&a=1")
	if err != nil {
		t.Fatalf("parse url failed: %v", err)
	}
	keys := buildPurgeCacheKeys(u)
	joined := strings.Join(keys, "\n")
	expectContains := []string{
		"example.com/static/a.png",
		"example.com/static/a.png?b=2&a=1",
		"example.com/static/a.png?a=1&b=2",
	}
	for _, expected := range expectContains {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected key %q in %v", expected, keys)
		}
	}
}

func TestPurgeURL_UsesResolvedCacheDirAndPurgesVariants(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	setCacheTestEnv(t, cacheDir)

	raw := "https://Example.com/static/a.png?b=2&a=1"
	u, _ := url.Parse(raw)
	keys := buildPurgeCacheKeys(u)
	for _, key := range keys {
		writeCacheFileByKey(t, cacheDir, key, false)
	}

	if err := purgeURL(raw); err != nil {
		t.Fatalf("purgeURL failed: %v", err)
	}
	for _, key := range keys {
		sum := md5.Sum([]byte(key))
		hash := fmt.Sprintf("%x", sum)
		path := filepath.Join(cacheDir, hash[0:1], hash[1:3], hash)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("cache file should be removed for key %q", key)
		}
	}
}

func TestPurgeDomains_RemovesOnlyTargetDomain(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	setCacheTestEnv(t, cacheDir)

	targetPath := writeCacheFileByKey(t, cacheDir, "tt.wmzih.cn/index.html", true)
	otherPath := writeCacheFileByKey(t, cacheDir, "other.example.com/index.html", true)

	if err := purgeDomains([]string{"TT.WMZIH.CN"}); err != nil {
		t.Fatalf("purgeDomains failed: %v", err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("target domain cache should be removed")
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("other domain cache should remain: %v", err)
	}
}

func TestParseCacheDirTarget(t *testing.T) {
	target, ok := parseCacheDirTarget("https://TT.WMZIH.CN/assets")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if target.host != "tt.wmzih.cn" {
		t.Fatalf("unexpected host: %s", target.host)
	}
	if target.pathPrefix != "/assets/" {
		t.Fatalf("unexpected path prefix: %s", target.pathPrefix)
	}
}
