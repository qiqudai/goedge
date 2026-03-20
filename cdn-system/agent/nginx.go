package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func nginxConfPath() string {
	rootDir := runtimeRoot()
	confPath := filepath.Join(rootDir, "conf", "nginx.conf")
	if abs, err := filepath.Abs(confPath); err == nil {
		confPath = abs
	}
	return confPath
}

func expectedNginxPidPath() string {
	rootDir := runtimeRoot()
	if rootDir == "" {
		return ""
	}
	pidPath := filepath.Join(rootDir, "logs", "nginx.pid")
	if abs, err := filepath.Abs(pidPath); err == nil {
		pidPath = abs
	}
	return pidPath
}

func isPidMissingOrInvalid(outputText string) bool {
	if outputText == "" {
		return false
	}
	lowerText := strings.ToLower(outputText)
	return strings.Contains(lowerText, "invalid pid number") ||
		strings.Contains(lowerText, "no such file or directory")
}

func findNginxMasterPID(confPath string) (int, error) {
	if runtime.GOOS == "windows" {
		return 0, nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	rootDir := runtimeRoot()
	absWorkDir := rootDir
	if abs, err := filepath.Abs(rootDir); err == nil {
		absWorkDir = abs
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(cmdlineBytes) == 0 {
			continue
		}
		cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
		if !strings.Contains(cmdline, "nginx: master process") {
			continue
		}
		if confPath != "" && strings.Contains(cmdline, confPath) {
			return pid, nil
		}
		if NginxBinPath != "" && strings.Contains(cmdline, NginxBinPath) {
			return pid, nil
		}
		if absWorkDir != "" && strings.Contains(cmdline, absWorkDir) {
			return pid, nil
		}
		if rootDir != "" && strings.Contains(cmdline, rootDir) {
			return pid, nil
		}
	}
	return 0, nil
}

func ensureNginxPidFile(pid int) (string, error) {
	pidPath := expectedNginxPidPath()
	if pidPath == "" {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(pidPath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return "", err
	}
	return pidPath, nil
}

func setNginxEnv(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		return
	}
	env := os.Environ()
	ldLibraryPath := ""
	filteredEnv := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "LD_LIBRARY_PATH=") {
			ldLibraryPath = strings.TrimPrefix(kv, "LD_LIBRARY_PATH=")
			continue
		}
		filteredEnv = append(filteredEnv, kv)
	}
	rootDir := runtimeRoot()
	lualibPath := filepath.Join(rootDir, "openresty", "lualib")
	luaUserPath := filepath.Join(rootDir, "lua")
	luaPath := strings.Join([]string{
		filepath.Join(lualibPath, "?.lua"),
		filepath.Join(lualibPath, "?", "init.lua"),
		filepath.Join(luaUserPath, "?.lua"),
		filepath.Join(luaUserPath, "?", "init.lua"),
	}, ";") + ";;"
	luaCPath := strings.Join([]string{
		filepath.Join(lualibPath, "?.so"),
		filepath.Join(luaUserPath, "?.so"),
	}, ";") + ";;"
	libDir := filepath.Join(rootDir, "openresty", "luajit", "lib")
	if info, err := os.Stat(libDir); err == nil && info.IsDir() {
		if ldLibraryPath == "" {
			ldLibraryPath = libDir
		} else if !strings.Contains(ldLibraryPath, libDir) {
			ldLibraryPath = libDir + ":" + ldLibraryPath
		}
	}
	if ldLibraryPath != "" {
		filteredEnv = append(filteredEnv, "LD_LIBRARY_PATH="+ldLibraryPath)
	}
	cmd.Env = append(filteredEnv, "LUA_PATH="+luaPath, "LUA_CPATH="+luaCPath)
}

