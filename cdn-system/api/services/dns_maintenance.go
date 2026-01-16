package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"errors"
	"strings"

	"gorm.io/gorm"
)

func RepairDNSRecords() []string {
	var groups []models.NodeGroup
	if err := db.DB.Find(&groups).Error; err != nil {
		return []string{err.Error()}
	}
	if len(groups) == 0 {
		return nil
	}

	errs := make([]string, 0)
	for _, group := range groups {
		resolvedGroup, err := dns.EnsureGroupDNSConfig(group.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		group = resolvedGroup

		var lines []models.Line
		if err := db.DB.Select("line_id", "line_name", "node_id", "node_ip_id", "enable").
			Where("node_group_id = ?", group.ID).
			Find(&lines).Error; err != nil {
			errs = append(errs, err.Error())
			continue
		}

		lineMap := map[string]*struct {
			Name    string
			NodeIDs []int64
		}{}
		for _, line := range lines {
			if !line.Enable {
				continue
			}
			lineKey := strings.TrimSpace(line.LineID)
			if lineKey == "" {
				lineKey = "default"
			}
			item := lineMap[lineKey]
			if item == nil {
				lineName := strings.TrimSpace(line.LineName)
				if lineName == "" {
					lineName = lineKey
				}
				item = &struct {
					Name    string
					NodeIDs []int64
				}{Name: lineName}
				lineMap[lineKey] = item
			}
			nodeID := line.NodeIPID
			if nodeID == 0 {
				nodeID = line.NodeID
			}
			if nodeID != 0 {
				item.NodeIDs = append(item.NodeIDs, nodeID)
			}
		}

		for lineKey, item := range lineMap {
			ids := uniqueInt64List(item.NodeIDs)
			if err := dns.SyncLineRecords(group.ID, lineKey, item.Name, "resync", ids); err != nil {
				errs = append(errs, err.Error())
			}
			if err := SyncPackageCnameForLineChange(group.ID, lineKey, item.Name, ids, "resync"); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	return errs
}

func CleanupInvalidDNSRecords() []string {
	cfg, err := LoadSystemConfig()
	if err != nil {
		return []string{err.Error()}
	}
	protected := parseDNSProtectHosts(cfg["dns_rs_protect"])
	return cleanupInvalidDNSRecords(protected)
}

func cleanupInvalidDNSRecords(protected map[string]struct{}) []string {
	var groups []models.NodeGroup
	if err := db.DB.Find(&groups).Error; err != nil {
		return []string{err.Error()}
	}
	if len(groups) == 0 {
		return nil
	}

	errs := make([]string, 0)
	allowed := map[string]map[string]struct{}{}
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		resolvedGroup, err := dns.EnsureGroupDNSConfig(group.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		group = resolvedGroup
		groupIDs = append(groupIDs, group.ID)
		domainKey := normalizeDomainInput(group.CnameDomain)
		if domainKey == "" {
			continue
		}
		host := normalizeGroupHostname(group.CnameHostname, domainKey)
		if host == "" {
			continue
		}
		if _, ok := allowed[domainKey]; !ok {
			allowed[domainKey] = map[string]struct{}{}
		}
		allowed[domainKey][normalizeRecordHost(host, domainKey)] = struct{}{}
	}
	if len(groupIDs) > 0 {
		if infos, _, err := loadSiteCnameInfos(groupIDs); err != nil {
			errs = append(errs, err.Error())
		} else {
			for _, info := range infos {
				domainKey := normalizeDomainInput(info.DomainKey)
				if domainKey == "" || strings.TrimSpace(info.Hostname) == "" {
					continue
				}
				if _, ok := allowed[domainKey]; !ok {
					allowed[domainKey] = map[string]struct{}{}
				}
				allowed[domainKey][normalizeRecordHost(info.Hostname, domainKey)] = struct{}{}
			}
		}
	}
	if len(allowed) == 0 {
		return nil
	}

	domainList := make([]string, 0, len(allowed))
	for domain := range allowed {
		domainList = append(domainList, domain)
	}
	var domainRows []models.CnameDomain
	if err := db.DB.Where("domain IN ?", domainList).Find(&domainRows).Error; err != nil {
		return []string{err.Error()}
	}
	if len(domainRows) == 0 {
		return nil
	}

	apis := map[int64]models.DNSAPI{}
	for _, domain := range domainRows {
		if domain.DNSProviderID == 0 {
			continue
		}
		api, ok := apis[domain.DNSProviderID]
		if !ok {
			if err := db.DB.Where("id = ?", domain.DNSProviderID).First(&api).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				errs = append(errs, err.Error())
				continue
			}
			apis[domain.DNSProviderID] = api
		}
		provider, err := dns.GetProvider(api.Type, api.Auth)
		if err != nil || provider == nil {
			if err == nil {
				err = errors.New("dns provider not available")
			}
			errs = append(errs, err.Error())
			continue
		}
		records, err := provider.GetRecords(domain.Domain)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		allowedHosts := allowed[normalizeDomainInput(domain.Domain)]
		for _, record := range records {
			if strings.EqualFold(record.Type, "NS") {
				continue
			}
			if isProtectedRecord(record.Name, domain.Domain, protected) {
				continue
			}
			recordHost := normalizeRecordHost(record.Name, domain.Domain)
			if strings.EqualFold(record.Type, "A") {
				if _, ok := allowedHosts[recordHost]; ok {
					continue
				}
			}
			if err := provider.DeleteRecord(domain.Domain, record); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	return errs
}

func parseDNSProtectHosts(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
	out := map[string]struct{}{}
	for _, part := range parts {
		host := strings.TrimSpace(strings.ToLower(part))
		host = strings.TrimSuffix(host, ".")
		if host != "" {
			out[host] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isProtectedRecord(recordName, domain string, protected map[string]struct{}) bool {
	if len(protected) == 0 {
		return false
	}
	name := normalizeRecordName(recordName)
	if name != "" {
		if _, ok := protected[name]; ok {
			return true
		}
	}
	host := normalizeRecordHost(recordName, domain)
	if host != "" {
		if _, ok := protected[host]; ok {
			return true
		}
	}
	domainKey := normalizeDomainInput(domain)
	if domainKey == "" {
		return false
	}
	if host == "@" {
		if _, ok := protected[domainKey]; ok {
			return true
		}
	}
	if host != "" && host != "@" {
		fqdn := host + "." + domainKey
		if _, ok := protected[fqdn]; ok {
			return true
		}
	}
	return false
}

func normalizeRecordHost(recordName, domain string) string {
	host := normalizeRecordName(recordName)
	if host == "" {
		return ""
	}
	domainKey := normalizeDomainInput(domain)
	if domainKey == "" {
		return host
	}
	if host == domainKey {
		return "@"
	}
	suffix := "." + domainKey
	if strings.HasSuffix(host, suffix) {
		host = strings.TrimSuffix(host, suffix)
	}
	return strings.TrimSuffix(host, ".")
}

func normalizeRecordName(input string) string {
	name := strings.TrimSpace(strings.ToLower(input))
	name = strings.TrimSuffix(name, ".")
	return name
}

func normalizeDomainInput(input string) string {
	domain := strings.TrimSpace(strings.ToLower(input))
	if strings.HasPrefix(domain, "http://") {
		domain = strings.TrimPrefix(domain, "http://")
	} else if strings.HasPrefix(domain, "https://") {
		domain = strings.TrimPrefix(domain, "https://")
	}
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "#"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "?"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	domain = strings.TrimRight(domain, ".")
	return domain
}

func normalizeGroupHostname(host, domain string) string {
	normalized := normalizeDomainInput(host)
	if normalized == "" {
		return ""
	}
	domain = normalizeDomainInput(domain)
	if domain != "" {
		if normalized == domain {
			return "@"
		}
		suffix := "." + domain
		if strings.HasSuffix(normalized, suffix) {
			normalized = strings.TrimSuffix(normalized, suffix)
		}
	}
	return strings.TrimSuffix(normalized, ".")
}
