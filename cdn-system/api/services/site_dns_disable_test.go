package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"cdn-api/services/dns/providers"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRemoveSiteDNSOnDisablePackageModeKeepsPackageCname(t *testing.T) {
	providers.ResetMemoryStore()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	oldDB := db.DB
	defer func() {
		db.DB = oldDB
	}()
	db.DB = gdb
	if err := db.DB.AutoMigrate(
		&models.DNSAPI{},
		&models.CnameDomain{},
		&models.UserPackage{},
		&models.Site{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	memProvider, err := providers.NewMemoryProvider("")
	if err != nil {
		t.Fatalf("memory provider: %v", err)
	}
	mem := memProvider.(*providers.MemoryProvider)

	if err := db.DB.Create(&models.DNSAPI{ID: 1, Name: "memory", Type: "memory", Auth: "{}"}).Error; err != nil {
		t.Fatalf("create dns api: %v", err)
	}
	if err := db.DB.Create(&models.CnameDomain{ID: 1, Domain: "311779.cc", DNSProviderID: 1}).Error; err != nil {
		t.Fatalf("create cname domain: %v", err)
	}
	if err := db.DB.Create(&models.UserPackage{
		ID:            20,
		CnameDomain:   "311779.cc",
		CnameHostname: "mv72qnys",
		CnameMode:     "package",
	}).Error; err != nil {
		t.Fatalf("create user package: %v", err)
	}
	site := models.Site{
		ID:            30,
		UserPackageID: 20,
		CnameDomain:   "311779.cc",
		CnameHostname: "mv72qnys.311779.cc",
		CnameMode:     "package",
		Domains:       []string{"icztev.cam"},
		Enable:        true,
	}
	if err := db.DB.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}

	if err := mem.AddRecord("311779.cc", dns.DNSRecord{Type: "CNAME", Name: "mv72qnys", Value: "line.311779.cc"}); err != nil {
		t.Fatalf("seed package cname: %v", err)
	}
	if err := mem.AddRecord("311779.cc", dns.DNSRecord{Type: "CNAME", Name: "siteonly", Value: "line.311779.cc"}); err != nil {
		t.Fatalf("seed site cname: %v", err)
	}

	site.Enable = false
	if errs := RemoveSiteDNSOnDisable(site); len(errs) > 0 {
		t.Fatalf("remove dns errors: %v", errs)
	}

	expectMemoryRecord(t, mem, "311779.cc", "CNAME", "mv72qnys", "line.311779.cc")
	expectMemoryRecord(t, mem, "311779.cc", "CNAME", "siteonly", "line.311779.cc")
}

func TestRemoveSiteDNSOnDisableDomainModeRemovesSiteCnameOnly(t *testing.T) {
	providers.ResetMemoryStore()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	oldDB := db.DB
	defer func() {
		db.DB = oldDB
	}()
	db.DB = gdb
	if err := db.DB.AutoMigrate(
		&models.DNSAPI{},
		&models.CnameDomain{},
		&models.UserPackage{},
		&models.Site{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	memProvider, err := providers.NewMemoryProvider("")
	if err != nil {
		t.Fatalf("memory provider: %v", err)
	}
	mem := memProvider.(*providers.MemoryProvider)

	if err := db.DB.Create(&models.DNSAPI{ID: 1, Name: "memory", Type: "memory", Auth: "{}"}).Error; err != nil {
		t.Fatalf("create dns api: %v", err)
	}
	if err := db.DB.Create(&models.CnameDomain{ID: 1, Domain: "311779.cc", DNSProviderID: 1}).Error; err != nil {
		t.Fatalf("create cname domain: %v", err)
	}
	if err := db.DB.Create(&models.UserPackage{
		ID:            21,
		CnameDomain:   "311779.cc",
		CnameHostname: "mv72qnys",
		CnameMode:     "package",
	}).Error; err != nil {
		t.Fatalf("create user package: %v", err)
	}
	site := models.Site{
		ID:            31,
		UserPackageID: 21,
		CnameDomain:   "311779.cc",
		CnameHostname: "icztev.311779.cc",
		CnameMode:     "domain",
		Domains:       []string{"icztev.cam"},
		Enable:        true,
	}
	if err := db.DB.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}

	if err := mem.AddRecord("311779.cc", dns.DNSRecord{Type: "CNAME", Name: "mv72qnys", Value: "line.311779.cc"}); err != nil {
		t.Fatalf("seed package cname: %v", err)
	}
	if err := mem.AddRecord("311779.cc", dns.DNSRecord{Type: "CNAME", Name: "icztev", Value: "line.311779.cc"}); err != nil {
		t.Fatalf("seed site cname: %v", err)
	}

	site.Enable = false
	if errs := RemoveSiteDNSOnDisable(site); len(errs) > 0 {
		t.Fatalf("remove dns errors: %v", errs)
	}

	expectMemoryRecord(t, mem, "311779.cc", "CNAME", "mv72qnys", "line.311779.cc")
	expectMissingMemoryRecord(t, mem, "311779.cc", "CNAME", "icztev", "line.311779.cc")
}

func expectMissingMemoryRecord(t *testing.T, mem *providers.MemoryProvider, zone, recordType, name, value string) {
	t.Helper()
	records, err := mem.GetRecords(zone)
	if err != nil {
		t.Fatalf("get records %s: %v", zone, err)
	}
	for _, record := range records {
		if record.Type == recordType && record.Name == name && record.Value == value {
			t.Fatalf("unexpected record %s %s %s -> %s", recordType, zone, name, value)
		}
	}
}
