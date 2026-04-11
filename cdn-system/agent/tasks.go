package main

import (
	"bufio"
	"cdn-common/acme"
	"cdn-common/i18n"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func startTaskPull() {
	log.Printf("[Info] Task pull disabled; waiting for WS dispatch")
}

func pullTasks() {
	log.Printf("[Info] Task pull disabled; waiting for WS dispatch")
}

type TaskProgressReporter func(percent int, message string) error

func processTask(id int64, taskType string, data string, report TaskProgressReporter) (string, error) {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "refresh_url":
		return "", purgeURLs(splitLines(data))
	case "refresh_dir":
		return "", purgeDirs(splitLines(data))
	case "clear_cache":
		var payload cacheClearPayload
		if raw := strings.TrimSpace(data); raw != "" {
			_ = json.Unmarshal([]byte(raw), &payload)
		}
		if len(payload.Domains) > 0 {
			return "", purgeDomains(payload.Domains)
		}
		cacheDir := resolveCacheDir()
		return "", clearCacheDir(cacheDir)
	case "preheat":
		return "", preheatURLs(splitLines(data))
	case "issue_cert":
		return issueCertTask(id, data)
	case "config_sync":
		if strings.TrimSpace(data) == "" {
			return "", nil
		}
		return applyConfigPayloadWithOptions([]byte(data), true)
	case "package_sync":
		return syncUserPackageTask(data)
	case i18n.T("agent.task_sync_package"):
		return syncUserPackageTask(data)
	case "agent_upgrade":
		return upgradeAgentPackage(data, report)
	default:
		return "", fmt.Errorf("unknown task type: %s", taskType)
	}
}

type issueCertItem struct {
	CertID  int64    `json:"cert_id"`
	Domains []string `json:"domains"`
}

type issueCertTaskPayload struct {
	CA       string          `json:"ca"`
	CADirURL string          `json:"ca_dir_url"`
	Email    string          `json:"email"`
	Items    []issueCertItem `json:"items"`
}

func issueCertTask(taskID int64, raw string) (string, error) {
	var payload issueCertTaskPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("invalid issue cert task payload")
	}
	if len(payload.Items) == 0 {
		return "", fmt.Errorf("no cert items")
	}
	webroot := filepath.Join(runtimeRoot(), "cert", "acme")
	fallbackCAs := buildIssueCAFallbackList(payload.CA)
	logLines := make([]string, 0, 16)

	for _, item := range payload.Items {
		if len(item.Domains) == 0 || item.CertID == 0 {
			continue
		}
		var lastErr error
		ok := false
		for idx, ca := range fallbackCAs {
			accountKey := filepath.Join(webroot, "account_"+ca+".key")
			caDir := resolveIssueCADirURL(ca, normalizeIssueCAName(payload.CA), payload.CADirURL)
			tokenStore := newAPITokenStore()
			issuer := acme.NewIssuer(acme.IssueOptions{
				Email:          payload.Email,
				CADirURL:       caDir,
				Webroot:        webroot,
				AccountKeyPath: accountKey,
				Timeout:        3 * time.Minute,
				TokenStore:     tokenStore,
			})

			result, err := issuer.Issue(item.Domains)
			if err != nil {
				lastErr = err
				logLines = append(logLines, fmt.Sprintf("cert=%d ca=%s failed: %s", item.CertID, ca, err.Error()))
				if idx+1 < len(fallbackCAs) {
					logLines = append(logLines, fmt.Sprintf("cert=%d switch_ca: %s -> %s", item.CertID, ca, fallbackCAs[idx+1]))
				}
				continue
			}
			if err := reportIssuedCert(taskID, item.CertID, result.CertPEM, result.KeyPEM); err != nil {
				lastErr = err
				logLines = append(logLines, fmt.Sprintf("cert=%d ca=%s issue ok but report failed: %s", item.CertID, ca, err.Error()))
				break
			}
			if ca == normalizeIssueCAName(payload.CA) {
				logLines = append(logLines, fmt.Sprintf("cert=%d issued by ca=%s", item.CertID, ca))
			} else {
				logLines = append(logLines, fmt.Sprintf("cert=%d issued by fallback ca=%s (primary=%s)", item.CertID, ca, normalizeIssueCAName(payload.CA)))
			}
			ok = true
			break
		}
		if !ok {
			ret := strings.Join(logLines, "\n")
			if lastErr == nil {
				lastErr = fmt.Errorf("unknown issue error")
			}
			return ret, fmt.Errorf("cert %d issue failed after fallback: %w", item.CertID, lastErr)
		}
	}
	return strings.Join(logLines, "\n"), nil
}

func normalizeIssueCAName(ca string) string {
	v := strings.ToLower(strings.TrimSpace(ca))
	switch v {
	case "", "lets", "let's encrypt", "lets encrypt":
		return "letsencrypt"
	case "letsencrypt", "zerossl", "google", "buypass":
		return v
	default:
		return "letsencrypt"
	}
}

