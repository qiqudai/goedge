package controllers

import (
	"cdn-api/models"
	"testing"
)

func TestUserPackageDNSFieldsChangedIgnoresResourceOnlyUpdates(t *testing.T) {
	current := models.UserPackage{
		NodeGroupID:     6,
		BackupNodeGroup: 10,
		EnableBackup:    true,
		CnameDomain:     "311779.cc",
		CnameHostname:   "mv72qnys",
		CnameMode:       "package",
	}
	updates := map[string]interface{}{
		"traffic":        int32(100),
		"bandwidth":      "10M",
		"month_price":    int64(12),
		"http_port":      int32(80),
		"custom_cc_rule": true,
	}
	if userPackageDNSFieldsChanged(current, updates) {
		t.Fatalf("resource-only package updates must not trigger DNS refresh")
	}
}

func TestUserPackageDNSFieldsChangedDetectsResolutionUpdates(t *testing.T) {
	current := models.UserPackage{
		NodeGroupID:   6,
		CnameDomain:   "311779.cc",
		CnameHostname: "mv72qnys",
		CnameMode:     "package",
	}
	cases := []map[string]interface{}{
		{"cname_hostname": "8klh0jkn"},
		{"cname_domain": "example.com"},
		{"cname_mode": "domain"},
		{"node_group_id": int64(10)},
		{"backup_node_group": int64(11)},
		{"enable_backup_group": true},
	}
	for _, updates := range cases {
		if !userPackageDNSFieldsChanged(current, updates) {
			t.Fatalf("DNS update not detected for %+v", updates)
		}
	}
}