func startNginx() error {
	if NginxBinPath == "" {
		return nil
	}
	confPath := nginxConfPath()
	cmd := exec.Command(NginxBinPath, "-p", runtimeRoot(), "-c", confPath)
	setNginxEnv(cmd)
	if runtime.GOOS == "windows" {
		if err := cmd.Start(); err != nil {
			return err
		}
		log.Println("[Success] Nginx Started")
		return nil
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println("[Success] Nginx Started")
	return nil
}

func stopNginx() error {
	if NginxBinPath == "" {
		return nil
	}
	confPath := nginxConfPath()
	cmd := exec.Command(NginxBinPath, "-p", runtimeRoot(), "-s", "stop", "-c", confPath)
	setNginxEnv(cmd)
	stopOutput, err := cmd.CombinedOutput()
	if err != nil {
		outputText := strings.TrimSpace(string(stopOutput))
		if runtime.GOOS != "windows" {
			pid, pidErr := findNginxMasterPID(confPath)
			if pidErr == nil && pid > 0 {
				if signalErr := sendSignal(pid, signalTerm); signalErr == nil {
					log.Printf("[Warn] Nginx stopped via master pid=%d", pid)
					return nil
				}
			} else if pidErr == nil && pid == 0 && isPidMissingOrInvalid(outputText) {
				log.Printf("[Warn] Nginx stop skipped: pid missing or invalid and no master process found")
				return nil
			}
		}
		if outputText != "" {
			return fmt.Errorf("nginx stop failed: %w: %s", err, outputText)
		}
		return fmt.Errorf("nginx stop failed: %w", err)
	}
	log.Println("[Success] Nginx Stopped")
	return nil
}

func isManagedNginxRunning() bool {
	if NginxBinPath == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		pidPath := expectedNginxPidPath()
		if pidPath == "" {
			return false
		}
		if _, err := os.Stat(pidPath); err == nil {
			return true
		}
		return false
	}
	pid, err := findNginxMasterPID(nginxConfPath())
	return err == nil && pid > 0
}

func startOrRestartManagedNginx() error {
	if NginxBinPath == "" {
		return nil
	}
	if isManagedNginxRunning() {
		log.Printf("[Info] Managed nginx detected under %s, restarting", runtimeRoot())
		if err := stopNginx(); err != nil {
			return err
		}
	} else {
		log.Printf("[Info] Managed nginx not running under %s, starting", runtimeRoot())
	}
	return startNginx()
}

func reloadNginx() error {
	return executeReload()
}

func executeReload() error {
	// Check if nginx is running first to avoid errors
	// Use absolute path to the extracted binary
	if NginxBinPath == "" {
		return nil // Should not happen if initEnvironment called
	}

	confPath := nginxConfPath()

	// nginx -t -c ... -p ...
	cmd := exec.Command(NginxBinPath, "-p", runtimeRoot(), "-t", "-c", confPath)
	setNginxEnv(cmd)

	testOutput, err := cmd.CombinedOutput()
	if err != nil {
		outputText := strings.TrimSpace(string(testOutput))
		if outputText != "" {
			return fmt.Errorf("nginx test failed: %w: %s", err, outputText)
		}
		return fmt.Errorf("nginx test failed: %w", err)
	}

	if runtime.GOOS != "windows" {
		if pid, pidErr := findNginxMasterPID(confPath); pidErr == nil && pid > 0 {
			if pidPath, writeErr := ensureNginxPidFile(pid); writeErr == nil && pidPath != "" {
				log.Printf("[Warn] Reload using master pid=%d pid_path=%s", pid, pidPath)
			}
			if signalErr := sendSignal(pid, signalHup); signalErr == nil {
				log.Println("[Success] Nginx Reloaded")
				return nil
			}
		}
	}

	// nginx -s reload -c ... -p ...
	cmd = exec.Command(NginxBinPath, "-p", runtimeRoot(), "-s", "reload", "-c", confPath)
	setNginxEnv(cmd)

	reloadOutput, err := cmd.CombinedOutput()
	if err != nil {
		outputText := strings.TrimSpace(string(reloadOutput))
		if runtime.GOOS != "windows" {
			if pid, pidErr := findNginxMasterPID(confPath); pidErr == nil && pid > 0 {
				if pidPath, writeErr := ensureNginxPidFile(pid); writeErr == nil {
					if signalErr := sendSignal(pid, signalHup); signalErr == nil {
						log.Printf("[Warn] Reload fallback: signaled master pid=%d pid_path=%s", pid, pidPath)
						return nil
					}
				}
			} else if pidErr == nil {
				if isPidMissingOrInvalid(outputText) {
					log.Printf("[Warn] Reload skipped: pid missing or invalid and no master process found")
					return nil
				}
				if startErr := startNginx(); startErr == nil {
					log.Printf("[Warn] Reload fallback: nginx was not running; started a new master")
					return nil
				}
			}
		}
		if outputText != "" {
			return fmt.Errorf("nginx reload failed: %w: %s", err, outputText)
		}
		return fmt.Errorf("nginx reload failed: %w", err)
	}
	log.Println("[Success] Nginx Reloaded")
	return nil
}
