package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
)

const TaskTypeHTTPSProbe = "https_probe"

type HTTPSProbePayload struct {
	SiteID         int64    `json:"site_id"`
	CertID         int64    `json:"cert_id"`
	Domains        []string `json:"domains"`
	Ports          []string `json:"ports"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type HTTPSProbeResult struct {
	Domain       string `json:"domain"`
	Port         string `json:"port"`
	OK           bool   `json:"ok"`
	TLSVersion   string `json:"tls_version,omitempty"`
	CertSubject  string `json:"cert_subject,omitempty"`
	CertNotAfter string `json:"cert_not_after,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	Error        string `json:"error,omitempty"`
}

func CreateHTTPSProbeTasksForSites(siteIDs []int64, certID int64) {
	if len(siteIDs) == 0 || certID == 0 {
		return
	}
	nodes := resolveScopedConfigSyncTargets("site", siteIDs)
	if len(nodes) == 0 {
		nodes = ConnectedNodeIDs()
	}
	var sites []models.Site
	if err := db.DB.Where("id IN ?", siteIDs).Find(&sites).Error; err != nil {
		log.Printf("[HTTPSProbe] load sites failed cert_id=%d err=%v", certID, err)
		return
	}
	for _, site := range sites {
		payload := HTTPSProbePayload{
			SiteID:         site.ID,
			CertID:         certID,
			Domains:        site.Domains,
			Ports:          resolveHTTPSProbePorts(site),
			TimeoutSeconds: 8,
		}
		if len(payload.Domains) == 0 || len(payload.Ports) == 0 {
			markSiteHTTPSFailed(site, "https probe has no domains or ports")
			continue
		}
		if len(nodes) == 0 {
			markSiteHTTPSFailed(site, "https probe has no online target nodes")
			continue
		}
		raw, _ := json.Marshal(payload)
		task := models.Task{
			Type:        TaskTypeHTTPSProbe,
			Name:        fmt.Sprintf("HTTPS Probe site=%d cert=%d", site.ID, certID),
			Data:        string(raw),
			State:       "waiting",
			Enable:      true,
			CreateAt:    time.Now(),
			TargetsJSON: NewTaskTargets(nodes).Marshal(),
		}
		if err := db.DB.Create(&task).Error; err != nil {
			log.Printf("[HTTPSProbe] create task failed site_id=%d cert_id=%d err=%v", site.ID, certID, err)
			markSiteHTTPSFailed(site, err.Error())
			continue
		}
	}
	TriggerDispatchPending()
}

func resolveHTTPSProbePorts(site models.Site) []string {
	ports := site.HttpsListen
	if len(ports) == 0 && strings.TrimSpace(site.HttpsListenRaw) != "" {
		_ = json.Unmarshal([]byte(site.HttpsListenRaw), &ports)
	}
	if len(ports) == 0 {
		return []string{"443"}
	}
	out := make([]string, 0, len(ports))
	seen := map[string]struct{}{}
	for _, port := range ports {
		port = strings.TrimSpace(port)
		if port == "" {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		out = append(out, port)
	}
	if len(out) == 0 {
		return []string{"443"}
	}
	return out
}

func HandleHTTPSProbeTaskFinished(taskID int64) {
	if taskID == 0 {
		return
	}
	var task models.Task
	if err := db.DB.Where("id = ? AND type = ?", taskID, TaskTypeHTTPSProbe).First(&task).Error; err != nil {
		return
	}
	var payload HTTPSProbePayload
	if err := json.Unmarshal([]byte(task.Data), &payload); err != nil {
		return
	}
	var site models.Site
	if err := db.DB.Where("id = ?", payload.SiteID).First(&site).Error; err != nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(task.State)) {
	case "done":
		activateSiteHTTPS(site, payload.CertID)
	case "fail":
		markSiteHTTPSFailed(site, strings.TrimSpace(task.Ret))
	}
}

func activateSiteHTTPS(site models.Site, certID int64) {
	settings := site.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}
	httpsCfg := getMap(settings, "https")
	if httpsCfg == nil {
		httpsCfg = map[string]interface{}{}
		settings["https"] = httpsCfg
	}
	httpsCfg["enable"] = true
	httpsCfg["state"] = "active"
	httpsCfg["certificate_id"] = certID
	httpsCfg["active_certificate_id"] = certID
	httpsCfg["pending_certificate_id"] = 0
	httpsCfg["last_error"] = ""
	httpsCfg["probe_at"] = time.Now().Format(time.RFC3339)
	httpsCfg["activated_at"] = time.Now().Format(time.RFC3339)
	raw, _ := json.Marshal(settings)
	updates := map[string]interface{}{
		"settings":  string(raw),
		"update_at": time.Now(),
	}
	if db.DB.Migrator().HasColumn(&models.Site{}, "cert_id") {
		updates["cert_id"] = certID
	}
	if err := db.DB.Model(&models.Site{}).Where("id = ?", site.ID).Updates(updates).Error; err != nil {
		log.Printf("[HTTPSProbe] activate site failed site_id=%d cert_id=%d err=%v", site.ID, certID, err)
		return
	}
	BumpConfigVersion("site", []int64{site.ID})
}

func markSiteHTTPSFailed(site models.Site, reason string) {
	settings := site.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}
	httpsCfg := getMap(settings, "https")
	if httpsCfg == nil {
		httpsCfg = map[string]interface{}{}
		settings["https"] = httpsCfg
	}
	httpsCfg["enable"] = false
	httpsCfg["state"] = "failed"
	httpsCfg["last_error"] = strings.TrimSpace(reason)
	raw, _ := json.Marshal(settings)
	if err := db.DB.Model(&models.Site{}).Where("id = ?", site.ID).Updates(map[string]interface{}{
		"settings":  string(raw),
		"update_at": time.Now(),
	}).Error; err != nil {
		log.Printf("[HTTPSProbe] mark site failed failed site_id=%d err=%v", site.ID, err)
		return
	}
	BumpConfigVersion("site", []int64{site.ID})
}
