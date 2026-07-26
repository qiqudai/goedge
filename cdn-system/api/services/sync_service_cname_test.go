package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBumpCnameConfigVersionTargetsEveryEnabledPrimaryNode(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.SysConfig{}, &models.Task{}, &models.Node{}); err != nil {
		t.Fatalf("migrate sync models: %v", err)
	}
	if err := gdb.Create(&[]models.Node{
		{ID: 11, Enable: true},
		{ID: 12, Enable: true},
		{ID: 13, Enable: false},
		{ID: 14, PID: 11, Enable: true},
	}).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	oldDB := db.DB
	db.DB = gdb
	t.Cleanup(func() { db.DB = oldDB })
	SetConnectedNodeProvider(func() []int64 { return []int64{99} })
	t.Cleanup(func() { SetConnectedNodeProvider(nil) })

	if got := BumpCnameConfigVersion([]int64{101, 102}); got != 1 {
		t.Fatalf("config version = %d, want 1", got)
	}
	var task models.Task
	if err := gdb.Where("type = ?", "config_sync").First(&task).Error; err != nil {
		t.Fatalf("load CNAME sync task: %v", err)
	}
	var change ConfigChange
	if err := json.Unmarshal([]byte(task.Data), &change); err != nil {
		t.Fatalf("decode change: %v", err)
	}
	if change.Resource != ConfigResourceCNAME {
		t.Fatalf("resource = %q, want %q", change.Resource, ConfigResourceCNAME)
	}
	targets := ParseTaskTargets(task.TargetsJSON)
	if targets.Total != 2 || targets.Nodes["11"] == nil || targets.Nodes["12"] == nil || targets.Nodes["99"] != nil {
		t.Fatalf("unexpected target set: %s", task.TargetsJSON)
	}
}

func TestBumpUserPackageConfigVersionTargetsEveryEnabledPrimaryNode(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.SysConfig{}, &models.Task{}, &models.Node{}); err != nil {
		t.Fatalf("migrate sync models: %v", err)
	}
	if err := gdb.Create(&[]models.Node{
		{ID: 11, Enable: true},
		{ID: 12, Enable: true},
		{ID: 13, Enable: false},
		{ID: 14, PID: 11, Enable: true},
	}).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	oldDB := db.DB
	db.DB = gdb
	t.Cleanup(func() { db.DB = oldDB })
	SetConnectedNodeProvider(func() []int64 { return []int64{99} })
	t.Cleanup(func() { SetConnectedNodeProvider(nil) })

	if got := BumpUserPackageConfigVersion([]int64{10}); got != 1 {
		t.Fatalf("config version = %d, want 1", got)
	}
	var task models.Task
	if err := gdb.Where("type = ?", "config_sync").First(&task).Error; err != nil {
		t.Fatalf("load user-package sync task: %v", err)
	}
	var change ConfigChange
	if err := json.Unmarshal([]byte(task.Data), &change); err != nil {
		t.Fatalf("decode change: %v", err)
	}
	if change.Resource != ConfigResourceUserPackage {
		t.Fatalf("resource = %q, want %q", change.Resource, ConfigResourceUserPackage)
	}
	targets := ParseTaskTargets(task.TargetsJSON)
	if targets.Total != 2 || targets.Nodes["11"] == nil || targets.Nodes["12"] == nil || targets.Nodes["99"] != nil {
		t.Fatalf("unexpected target set: %s", task.TargetsJSON)
	}
}
