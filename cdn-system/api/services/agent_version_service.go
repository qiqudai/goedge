package services

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(`v?(\d+(\.\d+)+)`)

func ReadAgentBinaryVersion() string {
	if v := strings.TrimSpace(os.Getenv("AGENT_VERSION")); v != "" {
		return v
	}
	path := ResolveAgentBinaryPath()
	if path == "" {
		return "unknown"
	}
	if v := readVersionFromBinary(path); v != "" {
		return v
	}
	if v := extractVersionFromName(filepath.Base(path)); v != "" {
		return v
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime().Format("20060102-150405")
	}
	return "unknown"
}

func ResolveAgentBinaryPath() string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join("agent", "cdn-agent.exe"),
		}
	} else {
		candidates = []string{
			filepath.Join("agent", "cdn-agent"),
		}
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func readVersionFromBinary(path string) string {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(info.Main.Version)
	if version == "" || version == "(devel)" {
		return ""
	}
	return version
}

func extractVersionFromName(name string) string {
	match := versionRegex.FindStringSubmatch(name)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// CompareVersion returns 1 if a>b, -1 if a<b, 0 if equal or not comparable.
func CompareVersion(a, b string) int {
	left := parseVersionSegments(a)
	right := parseVersionSegments(b)
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	max := len(left)
	if len(right) > max {
		max = len(right)
	}
	for i := 0; i < max; i++ {
		lv := 0
		rv := 0
		if i < len(left) {
			lv = left[i]
		}
		if i < len(right) {
			rv = right[i]
		}
		if lv > rv {
			return 1
		}
		if lv < rv {
			return -1
		}
	}
	return 0
}

func parseVersionSegments(raw string) []int {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			break
		}
		part = leadingDigits(part)
		if part == "" {
			break
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		out = append(out, value)
	}
	return out
}

func leadingDigits(raw string) string {
	for i, r := range raw {
		if r < '0' || r > '9' {
			return raw[:i]
		}
	}
	return raw
}
