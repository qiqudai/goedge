package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/acme"
)

// StartCertIssueWorker starts the worker to process certificate issuance tasks
func StartCertIssueWorker() {
	go func() {
		for {
			var tasks []models.Task
			if err := db.DB.Where("type = ? AND state IN ?", "issue_cert_v2", []string{"waiting", "retrying"}).
				Order("id asc").Limit(10).Find(&tasks).Error; err != nil {
				time.Sleep(5 * time.Second)
				continue
			}

			if len(tasks) == 0 {
				time.Sleep(2 * time.Second)
				continue
			}

			for _, task := range tasks {
				processIssueTask(task)
			}
		}
	}()
}

func processIssueTask(task models.Task) {
	// Update state to running
	db.DB.Model(&task).Updates(map[string]interface{}{"state": "running", "update_at": time.Now()})

	var payload struct {
		CertID int64 `json:"cert_id"`
	}
	if err := json.Unmarshal([]byte(task.Data), &payload); err != nil {
		failTask(task, "invalid payload")
		return
	}

	var cert models.Cert
	if err := db.DB.First(&cert, payload.CertID).Error; err != nil {
		failTask(task, "cert not found")
		return
	}

	// Update Cert State
	db.DB.Model(&cert).Update("state", "issuing")

	// Execute Issuance
	if err := issueCertLocal(cert, task.ID); err != nil {
		// Retry Logic
		errMsg := err.Error()
		if IsRegisterRateLimited(err) {
			// Backoff
			retryAt := time.Now().Add(1 * time.Hour)
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":     "retrying",
				"err":       errMsg,
				"err_times": task.ErrTimes + 1,
				"retry_at":  retryAt,
			})
			db.DB.Model(&cert).Update("state", "fail")
			return
		}
		
		// General Retry (Max 3 times)
		if task.ErrTimes < 3 {
			retryAt := time.Now().Add(time.Duration(task.ErrTimes*2) * time.Minute)
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":     "retrying",
				"err":       errMsg,
				"err_times": task.ErrTimes + 1,
				"retry_at":  retryAt,
			})
			db.DB.Model(&cert).Update("state", "fail") 
		} else {
			failTask(task, errMsg)
			db.DB.Model(&cert).Update("state", "fail")
		}
		return
	}

	// Success
	completeTask(task)
	// Create Deploy Task
	if err := CreateDeployTask(int64(cert.ID)); err != nil {
		fmt.Printf("Failed to create deploy task for cert %d: %v\n", cert.ID, err)
	}
}

func failTask(task models.Task, err string) {
	db.DB.Model(&task).Updates(map[string]interface{}{
		"state":     "fail",
		"err":       err,
		"update_at": time.Now(),
	})
}

func completeTask(task models.Task) {
	db.DB.Model(&task).Updates(map[string]interface{}{
		"state":     "success",
		"err":       "",
		"update_at": time.Now(),
	})
}

// IssueCertsAsync now schedules tasks
func IssueCertsAsync(ids []int64) {
	if len(ids) == 0 {
		return
	}
	go func() {
		for _, id := range ids {
			CreateIssueTask(id)
		}
	}()
}

func CreateIssueTask(certID int64) error {
	key := fmt.Sprintf("CERT_ISSUE:%d", certID)
	payload := map[string]interface{}{"cert_id": certID}
	data, _ := json.Marshal(payload)
	
	// Create Task using generic service (reusing CreateDNSTask logic but tailored here or generic)
	// We can reuse CreateDNSTask if we export it or duplicate logic. 
	// To minimize coupling, I'll inline idempotency logic here for now or rename CreateDNSTask to CreateGenericTask later.
	// For now: inline.
	
	var existing models.Task
	if err := db.DB.Where("idempotency_key = ? AND state IN ?", key, []string{"waiting", "running", "retrying"}).First(&existing).Error; err == nil {
		return nil
	}

	task := models.Task{
		Type:           "issue_cert_v2",
		Name:           "Issue Cert " + fmt.Sprint(certID),
		Data:           string(data),
		IdempotencyKey: key,
		State:          "waiting",
		Enable:         true,
		CreateAt:       time.Now(),
	}
	return db.DB.Create(&task).Error
}


func issueCertLocal(cert models.Cert, taskID int64) error {
	domains := splitCertDomains(cert.Domain)
	if len(domains) == 0 {
		return errors.New("cert domain is empty")
	}
	issuer := NewHTTP01Issuer(cert.Type)
	result, err := issuer.Issue(domains)
	if err != nil {
		return err
	}
	
	// Encrypt Key
	encryptedKey, err := Crypto.Encrypt(result.KeyPEM)
	if err != nil {
		return fmt.Errorf("encryption failed: %v", err)
	}

	return UpdateIssuedCert(int64(cert.ID), result.CertPEM, encryptedKey, result.NotBefore, result.NotAfter, taskID, cert.Version+1)
}

func splitCertDomains(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func UpdateIssuedCert(certID int64, certPEM string, keyCipher string, notBefore time.Time, notAfter time.Time, issueTaskID int64, newVersion int) error {
	updates := map[string]interface{}{
		"cert":          certPEM,
		"key":           keyCipher, // Already Encrypted
		"start_time":    notBefore,
		"expire_time":   notAfter,
		"update_at":     time.Now(),
		"enable":        true,
		"version":       newVersion,
		"state":         "ready",
		"issue_task_id": issueTaskID,
	}
	
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func ParseCertTimes(certPEM string) (time.Time, time.Time, error) {
	return acme.ParseCertTimes(certPEM)
}

// Deprecated: dispatchCertsToNodes is replaced by CERT_DEPLOY task
func dispatchCertsToNodes(certs []models.Cert) error {
	return nil
}

