package services

import (
	"cdn-api/models"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// NormalizeSiteCnamePart returns a DNS label chain without a trailing dot.
// Site CNAME prefixes are intentionally stored separately from their root.
func NormalizeSiteCnamePart(value string) string {
	return strings.Trim(strings.TrimSpace(strings.ToLower(value)), ".")
}

// ComposeSiteCname joins the stored site CNAME prefix and root domain.
func ComposeSiteCname(prefix, root string) string {
	prefix = NormalizeSiteCnamePart(prefix)
	root = NormalizeSiteCnamePart(root)
	if prefix == "" || root == "" {
		return ""
	}
	return prefix + "." + root
}

// SplitSiteCname extracts the prefix from a full CNAME that ends in root.
func SplitSiteCname(full, root string) (string, error) {
	full = NormalizeSiteCnamePart(full)
	root = NormalizeSiteCnamePart(root)
	if full == "" || root == "" {
		return "", fmt.Errorf("cname and root domain are required")
	}
	suffix := "." + root
	if !strings.HasSuffix(full, suffix) {
		return "", fmt.Errorf("cname %q does not end in root domain %q", full, root)
	}
	prefix := strings.TrimSuffix(full, suffix)
	if prefix == "" {
		return "", fmt.Errorf("cname %q has no host prefix", full)
	}
	return prefix, nil
}

// PropagateUserPackageCnameToSites applies package CNAME values to every site
// belonging to that sold package. Call this inside the package transaction.
func PropagateUserPackageCnameToSites(tx *gorm.DB, userPackageID int64, prefix, root string) ([]int64, error) {
	if tx == nil {
		return nil, fmt.Errorf("database transaction is required")
	}
	prefix = NormalizeSiteCnamePart(prefix)
	root = NormalizeSiteCnamePart(root)
	if userPackageID == 0 || prefix == "" || root == "" {
		return nil, fmt.Errorf("user package id, cname prefix, and cname root are required")
	}

	var siteIDs []int64
	if err := tx.Model(&models.Site{}).Where("user_package = ?", userPackageID).Pluck("id", &siteIDs).Error; err != nil {
		return nil, err
	}
	if len(siteIDs) == 0 {
		return siteIDs, nil
	}
	if err := tx.Model(&models.Site{}).Where("id IN ?", siteIDs).Updates(map[string]interface{}{
		"cname_domain":   prefix,
		"cname_hostname": root,
	}).Error; err != nil {
		return nil, err
	}
	return siteIDs, nil
}

// MigrateLegacySiteCnames converts sites that store a full CNAME in
// cname_hostname into the split (prefix, root) representation.
func MigrateLegacySiteCnames(tx *gorm.DB, sourceRoot, targetRoot string) ([]int64, error) {
	if tx == nil {
		return nil, fmt.Errorf("database transaction is required")
	}
	sourceRoot = NormalizeSiteCnamePart(sourceRoot)
	targetRoot = NormalizeSiteCnamePart(targetRoot)
	if sourceRoot == "" || targetRoot == "" || sourceRoot == targetRoot {
		return nil, fmt.Errorf("different source and target root domains are required")
	}

	var sites []models.Site
	if err := tx.Where("LOWER(cname_hostname) LIKE ?", "%."+sourceRoot).Find(&sites).Error; err != nil {
		return nil, err
	}
	siteIDs := make([]int64, 0, len(sites))
	for _, site := range sites {
		prefix, err := SplitSiteCname(site.CnameHostname, sourceRoot)
		if err != nil {
			return nil, fmt.Errorf("site %d: %w", site.ID, err)
		}
		if err := tx.Model(&models.Site{}).Where("id = ?", site.ID).Updates(map[string]interface{}{
			"cname_domain":   prefix,
			"cname_hostname": targetRoot,
		}).Error; err != nil {
			return nil, err
		}
		siteIDs = append(siteIDs, site.ID)
	}
	return siteIDs, nil
}
