package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepareHTTPGlobalConfigTestEnv(t *testing.T) string {
	t.Helper()
	prevWorkDir := WorkDir
	prevAPI := API_BaseURL
	WorkDir = t.TempDir()
	API_BaseURL = ""
	t.Cleanup(func() {
		WorkDir = prevWorkDir
		API_BaseURL = prevAPI
	})

	root := runtimeRoot()
	if err := os.MkdirAll(filepath.Join(root, "conf", "dynamic"), 0o755); err != nil {
		t.Fatalf("mkdir dynamic failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "logs"), 0o755); err != nil {
		t.Fatalf("mkdir logs failed: %v", err)
	}
	return root
}

func TestWriteHTTPGlobalConfig_DisableCachePathWhenNoCacheDomain(t *testing.T) {
	root := prepareHTTPGlobalConfigTestEnv(t)
	if err := writeHTTPGlobalConfig(nil, false); err != nil {
		t.Fatalf("writeHTTPGlobalConfig failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "conf", "dynamic", "http_global.conf"))
	if err != nil {
		t.Fatalf("read http_global.conf failed: %v", err)
	}
	out := string(data)
	if strings.Contains(out, "proxy_cache_path ") {
		t.Fatalf("proxy_cache_path should not be generated when cache is disabled")
	}
	if strings.Contains(out, "proxy_cache_methods ") {
		t.Fatalf("proxy_cache_methods should not be generated when cache is disabled")
	}
	if strings.Contains(out, "proxy_cache_revalidate ") {
		t.Fatalf("proxy_cache_revalidate should not be generated when cache is disabled")
	}
}

func TestWriteHTTPGlobalConfig_GenerateCachePathWhenCacheEnabled(t *testing.T) {
	root := prepareHTTPGlobalConfigTestEnv(t)
	if err := writeHTTPGlobalConfig(nil, true); err != nil {
		t.Fatalf("writeHTTPGlobalConfig failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "conf", "dynamic", "http_global.conf"))
	if err != nil {
		t.Fatalf("read http_global.conf failed: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "proxy_cache_path ") {
		t.Fatalf("proxy_cache_path should be generated when cache is enabled")
	}
	if !strings.Contains(out, "proxy_cache_methods GET HEAD;") {
		t.Fatalf("proxy_cache_methods default missing when cache is enabled")
	}
	if !strings.Contains(out, "proxy_cache_revalidate on;") {
		t.Fatalf("proxy_cache_revalidate default missing when cache is enabled")
	}
}