func buildIssueCAFallbackList(primary string) []string {
	all := []string{"letsencrypt", "zerossl", "google", "buypass"}
	first := normalizeIssueCAName(primary)
	out := make([]string, 0, len(all))
	out = append(out, first)
	for _, ca := range all {
		if ca != first {
			out = append(out, ca)
		}
	}
	return out
}

func resolveIssueCADirURL(ca string, primary string, primaryDirURL string) string {
	if ca == primary && strings.TrimSpace(primaryDirURL) != "" {
		return strings.TrimSpace(primaryDirURL)
	}
	switch ca {
	case "zerossl":
		return "https://acme.zerossl.com/v2/DV90"
	case "buypass":
		return "https://api.buypass.com/acme/directory"
	case "google":
		return "https://dv.acme-v02.api.pki.goog/directory"
	default:
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}

func reportIssuedCert(taskID int64, certID int64, certPEM string, keyPEM string) error {
	return sendCertIssued(taskID, certID, certPEM, keyPEM, false, 0)
}

func reportTask(id int64, state string, ret string) {
	status := "success"
	if strings.ToLower(strings.TrimSpace(state)) == "fail" {
		status = "fail"
	}
	sendTaskAck("", id, "", status, ret, "")
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
	targets := make([]cacheDirTarget, 0, len(urls))
	for _, raw := range urls {
		target, ok := parseCacheDirTarget(raw)
		if !ok {
			continue
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return nil
	}
	cacheDir := resolveCacheDir()
	return purgeCacheEntriesByMatch(cacheDir, func(cacheKey string) bool {
		host, path := splitCacheKeyHostPath(cacheKey)
		if host == "" {
			return false
		}
		for _, target := range targets {
			if host != target.host {
				continue
			}
			if target.pathPrefix == "/" || strings.HasPrefix(path, target.pathPrefix) {
				return true
			}
		}
		return false
	})
}

func purgeURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url: %s", raw)
	}
	cacheDir := resolveCacheDir()
	keys := buildPurgeCacheKeys(u)
	var lastErr error
	for _, cacheKey := range keys {
		sum := md5.Sum([]byte(cacheKey))
		hash := fmt.Sprintf("%x", sum)
		if len(hash) < 3 {
			continue
		}
		path := filepath.Join(cacheDir, hash[0:1], hash[1:3], hash)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	return lastErr
}

type cacheClearPayload struct {
	Action  string   `json:"action"`
	SiteIDs []int64  `json:"site_ids"`
	Domains []string `json:"domains"`
}

type cacheDirTarget struct {
	host       string
	pathPrefix string
}

func parseCacheDirTarget(raw string) (cacheDirTarget, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return cacheDirTarget{}, false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return cacheDirTarget{}, false
	}
	pathPrefix := u.EscapedPath()
	if pathPrefix == "" {
		pathPrefix = "/"
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix += "/"
	}
	return cacheDirTarget{host: host, pathPrefix: pathPrefix}, true
}

func clearCacheDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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

func buildPurgeCacheKeys(u *url.URL) []string {
	if u == nil {
		return nil
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return nil
	}
	escapedPath := strings.TrimSpace(u.EscapedPath())
	if escapedPath == "" {
		escapedPath = "/"
	}
	decodedPath := strings.TrimSpace(u.Path)
	if decodedPath == "" {
		decodedPath = escapedPath
	}
	rawQuery := strings.TrimSpace(u.RawQuery)
	normalizedQuery := ""
	if values, err := url.ParseQuery(rawQuery); err == nil {
		normalizedQuery = values.Encode()
	}

	addUnique := func(out *[]string, seen map[string]struct{}, key string) {
		if strings.TrimSpace(key) == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		*out = append(*out, key)
	}

	seen := make(map[string]struct{}, 8)
	out := make([]string, 0, 8)
	addUnique(&out, seen, host+escapedPath)
	addUnique(&out, seen, host+decodedPath)
	if rawQuery != "" {
		addUnique(&out, seen, host+escapedPath+"?"+rawQuery)
		addUnique(&out, seen, host+decodedPath+"?"+rawQuery)
	}
	if normalizedQuery != "" && normalizedQuery != rawQuery {
		addUnique(&out, seen, host+escapedPath+"?"+normalizedQuery)
		addUnique(&out, seen, host+decodedPath+"?"+normalizedQuery)
	}
	return out
}

func purgeDomains(domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		trimmed := strings.ToLower(strings.TrimSpace(domain))
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	cacheDir := resolveCacheDir()
	return purgeCacheEntriesByMatch(cacheDir, func(cacheKey string) bool {
		host, _ := splitCacheKeyHostPath(cacheKey)
		if host == "" {
			return false
		}
		_, ok := set[host]
		return ok
	})
}

