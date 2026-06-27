package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"errors"
	"log"
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
		errs = append(errs, resyncGroupDNSRecords(group.ID)...)
	}
	return errs
}

func ResyncDNSForProvider(providerID int64) []string {
	if providerID == 0 || db.DB == nil {
		return nil
	}
	var domains []models.CnameDomain
	if err := db.DB.Where("dns_provider_id = ?", providerID).Find(&domains).Error; err != nil {
		return []string{err.Error()}
	}
	if len(domains) == 0 {
		return nil
	}
	keys := make([]string, 0, len(domains))
	for _, domain := range domains {
		domainKey := normalizeDomainInput(domain.Domain)
		if domainKey != "" {
			keys = append(keys, domainKey)
		}
	}
	return ResyncDNSForCnameDomains(keys)
}

func ResyncDNSForCnameDomains(domains []string) []string {
	if db.DB == nil || len(domains) == 0 {
		return nil
	}
	domainSet := map[string]struct{}{}
	for _, domain := range domains {
		domainKey := normalizeDomainInput(domain)
		if domainKey == "" {
			continue
		}
		domainSet[domainKey] = struct{}{}
	}
	if len(domainSet) == 0 {
		return nil
	}
	domainList := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domainList = append(domainList, domain)
	}

	errs := make([]string, 0)
	var groups []models.NodeGroup
	if err := db.DB.
		Where("cname_domain IN ? OR cname_domain = '' OR cname_domain IS NULL", domainList).
		Find(&groups).Error; err != nil {
		return []string{err.Error()}
	}
	if len(groups) == 0 {
		return nil
	}

	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		resolvedGroup, err := dns.EnsureGroupDNSConfig(group.ID)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		domainKey := normalizeDomainInput(resolvedGroup.CnameDomain)
		if domainKey == "" {
			continue
		}
		if _, ok := domainSet[domainKey]; ok {
			groupIDs = append(groupIDs, resolvedGroup.ID)
		}
	}
	groupIDs = uniqueInt64List(groupIDs)
	for _, groupID := range groupIDs {
		errs = append(errs, resyncGroupDNSRecords(groupID)...)
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
	managedARecords := map[string]struct{}{}
	managedCnameRecords := map[string]struct{}{}
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
		addManagedRecordName(managedARecords, domainKey, group.CnameHostname)
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
				if info.Hostname != "" && info.Hostname != "@" {
					addAllowedCnameValue(allowedCnameValues, info.Hostname+"."+domainKey)
					addManagedRecordName(managedCnameRecords, domainKey, info.Hostname)
				}
			}
		}
	}
	var userPackages []models.UserPackage
	if err := db.DB.Select("cname_mode", "cname_hostname", "cname_domain", "record_id").Find(&userPackages).Error; err != nil {
		errs = append(errs, err.Error())
	} else {
		for _, pack := range userPackages {
			if strings.TrimSpace(strings.ToLower(pack.CnameMode)) != "package" {
				continue
			}
			domainKey, host := resolveSiteCnameTarget(models.Site{UserPackageID: pack.ID}, pack)
			if domainKey == "" {
				continue
			}
			domainSet[normalizeDomainInput(domainKey)] = struct{}{}
			if host != "" && host != "@" {
				addAllowedCnameValue(allowedCnameValues, host+"."+normalizeDomainInput(domainKey))
				addManagedRecordName(managedCnameRecords, domainKey, host)
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
			addManagedRecordNameFromFQDN(managedCnameRecords, row.CnameHostname)
			addManagedRecordNameFromFQDN(managedCnameRecords, row.CnameHostname2)
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
			if isProtectedPackageCNAMERecord(domain.Domain, record.Type, record.Name) {
				log.Printf("[DNS Cleanup] protected package cname kept domain=%s name=%s value=%s line=%s", domain.Domain, record.Name, record.Value, record.Line)
				continue
			}
			if strings.EqualFold(record.Type, "A") {
				if !isManagedRecordName(managedARecords, domain.Domain, record.Name) {
					continue
				}
				value := strings.TrimSpace(record.Value)
				if value != "" {
					if _, ok := allowedAValues[value]; ok {
						continue
					}
				}
			} else if strings.EqualFold(record.Type, "CNAME") {
				if !isManagedRecordName(managedCnameRecords, domain.Domain, record.Name) {
					continue
				}
				value := normalizeDomainInput(record.Value)
				if value != "" {
					if _, ok := allowedCnameValues[value]; ok {
						continue
					}
				}
			}
			log.Printf("[DNS Cleanup] delete invalid record domain=%s type=%s name=%s value=%s line=%s", domain.Domain, record.Type, record.Name, record.Value, record.Line)
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

func addManagedRecordName(records map[string]struct{}, domain, name string) {
	key := managedRecordKey(domain, name)
	if key == "" {
		return
	}
	records[key] = struct{}{}
}

func addManagedRecordNameFromFQDN(records map[string]struct{}, fqdn string) {
	host := normalizeDomainInput(fqdn)
	if host == "" || host == "@" {
		return
	}
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return
	}
	domain := strings.Join(parts[len(parts)-2:], ".")
	name := strings.Join(parts[:len(parts)-2], ".")
	addManagedRecordName(records, domain, name)
}

func isManagedRecordName(records map[string]struct{}, domain, name string) bool {
	key := managedRecordKey(domain, name)
	if key == "" {
		return false
	}
	_, ok := records[key]
	return ok
}

func managedRecordKey(domain, name string) string {
	domain = normalizeDomainInput(domain)
	name = normalizeRecordHost(name, domain)
	if domain == "" || name == "" {
		return ""
	}
	return domain + "|" + name
}

func resyncGroupDNSRecords(groupID int64) []string {
	if groupID == 0 || db.DB == nil {
		return nil
	}
	resolvedGroup, err := dns.EnsureGroupDNSConfig(groupID)
	if err != nil {
		return []string{err.Error()}
	}

	var lines []models.Line
	if err := db.DB.Select("line_id", "line_name", "node_id", "node_ip_id", "enable").
		Where("node_group_id = ?", resolvedGroup.ID).
		Find(&lines).Error; err != nil {
		return []string{err.Error()}
	}
	if len(lines) == 0 {
		return nil
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

	errs := make([]string, 0)
	for lineKey, item := range lineMap {
		ids := uniqueInt64List(item.NodeIDs)
		if err := dns.SyncLineRecords(resolvedGroup.ID, lineKey, item.Name, "resync", ids); err != nil {
			errs = append(errs, err.Error())
		}
		if err := SyncPackageCnameForLineChange(resolvedGroup.ID, lineKey, item.Name, ids, "resync"); err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}
