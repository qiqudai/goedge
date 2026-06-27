package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"fmt"
	"strings"
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
	payload.Domain = normalizeTaskSiteDomain(payload.Domain)
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
		if db.RecoverIfConnectionError(err) {
			_ = db.DB.Where("type = ? AND state IN ?", "site_create", []string{"waiting", "retrying"}).Find(&tasks).Error
		}
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
	payload.Domain = normalizeTaskSiteDomain(payload.Domain)
	if payload.Domain == "" {
		db.DB.Model(task).Updates(map[string]interface{}{"state": "fail", "ret": "domain is required", "end_at": time.Now()})
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
	}
	applySiteCnameForTask(site, payload.Domain)

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
		if conflictDomain, err := findTaskSiteDomainConflict(tx, site.Domains, 0); err != nil {
			return err
		} else if conflictDomain != "" {
			return fmt.Errorf("domain already exists: %s", conflictDomain)
		}

		dbTx := tx
		omitColumns := siteMissingColumnsForTask(tx)
		if len(omitColumns) > 0 {
			dbTx = dbTx.Omit(omitColumns...)
		}
		// Always omit cert_id on create to keep compatibility with schemas lacking this column.
		dbTx = dbTx.Omit("CertID", "cert_id")
		if site.RegionID == 0 {
			dbTx = dbTx.Omit("RegionID")
		}
		if site.NodeGroupID == 0 {
			dbTx = dbTx.Omit("NodeGroupID")
		}
		if !site.EnableBackupGroup || site.BackupNodeGroupID == 0 {
			dbTx = dbTx.Omit("BackupNodeGroupID")
		}
		selectCols := selectSiteCreateColumns(tx, site)
		if len(selectCols) > 0 {
			dbTx = dbTx.Select(selectCols)
		}
		if err := dbTx.Create(site).Error; err != nil {
			if isUnknownColumnError(err, "cert_id") {
				if retryErr := dbTx.Omit("CertID", "cert_id").Create(site).Error; retryErr == nil {
					goto created
				} else {
					return retryErr
				}
			}
			return err
		}
	created:
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

func applySiteCnameForTask(site *models.Site, domain string) {
	if site == nil || site.UserPackageID == 0 {
		return
	}
	var pkg models.UserPackage
	if err := db.DB.Select("cname_mode", "cname_hostname", "cname_domain", "record_id").
		Where("id = ?", site.UserPackageID).
		First(&pkg).Error; err != nil {
		return
	}

	pkgMode := strings.TrimSpace(pkg.CnameMode)
	pkgDomain := strings.TrimSpace(pkg.CnameDomain)
	if pkgDomain == "" {
		pkgDomain = "cdn.node.com"
	}

	if pkgMode == "package" {
		pkgHost := strings.TrimSpace(pkg.CnameHostname)
		if pkgHost == "" {
			pkgHost = strings.TrimSpace(pkg.RecordID)
		}
		if pkgHost != "" {
			site.CnameMode = "package"
			site.CnameDomain = pkgDomain
			site.CnameHostname = buildSiteCnameForTask(pkgHost, pkgDomain)
			return
		}
	}

	if strings.TrimSpace(domain) != "" {
		site.CnameMode = "domain"
		site.CnameDomain = pkgDomain
		site.CnameHostname = buildSiteCnameForTask(domain, pkgDomain)
	}
}

func buildSiteCnameForTask(hostname, cnameDomain string) string {
	hostname = strings.TrimSpace(hostname)
	cnameDomain = strings.TrimSpace(cnameDomain)
	if hostname == "" || cnameDomain == "" {
		return ""
	}
	return hostname + "." + cnameDomain
}

func normalizeTaskSiteDomain(value string) string {
	host := strings.TrimSpace(strings.ToLower(value))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimSuffix(host, ".")
	return strings.TrimSpace(host)
}

func decodeTaskSiteDomainRaw(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var domains []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &domains); err == nil {
			return domains
		}
	}
	return splitFields(raw)
}

func findTaskSiteDomainConflict(tx *gorm.DB, domains []string, excludeSiteID int64) (string, error) {
	targetSet := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		key := normalizeTaskSiteDomain(domain)
		if key == "" {
			continue
		}
		if _, exists := targetSet[key]; exists {
			return key, nil
		}
		targetSet[key] = struct{}{}
	}
	if len(targetSet) == 0 {
		return "", nil
	}

	query := tx.Model(&models.Site{}).Select("id, domain")
	if excludeSiteID > 0 {
		query = query.Where("id <> ?", excludeSiteID)
	}
	type siteDomainRow struct {
		ID        int64  `gorm:"column:id"`
		DomainRaw string `gorm:"column:domain"`
	}
	var rows []siteDomainRow
	if err := query.Find(&rows).Error; err != nil {
		return "", err
	}
	for _, row := range rows {
		for _, existing := range decodeTaskSiteDomainRaw(row.DomainRaw) {
			if _, exists := targetSet[normalizeTaskSiteDomain(existing)]; exists {
				return normalizeTaskSiteDomain(existing), nil
			}
		}
	}
	return "", nil
}

func siteMissingColumnsForTask(tx *gorm.DB) []string {
	migrator := tx.Migrator()
	missing := make([]string, 0, 3)
	if !migrator.HasColumn(&models.Site{}, "dns_provider_id") {
		missing = append(missing, "DNSProviderID")
	}
	if !migrator.HasColumn(&models.Site{}, "platform_dns_record_id") {
		missing = append(missing, "PlatformDNSRecordID")
	}
	if !migrator.HasColumn(&models.Site{}, "user_dns_record_id") {
		missing = append(missing, "UserDNSRecordID")
	}
	if !migrator.HasColumn(&models.Site{}, "settings") {
		missing = append(missing, "SettingsRaw")
	}
	if !migrator.HasColumn(&models.Site{}, "cname_hostname2") {
		missing = append(missing, "CnameHostname2")
	}
	if !migrator.HasColumn(&models.Site{}, "cert_id") {
		missing = append(missing, "CertID")
	}
	return missing
}

func isUnknownColumnError(err error, column string) bool {
	if err == nil || column == "" {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") && strings.Contains(msg, strings.ToLower(column))
}

func selectSiteCreateColumns(tx *gorm.DB, site *models.Site) []string {
	if tx == nil || site == nil {
		return nil
	}
	existing := map[string]struct{}{}
	if types, err := tx.Migrator().ColumnTypes(&models.Site{}); err == nil {
		for _, ct := range types {
			name := strings.TrimSpace(ct.Name())
			if name != "" {
				existing[name] = struct{}{}
			}
		}
	}

	stmt := &gorm.Statement{DB: tx}
	if err := stmt.Parse(site); err != nil || stmt.Schema == nil {
		return nil
	}
	cols := make([]string, 0, len(stmt.Schema.Fields))
	for _, field := range stmt.Schema.Fields {
		if !field.Creatable || field.DBName == "" {
			continue
		}
		if len(existing) > 0 {
			if _, ok := existing[field.DBName]; !ok {
				continue
			}
		} else if !tx.Migrator().HasColumn(&models.Site{}, field.DBName) {
			continue
		}

		switch field.DBName {
		case "cert_id":
			continue
		case "region_id":
			if site.RegionID == 0 {
				continue
			}
		case "node_group_id":
			if site.NodeGroupID == 0 {
				continue
			}
		case "backup_node_group":
			if !site.EnableBackupGroup || site.BackupNodeGroupID == 0 {
				continue
			}
		}
		cols = append(cols, field.DBName)
	}
	return cols
}
