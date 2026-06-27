package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadSiteCnameInfosIncludesPackageWhenSitesDisabled(t *testing.T) {
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
		&models.CnameDomain{},
		&models.Package{},
		&models.UserPackage{},
		&models.Site{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.DB.Create(&models.CnameDomain{ID: 1, Domain: "311779.cc", DNSProviderID: 1}).Error; err != nil {
		t.Fatalf("create cname domain: %v", err)
	}
	if err := db.DB.Create(&models.UserPackage{
		ID:            7,
		NodeGroupID:   6,
		CnameDomain:   "311779.cc",
		CnameHostname: "mv72qnys",
		CnameMode:     "package",
	}).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}
	site := models.Site{
		ID:            101,
		UserPackageID: 7,
		CnameMode:     "package",
		Domains:       []string{"a.example.com"},
		Enable:        false,
		State:         "stop",
	}
	if err := db.DB.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}

	infos, _, err := loadSiteCnameInfos([]int64{6})
	if err != nil {
		t.Fatalf("load infos: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected one package cname sync target, got %d: %+v", len(infos), infos)
	}
	if !infos[0].PackageMode || infos[0].Hostname != "mv72qnys" {
		t.Fatalf("unexpected info: %+v", infos[0])
	}
}
