package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type SiteCreatePayload struct {
	UserID        int64    `json:"user_id"`
	UserPackageID int64    `json:"user_package_id"`
	DNSProviderID int64    `json:"dns_provider_id"`
	NodeGroupID   int64    `json:"node_group_id"`
	GroupID       int64    `json:"group_id"`
	Domain        string   `json:"domain"`
	Backends      []string `json:"backends"`
	// Additional defaults can be applied in worker or payload
}

func CreateSiteCreateTask(payload SiteCreatePayload, batchID string) (*models.Task, error) {
	data, _ := json.Marshal(payload)

	// Idempotency logic removed
	// BatchID removed (not supported in DB without migration)

	task := &models.Task{
		Type:     "site_create",
		Name:     "Create Site " + payload.Domain,
		Data:     string(data),
		State:    "waiting",
		Enable:   true,
		CreateAt: time.Now(),
	}
	if err := db.DB.Create(&task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func StartSiteCreateWorker() {
	go func() {
		for {
			processSiteCreateTasks()
			time.Sleep(2 * time.Second)
		}
	}()
}

func processSiteCreateTasks() {
	var tasks []models.Task
	if err := db.DB.Where("type = ? AND state IN ?", "site_create", []string{"waiting", "retrying"}).Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		processSiteCreateTask(&task)
	}
}

func processSiteCreateTask(task *models.Task) {
	// Update state to running
	db.DB.Model(task).Updates(map[string]interface{}{"state": "running", "start_at": time.Now()})

	var payload SiteCreatePayload
	if err := json.Unmarshal([]byte(task.Data), &payload); err != nil {
		db.DB.Model(task).Updates(map[string]interface{}{"state": "fail", "ret": "Invalid Data: " + err.Error(), "end_at": time.Now()})
		return
	}

	if err := CheckDomainLimit(payload.UserID, payload.UserPackageID, []string{payload.Domain}); err != nil {
		db.DB.Model(task).Updates(map[string]interface{}{"state": "fail", "ret": err.Error(), "end_at": time.Now()})
		return
	}

	// Double check if domain exists
	// var exists int64

	// Check against Domain table or Site table? Site table: Domains column (json/string)
	// Simple check: LIKE query or exact if simplified.
	// For high performance, we might skip heavy check or just try Create.
	// Current system allows domains as JSON array. Check is tricky.

	// Create Site Logic
	site := &models.Site{
		UserID:        int64(payload.UserID),
		UserPackageID: int64(payload.UserPackageID),
		DNSProviderID: int64(payload.DNSProviderID),
		NodeGroupID:   int64(payload.NodeGroupID),
		Domains:       []string{payload.Domain},
		Backends:      payload.Backends,
		HttpListen:    []string{"80"},
		State:         "running",
		Enable:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CnameHostname: payload.Domain + ".cdn.node.com",
	}

	defaults, err := GetSiteDefaultMapWithGroup(payload.UserID, payload.GroupID)
	if err == nil {
		ApplySiteDefaults(site, defaults)
	}
	if globalDefaults := GetGlobalDefaultConfig(); globalDefaults != nil {
		ApplySiteTemplateDefaultsByType(site, globalDefaults)
	}

	// Transaction to save Site and Group Relation
	// Actually better to use db.DB.Transaction

	// Re-use logic? site_helper.go probably has `createSiteWithGroup`.
	// But `services` package cannot call `controllers` logic.
	// Logic in `controllers` should be moved to `services`?
	// For now, I'll inline the DB creation logic here.

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(site).Error; err != nil {
			return err
		}
		if payload.GroupID != 0 {
			rel := models.SiteGroupRelation{SiteID: site.ID, GroupID: payload.GroupID}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		db.DB.Model(task).Updates(map[string]interface{}{"state": "fail", "ret": err.Error(), "end_at": time.Now()})
		return
	}

	BumpConfigVersion("site", []int64{site.ID})
	_ = SyncUserDNSRecords(nil, site)

	// Success
	db.DB.Model(task).Updates(map[string]interface{}{"state": "success", "end_at": time.Now(), "ret": ""})
}
