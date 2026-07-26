package services

import (
	"cdn-api/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestComposeAndSplitSiteCname(t *testing.T) {
	if got := ComposeSiteCname("Api.", ".7PLVIP.COM."); got != "api.7plvip.com" {
		t.Fatalf("ComposeSiteCname() = %q", got)
	}
	prefix, err := SplitSiteCname("api.311779.cc.", "311779.cc")
	if err != nil || prefix != "api" {
		t.Fatalf("SplitSiteCname() = (%q, %v), want (api, nil)", prefix, err)
	}
	if _, err := SplitSiteCname("311779.cc", "311779.cc"); err == nil {
		t.Fatal("apex CNAME without a prefix must be rejected")
	}
}

func TestPropagateUserPackageCnameToSites(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Site{}); err != nil {
		t.Fatalf("migrate site: %v", err)
	}
	sites := []models.Site{
		{ID: 1, UserPackageID: 8, CnameDomain: "old", CnameHostname: "311779.cc"},
		{ID: 2, UserPackageID: 8, CnameDomain: "old", CnameHostname: "311779.cc"},
		{ID: 3, UserPackageID: 9, CnameDomain: "other", CnameHostname: "example.com"},
	}
	if err := db.Create(&sites).Error; err != nil {
		t.Fatalf("seed sites: %v", err)
	}

	ids, err := PropagateUserPackageCnameToSites(db, 8, "new-prefix", "7plvip.com")
	if err != nil || len(ids) != 2 {
		t.Fatalf("PropagateUserPackageCnameToSites() = (%v, %v)", ids, err)
	}
	var got []models.Site
	if err := db.Order("id").Find(&got).Error; err != nil {
		t.Fatalf("load sites: %v", err)
	}
	if got[0].CnameDomain != "new-prefix" || got[0].CnameHostname != "7plvip.com" || got[1].CnameDomain != "new-prefix" || got[2].CnameDomain != "other" {
		t.Fatalf("unexpected propagated sites: %+v", got)
	}
}

func TestMigrateLegacySiteCnames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Site{}); err != nil {
		t.Fatalf("migrate site: %v", err)
	}
	if err := db.Create(&[]models.Site{
		{ID: 1, CnameDomain: "311779.cc", CnameHostname: "api.311779.cc"},
		{ID: 2, CnameHostname: "keep.example.com"},
	}).Error; err != nil {
		t.Fatalf("seed sites: %v", err)
	}

	ids, err := MigrateLegacySiteCnames(db, "311779.cc", "7plvip.com")
	if err != nil || len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("MigrateLegacySiteCnames() = (%v, %v)", ids, err)
	}
	var site models.Site
	if err := db.First(&site, 1).Error; err != nil {
		t.Fatalf("load migrated site: %v", err)
	}
	if site.CnameDomain != "api" || site.CnameHostname != "7plvip.com" {
		t.Fatalf("unexpected migrated site: %+v", site)
	}
}
