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

type pkgAuthMeta struct {
	TTL int `json:"ttl"`
}

// SyncPackageLineRecords syncs DNS records for a package hostname and line.
// action: add, delete, resync
func SyncPackageLineRecords(domain models.CnameDomain, hostname string, groupID int64, lineID, lineName, action string, nodeIPIDs []int64) error {
	if db.DB == nil {
		return nil
	}
	root := strings.TrimSpace(domain.Domain)
	if root == "" {
		return errors.New("domain is empty")
	}
	if strings.TrimSpace(hostname) == "" {
		return errors.New("hostname is empty")
	}
	if domain.DNSProviderID == 0 {
		return errors.New("dns provider not configured")
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
	var meta pkgAuthMeta
	if err := json.Unmarshal([]byte(api.Auth), &meta); err == nil && meta.TTL > 0 {
		ttl = meta.TTL
	}
	lineValue := ResolveLineValue(api.Type, lineID, lineName)

	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "add", "delete", "enable", "disable", "resync":
	default:
		return errors.New("unsupported action")
	}

	recordName := strings.TrimSpace(hostname)
	if recordName == "" {
		return errors.New("hostname is empty")
	}

	if action == "resync" {
		log.Printf("[DNS] package cname resync start domain=%s host=%s line=%s", root, recordName, lineID)
		record := DNSRecord{Type: "A", Name: recordName, Line: lineValue}
		if err := deleteAllByLine(provider, root, record); err != nil {
			return err
		}
		if len(nodeIPIDs) == 0 {
			log.Printf("[DNS] package cname resync done domain=%s host=%s line=%s nodes=0", root, recordName, lineID)
			return nil
		}
		action = "add"
	}

	if len(nodeIPIDs) == 0 {
		return nil
	}
	var nodes []models.Node
	if err := db.DB.Select("id", "ip").Where("id IN ?", nodeIPIDs).Find(&nodes).Error; err != nil {
		return err
	}
	if len(nodes) == 0 {
		return errors.New("node list empty")
	}

	weightMap := map[int64]int{}
	if action == "add" || action == "enable" {
		weightMap = loadLineWeightMap(groupID, lineID, nodeIPIDs)
	}

	var errs []string
	for _, node := range nodes {
		if strings.TrimSpace(node.IP) == "" {
			continue
		}
		record := DNSRecord{
			Type:   "A",
			Name:   recordName,
			Value:  strings.TrimSpace(node.IP),
			TTL:    ttl,
			Line:   lineValue,
			Weight: weightMap[node.ID],
		}
		switch action {
		case "add", "enable":
			if err := provider.AddRecord(root, record); err != nil {
				errs = append(errs, fmt.Sprintf("add domain=%s name=%s value=%s line=%s err=%v", root, record.Name, record.Value, record.Line, err))
			}
		case "delete", "disable":
			if err := provider.DeleteRecord(root, record); err != nil {
				errs = append(errs, fmt.Sprintf("delete domain=%s name=%s value=%s line=%s err=%v", root, record.Name, record.Value, record.Line, err))
			}
		default:
			return errors.New("unsupported action")
		}
	}

	if len(errs) > 0 {
		msg := strings.Join(errs, "; ")
		log.Printf("[DNS] package cname sync failed host=%s.%s line=%s errors=%s", recordName, root, lineID, msg)
		return errors.New(msg)
	}
	log.Printf("[DNS] package cname sync success host=%s.%s line=%s action=%s", recordName, root, lineID, action)
	return nil
}

func deleteAllByLine(provider Provider, domain string, record DNSRecord) error {
	if deleter, ok := provider.(LineRecordDeleter); ok {
		return deleter.DeleteRecordsByLine(domain, record)
	}
	records, err := provider.GetRecords(domain)
	if err != nil {
		return err
	}
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
		if err := provider.DeleteRecord(domain, r); err != nil {
			return err
		}
	}
	return nil
}
