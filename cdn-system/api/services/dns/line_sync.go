package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

type dnsAuthMeta struct {
	TTL int `json:"ttl"`
}

type desiredDNSRecord struct {
	Value  string
	Weight int
}

func SyncLineRecords(groupID int64, lineID, lineName, action string, nodeIPIDs []int64) error {
	if groupID == 0 {
		return nil
	}
	group, err := EnsureGroupDNSConfig(groupID)
	if err != nil {
		return err
	}
	domainName := normalizeDomainName(group.CnameDomain)
	if domainName == "" {
		return errors.New("cname domain is empty")
	}
	if strings.TrimSpace(group.CnameHostname) == "" {
		return errors.New("cname hostname is empty")
	}
	recordName := normalizeRecordHostname(group.CnameHostname, domainName)
	if recordName == "" {
		return errors.New("cname hostname is empty")
	}

	var domain models.CnameDomain
	if err := db.DB.Where("domain = ?", domainName).First(&domain).Error; err != nil {
		return err
	}
	if domain.DNSProviderID == 0 {
		return errors.New("cname domain dns provider not configured")
	}

	var api models.DNSAPI
	if err := db.DB.Where("id = ?", domain.DNSProviderID).First(&api).Error; err != nil {
		return err
	}
	if api.Type == "dnspod_intl" {
		var auth struct {
			SecretID  string `json:"secret_id"`
			SecretKey string `json:"secret_key"`
		}
		if err := json.Unmarshal([]byte(api.Auth), &auth); err == nil {
			if strings.TrimSpace(auth.SecretID) == "" || strings.TrimSpace(auth.SecretKey) == "" {
				return errors.New("dnspod_intl requires secret_id/secret_key")
			}
		}
	}
	provider, err := GetProvider(api.Type, api.Auth)
	if err != nil || provider == nil {
		if err == nil {
			return errors.New("dns provider not available")
		}
		return err
	}
	ttl := 600
	var meta dnsAuthMeta
	if err := json.Unmarshal([]byte(api.Auth), &meta); err == nil && meta.TTL > 0 {
		ttl = meta.TTL
	}
	lineValue := ResolveLineValue(api.Type, lineID, lineName)

	action = strings.ToLower(strings.TrimSpace(action))
	logAction := action
	switch action {
	case "enable":
		action = "add"
	case "disable":
		action = "delete"
	}
	resync := action == "resync"
	if len(nodeIPIDs) == 0 && !resync {
		return nil
	}

	root := strings.TrimSpace(domain.Domain)
	if root == "" {
		return errors.New("cname domain is empty")
	}
	record := DNSRecord{
		Type: "A",
		Name: recordName,
		Line: lineValue,
		TTL:  ttl,
	}

	if resync {
		desiredNodeIDs := loadLineNodeIDs(groupID, lineID)
		var nodes []models.Node
		if len(desiredNodeIDs) > 0 {
			if err := db.DB.Select("id", "ip").Where("id IN ?", desiredNodeIDs).Find(&nodes).Error; err != nil {
				return err
			}
		}
		weightMap := loadLineWeightMap(groupID, lineID, desiredNodeIDs)
		ipWeights := map[string]int{}
		for _, node := range nodes {
			ip := strings.TrimSpace(node.IP)
			if ip == "" {
				continue
			}
			ipWeights[ip] = weightMap[node.ID]
		}

		log.Printf("[DNS] sync start provider=%s group=%d line=%s action=%s nodes=%d domains=%d", api.Type, groupID, lineID, logAction, len(ipWeights), 1)
		errs := make([]string, 0)
		if err := applyLineRecordSet(provider, root, record, ipWeights); err != nil {
			errs = append(errs, fmt.Sprintf("provider=%s resync domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
		}
		if len(errs) > 0 {
			msg := strings.Join(errs, "; ")
			log.Printf("[DNS] sync failed group=%d line=%s action=%s errors=%s", groupID, lineID, logAction, msg)
			return errors.New(msg)
		}
		log.Printf("[DNS] sync success group=%d line=%s action=%s", groupID, lineID, logAction)
		return nil
	}

	var nodes []models.Node
	if len(nodeIPIDs) > 0 {
		if err := db.DB.Select("id", "ip").Where("id IN ?", nodeIPIDs).Find(&nodes).Error; err != nil {
			return err
		}
		if len(nodes) == 0 && action != "resync" {
			return errors.New("node list empty")
		}
	}
	deleteAll := false
	if resync {
		deleteAll = true
		action = "add"
	}
	if action == "delete" {
		var remaining int64
		if err := db.DB.Model(&models.Line{}).Where("node_group_id = ? AND line_id = ? AND enable = ?", groupID, lineID, true).Count(&remaining).Error; err != nil {
			return err
		}
		deleteAll = remaining == 0
	}
	log.Printf("[DNS] sync start provider=%s group=%d line=%s action=%s nodes=%d domains=%d", api.Type, groupID, lineID, logAction, len(nodes), 1)
	errs := make([]string, 0)
	if deleteAll {
		if err := applyLineRecordSet(provider, root, record, map[string]int{}); err != nil {
			errs = append(errs, fmt.Sprintf("provider=%s clear domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
		}
		if len(errs) > 0 {
			msg := strings.Join(errs, "; ")
			log.Printf("[DNS] sync failed group=%d line=%s action=%s errors=%s", groupID, lineID, logAction, msg)
			return errors.New(msg)
		}
		log.Printf("[DNS] sync success group=%d line=%s action=%s (cleared)", groupID, lineID, logAction)
		return nil
	}
	weightMap := map[int64]int{}
	if action == "add" || action == "resync" {
		weightMap = loadLineWeightMap(groupID, lineID, nodeIPIDs)
	}

	if _, ok := provider.(RecordSetUpdater); ok {
		desiredIPs, err := loadLineNodeIPs(groupID, lineID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("provider=%s list domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
		} else {
			lineNodeIDs := loadLineNodeIDs(groupID, lineID)
			lineWeightMap := loadLineWeightMap(groupID, lineID, lineNodeIDs)
			ipWeights := make(map[string]int, len(desiredIPs))
			for _, ip := range desiredIPs {
				ipWeights[ip] = 0
			}
			if len(lineNodeIDs) > 0 {
				var lineNodes []models.Node
				if err := db.DB.Select("id", "ip").Where("id IN ?", lineNodeIDs).Find(&lineNodes).Error; err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s load nodes domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
				} else {
					for _, node := range lineNodes {
						ip := strings.TrimSpace(node.IP)
						if ip != "" {
							ipWeights[ip] = lineWeightMap[node.ID]
						}
					}
				}
			}
			if len(errs) == 0 {
				if err := applyLineRecordSet(provider, root, record, ipWeights); err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s upsert domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
				}
			}
		}
	} else {
		for _, node := range nodes {
			if strings.TrimSpace(node.IP) == "" {
				continue
			}
			record.Value = strings.TrimSpace(node.IP)
			record.Weight = weightMap[node.ID]
			switch action {
			case "add", "enable":
				if err := provider.AddRecord(root, record); err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s add domain=%s name=%s value=%s line=%s err=%v", api.Type, root, record.Name, record.Value, record.Line, err))
				}
			case "delete", "disable":
				if err := provider.DeleteRecord(root, record); err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s delete domain=%s name=%s value=%s line=%s err=%v", api.Type, root, record.Name, record.Value, record.Line, err))
				}
			default:
				return errors.New("unsupported action")
			}
		}
	}

	if len(errs) > 0 {
		msg := strings.Join(errs, "; ")
		log.Printf("[DNS] sync failed group=%d line=%s action=%s errors=%s", groupID, lineID, logAction, msg)
		return errors.New(msg)
	}
	log.Printf("[DNS] sync success group=%d line=%s action=%s", groupID, lineID, logAction)
	return nil
}

func applyLineRecordSet(provider Provider, domain string, record DNSRecord, desiredWeights map[string]int) error {
	if updater, ok := provider.(RecordSetUpdater); ok {
		values := make([]string, 0, len(desiredWeights))
		for value := range desiredWeights {
			value = strings.TrimSpace(value)
			if value != "" {
				values = append(values, value)
			}
		}
		return updater.UpsertRecordSet(domain, record, values)
	}
	return syncLineRecordSetLegacy(provider, domain, record, desiredWeights)
}

// ReconcileLineRecordSet syncs provider records for name/type/line to desired values.
func ReconcileLineRecordSet(provider Provider, domain string, record DNSRecord, values []string) error {
	desired := make(map[string]int, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			desired[value] = 0
		}
	}
	return syncLineRecordSetLegacy(provider, domain, record, desired)
}

