package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	lualibPath := filepath.Join(WorkDir, "openresty", "lualib")
	luaUserPath := filepath.Join(WorkDir, "lua")
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
	cmd.Env = append(env, "LUA_PATH="+luaPath, "LUA_CPATH="+luaCPath)
}

func startNginx() error {
	if NginxBinPath == "" {
		return nil
	}
	confPath := nginxConfPath()
	cmd := exec.Command(NginxBinPath, "-p", WorkDir, "-c", confPath)
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
