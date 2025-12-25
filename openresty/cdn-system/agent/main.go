package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
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
)

func main() {
	// 1. Argument Parsing
	configFile := flag.String("config", "agent.json", "Path to config file")
	apiFlag := flag.String("api", "", "API Server URL")
	tokenFlag := flag.String("token", "", "Node Auth Token")
	flag.Parse()

	// 2. Load from Config File
	if fileData, err := ioutil.ReadFile(*configFile); err == nil {
		var fileConfig struct {
			API   string `json:"api"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal(fileData, &fileConfig); err == nil {
			if fileConfig.API != "" { API_BaseURL = fileConfig.API }
			if fileConfig.Token != "" { AuthToken = fileConfig.Token }
			log.Printf("[Info] Loaded config from %s", *configFile)
		}
	}

	// 3. Override with Flags
	if *apiFlag != "" { API_BaseURL = *apiFlag }
	if *tokenFlag != "" { AuthToken = *tokenFlag }

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
	go startTaskPull()

	// 3. Keep Alive
	select {}
}


// ... imports ...

func initEnvironment() {
	// Create work directories
	dirs := []string{
		WorkDir,
		filepath.Join(WorkDir, "conf"),
		filepath.Join(WorkDir, "conf", "dynamic"),
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
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "http.conf"))
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "http_global.conf"))
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "main.conf"))
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "events.conf"))
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "stream.conf"))
	ensureDynamicConf(filepath.Join(WorkDir, "conf", "dynamic", "stream_global.conf"))

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

func ensureDynamicConf(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	_ = ioutil.WriteFile(path, []byte(""), 0644)
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

func startTaskPull() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		pullTasks()
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

		if err := generateDynamicConfigs(body); err != nil {
			log.Printf("[Error] Failed to generate dynamic configs: %v", err)
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

func pullTasks() {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Task Pull Failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[Warn] Task Pull Status: %d", resp.StatusCode)
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var payload struct {
		Tasks []struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
			Data string `json:"data"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[Error] Task Pull Decode Failed: %v", err)
		return
	}

	for _, task := range payload.Tasks {
		if err := processTask(task.ID, task.Type, task.Data); err != nil {
			reportTask(task.ID, "fail", err.Error())
		} else {
			reportTask(task.ID, "done", "ok")
		}
	}
}

func processTask(id int64, taskType string, data string) error {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "refresh_url":
		return purgeURLs(splitLines(data))
	case "refresh_dir":
		return purgeDirs(splitLines(data))
	case "preheat":
		return preheatURLs(splitLines(data))
	default:
		return fmt.Errorf("unknown task type: %s", taskType)
	}
}

func reportTask(id int64, state string, ret string) {
	payload := map[string]string{
		"state": state,
		"ret":   ret,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/agent/tasks/%d/finish", API_BaseURL, id), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Error] Task Report Failed: %v", err)
		return
	}
	resp.Body.Close()
}

func purgeURLs(urls []string) error {
	var lastErr error
	for _, raw := range urls {
		if raw == "" {
			continue
		}
		if err := purgeURL(raw); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func purgeDirs(urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	cacheDir := filepath.Join(WorkDir, "cache")
	return clearCacheDir(cacheDir)
}

func purgeURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url: %s", raw)
	}
	host := u.Hostname()
	uri := u.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	args := u.RawQuery
	cacheKey := host + uri
	if args != "" {
		cacheKey = cacheKey + "?" + args
	}
	sum := md5.Sum([]byte(cacheKey))
	hash := fmt.Sprintf("%x", sum)
	cacheDir := filepath.Join(WorkDir, "cache")
	if len(hash) < 3 {
		return nil
	}
	path := filepath.Join(cacheDir, hash[0:1], hash[1:3], hash)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func clearCacheDir(dir string) error {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		fp := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(fp); err != nil {
			return err
		}
	}
	return nil
}

