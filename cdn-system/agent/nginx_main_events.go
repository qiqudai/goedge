package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func writeMainConfig(cfg *edgeNginxConfig) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "main.conf")
	var b strings.Builder
	if cfg != nil {
		if v := strings.TrimSpace(cfg.WorkerProcesses); v != "" {
			b.WriteString("worker_processes " + v + ";\n")
		}
		if cfg.WorkerRlimitNofile > 0 {
			b.WriteString(fmt.Sprintf("worker_rlimit_nofile %d;\n", cfg.WorkerRlimitNofile))
		}
		if v := strings.TrimSpace(cfg.WorkerShutdownTimeout); v != "" {
			b.WriteString("worker_shutdown_timeout " + v + ";\n")
		}
	}
	logsDir := ""
	if cfg != nil {
		logsDir = strings.TrimSpace(cfg.LogsDir)
	}
	if logsDir == "" {
		logsDir = filepath.ToSlash(filepath.Join(rootDir, "logs"))
	} else {
		logsDir = strings.TrimRight(logsDir, "/")
	}
	b.WriteString("error_log " + logsDir + "/error.log warn;\n")
	pidPath := filepath.ToSlash(filepath.Join(rootDir, "logs", "nginx.pid"))
	b.WriteString("pid " + pidPath + ";\n")
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeEventsConfig(cfg *edgeNginxConfig) error {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "dynamic", "events.conf")
	var b strings.Builder
	if cfg != nil && cfg.WorkerConnections > 0 {
		b.WriteString(fmt.Sprintf("worker_connections %d;\n", cfg.WorkerConnections))
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}
