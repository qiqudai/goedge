package main

import (
	"path/filepath"
	"strings"
)

func resolveNginxLogsDir(raw string) string {
	rootDir := runtimeRoot()
	logsDir := filepath.Join(rootDir, "logs")
	if dir := strings.TrimSpace(raw); dir != "" {
		logsDir = dir
		if !filepath.IsAbs(logsDir) {
			logsDir = filepath.Join(rootDir, logsDir)
		}
	}
	return filepath.Clean(logsDir)
}

func resolveNginxLogsDirForConfig(raw string) string {
	return strings.TrimRight(filepath.ToSlash(resolveNginxLogsDir(raw)), "/")
}