func preheatURLs(urls []string) error {
	var lastErr error
	for _, raw := range urls {
		if raw == "" {
			continue
		}
		if err := preheatURL(raw); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func preheatURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url: %s", raw)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	localURL := fmt.Sprintf("%s://127.0.0.1:%s%s", scheme, port, u.RequestURI())
	req, _ := http.NewRequest("GET", localURL, nil)
	req.Host = u.Host

	client := &http.Client{Timeout: 15 * time.Second}
	if scheme == "https" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func splitLines(input string) []string {
	parts := strings.Split(input, "\n")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
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

type edgeStreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	Enable bool   `json:"enable"`
}

type edgeStream struct {
	ID           int64             `json:"id"`
	ListenPorts  []string          `json:"listen_ports"`
	Targets      []edgeStreamTarget `json:"targets"`
	BalanceWay   string            `json:"balance_way"`
	ProxyProtocol bool             `json:"proxy_protocol"`
	ProxyConnectTimeout string      `json:"proxy_connect_timeout"`
	ProxyTimeout        string      `json:"proxy_timeout"`
	ConnLimit           int          `json:"conn_limit"`
}

type edgeUpstreamTarget struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}

type edgeUpstream struct {
	ID      string               `json:"id"`
	Targets []edgeUpstreamTarget `json:"targets"`
}

type edgeCacheRule struct {
	Rule       string `json:"rule"`
	Ext        string `json:"ext"`
	URI        string `json:"uri"`
	Prefix     string `json:"prefix"`
	TTL        int    `json:"ttl"`
	Enable     *bool  `json:"enable"`
	NoCache    bool   `json:"no_cache"`
	ForceCache bool   `json:"force_cache"`
	Priority   int    `json:"priority"`
	IgnoreArgs bool   `json:"ignore_args"`
	CacheKey   string `json:"cache_key"`
}

type edgeCacheConfig struct {
	Enable     bool           `json:"enable"`
	DefaultTTL int            `json:"default_ttl"`
	Rules      []edgeCacheRule `json:"rules"`
}

type edgeDomain struct {
	Name              string            `json:"name"`
	UpstreamKey       string            `json:"upstream_key"`
	LoadBalancePolicy string            `json:"load_balance_policy"`
	Headers           map[string]string `json:"headers"`
	ResponseHeaders   map[string]string `json:"response_headers"`
	SSLCertData       string            `json:"ssl_cert_data"`
	SSLKeyData        string            `json:"ssl_key_data"`
	ACLDefaultAction  string            `json:"acl_default_action"`
	ACLRules          []struct {
		IP     string `json:"ip"`
		Action string `json:"action"`
	} `json:"acl_rules"`
	BlackIPs          []string          `json:"black_ips"`
	WhiteIPs          []string          `json:"white_ips"`
	CCRuleID          int64             `json:"cc_rule_id"`
	OriginProtocol    string            `json:"origin_protocol"`
	OriginHTTPPort    string            `json:"origin_http_port"`
	OriginHTTPSPort   string            `json:"origin_https_port"`
	Cache             *edgeCacheConfig  `json:"cache"`
	HttpListen        []string          `json:"http_listen"`
	HttpsListen       []string          `json:"https_listen"`
	HTTPSForce        bool              `json:"https_force"`
	HTTPSRedirectPort string            `json:"https_redirect_port"`
	HTTPSHSTS         bool              `json:"https_hsts"`
	HTTPSHTTP2        bool              `json:"https_http2"`
	HTTPSSSLProtocols string            `json:"https_ssl_protocols"`
	HTTPSSSLCiphers   string            `json:"https_ssl_ciphers"`
	HTTPSSSLPreferServerCiphers string   `json:"https_ssl_prefer_server_ciphers"`
	ProxyConnectTimeout string           `json:"proxy_connect_timeout"`
	ProxyReadTimeout    string           `json:"proxy_read_timeout"`
	ProxySendTimeout    string           `json:"proxy_send_timeout"`
	ProxyHTTPVersion    string           `json:"proxy_http_version"`
	ProxySSLProtocols   string           `json:"proxy_ssl_protocols"`
	EnableGzip          bool             `json:"enable_gzip"`
	GzipTypes           string           `json:"gzip_types"`
	EnableWebsocket     bool             `json:"enable_websocket"`
	EnableRange         bool             `json:"enable_range"`
	BodyLimit           int64            `json:"body_limit"`
	UpstreamKeepalive   bool             `json:"upstream_keepalive"`
	UpstreamKeepaliveConn int            `json:"upstream_keepalive_conn"`
	UpstreamKeepaliveTimeout int         `json:"upstream_keepalive_timeout"`
}

type edgeNginxConfig struct {
	LogsDir               string                 `json:"logs_dir"`
	WorkerProcesses       string                 `json:"worker_processes"`
	WorkerConnections     int                    `json:"worker_connections"`
	WorkerRlimitNofile    int                    `json:"worker_rlimit_nofile"`
	WorkerShutdownTimeout string                 `json:"worker_shutdown_timeout"`
	Resolver              string                 `json:"resolver"`
	ResolverTimeout       string                 `json:"resolver_timeout"`
	HTTP                  map[string]interface{} `json:"http"`
	Stream                map[string]interface{} `json:"stream"`
}

type edgeConfig struct {
	Domains  []edgeDomain  `json:"domains"`
	Upstreams []edgeUpstream `json:"upstreams"`
	Streams []edgeStream `json:"streams"`
	Nginx   *edgeNginxConfig `json:"nginx"`
}

func generateDynamicConfigs(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	var cfg edgeConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return err
	}
	if err := writeHTTPConfig(cfg); err != nil {
		return err
	}
	if err := writeStreamConfig(cfg.Streams); err != nil {
		return err
	}
	if err := writeMainConfig(cfg.Nginx); err != nil {
		return err
	}
	if err := writeEventsConfig(cfg.Nginx); err != nil {
		return err
	}
	if err := writeHTTPGlobalConfig(cfg.Nginx); err != nil {
		return err
	}
	return writeStreamGlobalConfig(cfg.Nginx)
}

