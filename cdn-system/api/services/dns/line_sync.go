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
		if updater, ok := provider.(RecordSetUpdater); ok {
			desiredIPs := make([]string, 0, len(ipWeights))
			for ip := range ipWeights {
				desiredIPs = append(desiredIPs, ip)
			}
			if err := updater.UpsertRecordSet(root, record, desiredIPs); err != nil {
				errs = append(errs, fmt.Sprintf("provider=%s upsert domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
			}
		} else {
			if err := syncLineRecordSetLegacy(provider, root, record, ipWeights); err != nil {
				errs = append(errs, fmt.Sprintf("provider=%s resync domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
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
	weightMap := map[int64]int{}
	if action == "add" || action == "resync" {
		weightMap = loadLineWeightMap(groupID, lineID, nodeIPIDs)
	}

	if updater, ok := provider.(RecordSetUpdater); ok {
		desiredIPs, err := loadLineNodeIPs(groupID, lineID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("provider=%s list domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
		} else {
			if deleteAll {
				desiredIPs = []string{}
			}
			if err := updater.UpsertRecordSet(root, record, desiredIPs); err != nil {
				errs = append(errs, fmt.Sprintf("provider=%s upsert domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
			}
		}
	} else {
		if deleteAll {
			if err := deleteAllByLine(provider, root, record); err != nil {
				errs = append(errs, fmt.Sprintf("provider=%s delete-all domain=%s name=%s line=%s err=%v", api.Type, root, record.Name, record.Line, err))
			}
		}
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

func syncLineRecordSetLegacy(provider Provider, domain string, record DNSRecord, desiredWeights map[string]int) error {
	records, err := provider.GetRecords(domain)
	if err != nil {
		return err
	}
	existing := make([]DNSRecord, 0)
	existingCount := map[string]int{}
	for _, r := range records {
		if r.Type != record.Type {
			continue
		}
		if r.Name != record.Name {
			continue
		}
		if strings.TrimSpace(record.Line) != "" && r.Line != record.Line {
			continue
		}
		existing = append(existing, r)
		existingCount[r.Value]++
	}
	if len(desiredWeights) == 0 {
		return deleteAllByLine(provider, domain, record)
	}

	replacer, canReplace := provider.(RecordValueReplacer)
	for value, weight := range desiredWeights {
		_, exists := existingCount[value]
		record.Value = value
		record.Weight = weight
		if !exists {
			if err := provider.AddRecord(domain, record); err != nil {
				return err
			}
			continue
		}
		for _, r := range existing {
			if r.Value != value {
				continue
			}
			needsUpdate := r.TTL != record.TTL || (weight > 0 && r.Weight != weight)
			if !needsUpdate {
				break
			}
			if canReplace {
				if err := replacer.ReplaceRecordValue(domain, record, value); err != nil {
					return err
				}
			} else {
				if err := provider.DeleteRecord(domain, record); err != nil {
					return err
				}
				if err := provider.AddRecord(domain, record); err != nil {
					return err
				}
			}
			break
		}
	}

	for value, count := range existingCount {
		if _, keep := desiredWeights[value]; keep {
			for i := 1; i < count; i++ {
				record.Value = value
				if err := provider.DeleteRecord(domain, record); err != nil {
					return err
				}
			}
			continue
		}
		for i := 0; i < count; i++ {
			record.Value = value
			if err := provider.DeleteRecord(domain, record); err != nil {
				return err
			}
		}
	}
	return nil
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