func purgeCacheEntriesByMatch(cacheDir string, matcher func(cacheKey string) bool) error {
	cacheDir = strings.TrimSpace(cacheDir)
	if cacheDir == "" || matcher == nil {
		return nil
	}
	var lastErr error
	_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			lastErr = err
			return nil
		}
		if info.IsDir() {
			return nil
		}
		cacheKey, ok := readCacheKey(path)
		if !ok {
			return nil
		}
		if !matcher(cacheKey) {
			return nil
		}
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			lastErr = rmErr
		}
		return nil
	})
	return lastErr
}

func readCacheKey(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	// nginx cache header keeps `KEY: ...` in the metadata header region.
	reader := bufio.NewReader(file)
	for i := 0; i < 64; i++ {
		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			if idx := strings.Index(line, "KEY: "); idx >= 0 {
				key := strings.TrimSpace(line[idx+5:])
				if key != "" {
					return key, true
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	return "", false
}

func splitCacheKeyHostPath(cacheKey string) (string, string) {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return "", ""
	}
	idx := strings.IndexAny(cacheKey, "/?")
	if idx <= 0 {
		return strings.ToLower(cacheKey), "/"
	}
	host := strings.ToLower(strings.TrimSpace(cacheKey[:idx]))
	path := cacheKey[idx:]
	if strings.HasPrefix(path, "?") {
		path = "/" + path
	}
	return host, path
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

type UserPackageSyncPayload struct {
	Packages []struct {
		PackageID int64           `json:"package_id"`
		Version   int             `json:"version"`
		Config    json.RawMessage `json:"config"`
	} `json:"packages"`
}

type AgentPackageConfig struct {
	Version int    `json:"version"`
	Status  string `json:"status"`
	Limits  struct {
		Traffic    int64  `json:"traffic"`
		Bandwidth  string `json:"bandwidth"`
		Connection int64  `json:"connection"`
		Domain     int64  `json:"domain"`
	} `json:"limits"`
	Features struct {
		Websocket    bool `json:"websocket"`
		CustomCCRule bool `json:"custom_cc_rule"`
		L2Origin     bool `json:"l2_origin"`
	} `json:"features"`
}

func syncUserPackageTask(raw string) (string, error) {
	var payload UserPackageSyncPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("invalid payload: %v", err)
	}

	packagesDir := filepath.Join(runtimeRoot(), "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return "", fmt.Errorf("create packages dir failed: %v", err)
	}

	var applied []map[string]interface{}

	for _, pkg := range payload.Packages {
		var parsed AgentPackageConfig
		if err := json.Unmarshal(pkg.Config, &parsed); err != nil {
			return "", fmt.Errorf("invalid package config: %v", err)
		}
		filename := fmt.Sprintf("%d.json", pkg.PackageID)
		targetPath := filepath.Join(packagesDir, filename)

		// Idempotency Check
		currentVersion := int64(0)
		if existing, err := ioutil.ReadFile(targetPath); err == nil {
			// Try to parse existing to get version
			var existingMeta struct {
				Version int `json:"version"`
			}
			if json.Unmarshal(existing, &existingMeta) == nil {
				currentVersion = int64(existingMeta.Version)
			}
		}

		if currentVersion >= int64(pkg.Version) {
			// Already up to date
			applied = append(applied, map[string]interface{}{
				"package_id": pkg.PackageID,
				"version":    pkg.Version,
				"status":     "skipped",
			})
			continue
		}

		// Write Atomic
		tmpPath := targetPath + ".tmp"
		// Ensure config is written as string/bytes
		if err := ioutil.WriteFile(tmpPath, pkg.Config, 0644); err != nil {
			return "", fmt.Errorf("write tmp failed: %v", err)
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			return "", fmt.Errorf("rename failed: %v", err)
		}

		localConfigMu.Lock()
		if LocalPackages == nil {
			LocalPackages = make(map[int64]AgentPackageConfig)
		}
		LocalPackages[pkg.PackageID] = parsed
		localConfigMu.Unlock()

		applied = append(applied, map[string]interface{}{
			"package_id": pkg.PackageID,
			"version":    pkg.Version,
			"status":     "updated",
		})
	}

	if applied == nil {
		applied = make([]map[string]interface{}, 0)
	}
	res, _ := json.Marshal(applied)
	return string(res), nil
}

func loadPersistedPackages() {
	packagesDir := filepath.Join(runtimeRoot(), "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return
	}

	loaded := 0
	localConfigMu.Lock()
	if LocalPackages == nil {
		LocalPackages = make(map[int64]AgentPackageConfig)
	}
	localConfigMu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		idStr := strings.TrimSuffix(name, ".json")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id == 0 {
			continue
		}
		data, err := os.ReadFile(filepath.Join(packagesDir, name))
		if err != nil {
			continue
		}
		var parsed AgentPackageConfig
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}
		localConfigMu.Lock()
		LocalPackages[id] = parsed
		localConfigMu.Unlock()
		loaded++
	}
	if loaded > 0 {
		log.Printf("[Info] Loaded %d package configs from disk", loaded)
	}
}