func writeStreamConfig(streams []edgeStream) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "stream.conf")
	if len(streams) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}

	var b strings.Builder
	for _, stream := range streams {
		if len(stream.ListenPorts) == 0 || len(stream.Targets) == 0 {
			continue
		}
		upstreamName := fmt.Sprintf("stream_up_%d", stream.ID)
		b.WriteString("upstream " + upstreamName + " {\n")
		switch strings.ToLower(stream.BalanceWay) {
		case "ip_hash":
			b.WriteString("    hash $remote_addr consistent;\n")
		case "least_conn":
			b.WriteString("    least_conn;\n")
		}
		for _, target := range stream.Targets {
			if !target.Enable || target.Addr == "" {
				continue
			}
			if target.Weight > 0 {
				b.WriteString(fmt.Sprintf("    server %s weight=%d;\n", target.Addr, target.Weight))
			} else {
				b.WriteString(fmt.Sprintf("    server %s;\n", target.Addr))
			}
		}
		b.WriteString("}\n")

		for _, port := range stream.ListenPorts {
			port = strings.TrimSpace(port)
			if port == "" {
				continue
			}
			b.WriteString("server {\n")
			if stream.ProxyProtocol {
				b.WriteString("    listen " + port + " proxy_protocol;\n")
			} else {
				b.WriteString("    listen " + port + ";\n")
			}
			b.WriteString("    proxy_pass " + upstreamName + ";\n")
			if stream.ProxyConnectTimeout != "" {
				b.WriteString("    proxy_connect_timeout " + stream.ProxyConnectTimeout + ";\n")
			} else {
				b.WriteString("    proxy_connect_timeout 10s;\n")
			}
			if stream.ProxyTimeout != "" {
				b.WriteString("    proxy_timeout " + stream.ProxyTimeout + ";\n")
			} else {
				b.WriteString("    proxy_timeout 60s;\n")
			}
			if stream.ConnLimit > 0 {
				b.WriteString(fmt.Sprintf("    limit_conn stream_conn %d;\n", stream.ConnLimit))
			}
			b.WriteString("}\n")
		}
	}

	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeHTTPConfig(cfg edgeConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "http.conf")
	if len(cfg.Domains) == 0 {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}

	upstreamKeepalive := map[string]edgeDomain{}
	for _, domain := range cfg.Domains {
		if domain.UpstreamKey != "" && domain.UpstreamKeepalive {
			upstreamKeepalive[domain.UpstreamKey] = domain
		}
	}

	var b strings.Builder
	for _, upstream := range cfg.Upstreams {
		if upstream.ID == "" || len(upstream.Targets) == 0 {
			continue
		}
		b.WriteString("upstream " + upstream.ID + " {\n")
		for _, target := range upstream.Targets {
			if target.Addr == "" {
				continue
			}
			if target.Weight > 0 {
				b.WriteString(fmt.Sprintf("    server %s weight=%d;\n", target.Addr, target.Weight))
			} else {
				b.WriteString(fmt.Sprintf("    server %s;\n", target.Addr))
			}
		}
		if keep, ok := upstreamKeepalive[upstream.ID]; ok {
			conn := keep.UpstreamKeepaliveConn
			if conn <= 0 {
				conn = 32
			}
			b.WriteString(fmt.Sprintf("    keepalive %d;\n", conn))
		}
		b.WriteString("}\n")
	}

	for _, domain := range cfg.Domains {
		if domain.Name == "" || domain.UpstreamKey == "" {
			continue
		}
		writeDomainServers(&b, domain)
	}

	b.WriteString("server {\n")
	b.WriteString("    listen 80 default_server;\n")
	b.WriteString("    server_name _;\n")
	b.WriteString("    return 404;\n")
	b.WriteString("}\n")

	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeDomainServers(b *strings.Builder, domain edgeDomain) {
	httpPorts := domain.HttpListen
	if len(httpPorts) == 0 {
		httpPorts = []string{"80"}
	}
	httpsPorts := domain.HttpsListen

	if domain.HTTPSForce && len(httpsPorts) > 0 {
		writeHTTPSRedirectServer(b, domain, httpPorts, httpsPorts)
	} else {
		for _, port := range httpPorts {
			writeHTTPServer(b, domain, port, false)
		}
	}

	for _, port := range httpsPorts {
		writeHTTPServer(b, domain, port, true)
	}
}

