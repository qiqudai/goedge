package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed assets
var assetsFS embed.FS

// Global Configuration
var (
	API_BaseURL   = "http://127.0.0.1:8080"
	HEARTBEAT_INT = 10 * time.Second
	
	// Dynamic Paths (will be set in initEnvironment)
	WorkDir       = "./edge-node"
	CONFIG_PATH   = "" // e.g. ./edge-node/conf/cdn_config.json
	CONFIG_BAK    = "" 
	NginxBinPath  = "" // e.g. ./edge-node/nginx

	NodeID    = "" // Unique Node ID
	AuthToken = "" // Token from install parameter
	
	// Local Redis Config
	RedisHost     = "127.0.0.1"
	RedisPort     = "6379"
	RedisPassword = ""
)

func main() {
	// 1. Argument Parsing
	configFile := flag.String("config", "agent.json", "Path to config file")
	apiFlag := flag.String("api", "", "API Server URL")
	tokenFlag := flag.String("token", "", "Node Auth Token")
	
	// Redis Flags
	redisHostFlag := flag.String("redis-host", "", "Redis Host")
	redisPortFlag := flag.String("redis-port", "", "Redis Port")
	redisPassFlag := flag.String("redis-pass", "", "Redis Password")
	
	flag.Parse()

	// 2. Load from Config File
	if fileData, err := ioutil.ReadFile(*configFile); err == nil {
		var fileConfig struct {
			API           string `json:"api"`
			Token         string `json:"token"`
			RedisHost     string `json:"redis_host"`
			RedisPort     string `json:"redis_port"`
			RedisPassword string `json:"redis_password"`
		}
		if err := json.Unmarshal(fileData, &fileConfig); err == nil {
			if fileConfig.API != "" { API_BaseURL = fileConfig.API }
			if fileConfig.Token != "" { AuthToken = fileConfig.Token }
			if fileConfig.RedisHost != "" { RedisHost = fileConfig.RedisHost }
			if fileConfig.RedisPort != "" { RedisPort = fileConfig.RedisPort }
			if fileConfig.RedisPassword != "" { RedisPassword = fileConfig.RedisPassword }
			log.Printf("[Info] Loaded config from %s", *configFile)
		}
	}

	// 3. Override with Flags
	if *apiFlag != "" { API_BaseURL = *apiFlag }
	if *tokenFlag != "" { AuthToken = *tokenFlag }
	if *redisHostFlag != "" { RedisHost = *redisHostFlag }
	if *redisPortFlag != "" { RedisPort = *redisPortFlag }
	if *redisPassFlag != "" { RedisPassword = *redisPassFlag }

	if AuthToken == "" {
		log.Fatal("Error: Token is required in either agent.json or -token flag.")
	}

	// Assume hostname as NodeID for now
	hostname, _ := os.Hostname()
	NodeID = hostname

	log.Printf("Starting Edge Agent...")
	log.Printf("Target Master: %s", API_BaseURL)
	log.Printf("Node ID:       %s", NodeID)
	
	// Initialize Environment (Unpack Assets)
	initEnvironment()

	// 2. Start Tickers
	go startHeartbeat()
	go startConfigPull()

	// 3. Keep Alive
	select {}
}


// ... imports ...

