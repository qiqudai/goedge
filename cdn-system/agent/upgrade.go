package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type agentUpgradePayload struct {
	Version     string `json:"version"`
	FileName    string `json:"file_name"`
	Sha256      string `json:"sha256"`
	DownloadURL string `json:"download_url"`
}

func upgradeAgentPackage(raw string, report TaskProgressReporter) (string, error) {
	reporter := wrapProgress(report)
	var payload agentUpgradePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("invalid payload")
	}
	version := strings.TrimSpace(payload.Version)
	if version == "" {
		return "", fmt.Errorf("version is required")
	}

	downloadURL := strings.TrimSpace(payload.DownloadURL)
	if downloadURL == "" {
		downloadURL = buildAgentPackageURL(version)
	}
	if downloadURL == "" {
		return "", fmt.Errorf("download url is required")
	}

	reporter(5, "准备下载")
	tempDir, err := os.MkdirTemp("", "cdn-agent-upgrade-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	filename := strings.TrimSpace(payload.FileName)
	if filename == "" {
		filename = filepath.Base(downloadURL)
	}
	if filename == "" || filename == "/" || filename == "." {
		filename = "agent-package"
	}
	packagePath := filepath.Join(tempDir, filename)
	if err := downloadFile(downloadURL, packagePath); err != nil {
		return "", err
	}
	reporter(30, "下载完成")

	if strings.TrimSpace(payload.Sha256) != "" {
		sum := fileSHA256(packagePath)
		if sum == "" || !strings.EqualFold(sum, strings.TrimSpace(payload.Sha256)) {
			return "", fmt.Errorf("sha256 mismatch")
		}
	}

	reporter(40, "解压文件")
	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", err
	}
	if err := extractPackage(packagePath, extractDir); err != nil {
		return "", err
	}

	edgeNodePath, agentPath := locateUpgradeAssets(extractDir)
	if edgeNodePath == "" && agentPath == "" {
		return "", fmt.Errorf("invalid package layout")
	}

	reporter(55, "更新基础资源")
	if edgeNodePath != "" {
		if err := applyEdgeNodeUpgrade(edgeNodePath, runtimeRoot()); err != nil {
			return "", err
		}
	}

	reporter(75, "重启 OpenResty")
	_ = stopNginx()
	if err := startNginx(); err != nil {
		log.Printf("[Upgrade] start openresty failed: %v", err)
	}

	restartScheduled := false
	if agentPath != "" && runtime.GOOS != "windows" {
		if replaced, err := replaceAgentBinary(agentPath); err == nil && replaced {
			restartScheduled = true
		} else if err != nil {
			log.Printf("[Upgrade] replace agent failed: %v", err)
		}
	}

	reporter(100, "完成")
	if restartScheduled {
		scheduleAgentRestart(3 * time.Second)
	}
	result := map[string]interface{}{
		"version": version,
		"restart": restartScheduled,
	}
	out, _ := json.Marshal(result)
	return string(out), nil
}

func wrapProgress(report TaskProgressReporter) func(int, string) {
	if report == nil {
		return func(int, string) {}
	}
	last := -1
	return func(percent int, message string) {
		if percent <= last {
			return
		}
		if percent > 100 {
			percent = 100
		}
		last = percent
		_ = report(percent, message)
	}
}

func buildAgentPackageURL(version string) string {
	if strings.TrimSpace(API_BaseURL) == "" {
		return ""
	}
	return strings.TrimRight(API_BaseURL, "/") + "/api/v1/agent/upgrade/package?version=" + url.QueryEscape(version)
}

func downloadFile(downloadURL string, dest string) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+AuthToken)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed status=%d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmpPath := dest + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, dest)
}

func extractPackage(path string, dest string) error {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(path, dest)
	case strings.HasSuffix(lower, ".tar.gz"):
		return extractTarGz(path, dest)
	default:
		return errors.New("unsupported package format")
	}
}

func extractZip(path string, dest string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, f := range reader.File {
		targetPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return errors.New("invalid zip path")
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			return err
		}
		src.Close()
		dst.Close()
	}
	return nil
}

func extractTarGz(path string, dest string) error {
	fp, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	gz, err := gzip.NewReader(fp)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return errors.New("invalid tar path")
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func locateUpgradeAssets(root string) (string, string) {
	var edgeNodePath string
	var agentPath string
	agentName := "cdn-agent"
	if runtime.GOOS == "windows" {
		agentName = "cdn-agent.exe"
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if edgeNodePath == "" && strings.EqualFold(d.Name(), "edge-node") {
				edgeNodePath = path
			}
			return nil
		}
		if agentPath == "" && d.Name() == agentName {
			agentPath = path
		}
		return nil
	})
	return edgeNodePath, agentPath
}

func applyEdgeNodeUpgrade(srcRoot string, destRoot string) error {
	if srcRoot == "" || destRoot == "" {
		return nil
	}
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipUpgradePath(filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(path, targetPath, 0)
	})
}

func shouldSkipUpgradePath(rel string) bool {
	rel = strings.TrimPrefix(rel, "/")
	switch {
	case strings.HasPrefix(rel, "cert/"):
		return true
	case strings.HasPrefix(rel, "logs/"):
		return true
	case strings.HasPrefix(rel, "cache/"):
		return true
	case strings.HasPrefix(rel, "packages/"):
		return true
	case strings.HasPrefix(rel, "conf/dynamic/"):
		return true
	case rel == "conf/cdn_config.json":
		return true
	default:
		return false
	}
}

func copyFile(src string, dest string, perm fs.FileMode) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if perm == 0 {
		perm = info.Mode()
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func fileSHA256(path string) string {
	fp, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer fp.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fp); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func replaceAgentBinary(src string) (bool, error) {
	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		return false, err
	}
	tmp := exe + ".new"
	if err := copyFile(src, tmp, 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, exe); err != nil {
		return false, err
	}
	return true, nil
}

func scheduleAgentRestart(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		exe, err := os.Executable()
		if err != nil || strings.TrimSpace(exe) == "" {
			return
		}
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Printf("[Upgrade] restart failed: %v", err)
			return
		}
		os.Exit(0)
	}()
}
