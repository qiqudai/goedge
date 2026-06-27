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

func TestCleanupInvalidDNSRecordsOnlyDeletesManagedStaleLineRecords(t *testing.T) {
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
		LineName:    "Default",
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

	records := []dns.DNSRecord{
		{Type: "A", Name: "uxt9f6bk", Value: "103.21.90.251"},
		{Type: "A", Name: "uxt9f6bk", Value: "198.51.100.10"},
		{Type: "A", Name: "manual", Value: "198.51.100.11"},
		{Type: "CNAME", Name: "mv72qnys", Value: "stale.311779.cc"},
		{Type: "CNAME", Name: "manualc", Value: "stale.311779.cc"},
	}
	for _, record := range records {
		if err := mem.AddRecord("311779.cc", record); err != nil {
			t.Fatalf("seed record %+v: %v", record, err)
		}
	}

	if errs := cleanupInvalidDNSRecords(nil); len(errs) > 0 {
		t.Fatalf("cleanup errors: %v", errs)
	}

	expectMemoryRecord(t, mem, "311779.cc", "A", "uxt9f6bk", "103.21.90.251")
	expectMissingMemoryRecord(t, mem, "311779.cc", "A", "uxt9f6bk", "198.51.100.10")
	expectMemoryRecord(t, mem, "311779.cc", "A", "manual", "198.51.100.11")
	expectMemoryRecord(t, mem, "311779.cc", "CNAME", "mv72qnys", "stale.311779.cc")
	expectMemoryRecord(t, mem, "311779.cc", "CNAME", "manualc", "stale.311779.cc")
}
