package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsProtectedPackageCNAMERecord(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	oldDB := db.DB
	defer func() {
		db.DB = oldDB
	}()
	db.DB = gdb
	if err := db.DB.AutoMigrate(&models.UserPackage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.DB.Create(&models.UserPackage{
		ID:            10,
		CnameMode:     "package",
		CnameDomain:   "311779.cc",
		CnameHostname: "8klh0jkn",
	}).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	if !isProtectedPackageCNAMERecord("311779.cc", "CNAME", "8klh0jkn") {
		t.Fatalf("package cname hostname must be protected")
	}
	if isProtectedPackageCNAMERecord("311779.cc", "CNAME", "siteonly") {
		t.Fatalf("ordinary site cname must not be protected")
	}
	if isProtectedPackageCNAMERecord("311779.cc", "A", "8klh0jkn") {
		t.Fatalf("line A records must not be protected by package cname guard")
	}
}
