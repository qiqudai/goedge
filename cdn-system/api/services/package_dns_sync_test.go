package services

import "testing"

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
