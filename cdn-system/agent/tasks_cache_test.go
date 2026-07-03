package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestPurgeURL_RemovesCacheFileByFullURLKeyHeader(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	setCacheTestEnv(t, cacheDir)

	targetPath := writeCacheFileByKey(t, cacheDir, "https://example.com/static/a.png?b=2&a=1", true)
	otherPath := writeCacheFileByKey(t, cacheDir, "https://example.com/static/other.png?b=2&a=1", true)

	if err := purgeURL("https://example.com/static/a.png?b=2&a=1"); err != nil {
		t.Fatalf("purgeURL failed: %v", err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("full URL cache key should be removed")
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("unmatched full URL cache key should remain: %v", err)
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

func TestPurgeDomains_WildcardRemovesAllSubdomainLevelsOnly(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	setCacheTestEnv(t, cacheDir)

	firstLevelPath := writeCacheFileByKey(t, cacheDir, "630.example.com/static/app.js", true)
	deepLevelPath := writeCacheFileByKey(t, cacheDir, "https://img.630.example.com/static/app.css?v=1", true)
	rootPath := writeCacheFileByKey(t, cacheDir, "example.com/static/app.js", true)
	suffixAttackPath := writeCacheFileByKey(t, cacheDir, "badexample.com/static/app.js", true)
	otherPath := writeCacheFileByKey(t, cacheDir, "630.other.com/static/app.js", true)

	if err := purgeDomains([]string{"*.example.com"}); err != nil {
		t.Fatalf("purgeDomains failed: %v", err)
	}
	if _, err := os.Stat(firstLevelPath); !os.IsNotExist(err) {
		t.Fatalf("first-level wildcard subdomain cache should be removed")
	}
	if _, err := os.Stat(deepLevelPath); !os.IsNotExist(err) {
		t.Fatalf("deep wildcard subdomain cache should be removed")
	}
	for label, path := range map[string]string{
		"root domain":        rootPath,
		"suffix attack host": suffixAttackPath,
		"other domain":       otherPath,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s cache should remain: %v", label, err)
		}
	}
}

func TestPurgeDirs_RemovesFullURLCacheKeys(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	setCacheTestEnv(t, cacheDir)

	targetPath := writeCacheFileByKey(t, cacheDir, "https://example.com/static/assets/app.js?v=1", true)
	otherPath := writeCacheFileByKey(t, cacheDir, "https://example.com/static/other/app.js?v=1", true)

	if err := purgeDirs([]string{"https://example.com/static/assets/"}); err != nil {
		t.Fatalf("purgeDirs failed: %v", err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("directory full URL cache key should be removed")
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("non-matching directory cache key should remain: %v", err)
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

func TestPreheatURL_UsesLocalPortAndOriginalHost(t *testing.T) {
	gotHost := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server url failed: %v", err)
	}
	raw := fmt.Sprintf("http://example.com:%s/preheat.js?x=1", u.Port())
	if err := preheatURL(raw); err != nil {
		t.Fatalf("preheatURL failed: %v", err)
	}
	if gotHost != "example.com:"+u.Port() {
		t.Fatalf("unexpected preheat host header: %q", gotHost)
	}
}
