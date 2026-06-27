package services

import (
	"cdn-api/db"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestShouldSyncPackageCnameForGroup(t *testing.T) {
	cases := []struct {
		name    string
		info    siteCnameInfo
		groupID int64
		want    bool
	}{
		{
			name:    "primary group allowed",
			info:    siteCnameInfo{PrimaryGroup: 10, BackupGroup: 6, EnableBackup: true},
			groupID: 10,
			want:    true,
		},
		{
			name:    "backup group is not allowed for normal sync",
			info:    siteCnameInfo{PrimaryGroup: 10, BackupGroup: 6, EnableBackup: true},
			groupID: 6,
			want:    false,
		},
		{
			name:    "legacy unbound site falls back",
			info:    siteCnameInfo{},
			groupID: 6,
			want:    true,
		},
		{
			name:    "zero group is ignored",
			info:    siteCnameInfo{PrimaryGroup: 10},
			groupID: 0,
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSyncPackageCnameForGroup(tc.info, tc.groupID); got != tc.want {
				t.Fatalf("shouldSyncPackageCnameForGroup() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSyncPackageCnameForLineChangeSkipsDelete(t *testing.T) {
	if err := SyncPackageCnameForLineChange(1, "default", "default", []int64{1}, "delete"); err != nil {
		t.Fatalf("delete action should be ignored: %v", err)
	}
	if err := SyncPackageCnameForLineChange(1, "default", "default", []int64{1}, "disable"); err != nil {
		t.Fatalf("disable action should be ignored: %v", err)
	}
}

func TestSyncPackageCnameForNodesSkipsDelete(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	oldDB := db.DB
	defer func() {
		db.DB = oldDB
	}()
	db.DB = gdb

	if err := SyncPackageCnameForNodes([]int64{1}, "delete"); err != nil {
		t.Fatalf("node delete action should be ignored: %v", err)
	}
	if err := SyncPackageCnameForNodes([]int64{1}, "disable"); err != nil {
		t.Fatalf("node disable action should be ignored: %v", err)
	}
}
