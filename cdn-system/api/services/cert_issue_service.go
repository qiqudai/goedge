package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/acme"
)

type IssueCertItem struct {
	CertID  int64    `json:"cert_id"`
	Domains []string `json:"domains"`
}

type IssueCertTaskPayload struct {
	CA       string          `json:"ca"`
	CADirURL string          `json:"ca_dir_url"`
	Email    string          `json:"email"`
	Items    []IssueCertItem `json:"items"`
}

type issueTaskMeta struct {
	TargetNodeID int64 `json:"target_node_id"`
}

const maxCertIssueAttempts = 3

// IssueCertsAsync creates tasks and starts processing
func IssueCertsAsync(batchID int64, ids []int64) {
	if len(ids) == 0 {
		return
	}
	go func() {
		certs, err := loadCertsForIssue(ids)
		if err != nil {
			log.Printf("[CertIssue] load certs failed: %v", err)
			return
		}
		if len(certs) == 0 {
			return
		}

		localCerts := make([]models.Cert, 0, len(certs))
		nodeCerts := make([]models.Cert, 0, len(certs))
		for _, cert := range certs {
			if requiresDNSChallenge(cert) {
				localCerts = append(localCerts, cert)
			} else {
				nodeCerts = append(nodeCerts, cert)
			}
		}

		if len(nodeCerts) > 0 {
			if err := dispatchCertsToNodes(batchID, nodeCerts); err != nil {
				log.Printf("[CertIssue] dispatch to nodes failed: %v", err)
				markCertsIssueFailed(nodeCerts, err.Error())
			}
		}

		for _, cert := range localCerts {
			certID := int64(cert.ID)
			go processUniqueIssueTask(batchID, certID)
		}
	}()
}

func processUniqueIssueTask(batchID int64, certID int64) {
	var task models.Task
	var cert models.Cert
	log.Printf("[CertIssue] start batch=%d cert_id=%d", batchID, certID)

	defer func() {
		if r := recover(); r != nil {
			errReason := fmt.Sprintf("Panic: %v", r)
			if task.ID != 0 {
				failTask(&task, errReason)
			}
			// Update cert state if cert found
			if cert.ID != 0 {
				db.DB.Model(&models.Cert{ID: cert.ID}).Update("state", "fail")
			}
		}
	}()

	// 1. Create Task
	task = models.Task{
		Type:     "issue_cert",
		Name:     "Issue Cert " + strings.TrimSpace(strconv.FormatInt(certID, 10)),
		State:    "waiting",
		PID:      batchID,
		Enable:   true,
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		log.Printf("[CertIssue] create task failed cert_id=%d err=%v", certID, err)
		return
	}

	// 2. Associate with Cert
	if err := db.DB.First(&cert, certID).Error; err != nil {
		log.Printf("[CertIssue] cert not found cert_id=%d err=%v", certID, err)
		failTask(&task, "Cert not found")
		return
	}
	db.DB.Model(&cert).Updates(map[string]interface{}{
		"issue_task_id": task.ID,
		"task_id":       task.ID,
		"state":         "waiting",
		"ret":           "",
	})

	// 3. Issue with retry
	started := false
	for {
		if err := db.DB.First(&cert, certID).Error; err != nil {
			failTask(&task, "Cert not found")
			return
		}
		if !started {
			now := time.Now()
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":    "running",
				"start_at": &now,
			})
			db.DB.Model(&models.Cert{ID: cert.ID}).Update("state", "issuing")
			started = true
		} else {
			db.DB.Model(&task).Update("state", "running")
		}

		if err := issueCertLocal(cert); err != nil {
			errMsg := err.Error()
			log.Printf("[CertIssue] issue failed cert_id=%d err=%v", certID, err)
			db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
				"ret": errMsg,
			})
			if isFatalIssueError(err) {
				failTask(&task, errMsg)
				db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
					"state": "fail",
					"ret":   errMsg,
				})
				return
			}

			delay := nextCertRetryDelay(task.ErrTimes)
			retryAt := time.Now().Add(delay)
			task.ErrTimes++
			if task.ErrTimes >= maxCertIssueAttempts {
				failTask(&task, errMsg)
				db.DB.Model(&models.Cert{ID: cert.ID}).Updates(map[string]interface{}{
					"state": "fail",
					"ret":   errMsg,
				})
				return
			}
			db.DB.Model(&task).Updates(map[string]interface{}{
				"state":     "retrying",
				"ret":       errMsg,
				"retry_at":  retryAt,
				"err_times": task.ErrTimes,
			})
			time.Sleep(delay)
			continue
		}

		// Success
		completeTask(&task)
		log.Printf("[CertIssue] success cert_id=%d", certID)
		return
	}
}

