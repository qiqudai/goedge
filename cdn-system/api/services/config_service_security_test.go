package services

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExtractRegionBlockAccessDisabledFallsThroughToSecurity(t *testing.T) {
	site := models.Site{
		Settings: map[string]interface{}{
			"access": map[string]interface{}{
				"region_block": map[string]interface{}{
					"mode":      "disabled",
					"countries": []interface{}{},
				},
			},
			"security": map[string]interface{}{
				"region_block": []interface{}{"us", "cn", "fr"},
			},
		},
	}
	got := extractRegionBlock(site)
	if len(got) != 3 {
		t.Fatalf("expected security.region_block when access is disabled, got %#v", got)
	}
}

func TestExtractRegionBlockAccessEnabledTakesPrecedence(t *testing.T) {
	site := models.Site{
		Settings: map[string]interface{}{
			"access": map[string]interface{}{
				"region_block": map[string]interface{}{
					"mode":      "custom",
					"countries": []interface{}{"jp"},
				},
			},
			"security": map[string]interface{}{
				"region_block": []interface{}{"us", "cn"},
			},
		},
	}
	got := extractRegionBlock(site)
	if len(got) != 1 || got[0] != "JP" {
		t.Fatalf("expected access.region_block to win, got %#v", got)
	}
}

func TestExtractRegionBlockProduction447(t *testing.T) {
	raw, err := os.ReadFile("/tmp/site447-settings.json")
	if err != nil {
		t.Skip("production fixture missing")
	}
	site := models.Site{
		ID:       447,
		Settings: map[string]interface{}{},
	}
	if err := json.Unmarshal(raw, &site.Settings); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	sec, _ := site.Settings["security"].(map[string]interface{})
	want := len(sec["region_block"].([]interface{}))
	got := extractRegionBlock(site)
	if len(got) != want {
		t.Fatalf("expected %d region codes from security.region_block, got %d: %#v", want, len(got), got[:min(5, len(got))])
	}
}

func TestGenerateConfigRegionBlock447(t *testing.T) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN not set")
	}
	config.App.DBDSN = dsn
	db.Init()
	cfg, err := NewConfigService().GenerateConfigForNode("56")
	if err != nil {
		t.Fatalf("GenerateConfigForNode: %v", err)
	}
	var domain *models.EdgeDomain
	for i := range cfg.Domains {
		if cfg.Domains[i].Name == "www.boisconfort235.com" {
			domain = &cfg.Domains[i]
			break
		}
	}
	var site models.Site
	if err := db.DB.First(&site, 447).Error; err != nil {
		t.Fatalf("load site 447: %v", err)
	}
	if !site.Enable {
		if domain != nil {
			t.Fatalf("disabled site www.boisconfort235.com must not be included in edge config")
		}
		return
	}
	if domain == nil {
		t.Fatalf("enabled domain www.boisconfort235.com not found in node 56 config")
	}
	want := len(extractRegionBlock(site))
	if len(domain.RegionBlock) != want || want == 0 {
		t.Fatalf("expected %d region codes, got %d: %#v", want, len(domain.RegionBlock), domain.RegionBlock[:min(5, len(domain.RegionBlock))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestBuildSearchEngineOriginConditionRequiresSpiderIPAllowlist(t *testing.T) {
	cond := buildSearchEngineOriginConditionWithAllowlist(map[string]interface{}{
		"search_engine_origin":    true,
		"search_engine_origin_ip": "10.0.0.10",
	}, "")
	if cond != nil {
		t.Fatalf("search engine origin must not fall back to spoofable user-agent matching: %#v", cond)
	}
}

func TestSiteConfigGroupMatchesIncludesEnabledBackupOnly(t *testing.T) {
	groups := int64Set([]int64{6})
	if !siteConfigGroupMatches(10, 6, true, groups) {
		t.Fatalf("expected enabled backup group to match")
	}
	if siteConfigGroupMatches(10, 6, false, groups) {
		t.Fatalf("disabled backup group must not match")
	}
	if !siteConfigGroupMatches(6, 0, false, groups) {
		t.Fatalf("primary group should match")
	}
}

func TestLoadSitesForConfigGroupsSkipsDisabledSites(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.Site{}, &models.UserPackage{}, &models.Package{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	oldDB := db.DB
	db.DB = gdb
	t.Cleanup(func() { db.DB = oldDB })

	sites := []models.Site{
		{ID: 1, NodeGroupID: 6, Enable: true, State: "running"},
		{ID: 2, NodeGroupID: 6, Enable: false, State: "stop"},
		{ID: 3, NodeGroupID: 10, BackupNodeGroupID: 6, EnableBackupGroup: true, Enable: false, State: "stop"},
		{ID: 4, NodeGroupID: 10, BackupNodeGroupID: 6, EnableBackupGroup: true, Enable: true, State: "running"},
	}
	for _, site := range sites {
		if err := gdb.Create(&site).Error; err != nil {
			t.Fatalf("create site %d: %v", site.ID, err)
		}
	}

	got, err := loadSitesForConfigGroups(gdb, []int64{6})
	if err != nil {
		t.Fatalf("loadSitesForConfigGroups: %v", err)
	}
	gotIDs := map[int64]bool{}
	for _, site := range got {
		gotIDs[site.ID] = true
	}
	if !gotIDs[1] || !gotIDs[4] {
		t.Fatalf("enabled primary and backup sites must be included, got %#v", gotIDs)
	}
	if gotIDs[2] || gotIDs[3] {
		t.Fatalf("disabled sites must not be included, got %#v", gotIDs)
	}
}

func TestBuildResponseHeaderMapSkipsOriginLeakHeaders(t *testing.T) {
	got := buildResponseHeaderMap(map[string]interface{}{
		"advanced": map[string]interface{}{
			"cdn_headers": []interface{}{
				map[string]interface{}{"name": "X-Origin-IP", "value": "10.0.0.10"},
				map[string]interface{}{"name": "X-Safe", "value": "ok"},
			},
		},
	})
	if _, ok := got["X-Origin-IP"]; ok {
		t.Fatalf("origin leak response header must be filtered: %#v", got)
	}
	if got["X-Safe"] != "ok" {
		t.Fatalf("safe response header should be preserved: %#v", got)
	}
}
