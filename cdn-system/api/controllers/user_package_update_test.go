package controllers

import (
	"bytes"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserPackageUpdateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(
		&models.UserPackage{},
		&models.ConfigItem{},
		&models.SysConfig{},
		&models.Node{},
		&models.Site{},
		&models.Task{},
	); err != nil {
		t.Fatalf("migrate user-package update models: %v", err)
	}
	oldDB := db.DB
	db.DB = gdb
	t.Cleanup(func() { db.DB = oldDB })
	return gdb
}

func assertEnabledPrimaryTargets(t *testing.T, raw string) {
	t.Helper()
	targets := services.ParseTaskTargets(raw)
	if targets.Total != 2 || targets.Nodes["11"] == nil || targets.Nodes["12"] == nil || targets.Nodes["13"] != nil || targets.Nodes["14"] != nil {
		t.Fatalf("unexpected target set: %s", raw)
	}
}

func TestSaveUserPackageBoolConfigUpdatesExistingConfigWithoutPrimaryKey(t *testing.T) {
	gdb := newUserPackageUpdateTestDB(t)
	createdAt := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	if err := gdb.Create(&models.ConfigItem{
		Name:      "http3_enabled",
		Value:     "0",
		Type:      "user_package_config",
		ScopeID:   10,
		ScopeName: "user_package",
		Enable:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	if err := saveUserPackageBoolConfig(10, "http3_enabled", true); err != nil {
		t.Fatalf("update bool config: %v", err)
	}

	var configs []models.ConfigItem
	if err := gdb.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "http3_enabled", "user_package_config", "user_package", 10).Find(&configs).Error; err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("config rows = %d, want 1", len(configs))
	}
	if configs[0].Value != "1" || !configs[0].Enable {
		t.Fatalf("updated config = %+v, want enabled value 1", configs[0])
	}
	if !configs[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at changed from %s to %s", createdAt, configs[0].CreatedAt)
	}
}

func TestUpdateUserPlanCommitsAllFieldsAndQueuesEveryEnabledPrimaryNode(t *testing.T) {
	gdb := newUserPackageUpdateTestDB(t)
	oldEndAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	if err := gdb.Create(&models.UserPackage{
		ID:            10,
		UserID:        131708,
		Name:          "CN2",
		CnameHostname: "old-prefix",
		CnameDomain:   "311779.cc",
		CnameMode:     "domain",
		EndAt:         oldEndAt,
		Version:       1,
	}).Error; err != nil {
		t.Fatalf("seed package: %v", err)
	}
	if err := gdb.Create(&models.ConfigItem{
		Name:      "http3_enabled",
		Value:     "0",
		Type:      "user_package_config",
		ScopeID:   10,
		ScopeName: "user_package",
		Enable:    false,
		CreatedAt: oldEndAt,
		UpdatedAt: oldEndAt,
	}).Error; err != nil {
		t.Fatalf("seed existing HTTP3 config: %v", err)
	}
	if err := gdb.Create(&models.Site{
		ID:            21,
		UserID:        131708,
		UserPackageID: 10,
		CnameDomain:   "old-prefix",
		CnameHostname: "311779.cc",
		CnameMode:     "domain",
	}).Error; err != nil {
		t.Fatalf("seed linked site: %v", err)
	}
	if err := gdb.Create(&[]models.Node{
		{ID: 11, Enable: true},
		{ID: 12, Enable: true},
		{ID: 13, Enable: false},
		{ID: 14, PID: 11, Enable: true},
	}).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/api/v1/admin/user_plans/:id", (&PlanController{}).UpdateUserPlan)
	body := []byte(`{"end_at":"2026-07-28 15:47:58","http3_enabled":true,"cname_hostname":"edge","cname_domain":"7plvip.com","cname_mode":"package"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/user_plans/10", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("update status = %d, body=%s", res.Code, res.Body.String())
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("response code = %d, want 0", response.Code)
	}

	var updated models.UserPackage
	if err := gdb.First(&updated, 10).Error; err != nil {
		t.Fatalf("load updated package: %v", err)
	}
	wantEndAt := time.Date(2026, 7, 28, 15, 47, 58, 0, time.UTC)
	if !updated.EndAt.Equal(wantEndAt) || updated.CnameHostname != "edge" || updated.CnameDomain != "7plvip.com" || updated.CnameMode != "package" {
		t.Fatalf("updated package = %+v", updated)
	}

	var config models.ConfigItem
	if err := gdb.Where("name = ? AND type = ? AND scope_name = ? AND scope_id = ?", "http3_enabled", "user_package_config", "user_package", 10).First(&config).Error; err != nil {
		t.Fatalf("load HTTP3 config: %v", err)
	}
	if config.Value != "1" || !config.Enable {
		t.Fatalf("HTTP3 config = %+v, want enabled value 1", config)
	}

	var site models.Site
	if err := gdb.First(&site, 21).Error; err != nil {
		t.Fatalf("load linked site: %v", err)
	}
	if site.CnameDomain != "edge" || site.CnameHostname != "7plvip.com" || site.CnameMode != "package" {
		t.Fatalf("linked site CNAME = %+v", site)
	}

	var tasks []models.Task
	if err := gdb.Order("id asc").Find(&tasks).Error; err != nil {
		t.Fatalf("load tasks: %v", err)
	}
	var packageTask, cnameTask *models.Task
	for i := range tasks {
		if bytes.Contains([]byte(tasks[i].Data), []byte(`"packages"`)) {
			packageTask = &tasks[i]
		}
		var change services.ConfigChange
		if json.Unmarshal([]byte(tasks[i].Data), &change) == nil && change.Resource == services.ConfigResourceCNAME {
			cnameTask = &tasks[i]
		}
	}
	if packageTask == nil {
		t.Fatalf("package sync task was not created: %+v", tasks)
	}
	assertEnabledPrimaryTargets(t, packageTask.TargetsJSON)
	if cnameTask == nil {
		t.Fatalf("global CNAME config task was not created: %+v", tasks)
	}
	assertEnabledPrimaryTargets(t, cnameTask.TargetsJSON)
}
