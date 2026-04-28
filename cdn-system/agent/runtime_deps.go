package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type linuxDistroInfo struct {
	ID      string
	Version string
}

func selectOpenRestyAsset() string {
	info := detectLinuxDistroInfo()
	switch info.ID {
	case "ubuntu":
		if strings.HasPrefix(info.Version, "24.04") {
			return "assets/openresty-ubuntu24.04.zip"
		}
		if strings.HasPrefix(info.Version, "22.04") {
			return "assets/openresty-ubuntu22.04.zip"
		}
	}
	return "assets/openresty.zip"
}

func detectLinuxDistroInfo() linuxDistroInfo {
	if runtime.GOOS != "linux" {
		return linuxDistroInfo{}
	}
	candidates := []string{"/etc/os-release", "/usr/lib/os-release"}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			info := parseOSRelease(string(data))
			if info.ID != "" || info.Version != "" {
				return info
			}
		}
	}
	return linuxDistroInfo{}
}

func parseOSRelease(raw string) linuxDistroInfo {
	info := linuxDistroInfo{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "ID":
			info.ID = strings.ToLower(value)
		case "VERSION_ID":
			info.Version = value
		}
	}
	return info
}

func ensureRuntimeDependencies() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if os.Geteuid() != 0 {
		return nil
	}
	if strings.TrimSpace(NginxBinPath) == "" {
		return nil
	}

	missing, err := detectMissingSharedLibraries(NginxBinPath)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	info := detectLinuxDistroInfo()
	pkgs := runtimePackagesForMissingLibraries(info, missing)
	if len(pkgs) == 0 {
		return nil
	}

	scriptPath, err := materializeRuntimeBootstrapScript()
	if err != nil {
		return err
	}
	log.Printf("[Info] Auto-install runtime deps: distro=%s version=%s missing=%v pkgs=%v", info.ID, info.Version, missing, pkgs)
	if out, err := exec.Command("bash", append([]string{scriptPath}, pkgs...)...).CombinedOutput(); err != nil {
		return fmt.Errorf("runtime dependency install failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func detectMissingSharedLibraries(binPath string) ([]string, error) {
	if strings.TrimSpace(binPath) == "" {
		return nil, errors.New("binary path is empty")
	}
	cmd := exec.Command("ldd", binPath)
	setNginxEnv(cmd)
	out, err := cmd.CombinedOutput()
	missing := parseMissingSharedLibraries(string(out))
	if len(missing) > 0 {
		return uniqueStrings(missing), nil
	}
	if err != nil {
		return nil, fmt.Errorf("ldd failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil, nil
}

func parseMissingSharedLibraries(output string) []string {
	if strings.TrimSpace(output) == "" {
		return []string{}
	}
	missing := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "not found") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		lib := strings.TrimSpace(fields[0])
		if lib == "" {
			continue
		}
		missing = append(missing, lib)
	}
	return uniqueStrings(missing)
}

func runtimePackagesForMissingLibraries(info linuxDistroInfo, missing []string) []string {
	if len(missing) == 0 {
		return []string{}
	}

	release := strings.ToLower(strings.TrimSpace(info.ID))
	pkgs := make([]string, 0, len(missing))
	for _, lib := range missing {
		lib = strings.TrimSpace(lib)
		switch {
		case strings.Contains(lib, "libpcre.so.3"), strings.Contains(lib, "libpcre.so.1"):
			if release == "ubuntu" || release == "debian" || release == "" {
				pkgs = append(pkgs, "libpcre3")
			} else {
				pkgs = append(pkgs, "pcre")
			}
		case strings.Contains(lib, "libluajit-5.1.so.2"):
			if release == "ubuntu" || release == "debian" || release == "" {
				pkgs = append(pkgs, "libluajit-5.1-2")
			} else {
				pkgs = append(pkgs, "luajit")
			}
		}
	}
	return uniqueStrings(pkgs)
}

func materializeRuntimeBootstrapScript() (string, error) {
	scriptBytes, err := assetsFS.ReadFile("assets/scripts/bootstrap_runtime_deps.sh")
	if err != nil {
		return "", err
	}
	scriptPath := filepath.Join(runtimeRoot(), "scripts", "bootstrap_runtime_deps.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(scriptPath, scriptBytes, 0o755); err != nil {
		return "", err
	}
	return scriptPath, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
