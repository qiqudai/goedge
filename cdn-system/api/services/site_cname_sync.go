package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"errors"
	"fmt"
	"strings"
)

func ResyncSiteCnameForSite(site models.Site) {
	_ = ResyncSiteCnameForSiteWithErrors(site)
}

func RestoreSiteDNSRecords(site models.Site) []error {
	errs := make([]error, 0)
	if err := SyncUserDNSRecords(nil, &site); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// RemoveSiteDNSOnDisable removes DNS created for a single site when it is stopped.
// Package-mode sites must keep shared package/line records; only per-site records are removed in domain mode.
func RemoveSiteDNSOnDisable(site models.Site) []error {
	if site.UserPackageID == 0 {
		return nil
	}
	var pkg models.UserPackage
	if err := db.DB.Where("id = ?", site.UserPackageID).First(&pkg).Error; err != nil {
		return []error{err}
	}
	if isPackageCnameMode(site, pkg) {
		return nil
	}
	if err := deletePlatformSiteCname(site, pkg); err != nil {
		return []error{err}
	}
	return nil
}

func deletePlatformSiteCname(site models.Site, pkg models.UserPackage) error {
	domainKey, host := resolveSiteCnameTarget(site, pkg)
	if domainKey == "" || host == "" || host == "@" {
		return nil
	}
	pkgDomainKey, pkgHost := resolveSiteCnameTarget(models.Site{UserPackageID: pkg.ID}, pkg)
	if pkgDomainKey == domainKey && pkgHost == host {
		return nil
	}

	var domain models.CnameDomain
	if err := db.DB.Where("domain = ?", domainKey).First(&domain).Error; err != nil {
		return nil
	}
	if domain.DNSProviderID == 0 {
		return nil
	}
	var api models.DNSAPI
	if err := db.DB.Where("id = ?", domain.DNSProviderID).First(&api).Error; err != nil {
		return err
	}
	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil || provider == nil {
		if err == nil {
			err = errors.New("dns provider not available")
		}
		return err
	}
	records, err := provider.GetRecords(domain.Domain)
	if err != nil {
		return fmt.Errorf("get records failed: %w", err)
	}
	for _, record := range records {
		if !strings.EqualFold(record.Type, "CNAME") {
			continue
		}
		if record.Name != host {
			continue
		}
		if err := provider.DeleteRecord(domain.Domain, record); err != nil {
			return fmt.Errorf("delete record failed: %w", err)
		}
	}
	return nil
}

func restoreSiteLineDNSWithErrors(site models.Site) []error {
	if site.UserPackageID == 0 && site.NodeGroupID == 0 {
		return nil
	}
	errs := make([]error, 0)
	groupID := resolveGroupIDFromSite(site)
	if groupID != 0 {
		errs = append(errs, dnsMessagesToErrors(resyncGroupDNSRecords(groupID))...)
	}

	backupGroup := site.BackupNodeGroupID
	enableBackup := site.EnableBackupGroup
	if !enableBackup && site.UserPackageID != 0 {
		var pkg models.UserPackage
		if err := db.DB.Select("backup_node_group", "enable_backup_group").
			Where("id = ?", site.UserPackageID).
			First(&pkg).Error; err == nil {
			if backupGroup == 0 {
				backupGroup = pkg.BackupNodeGroup
			}
			enableBackup = pkg.EnableBackup
		}
	}
	if enableBackup && backupGroup != 0 {
		errs = append(errs, dnsMessagesToErrors(resyncGroupDNSRecords(backupGroup))...)
	}
	return errs
}

func dnsMessagesToErrors(messages []string) []error {
	if len(messages) == 0 {
		return nil
	}
	errs := make([]error, 0, len(messages))
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message != "" {
			errs = append(errs, errors.New(message))
		}
	}
	return errs
}

func ResyncSiteCnameForSiteWithErrors(site models.Site) []error {
	if !shouldSyncSiteCname(site) {
		return nil
	}
	errs := make([]error, 0)
	if err := SyncUserDNSRecords(nil, &site); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func shouldSyncSiteCname(site models.Site) bool {
	if site.UserPackageID == 0 {
		return false
	}
	mode := strings.TrimSpace(strings.ToLower(site.CnameMode))
	if mode != "" {
		return mode != "package"
	}
	var pkg models.UserPackage
	if err := db.DB.Select("cname_mode").Where("id = ?", site.UserPackageID).First(&pkg).Error; err != nil {
		return true
	}
	return strings.TrimSpace(strings.ToLower(pkg.CnameMode)) != "package"
}

func ResyncGroupLineCnames(groupID int64) {
	_ = ResyncGroupLineCnamesWithErrors(groupID)
}

func ResyncGroupLineCnamesWithErrors(groupID int64) []error {
	if groupID == 0 || db.DB == nil {
		return nil
	}
	var lines []models.Line
	if err := db.DB.Select("line_id", "line_name").
		Where("node_group_id = ?", groupID).
		Find(&lines).Error; err != nil {
		return []error{err}
	}
	lineMap := map[string]string{}
	for _, line := range lines {
		lineID := strings.TrimSpace(line.LineID)
		if lineID == "" {
			lineID = "default"
		}
		lineName := strings.TrimSpace(line.LineName)
		if lineName == "" {
			lineName = lineID
		}
		if _, ok := lineMap[lineID]; !ok {
			lineMap[lineID] = lineName
		}
	}
	errs := make([]error, 0)
	for lineID, lineName := range lineMap {
		if err := SyncPackageCnameForLineChange(groupID, lineID, lineName, nil, "resync"); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func resolveGroupIDFromSite(site models.Site) int64 {
	if site.NodeGroupID != 0 {
		return site.NodeGroupID
	}
	if site.UserPackageID == 0 {
		return 0
	}
	var pkg models.UserPackage
	if err := db.DB.Select("node_group_id").Where("id = ?", site.UserPackageID).First(&pkg).Error; err != nil {
		return 0
	}
	return pkg.NodeGroupID
}