func failTask(task *models.Task, reason string) {
	db.DB.Model(task).Updates(map[string]interface{}{
		"state":  "fail",
		"ret":    reason,
		"end_at": time.Now(),
	})
}

func completeTask(task *models.Task) {
	db.DB.Model(task).Updates(map[string]interface{}{
		"state":  "success",
		"ret":    "",
		"end_at": time.Now(),
	})
}

func issueCertLocal(cert models.Cert) error {
	domains := splitCertDomains(cert.Domain)
	if len(domains) == 0 {
		return errors.New("cert domain is empty")
	}
	var issuer *acme.Issuer
	if requiresDNSChallenge(cert) {
		provider, err := BuildDNSChallengeProvider(cert)
		if err != nil {
			return err
		}
		issuer = NewDNS01Issuer(cert.Type, provider)
	} else {
		issuer = NewHTTP01Issuer(cert.Type)
	}
	result, err := issuer.Issue(domains)
	if err != nil {
		return err
	}
	return UpdateIssuedCert(int64(cert.ID), result.CertPEM, result.KeyPEM, result.NotBefore, result.NotAfter, 0)
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

func requiresDNSChallenge(cert models.Cert) bool {
	if cert.DNSAPI != nil && *cert.DNSAPI > 0 {
		return true
	}
	return hasWildcardDomain(splitCertDomains(cert.Domain))
}

func hasWildcardDomain(domains []string) bool {
	for _, domain := range domains {
		if strings.HasPrefix(strings.TrimSpace(domain), "*.") {
			return true
		}
	}
	return false
}

func UpdateIssuedCert(certID int64, certPEM string, keyPEM string, notBefore time.Time, notAfter time.Time, issueTaskID int64) error {
	updates := map[string]interface{}{
		"cert":        certPEM,
		"key":         keyPEM,
		"start_time":  notBefore,
		"expire_time": notAfter,
		"enable":      true,
		"state":       "ready",
		"ret":         "",
	}
	var current models.Cert
	if err := db.DB.Select("id", "type", "auto_renew").Where("id = ?", certID).First(&current).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(current.Type)) != "upload" && !current.AutoRenew {
			updates["auto_renew"] = true
		}
	}
	if issueTaskID != 0 {
		updates["issue_task_id"] = issueTaskID
	} else {
		updates["issue_task_id"] = nil
	}
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(updates).Error; err != nil {
		return err
	}
	BumpConfigVersion("cert", []int64{certID})
	return nil
}

func ParseCertTimes(certPEM string) (time.Time, time.Time, error) {
	return acme.ParseCertTimes(certPEM)
}

func dispatchCertsToNodes(batchID int64, certs []models.Cert) error {
	nodes, err := loadAvailableNodes()
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return errors.New("no available nodes for cert issue")
	}
	email := strings.TrimSpace(config.App.AcmeEmail)
	if email == "" {
		return errors.New("acme_email is required")
	}

	startIndex := int(batchID % int64(len(nodes)))
	nodeIndex := startIndex

	for _, cert := range certs {
		domains := splitCertDomains(cert.Domain)
		if len(domains) == 0 {
			markCertIssueFailed(int64(cert.ID), "cert domain is empty")
			continue
		}
		ca := strings.ToLower(strings.TrimSpace(cert.Type))
		if ca == "" {
			ca = "letsencrypt"
		}
		payload := IssueCertTaskPayload{
			CA:       ca,
			CADirURL: BuildCADirURL(ca),
			Email:    email,
			Items: []IssueCertItem{
				{
					CertID:  int64(cert.ID),
					Domains: domains,
				},
			},
		}
		target := nodes[nodeIndex%len(nodes)]
		nodeIndex++

		taskName := fmt.Sprintf("Issue Cert %d", cert.ID)
		task, err := createIssueTask(batchID, target.ID, taskName, payload)
		if err != nil {
			markCertIssueFailed(int64(cert.ID), err.Error())
			continue
		}
		if err := db.DB.Model(&models.Cert{}).Where("id = ?", cert.ID).Updates(map[string]interface{}{
			"issue_task_id": task.ID,
			"task_id":       task.ID,
			"state":         "waiting",
			"ret":           "",
		}).Error; err != nil {
			log.Printf("[CertIssue] update cert task id failed cert_id=%d err=%v", cert.ID, err)
		}
	}
	return nil
}

