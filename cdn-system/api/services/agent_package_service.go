package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	agentPackageType      = "agent_package"
	agentPackageScopeName = "global"
	agentPackageScopeID   = 0
)

type AgentPackage struct {
	Version     string    `json:"version"`
	Filename    string    `json:"filename"`
	Status      string    `json:"status"`
	GrayPercent int       `json:"gray_percent"`
	UploadTime  time.Time `json:"upload_time"`
	Size        int64     `json:"size"`
	Sha256      string    `json:"sha256"`
}

func ResolveAgentPackageDir() (string, error) {
	base, err := os.Getwd()
	if err != nil || strings.TrimSpace(base) == "" {
		return "", err
	}
	return filepath.Join(base, "agent"), nil
}

func EnsureAgentPackageDir() (string, error) {
	dir, err := ResolveAgentPackageDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ListAgentPackages() ([]AgentPackage, error) {
	var items []models.ConfigItem
	if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ?", agentPackageType, agentPackageScopeName, agentPackageScopeID).
		Find(&items).Error; err != nil {
		return nil, err
	}
	list := make([]AgentPackage, 0, len(items))
	for _, item := range items {
		var pkg AgentPackage
		if strings.TrimSpace(item.Value) != "" {
			_ = json.Unmarshal([]byte(item.Value), &pkg)
		}
		if pkg.Version == "" {
			pkg.Version = item.Name
		}
		if pkg.Version == "" {
			continue
		}
		if pkg.UploadTime.IsZero() && !item.CreatedAt.IsZero() {
			pkg.UploadTime = item.CreatedAt
		}
		list = append(list, pkg)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].UploadTime.Equal(list[j].UploadTime) {
			return CompareVersion(list[i].Version, list[j].Version) > 0
		}
		return list[i].UploadTime.After(list[j].UploadTime)
	})
	return list, nil
}

func GetAgentPackage(version string) (*AgentPackage, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, errors.New("version is required")
	}
	var item models.ConfigItem
	err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", agentPackageType, agentPackageScopeName, agentPackageScopeID, version).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	var pkg AgentPackage
	if strings.TrimSpace(item.Value) != "" {
		_ = json.Unmarshal([]byte(item.Value), &pkg)
	}
	if pkg.Version == "" {
		pkg.Version = version
	}
	if pkg.UploadTime.IsZero() && !item.CreatedAt.IsZero() {
		pkg.UploadTime = item.CreatedAt
	}
	return &pkg, nil
}

func UpsertAgentPackage(pkg AgentPackage) error {
	if strings.TrimSpace(pkg.Version) == "" {
		return errors.New("version is required")
	}
	raw, err := json.Marshal(pkg)
	if err != nil {
		return err
	}
	var item models.ConfigItem
	query := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", agentPackageType, agentPackageScopeName, agentPackageScopeID, pkg.Version)
	if err := query.First(&item).Error; err == nil {
		return query.Model(&models.ConfigItem{}).Updates(map[string]interface{}{
			"value":     string(raw),
			"update_at": time.Now(),
			"enable":    true,
		}).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	item = models.ConfigItem{
		Name:      pkg.Version,
		Value:     string(raw),
		Type:      agentPackageType,
		ScopeID:   agentPackageScopeID,
		ScopeName: agentPackageScopeName,
		Enable:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return db.DB.Create(&item).Error
}

func UpdateAgentPackageGray(version string, percent int) error {
	pkg, err := GetAgentPackage(version)
	if err != nil {
		return err
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	pkg.Status = "gray"
	pkg.GrayPercent = percent
	if pkg.UploadTime.IsZero() {
		pkg.UploadTime = time.Now()
	}
	return UpsertAgentPackage(*pkg)
}

func SetAgentPackageStable(version string) error {
	items, err := ListAgentPackages()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("version not found")
	}
	found := false
	for i := range items {
		if items[i].Version == version {
			items[i].Status = "stable"
			found = true
		} else if items[i].Status == "stable" {
			items[i].Status = "history"
		}
		if items[i].UploadTime.IsZero() {
			items[i].UploadTime = time.Now()
		}
	}
	if !found {
		return errors.New("version not found")
	}
	for _, item := range items {
		if err := UpsertAgentPackage(item); err != nil {
			return err
		}
	}
	return nil
}

func ResolveLatestAgentVersion(prefer string) (string, error) {
	if strings.TrimSpace(prefer) != "" {
		return strings.TrimSpace(prefer), nil
	}
	items, err := ListAgentPackages()
	if err != nil {
		return "", err
	}
	stable := ""
	for _, item := range items {
		if item.Status == "stable" && item.Version != "" {
			stable = item.Version
			break
		}
	}
	if stable != "" {
		return stable, nil
	}
	best := ""
	for _, item := range items {
		if item.Version == "" {
			continue
		}
		if best == "" || CompareVersion(item.Version, best) > 0 {
			best = item.Version
		}
	}
	return best, nil
}