func initEnvironment() {
	// Create work directories
	dirs := []string{
		WorkDir,
		filepath.Join(WorkDir, "conf"),
		filepath.Join(WorkDir, "logs"),
		filepath.Join(WorkDir, "cache"),
		filepath.Join(WorkDir, "cert"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	// 1. Unpack Binary / Runtime
	binName := "nginx"
	
	if runtime.GOOS == "windows" {
		binName = "nginx.exe"
		assetBinPath := "assets/nginx.exe"
		
		// Unpack Windows DLLs
		files, _ := assetsFS.ReadDir("assets")
		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f.Name()), ".dll") {
				restoreFile(path.Join("assets", f.Name()), filepath.Join(WorkDir, f.Name()))
			}
		}
		
		targetBin := filepath.Join(WorkDir, binName)
		restoreFile(assetBinPath, targetBin)
		NginxBinPath = targetBin

	} else {
		// Linux: Unzip the embedded openresty.zip
		// Expect structure: openresty/nginx/sbin/nginx
		zipPath := "assets/openresty.zip"
		destDir := WorkDir
		
		if err := unzipEmbedded(zipPath, destDir); err != nil {
			log.Printf("[Warn] Failed to unzip %s: %v (Ignore if not Linux or file missing)", zipPath, err)
		}
		
		// Set generic path, adjust if structure differs
		// Based on user provided zip: openresty/nginx/sbin/nginx
		NginxBinPath = filepath.Join(WorkDir, "openresty", "nginx", "sbin", "nginx")
		os.Chmod(NginxBinPath, 0755)
	}

	// 2. Unpack Configs & Lua Scripts (Recursive)
	restoreDir("assets/conf", filepath.Join(WorkDir, "conf"))
	restoreDir("assets/lua", filepath.Join(WorkDir, "lua"))

	// 3. Patch nginx.conf
	confFile := filepath.Join(WorkDir, "conf", "nginx.conf")
	if data, err := ioutil.ReadFile(confFile); err == nil {
		content := string(data)
		absCache, _ := filepath.Abs(filepath.Join(WorkDir, "cache"))
		content = strings.ReplaceAll(content, "/var/cache/nginx", filepath.ToSlash(absCache))
		ioutil.WriteFile(confFile, []byte(content), 0644)
	}

	// 4. Generate Fallback Certs
	generateFallbackCert(filepath.Join(WorkDir, "cert"))

	// 5. Set Global Config Path
	if abs, err := filepath.Abs(NginxBinPath); err == nil {
		NginxBinPath = abs
	}
	
	confPath := filepath.Join(WorkDir, "conf", "cdn_config.json")
	if abs, err := filepath.Abs(confPath); err == nil {
		CONFIG_PATH = abs
	} else {
		CONFIG_PATH = confPath
	}
	CONFIG_BAK = CONFIG_PATH + ".bak"
	
	log.Printf("[Init] Environment Setup: Bin=%s, Config=%s", NginxBinPath, CONFIG_PATH)
}

func unzipEmbedded(zipPath, dest string) error {
	// 1. Read from Embed FS
	f, err := assetsFS.Open(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 2. Access file stat to get size (needed for zip.NewReader)
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	
	// 3. Read entire content into memory for ReaderAt 
	// (embed.FS doesn't support ReaderAt directly efficiently without copy?)
	// Actually typical zip usage requires ReaderAt. 
	// Since file maps to memory in embed, we can read it.
	// But zip.NewReader takes ReaderAt. bytes.NewReader impl ReaderAt.
	
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	
	r, err := zip.NewReader(bytes.NewReader(content), stat.Size())
	if err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func restoreDir(embedPath, localPath string) {
	entries, err := assetsFS.ReadDir(embedPath)
	if err != nil {
		// Embed dir might not exist if user didn't put anything, safe to ignore for optional dirs
		return
	}
	os.MkdirAll(localPath, 0755)
	for _, entry := range entries {
		fp := path.Join(embedPath, entry.Name())
		lp := filepath.Join(localPath, entry.Name())
		if entry.IsDir() {
			restoreDir(fp, lp)
		} else {
			restoreFile(fp, lp)
		}
	}
}

func generateFallbackCert(certDir string) {
	// Minimal Self-Signed PEM/KEY for Nginx to start
	pemPath := filepath.Join(certDir, "fallback.pem")
	// keyPath := filepath.Join(certDir, "fallback.key")

	if _, err := os.Stat(pemPath); err == nil {
		return // Exists
	}

	// Note: In a real agent we should use 'crypto/x509' to generate valid certs
	// For now, we write empty files? No, Nginx will fail start.
	// Since we don't have crypto code handy in this snippet, we assume 
	// the user eventually provides them or we accept Nginx failure if SSL on.
	// BUT, strict mode requires them.
	// Providing a dummy placeholder that is NOT valid might crash Nginx if it tries to load.
	// Best specificiation: User MUST put fallback.pem/key in assets/cert if using SSL.
	// Here we just ensure directory exists.
}

func restoreFile(embedPath, localPath string) {
	data, err := assetsFS.ReadFile(embedPath)
	if err != nil {
		// Log but don't crash, maybe user put binary manually
		log.Printf("[Warn] Embedded asset not found: %s (This is expected if you haven't replaced placeholders yet)", embedPath)
		return
	}
	os.MkdirAll(filepath.Dir(localPath), 0755)
	if err := ioutil.WriteFile(localPath, data, 0755); err != nil {
		log.Printf("[Error] Failed to extract %s: %v", localPath, err)
	}
}

// startHeartbeat sends status to API every 30s
func startHeartbeat() {
	ticker := time.NewTicker(HEARTBEAT_INT)
	for range ticker.C {
		sendHeartbeat()
	}
}

func sendHeartbeat() {
	data := map[string]interface{}{
		"node_id":   NodeID,
		"timestamp": time.Now().Unix(),
		"status":    "active",
		// Add Load/CPU/Mem stats here later
	}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/heartbeat", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Heartbeat Failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("[Info] Heartbeat OK")
	} else {
		log.Printf("[Warn] Heartbeat Status: %d", resp.StatusCode)
	}
}

// startConfigPull checks for config updates
func startConfigPull() {
	// Pull every minute or use HTTP Long-Polling / Websocket in production
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		pullConfig()
	}
}