func groupCertsByCA(certs []models.Cert) map[string][]IssueCertItem {
	result := map[string][]IssueCertItem{}
	for _, cert := range certs {
		ca := strings.ToLower(strings.TrimSpace(cert.Type))
		if ca == "" {
			ca = "letsencrypt"
		}
		item := IssueCertItem{
			CertID:  int64(cert.ID),
			Domains: splitCertDomains(cert.Domain),
		}
		if len(item.Domains) == 0 {
			continue
		}
		result[ca] = append(result[ca], item)
	}
	return result
}

func createIssueTask(batchID int64, nodeID int64, name string, payload IssueCertTaskPayload) (models.Task, error) {
	data, _ := json.Marshal(payload)
	meta, _ := json.Marshal(issueTaskMeta{TargetNodeID: nodeID})
	task := models.Task{
		PID:      batchID,
		Name:     name,
		Type:     "issue_cert",
		Data:     string(data),
		Res:      string(meta),
		Enable:   true,
		State:    "waiting",
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return task, err
	}
	for _, item := range payload.Items {
		_ = db.DB.Model(&models.Cert{}).Where("id = ?", item.CertID).Update("issue_task_id", task.ID).Error
	}
	TriggerDispatchPending()
	return task, nil
}

func loadAvailableNodes() ([]models.Node, error) {
	var nodes []models.Node
	if err := db.DB.Where("pid = 0 AND enable = ?", true).Order("id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	candidates := make([]models.Node, 0, len(nodes))
	online := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		if IsNodeRateLimited(node.ID) {
			continue
		}
		candidates = append(candidates, node)
		if IsNodeOnline(node.ID, 90*time.Second) {
			online = append(online, node)
		}
	}
	if len(online) > 0 {
		return online, nil
	}
	return candidates, nil
}

func loadCertsForIssue(ids []int64) ([]models.Cert, error) {
	var certs []models.Cert
	if err := db.DB.Where("id IN ?", ids).Order("id asc").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

func markCertsIssueFailed(certs []models.Cert, reason string) {
	for _, cert := range certs {
		markCertIssueFailed(int64(cert.ID), reason)
	}
}

func markCertIssueFailed(certID int64, reason string) {
	if certID == 0 {
		return
	}
	updates := map[string]interface{}{
		"state": "fail",
		"ret":   strings.TrimSpace(reason),
	}
	if err := db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(updates).Error; err != nil {
		log.Printf("[CertIssue] mark cert failed cert_id=%d err=%v", certID, err)
	}
}

func MarkIssueTaskFailed(taskID int64, reason string) {
	if taskID == 0 {
		return
	}
	updates := map[string]interface{}{
		"state": "fail",
		"ret":   strings.TrimSpace(reason),
	}
	if err := db.DB.Model(&models.Cert{}).
		Where("issue_task_id = ? AND state <> ?", taskID, "ready").
		Updates(updates).Error; err != nil {
		log.Printf("[CertIssue] mark task certs failed task_id=%d err=%v", taskID, err)
	}
}

func nextCertRetryDelay(errTimes int) time.Duration {
	delays := []int{5, 10, 20, 30, 60, 60, 60}
	if errTimes < len(delays) {
		return time.Duration(delays[errTimes]) * time.Minute
	}
	return 60 * time.Minute
}

func isFatalIssueError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "cert not found") ||
		strings.Contains(msg, "domain is empty") ||
		strings.Contains(msg, "acme:error") ||
		strings.Contains(msg, "dns problem") ||
		strings.Contains(msg, "nxdomain") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "timeout")
}