func writeHTTPSRedirectServer(b *strings.Builder, domain edgeDomain, httpPorts []string, httpsPorts []string) {
	redirectPort := domain.HTTPSRedirectPort
	if redirectPort == "" {
		redirectPort = "443"
	}
	for _, port := range httpPorts {
		if strings.TrimSpace(port) == "" {
			continue
		}
		b.WriteString("server {\n")
		b.WriteString("    listen " + port + ";\n")
		b.WriteString("    server_name " + domain.Name + ";\n")
		b.WriteString("    return 301 https://$host:" + redirectPort + "$request_uri;\n")
		b.WriteString("}\n")
	}
}

func writeHTTPServer(b *strings.Builder, domain edgeDomain, port string, tls bool) {
	port = strings.TrimSpace(port)
	if port == "" {
		return
	}
	b.WriteString("server {\n")
	if tls {
		if domain.HTTPSHTTP2 {
			b.WriteString("    listen " + port + " ssl http2;\n")
		} else {
			b.WriteString("    listen " + port + " ssl;\n")
		}
		b.WriteString("    ssl_certificate cert/fallback.pem;\n")
		b.WriteString("    ssl_certificate_key cert/fallback.key;\n")
		b.WriteString("    ssl_certificate_by_lua_block {\n")
		b.WriteString("        local ssl_mgr = require \"lua.ssl_manager\"\n")
		b.WriteString("        ssl_mgr.set_certificate()\n")
		b.WriteString("    }\n")
		if domain.HTTPSSSLProtocols != "" {
			b.WriteString("    ssl_protocols " + domain.HTTPSSSLProtocols + ";\n")
		}
		if domain.HTTPSSSLCiphers != "" {
			b.WriteString("    ssl_ciphers " + domain.HTTPSSSLCiphers + ";\n")
		}
		if domain.HTTPSSSLPreferServerCiphers != "" {
			b.WriteString("    ssl_prefer_server_ciphers " + domain.HTTPSSSLPreferServerCiphers + ";\n")
		}
		if domain.HTTPSHSTS {
			b.WriteString("    add_header Strict-Transport-Security \"max-age=31536000\" always;\n")
		}
	} else {
		b.WriteString("    listen " + port + ";\n")
	}
	b.WriteString("    server_name " + domain.Name + ";\n")
	if domain.BodyLimit > 0 {
		b.WriteString(fmt.Sprintf("    client_max_body_size %dm;\n", domain.BodyLimit))
	}
	if domain.EnableGzip {
		b.WriteString("    gzip on;\n")
		if domain.GzipTypes != "" {
			b.WriteString("    gzip_types " + domain.GzipTypes + ";\n")
		}
	}

	b.WriteString("    set $cc_rule_id " + fmt.Sprintf("%d", domain.CCRuleID) + ";\n")

	writeCacheLocations(b, domain, tls)

	b.WriteString("}\n")
}

