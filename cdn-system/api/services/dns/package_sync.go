package dns

import (
	"cdn-api/db"
	"cdn-api/models"
	"encoding/json"
	"errors"
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
	if groupID == 0 {
		return nil
	}

	group, err := EnsureGroupDNSConfig(groupID)
	if err != nil {
		return err
	}
	lineDomain := normalizeDomainName(group.CnameDomain)
	if lineDomain == "" {
		return errors.New("line cname domain is empty")
	}
	lineHost := normalizeRecordHostname(group.CnameHostname, lineDomain)
	if lineHost == "" {
		return errors.New("line cname hostname is empty")
	}
	if lineHost == "@" {
		lineHost = lineDomain
	} else {
		lineHost = lineHost + "." + lineDomain
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

	log.Printf("[DNS] package cname resync start domain=%s host=%s line=%s", root, recordName, lineID)

	hasNodes := false
	lineIPs, err := loadLineNodeIPs(groupID, lineID)
	if err != nil {
		return err
	}
	if len(lineIPs) > 0 {
		hasNodes = true
	}

	record := DNSRecord{
		Type:  "CNAME",
		Name:  recordName,
		Value: lineHost,
		TTL:   ttl,
		Line:  lineValue,
	}

	if updater, ok := provider.(RecordSetUpdater); ok {
		values := []string{}
		if hasNodes {
			values = []string{lineHost}
		}
		if err := updater.UpsertRecordSet(root, DNSRecord{Type: "CNAME", Name: recordName, TTL: ttl, Line: lineValue}, values); err != nil {
			log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
			return err
		}
		log.Printf("[DNS] package cname sync success host=%s.%s line=%s action=%s", recordName, root, lineID, action)
		return nil
	}

	if !hasNodes {
		if err := deleteAllByLine(provider, root, DNSRecord{Type: "CNAME", Name: recordName, Line: lineValue}); err != nil {
			log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
			return err
		}
		log.Printf("[DNS] package cname resync done domain=%s host=%s line=%s nodes=0", root, recordName, lineID)
		return nil
	}

	records, err := provider.GetRecords(root)
	if err != nil {
		log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
		return err
	}
	existing := make([]DNSRecord, 0)
	hasDesired := false
	for _, r := range records {
		if r.Type != "CNAME" {
			continue
		}
		if r.Name != recordName {
			continue
		}
		if strings.TrimSpace(lineValue) != "" && r.Line != lineValue {
			continue
		}
		existing = append(existing, r)
		if r.Value == lineHost {
			hasDesired = true
		}
	}

	if !hasDesired {
		if len(existing) > 0 {
			if replacer, ok := provider.(RecordValueReplacer); ok {
				if err := replacer.ReplaceRecordValue(root, DNSRecord{Type: "CNAME", Name: recordName, Line: lineValue, TTL: ttl}, lineHost); err != nil {
					log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
					return err
				}
				hasDesired = true
			} else {
				if err := deleteAllByLine(provider, root, DNSRecord{Type: "CNAME", Name: recordName, Line: lineValue}); err != nil {
					log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
					return err
				}
			}
		}
		if !hasDesired {
			if err := provider.AddRecord(root, record); err != nil {
				log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
				return err
			}
		}
	}

	for _, r := range existing {
		if r.Value == lineHost {
			continue
		}
		if err := provider.DeleteRecord(root, r); err != nil {
			log.Printf("[DNS] package cname sync failed host=%s.%s line=%s err=%v", recordName, root, lineID, err)
			return err
		}
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
