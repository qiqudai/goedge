package services

import (
	"context"
	"database/sql"
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func requireTestDB(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN not set")
	}
	config.App.DBDSN = dsn
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("database open failed: %v", err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		t.Skipf("database unreachable: %v", err)
	}
	db.Init()
}

func TestL3UpstreamKeyFormat(t *testing.T) {
	groupID := int64(5)
	l1Key := "l1_upstream_" + strconv.FormatInt(groupID, 10)
	l2Key := "l2_upstream_" + strconv.FormatInt(groupID, 10)
	if l1Key != "l1_upstream_5" || l2Key != "l2_upstream_5" {
		t.Fatalf("unexpected upstream keys: %s %s", l1Key, l2Key)
	}
}

func TestL3DomainFieldsFromParentFetchConfig(t *testing.T) {
	cfg := ParentFetchConfig{
		ParentGroupID:   5,
		ParentFetchMode: ParentFetchL1,
		L1RespectL2:     false,
	}
	groupID := cfg.ParentGroupID
	parentL1Key := "l1_upstream_" + strconv.FormatInt(groupID, 10)
	parentL2Key := "l2_upstream_" + strconv.FormatInt(groupID, 10)

	domain := models.EdgeDomain{
		Name:                "test.example.com",
		UpstreamKey:         "upstream_100",
		ParentFetchMode:     cfg.ParentFetchMode,
		ParentL1UpstreamKey: parentL1Key,
		ParentL2UpstreamKey: parentL2Key,
		ParentHTTPPort:      "80",
		ParentHTTPSPort:     "443",
		L1RespectL2:         cfg.L1RespectL2,
	}
	if domain.ParentFetchMode != ParentFetchL1 {
		t.Fatalf("ParentFetchMode = %q", domain.ParentFetchMode)
	}
	if domain.ParentL1UpstreamKey != "l1_upstream_5" {
		t.Fatalf("ParentL1UpstreamKey = %q", domain.ParentL1UpstreamKey)
	}
	if domain.ParentL2UpstreamKey != "l2_upstream_5" {
		t.Fatalf("ParentL2UpstreamKey = %q", domain.ParentL2UpstreamKey)
	}
	if domain.L1RespectL2 {
		t.Fatal("expected L1RespectL2=false")
	}
}

func TestGenerateConfigForNodeL3Integration(t *testing.T) {
	requireTestDB(t)

	var node models.Node
	err := db.DB.Where("level = ? AND enable = ?", 3, true).First(&node).Error
	if err != nil {
		t.Skip("no enabled L3 node in database")
	}

	cfg, err := NewConfigService().GenerateConfigForNode(strconv.FormatInt(node.ID, 10))
	if err != nil {
		t.Fatalf("GenerateConfigForNode: %v", err)
	}
	if cfg.NodeLevel != 3 {
		t.Fatalf("NodeLevel = %d, want 3", cfg.NodeLevel)
	}

	parentCfg := LoadParentFetchConfig(node.ID)
	if NormalizeParentFetchMode(cfg.ParentFetchMode) != parentCfg.ParentFetchMode {
		t.Fatalf("ParentFetchMode mismatch: payload=%q db=%q", cfg.ParentFetchMode, parentCfg.ParentFetchMode)
	}
	if parentCfg.ParentGroupID > 0 && cfg.ParentGroupID != parentCfg.ParentGroupID {
		t.Fatalf("ParentGroupID mismatch: payload=%d db=%d", cfg.ParentGroupID, parentCfg.ParentGroupID)
	}

	if parentCfg.ParentFetchMode == ParentFetchL1 || parentCfg.ParentFetchMode == ParentFetchL2 {
		if parentCfg.ParentGroupID <= 0 {
			t.Fatal("L3 l1/l2 mode requires parent_group_id in node_config")
		}
		l1Key := "l1_upstream_" + strconv.FormatInt(parentCfg.ParentGroupID, 10)
		l2Key := "l2_upstream_" + strconv.FormatInt(parentCfg.ParentGroupID, 10)
		hasL1 := upstreamExists(cfg.Upstreams, l1Key)
		hasL2 := upstreamExists(cfg.Upstreams, l2Key)
		if parentCfg.ParentFetchMode == ParentFetchL1 && !hasL1 && !hasL2 {
			t.Fatalf("expected parent L1/L2 upstreams for group %d", parentCfg.ParentGroupID)
		}
		if parentCfg.ParentFetchMode == ParentFetchL2 && !hasL2 {
			t.Fatalf("expected parent L2 upstream %s", l2Key)
		}
		for i := range cfg.Domains {
			d := cfg.Domains[i]
			if strings.TrimSpace(d.ParentFetchMode) == "" {
				continue
			}
			if d.ParentFetchMode != parentCfg.ParentFetchMode {
				t.Fatalf("domain %s ParentFetchMode=%q want %q", d.Name, d.ParentFetchMode, parentCfg.ParentFetchMode)
			}
			if d.ParentL1UpstreamKey != "" && !strings.HasPrefix(d.ParentL1UpstreamKey, "l1_upstream_") {
				t.Fatalf("domain %s invalid ParentL1UpstreamKey=%q", d.Name, d.ParentL1UpstreamKey)
			}
			if d.ParentL2UpstreamKey != "" && !strings.HasPrefix(d.ParentL2UpstreamKey, "l2_upstream_") {
				t.Fatalf("domain %s invalid ParentL2UpstreamKey=%q", d.Name, d.ParentL2UpstreamKey)
			}
		}
	}
}

func upstreamExists(upstreams []models.EdgeUpstream, id string) bool {
	for _, u := range upstreams {
		if u.ID == id && len(u.Targets) > 0 {
			return true
		}
	}
	return false
}

func TestCountParentGroupReferencesIntegration(t *testing.T) {
	requireTestDB(t)

	count, err := CountParentGroupReferences(-1)
	if err != nil {
		t.Fatalf("CountParentGroupReferences: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 refs for invalid group, got %d", count)
	}
}
