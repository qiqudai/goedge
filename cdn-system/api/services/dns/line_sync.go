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
	var group models.NodeGroup
	if err := db.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return err
	}
	if strings.TrimSpace(group.CnameHostname) == "" {
		return errors.New("cname hostname is empty")
	}

	var domains []models.CnameDomain
	if err := db.DB.Find(&domains).Error; err != nil {
		return err
	}
	if len(domains) == 0 {
		return errors.New("cname domains not configured")
	}

	providerIDs := make([]int64, 0)
	providerIDSet := make(map[int64]struct{})
	for _, domain := range domains {
		if domain.DNSProviderID == 0 {
			return errors.New("cname domain dns provider not configured")
		}
		if _, ok := providerIDSet[domain.DNSProviderID]; !ok {
			providerIDSet[domain.DNSProviderID] = struct{}{}
			providerIDs = append(providerIDs, domain.DNSProviderID)
		}
	}
	if len(providerIDs) == 0 {
		return errors.New("dns provider not configured")
	}

	var apis []models.DNSAPI
	if err := db.DB.Where("id IN ?", providerIDs).Find(&apis).Error; err != nil {
		return err
	}
	if len(apis) == 0 {
		return errors.New("dns provider not configured")
	}

	type providerState struct {
		api       models.DNSAPI
		provider  Provider
		ttl       int
		lineValue string
	}
	providerStates := make(map[int64]*providerState)
	for _, api := range apis {
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
		providerStates[api.ID] = &providerState{
			api:       api,
			provider:  provider,
			ttl:       ttl,
			lineValue: ResolveLineValue(api.Type, lineID, lineName),
		}
	}

	action = strings.ToLower(strings.TrimSpace(action))
	logAction := action
	switch action {
	case "enable":
		action = "add"
	case "disable":
		action = "delete"
	}
	resync := action == "resync"
	if len(nodeIPIDs) == 0 && action != "resync" {
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
		if err := db.DB.Model(&models.Line{}).Where("node_group_id = ? AND line_id = ?", groupID, lineID).Count(&remaining).Error; err != nil {
			return err
		}
		deleteAll = remaining == 0
	}
	domainCounts := make(map[int64]int)
	for _, domain := range domains {
		domainCounts[domain.DNSProviderID]++
	}
	for providerID, count := range domainCounts {
		state := providerStates[providerID]
		if state == nil || count == 0 {
			continue
		}
		log.Printf("[DNS] sync start provider=%s group=%d line=%s action=%s nodes=%d domains=%d", state.api.Type, groupID, lineID, logAction, len(nodes), count)
	}
	errs := make([]string, 0)
	deletedDomains := map[string]struct{}{}
	weightMap := map[int64]int{}
	if action == "add" || action == "resync" {
		weightMap = loadLineWeightMap(groupID, lineID, nodeIPIDs)
	}
	for _, domain := range domains {
		state := providerStates[domain.DNSProviderID]
		if state == nil {
			return errors.New("dns provider not configured")
		}
		root := strings.TrimSpace(domain.Domain)
		if root == "" {
			continue
		}
		if deleteAll {
			key := fmt.Sprintf("%d:%s", domain.DNSProviderID, root)
			if _, ok := deletedDomains[key]; ok {
				continue
			}
			deletedDomains[key] = struct{}{}
			record := DNSRecord{
				Type: "A",
				Name: strings.TrimSpace(group.CnameHostname),
				Line: state.lineValue,
			}
			if err := deleteAllByLine(state.provider, root, record); err != nil {
				errs = append(errs, fmt.Sprintf("provider=%s delete-all domain=%s name=%s line=%s err=%v", state.api.Type, root, record.Name, record.Line, err))
			}
		}
		for _, node := range nodes {
			if strings.TrimSpace(node.IP) == "" {
				continue
			}
			record := DNSRecord{
				Type:   "A",
				Name:   strings.TrimSpace(group.CnameHostname),
				Value:  strings.TrimSpace(node.IP),
				TTL:    state.ttl,
				Line:   state.lineValue,
				Weight: weightMap[node.ID],
			}
			switch action {
			case "add", "enable":
				if err := state.provider.AddRecord(root, record); err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s add domain=%s name=%s value=%s line=%s err=%v", state.api.Type, root, record.Name, record.Value, record.Line, err))
				}
			case "delete", "disable":
				if err := state.provider.DeleteRecord(root, record); err != nil {
					errs = append(errs, fmt.Sprintf("provider=%s delete domain=%s name=%s value=%s line=%s err=%v", state.api.Type, root, record.Name, record.Value, record.Line, err))
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
