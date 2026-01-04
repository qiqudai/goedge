package main

import (
	"bytes"
	"cdn-common/acme"
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
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		pullTasks()
	}
}

func pullTasks() {
	req, _ := http.NewRequest("GET", API_BaseURL+"/api/v1/agent/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	body, status, err := doRequest(req, 10*time.Second, true)
	if err != nil {
		log.Printf("[Error] Task Pull Failed: %v", err)
		return
	}

	if status != 200 {
		debugLogInteraction("GET", req.URL.String(), status, nil, nil)
		log.Printf("[Warn] Task Pull Status: %d", status)
		return
	}

	debugLogInteraction("GET", req.URL.String(), status, nil, body)
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
		if ret, err := processTask(task.ID, task.Type, task.Data); err != nil {
			reportTask(task.ID, "fail", err.Error())
		} else {
			if ret == "" {
				ret = "ok"
			}
			reportTask(task.ID, "done", ret)
		}
	}
}

func processTask(id int64, taskType string, data string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "refresh_url":
		return "", purgeURLs(splitLines(data))
	case "refresh_dir":
		return "", purgeDirs(splitLines(data))
	case "preheat":
		return "", preheatURLs(splitLines(data))
	case "issue_cert":
		return "", issueCertTask(id, data)
	case "config_sync":
		return "", pullConfig()
	case "package_sync":
		return syncUserPackageTask(data)
	case "套餐同步":
		return syncUserPackageTask(data)
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

func issueCertTask(taskID int64, raw string) error {
	var payload issueCertTaskPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("invalid issue cert task payload")
	}
	if len(payload.Items) == 0 {
		return fmt.Errorf("no cert items")
	}
	webroot := filepath.Join(WorkDir, "cert", "acme")
	accountKey := filepath.Join(webroot, "account_"+strings.ToLower(payload.CA)+".key")
	issuer := acme.NewIssuer(acme.IssueOptions{
		Email:          payload.Email,
		CADirURL:       payload.CADirURL,
		Webroot:        webroot,
		AccountKeyPath: accountKey,
		Timeout:        60 * time.Second,
	})

	for _, item := range payload.Items {
		if len(item.Domains) == 0 || item.CertID == 0 {
			continue
		}
		result, err := issuer.Issue(item.Domains)
		if err != nil {
			if acme.IsRegisterRateLimited(err) {
				return fmt.Errorf("RATE_LIMITED: %s", err.Error())
			}
			return err
		}
		if err := reportIssuedCert(taskID, item.CertID, result.CertPEM, result.KeyPEM); err != nil {
			return err
		}
	}
	return nil
}

func reportIssuedCert(taskID int64, certID int64, certPEM string, keyPEM string) error {
	payload := map[string]interface{}{
		"cert_id":       certID,
		"cert":          certPEM,
		"key":           keyPEM,
		"issue_task_id": taskID,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", API_BaseURL+"/api/v1/agent/certs/issued", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+AuthToken)
	req.Header.Set("Content-Type", "application/json")
	respBody, status, err := doRequest(req, 15*time.Second, DebugMode)
	if err != nil {
		return err
	}
	debugLogInteraction("POST", req.URL.String(), status, body, respBody)
	if status != 200 {
		return fmt.Errorf("cert report failed: %d", status)
	}
	return nil
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

	readBody := DebugMode
	respBody, status, err := doRequest(req, 5*time.Second, readBody)
	if err != nil {
		log.Printf("[Error] Task Report Failed: %v", err)
		return
	}
	debugLogInteraction("POST", req.URL.String(), status, body, respBody)
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

type UserPackageSyncPayload struct {
	Packages []struct {
		PackageID int64           `json:"package_id"`
		Version   int             `json:"version"`
		Config    json.RawMessage `json:"config"`
	} `json:"packages"`
}

type AgentPackageConfig struct {
	Version  int    `json:"version"`
	Status   string `json:"status"`
	Limits   struct {
		Traffic    int64 `json:"traffic"`
		Bandwidth  int64 `json:"bandwidth"`
		Connection int64 `json:"connection"`
		Domain     int64 `json:"domain"`
	} `json:"limits"`
	Features struct {
		Websocket    bool `json:"websocket"`
		CustomCCRule bool `json:"custom_cc_rule"`
	} `json:"features"`
}

func syncUserPackageTask(raw string) (string, error) {
	var payload UserPackageSyncPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("invalid payload: %v", err)
	}

	packagesDir := filepath.Join(WorkDir, "packages")
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
	packagesDir := filepath.Join(WorkDir, "packages")
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
