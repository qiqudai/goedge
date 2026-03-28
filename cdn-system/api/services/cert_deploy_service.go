package services

import (
	"encoding/json"
	"fmt"
	"time"

	"cdn-api/db"
	"cdn-api/models"
)

type DeployCertPayload struct {
	CertID  int64   `json:"cert_id"`
	Version int     `json:"version"`
	NodeIDs []int64 `json:"node_ids"`
}

// CreateDeployTask creates a task to deploy a certificate to all relevant nodes
func CreateDeployTask(certID int64) error {
	var cert models.Cert
	if err := db.DB.First(&cert, certID).Error; err != nil {
		return err
	}

	nodeIDs, err := NewNodeService().GetUserNodes(int64(cert.UserID))
	if err != nil {
		return err
	}

	if len(nodeIDs) == 0 {
		return nil // No nodes to deploy to
	}

	payload := DeployCertPayload{
		CertID:  certID,
		Version: cert.Version,
		NodeIDs: nodeIDs,
	}

	data, _ := json.Marshal(payload)
	// Idempotency logic removed to match DB schema constraints

	task := models.Task{
		Type:     "deploy_cert",
		Name:     fmt.Sprintf("Deploy Cert %d v%d", certID, cert.Version),
		Data:     string(data),
		State:    "waiting",
		Enable:   true,
		CreateAt: time.Now(),
	}

	if err := db.DB.Create(&task).Error; err != nil {
		return err
	}
	TriggerDispatchPending()
	return nil
}
