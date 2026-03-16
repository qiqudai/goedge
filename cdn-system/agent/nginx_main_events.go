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
	workerConnections := resolveWorkerConnections(cfg)
	b.WriteString("worker_processes " + resolveWorkerProcesses(cfg) + ";\n")
	b.WriteString(fmt.Sprintf("worker_rlimit_nofile %d;\n", resolveWorkerRlimitNofile(cfg, workerConnections)))
	b.WriteString("worker_shutdown_timeout " + resolveWorkerShutdownTimeout(cfg) + ";\n")
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
	b.WriteString(fmt.Sprintf("worker_connections %d;\n", resolveWorkerConnections(cfg)))
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}