func writeCacheLocations(b *strings.Builder, domain edgeDomain, tls bool) {
	cacheCfg := domain.Cache
	rules := make([]edgeCacheRule, 0)
	if cacheCfg != nil && len(cacheCfg.Rules) > 0 {
		rules = append(rules, cacheCfg.Rules...)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	for _, rule := range rules {
		location := buildRuleLocation(rule)
		if location == "" {
			continue
		}
		b.WriteString("    location " + location + " {\n")
		writeProxyBlock(b, domain, tls, cacheCfg, &rule)
		b.WriteString("    }\n")
	}

	b.WriteString("    location / {\n")
	writeProxyBlock(b, domain, tls, cacheCfg, nil)
	b.WriteString("    }\n")
}

func buildRuleLocation(rule edgeCacheRule) string {
	if rule.Rule != "" {
		return normalizeRuleLocation(rule.Rule)
	}
	if rule.URI != "" {
		return "= " + rule.URI
	}
	if rule.Prefix != "" {
		return "^~ " + rule.Prefix
	}
	if rule.Ext != "" {
		ext := rule.Ext
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		return "~* \\" + ext + "$"
	}
	return ""
}

func normalizeRuleLocation(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	if strings.HasPrefix(rule, "=") {
		return rule
	}
	if strings.HasPrefix(rule, "^~") || strings.HasPrefix(rule, "~") {
		return rule
	}
	if strings.HasPrefix(rule, "/") {
		return "^~ " + rule
	}
	if strings.HasPrefix(rule, ".") {
		return "~* \\" + rule + "$"
	}
	return "~* " + rule
}

func writeProxyBlock(b *strings.Builder, domain edgeDomain, tls bool, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	b.WriteString("        limit_req zone=cc_limit burst=20 nodelay;\n")
	b.WriteString("        limit_conn addr_conn 50;\n")
	b.WriteString("        access_by_lua_file lua/access_guard.lua;\n")
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	if domain.EnableWebsocket {
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection $connection_upgrade;\n")
	} else if domain.ProxyHTTPVersion != "" {
		b.WriteString("        proxy_http_version " + domain.ProxyHTTPVersion + ";\n")
	}
	if domain.ProxyConnectTimeout != "" {
		b.WriteString("        proxy_connect_timeout " + domain.ProxyConnectTimeout + ";\n")
	}
	if domain.ProxyReadTimeout != "" {
		b.WriteString("        proxy_read_timeout " + domain.ProxyReadTimeout + ";\n")
	}
	if domain.ProxySendTimeout != "" {
		b.WriteString("        proxy_send_timeout " + domain.ProxySendTimeout + ";\n")
	}
	if domain.EnableRange {
		b.WriteString("        proxy_force_ranges on;\n")
	}
	for k, v := range domain.Headers {
		if k == "" || v == "" {
			continue
		}
		b.WriteString("        proxy_set_header " + k + " " + v + ";\n")
	}
	for k, v := range domain.ResponseHeaders {
		if k == "" || v == "" {
			continue
		}
		b.WriteString("        add_header " + k + " " + v + " always;\n")
	}

	scheme := "http"
	if strings.ToLower(domain.OriginProtocol) == "https" {
		scheme = "https"
	}
	b.WriteString("        proxy_pass " + scheme + "://" + domain.UpstreamKey + ";\n")
	if scheme == "https" {
		b.WriteString("        proxy_ssl_server_name on;\n")
		if domain.ProxySSLProtocols != "" {
			b.WriteString("        proxy_ssl_protocols " + domain.ProxySSLProtocols + ";\n")
		}
	}

	applyCacheDirectives(b, cacheCfg, rule)
}

func applyCacheDirectives(b *strings.Builder, cacheCfg *edgeCacheConfig, rule *edgeCacheRule) {
	if cacheCfg == nil || !cacheCfg.Enable {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	enabled := true
	if rule != nil && rule.Enable != nil && !*rule.Enable {
		enabled = false
	}
	if rule != nil && rule.NoCache {
		enabled = false
	}
	if !enabled {
		b.WriteString("        proxy_no_cache 1;\n")
		b.WriteString("        proxy_cache_bypass 1;\n")
		return
	}
	b.WriteString("        proxy_cache my_cache;\n")
	b.WriteString("        proxy_cache_lock on;\n")
	b.WriteString("        proxy_cache_lock_timeout 5s;\n")
	b.WriteString("        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;\n")
	b.WriteString("        proxy_cache_background_update on;\n")
	if rule != nil && rule.ForceCache {
		b.WriteString("        proxy_ignore_headers Cache-Control Expires;\n")
	}
	ttl := cacheCfg.DefaultTTL
	if rule != nil && rule.TTL > 0 {
		ttl = rule.TTL
	}
	if ttl > 0 {
		b.WriteString(fmt.Sprintf("        proxy_cache_valid 200 302 %ds;\n", ttl))
	}
	cacheKey := ""
	if rule != nil && rule.CacheKey != "" {
		cacheKey = rule.CacheKey
	} else if rule != nil && rule.IgnoreArgs {
		cacheKey = "$host$uri"
	} else {
		cacheKey = "$host$uri$is_args$args"
	}
	b.WriteString("        proxy_cache_key " + cacheKey + ";\n")
}

func writeHTTPGlobalConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "http_global.conf")
	var b strings.Builder
	cacheDir := filepath.ToSlash(filepath.Join(WorkDir, "cache"))
	cacheMaxSize := ""
	cacheZoneSize := ""
	if cfg != nil && cfg.HTTP != nil {
		cacheDir = fallbackString(toString(cfg.HTTP["proxy_cache_dir"]), cacheDir)
		cacheMaxSize = toString(cfg.HTTP["proxy_cache_max_size"])
		cacheZoneSize = toString(cfg.HTTP["proxy_cache_keys_zone_size"])
	}
	zoneSize := "50m"
	if cacheZoneSize != "" {
		zoneSize = cacheZoneSize
	}
	cacheLine := "proxy_cache_path " + cacheDir + " levels=1:2 keys_zone=my_cache:" + zoneSize + " inactive=24h use_temp_path=off"
	if cacheMaxSize != "" {
		cacheLine = cacheLine + " max_size=" + cacheMaxSize
	}
	b.WriteString(cacheLine + ";\n")

	if cfg != nil && cfg.HTTP != nil {
		writeHTTPDirectives(&b, cfg.HTTP)
		if v := toString(cfg.HTTP["proxy_cache_methods"]); v != "" {
			b.WriteString("proxy_cache_methods " + v + ";\n")
		}
		if v := toString(cfg.HTTP["custom_snippet"]); v != "" {
			if !strings.HasSuffix(v, "\n") {
				v += "\n"
			}
			b.WriteString(v)
		}
	}
	if cfg != nil {
		if v := strings.TrimSpace(cfg.Resolver); v != "" {
			b.WriteString("resolver " + v + ";\n")
		}
		if v := strings.TrimSpace(cfg.ResolverTimeout); v != "" {
			b.WriteString("resolver_timeout " + v + ";\n")
		}
		if logs := strings.TrimSpace(cfg.LogsDir); logs != "" {
			logs = strings.TrimRight(logs, "/")
			b.WriteString("access_log " + logs + "/access.json json_analytics;\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeStreamGlobalConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "stream_global.conf")
	var b strings.Builder
	if cfg != nil && cfg.Stream != nil {
		if v := toString(cfg.Stream["proxy_connect_timeout"]); v != "" {
			b.WriteString("proxy_connect_timeout " + v + ";\n")
		}
		if v := toString(cfg.Stream["proxy_timeout"]); v != "" {
			b.WriteString("proxy_timeout " + v + ";\n")
		}
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeMainConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "main.conf")
	var b strings.Builder
	if cfg == nil {
		return ioutil.WriteFile(confPath, []byte(""), 0644)
	}
	if v := strings.TrimSpace(cfg.WorkerProcesses); v != "" {
		b.WriteString("worker_processes " + v + ";\n")
	}
	if cfg.WorkerRlimitNofile > 0 {
		b.WriteString(fmt.Sprintf("worker_rlimit_nofile %d;\n", cfg.WorkerRlimitNofile))
	}
	if v := strings.TrimSpace(cfg.WorkerShutdownTimeout); v != "" {
		b.WriteString("worker_shutdown_timeout " + v + ";\n")
	}
	if logs := strings.TrimSpace(cfg.LogsDir); logs != "" {
		logs = strings.TrimRight(logs, "/")
		b.WriteString("error_log " + logs + "/error.log warn;\n")
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func writeEventsConfig(cfg *edgeNginxConfig) error {
	confPath := filepath.Join(WorkDir, "conf", "dynamic", "events.conf")
	var b strings.Builder
	if cfg != nil && cfg.WorkerConnections > 0 {
		b.WriteString(fmt.Sprintf("worker_connections %d;\n", cfg.WorkerConnections))
	}
	return ioutil.WriteFile(confPath, []byte(b.String()), 0644)
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func writeHTTPDirectives(b *strings.Builder, httpCfg map[string]interface{}) {
	if httpCfg == nil {
		return
	}
	directives := map[string]string{
		"proxy_request_buffering":     "proxy_request_buffering",
		"proxy_buffering":             "proxy_buffering",
		"proxy_http_version":          "proxy_http_version",
		"proxy_next_upstream":         "proxy_next_upstream",
		"proxy_max_temp_file_size":    "proxy_max_temp_file_size",
		"proxy_connect_timeout":       "proxy_connect_timeout",
		"proxy_send_timeout":          "proxy_send_timeout",
		"proxy_read_timeout":          "proxy_read_timeout",
		"client_max_body_size":        "client_max_body_size",
		"large_client_header_buffers": "large_client_header_buffers",
		"gzip":                        "gzip",
		"keepalive_timeout":           "keepalive_timeout",
		"keepalive_requests":          "keepalive_requests",
		"gzip_comp_level":             "gzip_comp_level",
		"gzip_http_version":           "gzip_http_version",
		"gzip_min_length":             "gzip_min_length",
		"gzip_vary":                   "gzip_vary",
		"server_tokens":               "server_tokens",
		"log_not_found":               "log_not_found",
		"default_type":                "default_type",
		"server":                      "server",
	}
	for key, directive := range directives {
		if value, ok := httpCfg[key]; ok {
			if rendered := renderValue(value); rendered != "" {
				b.WriteString(directive + " " + rendered + ";\n")
			}
		}
	}
}

func renderValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case bool:
		if v {
			return "on"
		}
		return "off"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return toString(value)
	}
}
