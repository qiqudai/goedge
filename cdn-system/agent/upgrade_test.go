package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchesUpgradeAgentBinaryName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "cdn-agent", want: true},
		{name: "cdn-agent-linux-amd64", want: true},
		{name: "cdn-agent-darwin-arm64", want: true},
		{name: "cdn-agent.exe", want: false},
		{name: "nginx", want: false},
		{name: "", want: false},
	}

	for _, tt := range tests {
		if got := matchesUpgradeAgentBinaryName(tt.name); got != tt.want {
			t.Fatalf("matchesUpgradeAgentBinaryName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestLocateUpgradeAssetsFindsAgentArtifacts(t *testing.T) {
	root := t.TempDir()
	edgeNodeDir := filepath.Join(root, "pkg", "edge-node")
	if err := os.MkdirAll(edgeNodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentPath := filepath.Join(root, "pkg", "cdn-agent-linux-amd64")
	if err := os.WriteFile(agentPath, []byte("agent"), 0o755); err != nil {
		t.Fatal(err)
	}

	gotEdgeNode, gotAgent := locateUpgradeAssets(root)
	if gotEdgeNode != edgeNodeDir {
		t.Fatalf("edge-node path mismatch: got %q want %q", gotEdgeNode, edgeNodeDir)
	}
	if gotAgent != agentPath {
		t.Fatalf("agent path mismatch: got %q want %q", gotAgent, agentPath)
	}
}

func TestPostProcessRuntimeUpgradePatchesNginxConfigPaths(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join([]string{
		"proxy_cache_path /var/cache/nginx;",
		`file = "/opt/cdn-agent/data/ip2region.xdb"`,
	}, "\n")
	confPath := filepath.Join(confDir, "nginx.conf")
	if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := postProcessRuntimeUpgrade(root); err != nil {
		t.Fatal(err)
	}

	patched, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(patched)
	if strings.Contains(got, "/var/cache/nginx") {
		t.Fatalf("cache path placeholder was not replaced: %s", got)
	}
	if strings.Contains(got, "/opt/cdn-agent/data/ip2region.xdb") {
		t.Fatalf("geo path placeholder was not replaced: %s", got)
	}
}

func TestApplyEdgeNodeUpgradeOverwritesBundledConfAndSkipsRuntime(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")

	files := map[string]string{
		filepath.Join(src, "conf", "nginx.conf"):                   "new nginx",
		filepath.Join(src, "conf", "dynamic", "main.conf"):         "new main",
		filepath.Join(src, "openresty", "nginx", "sbin", "nginx"):  "new runtime",
		filepath.Join(dest, "conf", "nginx.conf"):                  "old nginx",
		filepath.Join(dest, "conf", "dynamic", "main.conf"):        "old main",
		filepath.Join(dest, "openresty", "nginx", "sbin", "nginx"): "old runtime",
		filepath.Join(dest, "conf", "cdn_config.json"):             "live config",
		filepath.Join(src, "conf", "cdn_config.json"):              "package config",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := applyEdgeNodeUpgrade(src, dest); err != nil {
		t.Fatal(err)
	}

	assertFileContent := func(path, want string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("%s = %q, want %q", path, string(data), want)
		}
	}

	assertFileContent(filepath.Join(dest, "conf", "nginx.conf"), "new nginx")
	assertFileContent(filepath.Join(dest, "conf", "dynamic", "main.conf"), "new main")
	assertFileContent(filepath.Join(dest, "openresty", "nginx", "sbin", "nginx"), "old runtime")
	assertFileContent(filepath.Join(dest, "conf", "cdn_config.json"), "live config")
}

func TestExtractZipRestoresSymlinkEntries(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("symlink extraction test is unix-only")
	}

	root := t.TempDir()
	zipPath := filepath.Join(root, "agent.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)

	targetHeader := &zip.FileHeader{Name: "edge-node/openresty/luajit/lib/libluajit-5.1.so.2.1.0"}
	targetHeader.SetMode(0o755)
	targetWriter, err := zw.CreateHeader(targetHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targetWriter.Write([]byte("real-library")); err != nil {
		t.Fatal(err)
	}

	linkHeader := &zip.FileHeader{Name: "edge-node/openresty/luajit/lib/libluajit-5.1.so.2"}
	linkHeader.SetMode(os.ModeSymlink | 0o777)
	linkWriter, err := zw.CreateHeader(linkHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := linkWriter.Write([]byte("libluajit-5.1.so.2.1.0")); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "extract")
	if err := extractZip(zipPath, dest); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(dest, "edge-node", "openresty", "luajit", "lib", "libluajit-5.1.so.2")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got mode %v", info.Mode())
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if target != "libluajit-5.1.so.2.1.0" {
		t.Fatalf("symlink target mismatch: got %q", target)
	}
}

func TestValidateAgentBinaryForCurrentPlatformCurrentExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAgentBinaryForCurrentPlatform(exe); err != nil {
		t.Fatalf("expected current executable to validate, got %v", err)
	}
}
