package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-common/i18n"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type DomainUsage struct {
	TotalDomains     int    `json:"total_domains"`
	TotalMainDomains int    `json:"total_main_domains"`
	DomainLimit      int    `json:"domain_limit"`
	MainDomainLimit  int    `json:"main_domain_limit"`
	Exceeded         bool   `json:"exceeded"`
	Message          string `json:"message"`
}

func GetDomainUsage(userID, userPackageID int64) (DomainUsage, error) {
	usage := DomainUsage{}
	if userID == 0 || userPackageID == 0 {
		return usage, errors.New("invalid user/package")
	}

	totalLimit, mainLimit, err := loadDomainLimits(userPackageID)
	if err != nil {
		return usage, err
	}
	usage.DomainLimit = totalLimit
	usage.MainDomainLimit = mainLimit

	domainSet, mainSet, err := loadUserDomainSets(userID, userPackageID)
	if err != nil {
		return usage, err
	}
	usage.TotalDomains = len(domainSet)
	usage.TotalMainDomains = len(mainSet)

	if totalLimit > 0 && usage.TotalDomains > totalLimit {
		usage.Exceeded = true
		usage.Message = buildDomainLimitMessage("domain", usage.TotalDomains)
	} else if mainLimit > 0 && usage.TotalMainDomains > mainLimit {
		usage.Exceeded = true
		usage.Message = buildDomainLimitMessage("main_domain", usage.TotalMainDomains)
	}

	return usage, nil
}

func CheckDomainLimit(userID, userPackageID int64, newDomains []string) error {
	if userID == 0 || userPackageID == 0 {
		return errors.New("invalid user/package")
	}
	totalLimit, mainLimit, err := loadDomainLimits(userPackageID)
	if err != nil {
		return err
	}
	if totalLimit <= 0 && mainLimit <= 0 {
		return nil
	}

	domainSet, mainSet, err := loadUserDomainSets(userID, userPackageID)
	if err != nil {
		return err
	}
	addDomains(domainSet, mainSet, newDomains)

	totalCount := len(domainSet)
	mainCount := len(mainSet)

	if totalLimit > 0 && totalCount > totalLimit {
		return errors.New(buildDomainLimitMessage("domain", totalCount))
	}
	if mainLimit > 0 && mainCount > mainLimit {
		return errors.New(buildDomainLimitMessage("main_domain", mainCount))
	}
	return nil
}

func CheckDomainLimitForUpdate(userID, userPackageID, siteID int64, newDomains []string) error {
	if userID == 0 || userPackageID == 0 {
		return errors.New("invalid user/package")
	}
	totalLimit, mainLimit, err := loadDomainLimits(userPackageID)
	if err != nil {
		return err
	}
	if totalLimit <= 0 && mainLimit <= 0 {
		return nil
	}

	domainSet, mainSet, err := loadUserDomainSetsExceptSite(userID, userPackageID, siteID)
	if err != nil {
		return err
	}
	addDomains(domainSet, mainSet, newDomains)

	totalCount := len(domainSet)
	mainCount := len(mainSet)

	if totalLimit > 0 && totalCount > totalLimit {
		return errors.New(buildDomainLimitMessage("domain", totalCount))
	}
	if mainLimit > 0 && mainCount > mainLimit {
		return errors.New(buildDomainLimitMessage("main_domain", mainCount))
	}
	return nil
}

func loadDomainLimits(userPackageID int64) (int, int, error) {
	var pack models.UserPackage
	if err := db.DB.First(&pack, userPackageID).Error; err != nil {
		return 0, 0, err
	}
	totalLimit := int(pack.DomainLimit)
	mainLimit := int(pack.MainDomainLimit)

	if mainLimit <= 0 {
		var cfg models.ConfigItem
		if err := db.DB.Where("type = ? AND scope_name = ? AND scope_id = ? AND name = ?", "user_package_config", "user_package", userPackageID, "main_domain_limit").First(&cfg).Error; err == nil {
			mainLimit = parseIntConfig(cfg.Value)
		}
	}

	return totalLimit, mainLimit, nil
}

func loadUserDomainSets(userID, userPackageID int64) (map[string]struct{}, map[string]struct{}, error) {
	domainSet := map[string]struct{}{}
	mainSet := map[string]struct{}{}

	var sites []models.Site
	if err := db.DB.Where("uid = ? AND user_package = ?", userID, userPackageID).Find(&sites).Error; err != nil {
		return nil, nil, err
	}
	for _, site := range sites {
		addDomains(domainSet, mainSet, site.Domains)
	}
	return domainSet, mainSet, nil
}

func loadUserDomainSetsExceptSite(userID, userPackageID, excludeSiteID int64) (map[string]struct{}, map[string]struct{}, error) {
	domainSet := map[string]struct{}{}
	mainSet := map[string]struct{}{}

	var sites []models.Site
	if err := db.DB.Where("uid = ? AND user_package = ? AND id <> ?", userID, userPackageID, excludeSiteID).Find(&sites).Error; err != nil {
		return nil, nil, err
	}
	for _, site := range sites {
		addDomains(domainSet, mainSet, site.Domains)
	}
	return domainSet, mainSet, nil
}

func addDomains(domainSet map[string]struct{}, mainSet map[string]struct{}, domains []string) {
	for _, domain := range domains {
		normalized := normalizeDomain(domain)
		if normalized == "" {
			continue
		}
		if _, ok := domainSet[normalized]; !ok {
			domainSet[normalized] = struct{}{}
		}
		mainKey := mainDomainKey(normalized)
		if mainKey == "" {
			continue
		}
		if _, ok := mainSet[mainKey]; !ok {
			mainSet[mainKey] = struct{}{}
		}
	}
}

func normalizeDomain(input string) string {
	host := strings.TrimSpace(strings.ToLower(input))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "#"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "?"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.TrimPrefix(host, "*.")
	host = strings.TrimRight(host, ".")
	return host
}

func mainDomainKey(domain string) string {
	if domain == "" {
		return ""
	}
	if net.ParseIP(domain) != nil {
		return domain
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func parseIntConfig(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return 0
}

func buildDomainLimitMessage(limitType string, count int) string {
	switch limitType {
	case "main_domain":
		return fmt.Sprintf(i18n.T("Main domain limit exceeded: %d"), count)
	default:
		return fmt.Sprintf(i18n.T("Domain limit exceeded: %d"), count)
	}
}
