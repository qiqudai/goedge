package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCnameDomainDeleteGuardUsesSiteRootColumn(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.Site{}); err != nil {
		t.Fatalf("migrate site: %v", err)
	}
	oldDB := db.DB
	db.DB = gdb
	defer func() { db.DB = oldDB }()
	if err := gdb.Create(&models.Site{CnameDomain: "api", CnameHostname: "7plvip.com"}).Error; err != nil {
		t.Fatalf("seed site: %v", err)
	}
	inUse, err := isCnameDomainInUse("7plvip.com")
	if err != nil || !inUse {
		t.Fatalf("target root must be protected, got inUse=%v err=%v", inUse, err)
	}
}
