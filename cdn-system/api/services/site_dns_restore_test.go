package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns/providers"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRestoreSiteDNSRecordsOnlyRestoresSiteCNAME(t *testing.T) {
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
		&models.Package{},
		&models.NodeGroup{},
		&models.Node{},
		&models.Line{},
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
	if err := db.DB.Create(&models.NodeGroup{ID: 10, Name: "primary", CnameHostname: "uxt9f6bk", CnameDomain: "311779.cc"}).Error; err != nil {
		t.Fatalf("create node group: %v", err)
	}
	if err := db.DB.Create(&models.Node{ID: 101, Name: "node-1", IP: "103.21.90.251", Enable: true}).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := db.DB.Create(&models.Line{
		ID:          1001,
		NodeGroupID: 10,
		NodeID:      101,
		LineID:      "default",
		LineName:    "default",
		Enable:      true,
	}).Error; err != nil {
		t.Fatalf("create line: %v", err)
	}
	if err := db.DB.Create(&models.UserPackage{
		ID:            20,
		NodeGroupID:   10,
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
		State:         "running",
	}
	if err := db.DB.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}
	if err := db.DB.First(&site, 30).Error; err != nil {
		t.Fatalf("reload site: %v", err)
	}

	if errs := RestoreSiteDNSRecords(site); len(errs) > 0 {
		t.Fatalf("restore dns errors: %v", errs)
	}

	expectMemoryRecord(t, mem, "icztev.cam", "CNAME", "@", "mv72qnys.311779.cc")
	expectMissingMemoryRecord(t, mem, "311779.cc", "CNAME", "mv72qnys", "uxt9f6bk.311779.cc")
	expectMissingMemoryRecord(t, mem, "311779.cc", "A", "uxt9f6bk", "103.21.90.251")
}

func expectMemoryRecord(t *testing.T, mem *providers.MemoryProvider, zone, recordType, name, value string) {
	t.Helper()
	records, err := mem.GetRecords(zone)
	if err != nil {
		t.Fatalf("get records %s: %v", zone, err)
	}
	for _, record := range records {
		if record.Type == recordType && record.Name == name && record.Value == value {
			return
		}
	}
	t.Fatalf("missing %s %s %s -> %s in %s records=%+v", recordType, zone, name, value, zone, records)
}
