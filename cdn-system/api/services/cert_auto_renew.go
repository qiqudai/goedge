package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"errors"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	certAutoRenewAdvance      = 7 * 24 * time.Hour
	certAutoRenewInterval     = 30 * time.Minute
	certAutoRenewRetryDelay   = 1 * time.Hour
	certAutoRenewInitFlagName = "cert_auto_renew_initialized"
)

// StartCertAutoRenewWorker checks expiring certs and auto-issues renewals.
func StartCertAutoRenewWorker() {
	go func() {
		ensureCertAutoRenewDefaults()
		runCertAutoRenew()
		ticker := time.NewTicker(certAutoRenewInterval)
		defer ticker.Stop()
		for range ticker.C {
			runCertAutoRenew()
		}
	}()
}

// RunCertAutoRenewOnce is a helper for manual validation.
func RunCertAutoRenewOnce() {
	ensureCertAutoRenewDefaults()
	runCertAutoRenew()
}

func runCertAutoRenew() {
	if db.DB == nil {
		return
	}
	now := time.Now()
	due := now.Add(certAutoRenewAdvance)

	var certs []models.Cert
	if err := db.DB.Where("enable = ? AND auto_renew = ? AND type <> ? AND (state = ? OR state = ?) AND expire_time IS NOT NULL AND expire_time <= ?",
		true, true, "upload", "ready", "success", due).
		Find(&certs).Error; err != nil {
		log.Printf("[CertAutoRenew] load certs failed: %v", err)
		return
	}
	if len(certs) == 0 {
		return
	}

	taskIDs := make([]int64, 0, len(certs))
	for _, cert := range certs {
		if cert.IssueTaskID > 0 {
			taskIDs = append(taskIDs, cert.IssueTaskID)
		}
	}
	taskMap := map[int64]models.Task{}
	if len(taskIDs) > 0 {
		var tasks []models.Task
		if err := db.DB.Select("id", "state", "end_at", "retry_at").Where("id IN ?", taskIDs).Find(&tasks).Error; err == nil {
			for _, task := range tasks {
				taskMap[task.ID] = task
			}
		}
	}

	renewIDs := make([]int64, 0, len(certs))
	for _, cert := range certs {
		if shouldSkipCertAutoRenew(cert, taskMap, now) {
			continue
		}
		renewIDs = append(renewIDs, int64(cert.ID))
	}
	if len(renewIDs) == 0 {
		return
	}
	log.Printf("[CertAutoRenew] reissue certs=%v", renewIDs)
	IssueCertsAsync(time.Now().Unix(), renewIDs)
}

func shouldSkipCertAutoRenew(cert models.Cert, taskMap map[int64]models.Task, now time.Time) bool {
	state := strings.ToLower(strings.TrimSpace(cert.State))
	switch state {
	case "waiting", "issuing", "dns_pending":
		return true
	}
	if cert.IssueTaskID == 0 {
		return false
	}
	task, ok := taskMap[cert.IssueTaskID]
	if !ok {
		return false
	}
	if task.RetryAt != nil && task.RetryAt.After(now) {
		return true
	}
	taskState := strings.ToLower(strings.TrimSpace(task.State))
	switch taskState {
	case "waiting", "running":
		return true
	case "retrying":
		return task.RetryAt == nil || task.RetryAt.After(now)
	}
	if task.EndAt != nil && now.Sub(*task.EndAt) < certAutoRenewRetryDelay {
		return true
	}
	return false
}

func ensureCertAutoRenewDefaults() {
	if db.DB == nil {
		return
	}
	enabled, err := loadCertAutoRenewFlag()
	if err != nil {
		log.Printf("[CertAutoRenew] load init flag failed: %v", err)
		return
	}
	if enabled {
		return
	}

	if err := db.DB.Model(&models.Cert{}).
		Where("enable = ? AND auto_renew = ? AND type <> ? AND (state = ? OR state = ? OR state = '' OR state IS NULL)",
			true, false, "upload", "ready", "success").
		Update("auto_renew", true).Error; err != nil {
		log.Printf("[CertAutoRenew] default enable failed: %v", err)
		return
	}
	if err := saveCertAutoRenewFlag(); err != nil {
		log.Printf("[CertAutoRenew] save init flag failed: %v", err)
	}
}

func loadCertAutoRenewFlag() (bool, error) {
	var cfg models.SysConfig
	err := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?",
		certAutoRenewInitFlagName, "system", "global", 0).
		First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return ParseBoolFlag(cfg.Value), nil
}

func saveCertAutoRenewFlag() error {
	now := time.Now()
	var cfg models.SysConfig
	query := db.DB.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?",
		certAutoRenewInitFlagName, "system", "global", 0)
	if err := query.First(&cfg).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		cfg = models.SysConfig{
			Name:      certAutoRenewInitFlagName,
			Value:     "1",
			Type:      "system",
			ScopeName: "global",
			ScopeID:   0,
			Enable:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return db.DB.Create(&cfg).Error
	}
	return query.Updates(map[string]interface{}{
		"value":     "1",
		"enable":    true,
		"update_at": now,
	}).Error
}