func syncLineRecordSetLegacy(provider Provider, domain string, record DNSRecord, desiredWeights map[string]int) error {
	records, err := provider.GetRecords(domain)
	if err != nil {
		return err
	}
	desired := make(map[string]desiredDNSRecord, len(desiredWeights))
	for value, weight := range desiredWeights {
		key := normalizeDNSRecordValue(record.Type, value)
		if key == "" {
			continue
		}
		desired[key] = desiredDNSRecord{Value: strings.TrimSpace(value), Weight: weight}
	}
	activeRecords := map[string][]DNSRecord{}
	allRecords := map[string][]DNSRecord{}
	for _, r := range records {
		if r.Type != record.Type {
			continue
		}
		if r.Name != record.Name {
			continue
		}
		if !dnsLineMatches(record.Line, r.Line) {
			continue
		}
		key := normalizeDNSRecordValue(record.Type, r.Value)
		if key == "" {
			continue
		}
		allRecords[key] = append(allRecords[key], r)
		if dnsRecordIsActive(r) {
			activeRecords[key] = append(activeRecords[key], r)
		}
	}
	if len(desired) == 0 {
		for _, records := range allRecords {
			for _, item := range records {
				if err := provider.DeleteRecord(domain, item); err != nil {
					return err
				}
			}
		}
		return nil
	}

	replacer, canReplace := provider.(RecordValueReplacer)
	for key, wanted := range desired {
		active := activeRecords[key]
		exists := len(active) > 0
		record.Value = wanted.Value
		record.Weight = wanted.Weight
		if !exists {
			if err := provider.AddRecord(domain, record); err != nil {
				return err
			}
			continue
		}
		for _, r := range active {
			needsUpdate := r.TTL != record.TTL || (wanted.Weight > 0 && r.Weight != wanted.Weight)
			if !needsUpdate {
				break
			}
			if canReplace {
				updateRecord := record
				updateRecord.Value = r.Value
				if err := replacer.ReplaceRecordValue(domain, updateRecord, wanted.Value); err != nil {
					return err
				}
			} else {
				// Avoid delete-then-add for TTL/weight drift on providers without
				// an in-place update API; removing live CDN records creates outages.
				break
			}
			break
		}
	}

	for key, records := range allRecords {
		if _, keep := desired[key]; keep {
			continue
		}
		for _, item := range records {
			if err := provider.DeleteRecord(domain, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func dnsLineMatches(want, got string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	return normalizePackageDNSLine(want) == normalizePackageDNSLine(got)
}

func normalizeDNSRecordValue(recordType, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(recordType, "CNAME") {
		return normalizeDomainName(value)
	}
	return value
}

func dnsRecordIsActive(record DNSRecord) bool {
	status := strings.ToLower(strings.TrimSpace(record.Status))
	switch status {
	case "", "enable", "enabled", "normal", "active", "1", "ok":
		return true
	case "disable", "disabled", "pause", "paused", "0":
		return false
	default:
		return true
	}
}

func normalizeDomainName(input string) string {
	domain := strings.TrimSpace(strings.ToLower(input))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
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
	return strings.TrimRight(domain, ".")
}

func normalizeRecordHostname(input, domain string) string {
	host := normalizeDomainName(input)
	if host == "" {
		return ""
	}
	domain = normalizeDomainName(domain)
	if domain == "" {
		return host
	}
	if host == domain {
		return "@"
	}
	suffix := "." + domain
	if strings.HasSuffix(host, suffix) {
		return strings.TrimSuffix(host, suffix)
	}
	return host
}

func loadLineNodeIPs(groupID int64, lineID string) ([]string, error) {
	if groupID == 0 {
		return []string{}, nil
	}
	var lines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id = ? AND line_id = ? AND enable = ?", groupID, lineID, true).
		Find(&lines).Error; err != nil {
		return nil, err
	}
	nodeIDs := make([]int64, 0, len(lines))
	seen := map[int64]struct{}{}
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID == 0 {
			continue
		}
		if _, ok := seen[nodeID]; ok {
			continue
		}
		seen[nodeID] = struct{}{}
		nodeIDs = append(nodeIDs, nodeID)
	}
	if len(nodeIDs) == 0 {
		return []string{}, nil
	}
	var nodes []models.Node
	if err := db.DB.Select("id", "ip", "enable").Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(nodes))
	ipSeen := map[string]struct{}{}
	for _, node := range nodes {
		if !node.Enable {
			continue
		}
		ip := strings.TrimSpace(node.IP)
		if ip == "" {
			continue
		}
		if _, ok := ipSeen[ip]; ok {
			continue
		}
		ipSeen[ip] = struct{}{}
		ips = append(ips, ip)
	}
	return ips, nil
}

func loadLineNodeIDs(groupID int64, lineID string) []int64 {
	if groupID == 0 || db.DB == nil {
		return []int64{}
	}
	var lines []models.Line
	if err := db.DB.Select("node_id", "node_ip_id").
		Where("node_group_id = ? AND line_id = ? AND enable = ?", groupID, lineID, true).
		Find(&lines).Error; err != nil {
		return []int64{}
	}
	seen := map[int64]struct{}{}
	ids := make([]int64, 0, len(lines))
	for _, line := range lines {
		nodeID := line.NodeIPID
		if nodeID == 0 {
			nodeID = line.NodeID
		}
		if nodeID == 0 {
			continue
		}
		if _, ok := seen[nodeID]; ok {
			continue
		}
		seen[nodeID] = struct{}{}
		ids = append(ids, nodeID)
	}
	return ids
}
