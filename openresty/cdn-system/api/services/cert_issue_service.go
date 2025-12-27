package services

import (
	"encoding/json"
	"errors"
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

func IssueCertsAsync(ids []int64) {
	if len(ids) == 0 {
		return
	}
	go func() {
		_ = IssueCerts(ids)
	}()
}

func IssueCerts(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var certs []models.Cert
	if err := db.DB.Where("id IN ?", ids).Find(&certs).Error; err != nil {
		return err
	}
	pending := make([]models.Cert, 0)
	for _, cert := range certs {
		if strings.TrimSpace(cert.Type) == "" || strings.ToLower(cert.Type) == "upload" {
			continue
		}
		pending = append(pending, cert)
	}
	if len(pending) == 0 {
		return nil
	}

	rateLimited := make([]models.Cert, 0)
	for _, cert := range pending {
		if err := issueCertLocal(cert); err != nil {
			if IsRegisterRateLimited(err) {
				rateLimited = append(rateLimited, cert)
				continue
			}
			return err
		}
	}
	if len(rateLimited) == 0 {
		return nil
	}

	return dispatchCertsToNodes(rateLimited)
}

func issueCertLocal(cert models.Cert) error {
	domains := splitCertDomains(cert.Domain)
	if len(domains) == 0 {
		return errors.New("cert domain is empty")
	}
	issuer := NewHTTP01Issuer(cert.Type)
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

func UpdateIssuedCert(certID int64, certPEM string, keyPEM string, notBefore time.Time, notAfter time.Time, issueTaskID int64) error {
	updates := map[string]interface{}{
		"cert":        certPEM,
		"key":         keyPEM,
		"start_time":  notBefore,
		"expire_time": notAfter,
		"update_at":   time.Now(),
		"enable":      true,
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

func dispatchCertsToNodes(certs []models.Cert) error {
	nodes, err := loadAvailableNodes()
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return errors.New("no available nodes for cert issue")
	}

	grouped := groupCertsByCA(certs)
	for ca, items := range grouped {
		if err := dispatchByCA(ca, items, nodes); err != nil {
			return err
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

func dispatchByCA(ca string, items []IssueCertItem, nodes []models.Node) error {
	batchSize := 10
	nodeIndex := 0
	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		target := nodes[nodeIndex%len(nodes)]
		nodeIndex++

		payload := IssueCertTaskPayload{
			CA:       ca,
			CADirURL: BuildCADirURL(ca),
			Email:    strings.TrimSpace(config.App.AcmeEmail),
			Items:    items[start:end],
		}
		if err := createIssueTask(target.ID, payload); err != nil {
			return err
		}
	}
	return nil
}

func createIssueTask(nodeID int64, payload IssueCertTaskPayload) error {
	data, _ := json.Marshal(payload)
	meta, _ := json.Marshal(issueTaskMeta{TargetNodeID: nodeID})
	task := models.Task{
		Name:     "issue_cert",
		Type:     "issue_cert",
		Data:     string(data),
		Res:      string(meta),
		Enable:   true,
		State:    "waiting",
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return err
	}
	for _, item := range payload.Items {
		_ = db.DB.Model(&models.Cert{}).Where("id = ?", item.CertID).Update("issue_task_id", task.ID).Error
	}
	return nil
}

func loadAvailableNodes() ([]models.Node, error) {
	var nodes []models.Node
	if err := db.DB.Where("pid = 0 AND enable = ?", true).Find(&nodes).Error; err != nil {
		return nil, err
	}
	filtered := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		if IsNodeRateLimited(node.ID) {
			continue
		}
		filtered = append(filtered, node)
	}
	return filtered, nil
}