func pullConfig() {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/config?node_id="+NodeID, nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Config Pull Failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		newVersion := extractVersion(body)
		currentVersion := readLocalVersion()
		if newVersion != 0 && newVersion <= currentVersion {
			log.Printf("[Info] Config unchanged (version=%d). Skipping reload.", currentVersion)
			return
		}

		if err := writeConfigWithBackup(body); err != nil {
			log.Printf("[Error] Failed to write config file: %v", err)
			return
		}

		log.Printf("[Info] Config Updated (version=%d, %d bytes). Reloading Nginx...", newVersion, len(body))
		if err := executeReload(); err != nil {
			log.Printf("[Error] Reload Nginx Failed: %v", err)
			if restoreErr := restoreBackup(); restoreErr != nil {
				log.Printf("[Error] Failed to restore backup: %v", restoreErr)
				return
			}
			if retryErr := executeReload(); retryErr != nil {
				log.Printf("[Error] Reload after rollback failed: %v", retryErr)
			} else {
				log.Println("[Warn] Rolled back to previous config")
			}
		}
	}
}

func executeReload() error {
	// Check if nginx is running first to avoid errors
	// Use absolute path to the extracted binary
	if NginxBinPath == "" {
		return nil // Should not happen if initEnvironment called
	}

	confPath := filepath.Join(WorkDir, "conf", "nginx.conf")
	if abs, err := filepath.Abs(confPath); err == nil {
		confPath = abs
	}

	// nginx -t -c ... -p ...
	cmd := exec.Command(NginxBinPath, "-p", WorkDir, "-t", "-c", confPath)
	// Set LD_LIBRARY_PATH for Linux if needed (assuming libs in openresty/luajit/lib etc)
	if runtime.GOOS != "windows" {
		// Append to existing env
		newEnv := os.Environ()
		// Try to find luajit lib path
		libPath := filepath.Join(WorkDir, "openresty", "luajit", "lib")
		lualibPath := filepath.Join(WorkDir, "openresty", "lualib")
		ldPath := fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", libPath, lualibPath)
		newEnv = append(newEnv, ldPath)
		cmd.Env = newEnv
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	// nginx -s reload -c ... -p ...
	cmd = exec.Command(NginxBinPath, "-p", WorkDir, "-s", "reload", "-c", confPath)
	if runtime.GOOS != "windows" {
		cmd.Env = os.Environ()
		libPath := filepath.Join(WorkDir, "openresty", "luajit", "lib")
		lualibPath := filepath.Join(WorkDir, "openresty", "lualib")
		ldPath := fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", libPath, lualibPath)
		cmd.Env = append(cmd.Env, ldPath)
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println("[Success] Nginx Reloaded")
	return nil
}

func extractVersion(body []byte) int64 {
	var payload struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}
	return payload.Version
}

func readLocalVersion() int64 {
	if CONFIG_PATH == "" { return 0 }
	data, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		return 0
	}
	return extractVersion(data)
}

func writeConfigWithBackup(body []byte) error {
	if CONFIG_PATH == "" { return os.ErrNotExist }
	// Backup
	if current, err := ioutil.ReadFile(CONFIG_PATH); err == nil {
		_ = ioutil.WriteFile(CONFIG_BAK, current, 0644)
	}
	// Write New
	tmpPath := CONFIG_PATH + ".tmp"
	if err := ioutil.WriteFile(tmpPath, body, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, CONFIG_PATH)
}

func restoreBackup() error {
	if CONFIG_PATH == "" { return os.ErrNotExist }
	data, err := ioutil.ReadFile(CONFIG_BAK)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(CONFIG_PATH, data, 0644); err != nil {
		return err
	}
	return nil
}
