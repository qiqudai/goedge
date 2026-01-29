package main

import (
	"path/filepath"
	"strings"
)

const runtimeDirName = "edge-node"

func runtimeDir() string {
	if strings.TrimSpace(WorkDir) == "" {
		return ""
	}
	return filepath.Join(WorkDir, runtimeDirName)
}

func runtimeRoot() string {
	if dir := runtimeDir(); dir != "" {
		return dir
	}
	return WorkDir
}
