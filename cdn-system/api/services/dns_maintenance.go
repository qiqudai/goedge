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
	allowedAValues := map[string]struct{}{}
	allowedCnameValues := map[string]struct{}{}
	domainSet := map[string]struct{}{}
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
		domainSet[domainKey] = struct{}{}
		lineValue := buildLineCnameValue(domainKey, group.CnameHostname)
		addAllowedCnameValue(allowedCnameValues, lineValue)
	}
	if len(groupIDs) > 0 {
		var lines []models.Line
		if err := db.DB.Select("node_id", "node_ip_id", "enable").
			Where("node_group_id IN ?", groupIDs).
			Find(&lines).Error; err != nil {
			errs = append(errs, err.Error())
		} else {
			nodeIDs := make([]int64, 0, len(lines))
			for _, line := range lines {
				if !line.Enable {
					continue
				}
				nodeID := line.NodeIPID
				if nodeID == 0 {
					nodeID = line.NodeID
				}
				if nodeID != 0 {
					nodeIDs = append(nodeIDs, nodeID)
				}
			}
			nodeIDs = uniqueInt64List(nodeIDs)
			if len(nodeIDs) > 0 {
				var nodes []models.Node
				if err := db.DB.Select("id", "ip").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
					errs = append(errs, err.Error())
				} else {
					for _, node := range nodes {
						addAllowedAValue(allowedAValues, node.IP)
					}
				}
			}
		}
		if infos, _, err := loadSiteCnameInfos(groupIDs); err != nil {
			errs = append(errs, err.Error())
		} else {
			for _, info := range infos {
				domainKey := normalizeDomainInput(info.DomainKey)
				if domainKey == "" {
					continue
				}
				domainSet[domainKey] = struct{}{}
			}
		}
	}
	type siteCnameRow struct {
		CnameHostname  string `gorm:"column:cname_hostname"`
		CnameHostname2 string `gorm:"column:cname_hostname2"`
	}
	siteCols := []string{"cname_hostname"}
	if db.DB.Migrator().HasColumn(&models.Site{}, "cname_hostname2") {
		siteCols = append(siteCols, "cname_hostname2")
	}
	var siteRows []siteCnameRow
	if err := db.DB.Model(&models.Site{}).Select(siteCols).Find(&siteRows).Error; err != nil {
		errs = append(errs, err.Error())
	} else {
		for _, row := range siteRows {
			addAllowedCnameValue(allowedCnameValues, row.CnameHostname)
			addAllowedCnameValue(allowedCnameValues, row.CnameHostname2)
		}
	}
	if len(domainSet) == 0 {
		return nil
	}
	if len(allowedAValues) == 0 && len(allowedCnameValues) == 0 {
		return nil
	}

	domainList := make([]string, 0, len(domainSet))
	for domain := range domainSet {
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
		for _, record := range records {
			if strings.EqualFold(record.Type, "NS") {
				continue
			}
			if !strings.EqualFold(record.Type, "A") && !strings.EqualFold(record.Type, "CNAME") {
				continue
			}
			if isProtectedRecord(record.Name, domain.Domain, protected) {
				continue
			}
			if strings.EqualFold(record.Type, "A") {
				value := strings.TrimSpace(record.Value)
				if value != "" {
					if _, ok := allowedAValues[value]; ok {
						continue
					}
				}
			} else if strings.EqualFold(record.Type, "CNAME") {
				value := normalizeDomainInput(record.Value)
				if value != "" {
					if _, ok := allowedCnameValues[value]; ok {
						continue
					}
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

func buildLineCnameValue(domainKey, host string) string {
	domainKey = normalizeDomainInput(domainKey)
	if domainKey == "" {
		return ""
	}
	host = normalizeDomainInput(host)
	if host == "" {
		return ""
	}
	recordHost := normalizeRecordHost(host, domainKey)
	if recordHost == "" {
		return ""
	}
	if recordHost == "@" {
		return domainKey
	}
	return recordHost + "." + domainKey
}

func addAllowedAValue(values map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	values[value] = struct{}{}
}

func addAllowedCnameValue(values map[string]struct{}, value string) {
	value = normalizeDomainInput(value)
	if value == "" {
		return
	}
	values[value] = struct{}{}
}
