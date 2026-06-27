package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"errors"
	"fmt"
	"log"
	"strings"
)

// SyncUserDNSRecords ensures user CNAME records are kept in sync without async tasks.
// When newSite is nil, it deletes records from oldSite; when oldSite is nil, it upserts newSite.
func SyncUserDNSRecords(oldSite, newSite *models.Site) error {
	if oldSite == nil && newSite == nil {
		return nil
	}

	if oldSite != nil && (newSite == nil || newSite.DNSProviderID != oldSite.DNSProviderID) {
		if api, err := resolveDNSAPIForSite(oldSite); err == nil {
			_ = deleteSiteDomains(api, oldSite.Domains, oldSite.CnameHostname)
		}
	}

	if newSite == nil || len(newSite.Domains) == 0 {
		return nil
	}

	api, err := resolveDNSAPIForSite(newSite)
	if err != nil {
		return err
	}

	if oldSite != nil && oldSite.DNSProviderID == newSite.DNSProviderID {
		removed := diffDomains(oldSite.Domains, newSite.Domains)
		if len(removed) > 0 {
			_ = deleteSiteDomains(api, removed, oldSite.CnameHostname)
		}
	}

	return upsertSiteDomains(api, newSite.Domains, newSite.CnameHostname)
}

func loadDNSAPI(id, uid int64) (*models.DNSAPI, error) {
	if id == 0 {
		return nil, errors.New("dnsapi id is required")
	}
	var api models.DNSAPI
	query := db.DB.Where("id = ?", id)
	if uid != 0 {
		query = query.Where("uid = ?", uid)
	}
	if err := query.First(&api).Error; err != nil {
		return nil, err
	}
	return &api, nil
}

func resolveDNSAPIForSite(site *models.Site) (*models.DNSAPI, error) {
	if site == nil {
		return nil, errors.New("site is nil")
	}
	if site.DNSProviderID != 0 {
		return loadDNSAPI(site.DNSProviderID, site.UserID)
	}
	domainKey := normalizeCnameDomain(site.CnameDomain)
	if domainKey == "" && strings.TrimSpace(site.CnameHostname) != "" {
		root, _ := splitRootDomain(site.CnameHostname)
		domainKey = normalizeCnameDomain(root)
	}
	if domainKey == "" {
		return nil, errors.New("dns provider not configured")
	}
	var cname models.CnameDomain
	if err := db.DB.Where("domain = ?", domainKey).First(&cname).Error; err != nil {
		return nil, err
	}
	if cname.DNSProviderID == 0 {
		return nil, errors.New("dns provider not configured")
	}
	var api models.DNSAPI
	if err := db.DB.Where("id = ?", cname.DNSProviderID).First(&api).Error; err != nil {
		return nil, err
	}
	return &api, nil
}

func normalizeCnameDomain(input string) string {
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
	return strings.TrimRight(host, ".")
}

func upsertSiteDomains(api *models.DNSAPI, domains []string, cname string) error {
	if api == nil || len(domains) == 0 || strings.TrimSpace(cname) == "" {
		return nil
	}
	for _, domain := range domains {
		root, name := splitRootDomain(domain)
		if root == "" {
			continue
		}
		if err := upsertDNSRecordSimple(*api, root, "CNAME", name, cname, 600); err != nil {
			return err
		}
	}
	return nil
}

func deleteSiteDomains(api *models.DNSAPI, domains []string, cname string) error {
	if api == nil || len(domains) == 0 || strings.TrimSpace(cname) == "" {
		return nil
	}
	for _, domain := range domains {
		root, name := splitRootDomain(domain)
		if root == "" {
			continue
		}
		if err := deleteDNSRecord(*api, root, "CNAME", name, cname, 600); err != nil {
			return err
		}
	}
	return nil
}

func upsertDNSRecordSimple(api models.DNSAPI, zone, rType, name, value string, ttl int) error {
	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil {
		return fmt.Errorf("get provider failed: %v", err)
	}
	if provider == nil {
		return fmt.Errorf("provider %s not found", api.Type)
	}

	records, err := provider.GetRecords(zone)
	if err != nil {
		return fmt.Errorf("get records failed: %v", err)
	}

	var existing *dns.DNSRecord
	for _, r := range records {
		if r.Type == rType && r.Name == name {
			existing = &r
			break
		}
	}

	if existing != nil {
		if existing.Value == value {
			return nil
		}
		if err := provider.DeleteRecord(zone, *existing); err != nil {
			return fmt.Errorf("delete record failed: %v", err)
		}
	}

	newRecord := dns.DNSRecord{
		Type:  rType,
		Name:  name,
		Value: value,
		TTL:   ttl,
	}
	if err := provider.AddRecord(zone, newRecord); err != nil {
		return fmt.Errorf("add record failed: %v", err)
	}
	return nil
}

func deleteDNSRecord(api models.DNSAPI, zone, rType, name, value string, ttl int) error {
	if isProtectedPackageCNAMERecord(zone, rType, name) {
		log.Printf("[DNS] protected package cname delete skipped zone=%s name=%s value=%s", zone, name, value)
		return nil
	}
	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil {
		return fmt.Errorf("get provider failed: %v", err)
	}
	if provider == nil {
		return fmt.Errorf("provider %s not found", api.Type)
	}
	return provider.DeleteRecord(zone, dns.DNSRecord{
		Type:  rType,
		Name:  name,
		Value: value,
		TTL:   ttl,
	})
}

func isProtectedPackageCNAMERecord(zone, recordType, name string) bool {
	if db.DB == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(recordType), "CNAME") {
		return false
	}
	zone = normalizeCnameDomain(zone)
	name = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(name, ".")))
	if zone == "" || name == "" || name == "@" {
		return false
	}
	var count int64
	if err := db.DB.Model(&models.UserPackage{}).
		Where("LOWER(cname_mode) = ? AND cname_domain = ? AND (LOWER(cname_hostname) = ? OR LOWER(record_id) = ?)",
			"package", zone, name, name).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func diffDomains(oldDomains, newDomains []string) []string {
	if len(oldDomains) == 0 {
		return nil
	}
	lookup := map[string]struct{}{}
	for _, d := range newDomains {
		key := normalizeDomainHost(d)
		if key != "" {
			lookup[key] = struct{}{}
		}
	}
	removed := make([]string, 0)
	for _, d := range oldDomains {
		key := normalizeDomainHost(d)
		if key == "" {
			continue
		}
		if _, ok := lookup[key]; !ok {
			removed = append(removed, d)
		}
	}
	return removed
}

func splitRootDomain(domain string) (string, string) {
	return SplitDNSZoneAndRecord(domain)
}

func normalizeDomainHost(input string) string {
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
	return strings.TrimRight(host, ".")
}
