package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	lowDiskFreePercent  = 5
	lowDiskFreeBytes    = 2 * 1024 * 1024 * 1024
	diskCleanupCooldown = 30 * time.Minute
)

var (
	diskCleanupMu     sync.Mutex
	lastDiskCleanupAt time.Time
)

func handleLowDiskCleanup() {
	strategy := getNodeLogCleanStrategy()
	if strategy == "" || strategy == "none" {
		return
	}

	root := runtimeRoot()
	if strings.TrimSpace(root) == "" {
		return
	}

	space, err := getDiskSpace(root)
	if err != nil {
		log.Printf("[Warn] Low disk check failed: %v", err)
		return
	}
	if !isLowDisk(space) {
		return
	}

	if !allowDiskCleanup() {
		return
	}

	log.Printf("[Warn] Low disk detected (free=%d total=%d). Strategy=%s", space.free, space.total, strategy)

	switch strategy {
	case "log_only":
		purgeLogFiles()
	case "log_cache":
		purgeLogFiles()
		purgeCacheFiles()
	default:
		return
	}
}

func getNodeLogCleanStrategy() string {
	localConfigMu.RLock()
	cfg := LocalWAFConfig
	localConfigMu.RUnlock()
	if cfg == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(cfg.NodeLogCleanStrategy))
}

func allowDiskCleanup() bool {
	diskCleanupMu.Lock()
	defer diskCleanupMu.Unlock()
	if !lastDiskCleanupAt.IsZero() && time.Since(lastDiskCleanupAt) < diskCleanupCooldown {
		return false
	}
	lastDiskCleanupAt = time.Now()
	return true
}

func isLowDisk(space diskSpace) bool {
	if space.total == 0 {
		return false
	}
	freePercent := int(float64(space.free) / float64(space.total) * 100)
	return freePercent <= lowDiskFreePercent || space.free <= lowDiskFreeBytes
}

func purgeLogFiles() {
	dirs := resolveLogDirs()
	for _, dir := range dirs {
		if dir == "" || !isSafeCleanupPath(dir) {
			continue
		}
		removeDirContents(dir, true)
	}
	resetLogOffsetFiles(dirs)
}

func resetLogOffsetFiles(dirs []string) {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		for _, name := range []string{"access.offset", "stream_access.offset"} {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

func purgeCacheFiles() {
	dir := resolveCacheDir()
	if dir == "" || !isSafeCleanupPath(dir) {
		return
	}
	removeDirContents(dir, false)
}

func resolveLogDirs() []string {
	dirs := make(map[string]struct{})

	accessLog, _ := getAccessLogPaths()
	if accessLog != "" {
		dirs[filepath.Dir(accessLog)] = struct{}{}
	}

	if dir, _ := getLogStorageSettings(); strings.TrimSpace(dir) != "" {
		dirs[dir] = struct{}{}
	}

	out := make([]string, 0, len(dirs))
	for dir := range dirs {
		out = append(out, dir)
	}
	return out
}

func resolveCacheDir() string {
	root := runtimeRoot()
	if strings.TrimSpace(root) == "" {
		return ""
	}

	cacheDir := filepath.Join(root, "cache")
	localConfigMu.RLock()
	nginxCfg := LocalNginxConfig
	localConfigMu.RUnlock()
	if nginxCfg != nil && nginxCfg.HTTP != nil {
		if v := sanitizeNginxValue(toString(nginxCfg.HTTP["proxy_cache_dir"])); v != "" {
			cacheDir = v
		}
	}
	if strings.TrimSpace(cacheDir) == "" {
		return ""
	}
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(root, cacheDir)
	}
	return filepath.Clean(cacheDir)
}

func removeDirContents(dir string, truncateFiles bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		target := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			_ = os.RemoveAll(target)
			continue
		}
		if truncateFiles {
			if err := os.Truncate(target, 0); err == nil {
				continue
			}
		}
		_ = os.Remove(target)
	}
}

func isSafeCleanupPath(path string) bool {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return false
	}
	if vol := filepath.VolumeName(cleaned); vol != "" && cleaned == vol+string(filepath.Separator) {
		return false
	}
	return true
}
