package dns_test

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"cdn-api/services/dns/providers"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDNSRobustnessDB(t *testing.T) (*providers.MemoryProvider, int64) {
	t.Helper()
	providers.ResetMemoryStore()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.DB = gdb
	if err := gdb.AutoMigrate(
		&models.NodeGroup{},
		&models.Node{},
		&models.Line{},
		&models.CnameDomain{},
		&models.DNSAPI{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	provider, err := providers.NewMemoryProvider("")
	if err != nil {
		t.Fatalf("memory provider: %v", err)
	}
	mem := provider.(*providers.MemoryProvider)

	now := time.Now()
	api := models.DNSAPI{ID: 1, UserID: 0, Name: "test", Type: "memory", Auth: "{}"}
	domain := models.CnameDomain{ID: 1, Domain: "test.local", DNSProviderID: 1, CreatedAt: now, UpdatedAt: now}
	group := models.NodeGroup{
		ID:            1,
		Name:          "robustness-group",
		CnameHostname: "line-a",
		CnameDomain:   "test.local",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	for _, row := range []interface{}{&api, &domain, &group} {
		if err := gdb.Create(row).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	nodeIDs := make([]int64, 0, 10)
	for i := 1; i <= 10; i++ {
		node := models.Node{
			ID:        int64(i),
			Name:      fmt.Sprintf("node-%d", i),
			IP:        fmt.Sprintf("203.0.113.%d", i),
			Enable:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := gdb.Create(&node).Error; err != nil {
			t.Fatalf("seed node: %v", err)
		}
		nodeIDs = append(nodeIDs, node.ID)
	}

	for _, nodeID := range nodeIDs {
		line := models.Line{
			NodeGroupID: 1,
			NodeID:      nodeID,
			NodeIPID:    nodeID,
			LineID:      "default",
			LineName:    "Default",
			Weight:      "1",
			Enable:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := gdb.Create(&line).Error; err != nil {
			t.Fatalf("seed line: %v", err)
		}
	}

	return mem, group.ID
}

func expectLineA(t *testing.T, mem *providers.MemoryProvider, want []string) {
	t.Helper()
	got := mem.LineAValues("test.local", "line-a", "")
	if len(got) != len(want) {
		t.Fatalf("dns A values = %v, want %v", got, want)
	}
	lookup := make(map[string]struct{}, len(want))
	for _, ip := range want {
		lookup[ip] = struct{}{}
	}
	for _, ip := range got {
		if _, ok := lookup[ip]; !ok {
			t.Fatalf("dns A values = %v, want %v", got, want)
		}
	}
}

func TestSyncLineRecordsResyncMatchesAllNodes(t *testing.T) {
	mem, groupID := setupDNSRobustnessDB(t)
	if err := dns.SyncLineRecords(groupID, "default", "Default", "resync", nil); err != nil {
		t.Fatalf("resync: %v", err)
	}
	want := []string{
		"203.0.113.1", "203.0.113.2", "203.0.113.3", "203.0.113.4", "203.0.113.5",
		"203.0.113.6", "203.0.113.7", "203.0.113.8", "203.0.113.9", "203.0.113.10",
	}
	expectLineA(t, mem, want)
}

func TestSyncLineRecordsDeleteAllThenReAdd(t *testing.T) {
	mem, groupID := setupDNSRobustnessDB(t)
	if err := dns.SyncLineRecords(groupID, "default", "Default", "resync", nil); err != nil {
		t.Fatalf("initial resync: %v", err)
	}

	// Simulate disable + delete all lines, then re-create them.
	if err := db.DB.Model(&models.Line{}).Where("node_group_id = ?", groupID).Update("enable", false).Error; err != nil {
		t.Fatalf("disable lines: %v", err)
	}
	if err := dns.SyncLineRecords(groupID, "default", "Default", "delete", []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	expectLineA(t, mem, nil)

	if err := db.DB.Where("node_group_id = ?", groupID).Delete(&models.Line{}).Error; err != nil {
		t.Fatalf("remove lines: %v", err)
	}

	now := time.Now()
	for i := 1; i <= 10; i++ {
		line := models.Line{
			NodeGroupID: groupID,
			NodeID:      int64(i),
			NodeIPID:    int64(i),
			LineID:      "default",
			LineName:    "Default",
			Weight:      "1",
			Enable:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := db.DB.Create(&line).Error; err != nil {
			t.Fatalf("re-add line: %v", err)
		}
	}
	nodeIDs := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if err := dns.SyncLineRecords(groupID, "default", "Default", "add", nodeIDs); err != nil {
		t.Fatalf("re-add sync: %v", err)
	}
	expectLineA(t, mem, []string{
		"203.0.113.1", "203.0.113.2", "203.0.113.3", "203.0.113.4", "203.0.113.5",
		"203.0.113.6", "203.0.113.7", "203.0.113.8", "203.0.113.9", "203.0.113.10",
	})
}

func TestSyncLineRecordsChaosCycles(t *testing.T) {
	mem, groupID := setupDNSRobustnessDB(t)
	nodeIDs := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	wantAll := []string{
		"203.0.113.1", "203.0.113.2", "203.0.113.3", "203.0.113.4", "203.0.113.5",
		"203.0.113.6", "203.0.113.7", "203.0.113.8", "203.0.113.9", "203.0.113.10",
	}

	ops := []struct {
		name   string
		run    func() error
		expect []string
	}{
		{
			name: "resync all",
			run: func() error {
				return dns.SyncLineRecords(groupID, "default", "Default", "resync", nil)
			},
			expect: wantAll,
		},
		{
			name: "delete one",
			run: func() error {
				if err := db.DB.Model(&models.Line{}).Where("node_ip_id = ?", 10).Update("enable", false).Error; err != nil {
					return err
				}
				return dns.SyncLineRecords(groupID, "default", "Default", "delete", []int64{10})
			},
			expect: wantAll[:9],
		},
		{
			name: "add one back",
			run: func() error {
				if err := db.DB.Model(&models.Line{}).Where("node_ip_id = ?", 10).Update("enable", true).Error; err != nil {
					return err
				}
				return dns.SyncLineRecords(groupID, "default", "Default", "add", []int64{10})
			},
			expect: wantAll,
		},
		{
			name: "disable batch",
			run: func() error {
				if err := db.DB.Model(&models.Line{}).Where("node_ip_id IN ?", []int64{8, 9, 10}).Update("enable", false).Error; err != nil {
					return err
				}
				return dns.SyncLineRecords(groupID, "default", "Default", "delete", []int64{8, 9, 10})
			},
			expect: wantAll[:7],
		},
		{
			name: "enable batch",
			run: func() error {
				if err := db.DB.Model(&models.Line{}).Where("node_ip_id IN ?", []int64{8, 9, 10}).Update("enable", true).Error; err != nil {
					return err
				}
				return dns.SyncLineRecords(groupID, "default", "Default", "add", []int64{8, 9, 10})
			},
			expect: wantAll,
		},
		{
			name: "delete all lines from db",
			run: func() error {
				if err := db.DB.Model(&models.Line{}).Where("node_group_id = ?", groupID).Update("enable", false).Error; err != nil {
					return err
				}
				if err := dns.SyncLineRecords(groupID, "default", "Default", "delete", nodeIDs); err != nil {
					return err
				}
				return db.DB.Where("node_group_id = ?", groupID).Delete(&models.Line{}).Error
			},
			expect: nil,
		},
		{
			name: "re-create all lines",
			run: func() error {
				now := time.Now()
				for _, id := range nodeIDs {
					line := models.Line{
						NodeGroupID: groupID,
						NodeID:      id,
						NodeIPID:    id,
						LineID:      "default",
						LineName:    "Default",
						Weight:      "1",
						Enable:      true,
						CreatedAt:   now,
						UpdatedAt:   now,
					}
					if err := db.DB.Create(&line).Error; err != nil {
						return err
					}
				}
				return dns.SyncLineRecords(groupID, "default", "Default", "add", nodeIDs)
			},
			expect: wantAll,
		},
	}

	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			if err := op.run(); err != nil {
				t.Fatalf("%s failed: %v", op.name, err)
			}
			expectLineA(t, mem, op.expect)
		})
	}
}

func TestReconcileLineRecordSetClearsStaleValues(t *testing.T) {
	providers.ResetMemoryStore()
	mem, _ := providers.NewMemoryProvider("")
	provider := mem.(*providers.MemoryProvider)
	record := dns.DNSRecord{Type: "A", Name: "host", Line: "Default", TTL: 600}
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		if err := provider.AddRecord("example.com", dns.DNSRecord{Type: record.Type, Name: record.Name, Line: record.Line, Value: ip, TTL: record.TTL}); err != nil {
			t.Fatalf("seed record: %v", err)
		}
	}
	if err := dns.ReconcileLineRecordSet(provider, "example.com", record, []string{"2.2.2.2", "4.4.4.4"}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got := provider.LineAValues("example.com", "host", "Default")
	want := []string{"2.2.2.2", "4.4.4.4"}
	if len(got) != len(want) {
		t.Fatalf("values = %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("values = %v want %v", got, want)
		}
	}
}
