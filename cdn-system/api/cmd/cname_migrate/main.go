package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"
)

type migrationSummary struct {
	SourceRoot      string `json:"source_root"`
	TargetRoot      string `json:"target_root"`
	Apply           bool   `json:"apply"`
	Sites           int64  `json:"sites"`
	UserPackages    int64  `json:"user_packages"`
	Packages        int64  `json:"packages"`
	NodeGroups      int64  `json:"node_groups"`
	Forwards        int64  `json:"forwards"`
	RegistryDeleted bool   `json:"registry_deleted"`
	ConfigVersion   int64  `json:"config_version,omitempty"`
}

func main() {
	sourceRoot := flag.String("source", "311779.cc", "legacy CNAME root domain")
	targetRoot := flag.String("target", "7plvip.com", "target CNAME root domain")
	apply := flag.Bool("apply", false, "apply the migration; without this flag the command is read-only")
	syncAll := flag.Bool("sync-all", false, "queue every target-root site to every enabled primary node; requires -apply")
	config.Load()
	db.Init()
	if err := db.Ensure(); err != nil {
		log.Fatal(err)
	}

	source := services.NormalizeSiteCnamePart(*sourceRoot)
	target := services.NormalizeSiteCnamePart(*targetRoot)
	if source == "" || target == "" || source == target {
		log.Fatal("different source and target root domains are required")
	}
	if *syncAll {
		if !*apply {
			log.Fatal("-sync-all requires -apply")
		}
		var siteIDs []int64
		if err := db.DB.Model(&models.Site{}).Where("cname_hostname = ?", target).Pluck("id", &siteIDs).Error; err != nil {
			log.Fatal(err)
		}
		if len(siteIDs) == 0 {
			log.Fatalf("no sites use target CNAME root %q", target)
		}
		summary := migrationSummary{SourceRoot: source, TargetRoot: target, Apply: true, Sites: int64(len(siteIDs))}
		summary.ConfigVersion = services.BumpCnameConfigVersion(siteIDs)
		writeSummary(summary)
		return
	}

	summary, err := inspect(source, target)
	if err != nil {
		log.Fatal(err)
	}
	summary.Apply = *apply
	if !*apply {
		writeSummary(summary)
		return
	}

	var siteIDs []int64
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := requireRegistryDomain(tx, source); err != nil {
			return err
		}
		if err := requireRegistryDomain(tx, target); err != nil {
			return err
		}

		var err error
		siteIDs, err = services.MigrateLegacySiteCnames(tx, source, target)
		if err != nil {
			return err
		}
		if err := updateRootReference(tx, &models.UserPackage{}, source, target, &summary.UserPackages); err != nil {
			return err
		}
		if err := updateRootReference(tx, &models.Package{}, source, target, &summary.Packages); err != nil {
			return err
		}
		if err := updateRootReference(tx, &models.NodeGroup{}, source, target, &summary.NodeGroups); err != nil {
			return err
		}
		result := tx.Model(&models.Forward{}).Where("cname_domain = ?", source).Updates(map[string]interface{}{
			"cname_domain":   target,
			"cname_hostname": gorm.Expr("REPLACE(cname_hostname, ?, ?)", "."+source, "."+target),
		})
		if result.Error != nil {
			return result.Error
		}
		summary.Forwards = result.RowsAffected
		summary.Sites = int64(len(siteIDs))

		if err := ensureNoLegacyReferences(tx, source); err != nil {
			return err
		}
		result = tx.Where("domain = ?", source).Delete(&models.CnameDomain{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("expected one %s registry row to be deleted, got %d", source, result.RowsAffected)
		}
		summary.RegistryDeleted = true
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(siteIDs) > 0 {
		summary.ConfigVersion = services.BumpCnameConfigVersion(siteIDs)
	}
	writeSummary(summary)
}

func inspect(source, target string) (migrationSummary, error) {
	summary := migrationSummary{SourceRoot: source, TargetRoot: target}
	if err := db.DB.Model(&models.Site{}).Where("LOWER(cname_hostname) LIKE ?", "%"+source).Count(&summary.Sites).Error; err != nil {
		return summary, err
	}
	if err := db.DB.Model(&models.UserPackage{}).Where("cname_domain = ?", source).Count(&summary.UserPackages).Error; err != nil {
		return summary, err
	}
	if err := db.DB.Model(&models.Package{}).Where("cname_domain = ?", source).Count(&summary.Packages).Error; err != nil {
		return summary, err
	}
	if err := db.DB.Model(&models.NodeGroup{}).Where("cname_domain = ?", source).Count(&summary.NodeGroups).Error; err != nil {
		return summary, err
	}
	if err := db.DB.Model(&models.Forward{}).Where("cname_domain = ?", source).Count(&summary.Forwards).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func requireRegistryDomain(tx *gorm.DB, domain string) error {
	var row models.CnameDomain
	if err := tx.Where("domain = ?", domain).First(&row).Error; err != nil {
		return fmt.Errorf("cname registry domain %q: %w", domain, err)
	}
	return nil
}

func updateRootReference(tx *gorm.DB, model interface{}, source, target string, affected *int64) error {
	result := tx.Model(model).Where("cname_domain = ?", source).Update("cname_domain", target)
	if result.Error != nil {
		return result.Error
	}
	*affected = result.RowsAffected
	return nil
}

func ensureNoLegacyReferences(tx *gorm.DB, source string) error {
	checks := []struct {
		model interface{}
		where string
		args  []interface{}
	}{
		{&models.Site{}, "LOWER(cname_hostname) LIKE ? OR cname_domain = ?", []interface{}{"%" + source, source}},
		{&models.UserPackage{}, "cname_domain = ?", []interface{}{source}},
		{&models.Package{}, "cname_domain = ?", []interface{}{source}},
		{&models.NodeGroup{}, "cname_domain = ?", []interface{}{source}},
		{&models.Forward{}, "cname_domain = ? OR LOWER(cname_hostname) LIKE ?", []interface{}{source, "%" + source}},
	}
	for _, check := range checks {
		var count int64
		query := tx.Model(check.model).Where(check.where, check.args...)
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%d legacy %s references remain", count, source)
		}
	}
	return nil
}

func writeSummary(summary migrationSummary) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		log.Fatal(err)
	}
}
