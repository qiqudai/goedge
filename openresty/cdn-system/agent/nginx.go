package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func nginxConfPath() string {
	confPath := filepath.Join(WorkDir, "conf", "nginx.conf")
	if abs, err := filepath.Abs(confPath); err == nil {
		confPath = abs
	}
	return confPath
}

func setNginxEnv(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		return
	}
	env := os.Environ()
	libPath := filepath.Join(WorkDir, "openresty", "luajit", "lib")
	lualibPath := filepath.Join(WorkDir, "openresty", "lualib")
	ldPath := fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", libPath, lualibPath)
	cmd.Env = append(env, ldPath)
}

func startNginx() error {
	if NginxBinPath == "" {
		return nil
	}
	confPath := nginxConfPath()
	cmd := exec.Command(NginxBinPath, "-p", WorkDir, "-c", confPath)
	setNginxEnv(cmd)
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
	cmd := exec.Command(NginxBinPath, "-p", WorkDir, "-s", "stop", "-c", confPath)
	setNginxEnv(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println("[Success] Nginx Stopped")
	return nil
}

func executeReload() error {
	// Check if nginx is running first to avoid errors
	// Use absolute path to the extracted binary
	if NginxBinPath == "" {
		return nil // Should not happen if initEnvironment called
	}

	confPath := nginxConfPath()

	// nginx -t -c ... -p ...
	cmd := exec.Command(NginxBinPath, "-p", WorkDir, "-t", "-c", confPath)
	// Set LD_LIBRARY_PATH for Linux if needed (assuming libs in openresty/luajit/lib etc)
	setNginxEnv(cmd)

	if err := cmd.Run(); err != nil {
		return err
	}

	// nginx -s reload -c ... -p ...
	cmd = exec.Command(NginxBinPath, "-p", WorkDir, "-s", "reload", "-c", confPath)
	setNginxEnv(cmd)

	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println("[Success] Nginx Reloaded")
	return nil
}
