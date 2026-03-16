package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const agentSystemdServicePath = "/etc/systemd/system/cdn-agent.service"

func maybeConfigureAutostart(configPath string) {
	if runtime.GOOS != "linux" || !AutoInstallService {
		return
	}
	if os.Geteuid() != 0 {
		log.Printf("[Info] Auto service install skipped: root required")
		return
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		log.Printf("[Info] Auto service install skipped: systemctl not found")
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[Warn] Auto service install skipped: resolve executable failed: %v", err)
		return
	}
	if abs, err := filepath.Abs(exePath); err == nil {
		exePath = abs
	}

	cfgPath := strings.TrimSpace(configPath)
	if cfgPath == "" {
		cfgPath = filepath.Join(filepath.Dir(exePath), "agent.json")
	}
	if abs, err := filepath.Abs(cfgPath); err == nil {
		cfgPath = abs
	}

	workDir := strings.TrimSpace(WorkDir)
	if workDir == "" {
		workDir = filepath.Dir(exePath)
	}
	if abs, err := filepath.Abs(workDir); err == nil {
		workDir = abs
	}

	libDir := filepath.Join(runtimeRoot(), "openresty", "luajit", "lib")
	serviceText := buildAgentSystemdService(exePath, cfgPath, workDir, libDir)

	needReload := false
	current, err := os.ReadFile(agentSystemdServicePath)
	if err != nil || string(current) != serviceText {
		if err := os.WriteFile(agentSystemdServicePath, []byte(serviceText), 0o644); err != nil {
			log.Printf("[Warn] Auto service install failed: write unit failed: %v", err)
			return
		}
		needReload = true
	}

	enabled := runCommandSilent("systemctl", "is-enabled", "cdn-agent") == nil
	if needReload {
		if out, err := runCommand("systemctl", "daemon-reload"); err != nil {
			log.Printf("[Warn] Auto service install failed: daemon-reload failed: %v output=%s", err, out)
			return
		}
	}
	if !enabled || needReload {
		if out, err := runCommand("systemctl", "enable", "cdn-agent"); err != nil {
			log.Printf("[Warn] Auto service install failed: enable failed: %v output=%s", err, out)
			return
		}
		log.Printf("[Info] Auto service installed: %s", agentSystemdServicePath)
	}
}

func buildAgentSystemdService(exePath, configPath, workDir, libDir string) string {
	ldPath := strings.TrimSpace(libDir)
	if ldPath == "" {
		ldPath = filepath.Join(workDir, "edge-node", "openresty", "luajit", "lib")
	}
	return fmt.Sprintf(`[Unit]
Description=CDN Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
Environment=LD_LIBRARY_PATH=%s
ExecStart=%s -config %s
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, workDir, ldPath, exePath, configPath)
}

func runStartupDiagnostics() {
	if runtime.GOOS != "linux" {
		return
	}
	if strings.TrimSpace(NginxBinPath) == "" {
		log.Printf("[Warn] Startup check skipped: nginx path is empty")
		return
	}
	if _, err := os.Stat(NginxBinPath); err != nil {
		log.Printf("[Warn] Startup check failed: nginx binary missing: %v", err)
		return
	}
	libPath := filepath.Join(runtimeRoot(), "openresty", "luajit", "lib", "libluajit-5.1.so.2")
	if _, err := os.Stat(libPath); err != nil {
		log.Printf("[Warn] Startup check: libluajit not found at %s", libPath)
	}
	cmd := exec.Command(NginxBinPath, "-p", runtimeRoot(), "-t", "-c", nginxConfPath())
	setNginxEnv(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Warn] Startup check failed: nginx -t error=%v output=%s", err, strings.TrimSpace(string(out)))
		return
	}
	log.Printf("[Info] Startup check passed: nginx -t")
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func runCommandSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	return cmd.Run()
}
