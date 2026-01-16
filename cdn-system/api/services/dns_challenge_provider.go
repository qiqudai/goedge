package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services/dns"
	"cdn-common/acme"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"
	"golang.org/x/net/publicsuffix"
)

const (
	defaultDNSTTL                = 300
	manualDNSPropagationTimeout  = 10 * time.Minute
	manualDNSPropagationInterval = 10 * time.Second
	dnsAPIPropagationTimeout     = 30 * time.Minute
	dnsAPIPropagationInterval    = 15 * time.Second
)

type DNSChallengeInfo struct {
	Domain      string `json:"domain"`
	FQDN        string `json:"fqdn"`
	RecordName  string `json:"record_name"`
	RecordValue string `json:"record_value"`
	RecordType  string `json:"record_type"`
	Zone        string `json:"zone"`
}

func BuildDNSChallengeProvider(cert models.Cert) (acme.ChallengeProvider, error) {
	if cert.DNSAPI == nil || *cert.DNSAPI == 0 {
		return &manualDNSChallengeProvider{certID: int64(cert.ID)}, nil
	}

	var api models.DNSAPI
	if err := db.DB.Where("id = ?", *cert.DNSAPI).First(&api).Error; err != nil {
		return nil, err
	}

	provider, err := dns.GetProvider(api.Type, api.Auth)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("dns provider %s not found", api.Type)
	}

	return &dnsAPIChallengeProvider{provider: provider}, nil
}

func ParseDNSChallengeInfo(raw string) (*DNSChallengeInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("dns challenge info is empty")
	}
	var info DNSChallengeInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func CheckDNSChallengeTXT(fqdn string, expected string) (bool, error) {
	fqdn = strings.TrimSuffix(strings.TrimSpace(fqdn), ".")
	expected = strings.TrimSpace(expected)
	if fqdn == "" || expected == "" {
		return false, errors.New("fqdn or expected value is empty")
	}
	records, err := net.LookupTXT(fqdn)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		if strings.TrimSpace(record) == expected {
			return true, nil
		}
	}
	return false, nil
}

func buildDNSChallengeInfo(domain, keyAuth string) (DNSChallengeInfo, error) {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := strings.TrimSuffix(info.EffectiveFQDN, ".")
	zone, err := resolveDNSZone(fqdn, domain)
	if err != nil {
		zone = strings.TrimSuffix(domain, ".")
	}
	name := resolveDNSRecordName(fqdn, zone)

	return DNSChallengeInfo{
		Domain:      domain,
		FQDN:        fqdn,
		RecordName:  name,
		RecordValue: info.Value,
		RecordType:  "TXT",
		Zone:        zone,
	}, nil
}

func storeDNSChallengeInfo(certID int64, info DNSChallengeInfo) error {
	payload, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return db.DB.Model(&models.Cert{}).Where("id = ?", certID).Updates(map[string]interface{}{
		"state": "dns_pending",
		"ret":   string(payload),
	}).Error
}

func resolveDNSZone(fqdn string, domain string) (string, error) {
	if fqdn == "" {
		return "", errors.New("fqdn is empty")
	}
	if zone, err := dns01.FindZoneByFqdn(fqdn + "."); err == nil && zone != "" {
		return strings.TrimSuffix(zone, "."), nil
	}

	base := strings.TrimPrefix(domain, "*.")
	base = strings.TrimSuffix(base, ".")
	if base == "" {
		return "", errors.New("domain is empty")
	}
	root, err := publicsuffix.EffectiveTLDPlusOne(base)
	if err != nil {
		return base, nil
	}
	return root, nil
}

func resolveDNSRecordName(fqdn string, zone string) string {
	fqdn = strings.TrimSuffix(fqdn, ".")
	zone = strings.TrimSuffix(zone, ".")
	if fqdn == "" {
		return ""
	}
	if zone == "" || fqdn == zone {
		return "@"
	}
	if strings.HasSuffix(fqdn, "."+zone) {
		name := strings.TrimSuffix(fqdn, "."+zone)
		if name == "" {
			return "@"
		}
		return name
	}
	return fqdn
}

type dnsAPIChallengeProvider struct {
	provider dns.Provider
}

func (p *dnsAPIChallengeProvider) Present(domain, token, keyAuth string) error {
	info, err := buildDNSChallengeInfo(domain, keyAuth)
	if err != nil {
		return err
	}
	record := dns.DNSRecord{
		Type:  info.RecordType,
		Name:  info.RecordName,
		Value: info.RecordValue,
		TTL:   defaultDNSTTL,
	}
	existing, err := p.provider.GetRecords(info.Zone)
	if err != nil {
		existing = nil
	}
	matches, values, hasDesired := collectChallengeValues(existing, record)
	if hasDesired {
		return nil
	}
	if updater, ok := p.provider.(dns.RecordSetUpdater); ok {
		return updater.UpsertRecordSet(info.Zone, record, values)
	}
	if replacer, ok := p.provider.(dns.RecordValueReplacer); ok {
		if len(matches) > 0 {
			return replacer.ReplaceRecordValue(info.Zone, dns.DNSRecord{
				Type: record.Type,
				Name: record.Name,
				Line: record.Line,
				TTL:  record.TTL,
			}, record.Value)
		}
		return p.provider.AddRecord(info.Zone, record)
	}
	if len(matches) == 0 {
		return p.provider.AddRecord(info.Zone, record)
	}
	if err := p.provider.AddRecord(info.Zone, record); err == nil {
		refreshed, refreshErr := p.provider.GetRecords(info.Zone)
		if refreshErr == nil {
			_, _, hasDesired = collectChallengeValues(refreshed, record)
			if hasDesired {
				return nil
			}
			matches = filterChallengeMatches(refreshed, record)
		}
	}
	for _, item := range matches {
		if strings.TrimSpace(item.Value) == strings.TrimSpace(record.Value) {
			continue
		}
		if err := p.provider.DeleteRecord(info.Zone, item); err != nil {
			return err
		}
	}
	return p.provider.AddRecord(info.Zone, record)
}

func (p *dnsAPIChallengeProvider) CleanUp(domain, token, keyAuth string) error {
	info, err := buildDNSChallengeInfo(domain, keyAuth)
	if err != nil {
		return err
	}
	record := dns.DNSRecord{
		Type:  info.RecordType,
		Name:  info.RecordName,
		Value: info.RecordValue,
		TTL:   defaultDNSTTL,
	}
	return p.provider.DeleteRecord(info.Zone, record)
}

func (p *dnsAPIChallengeProvider) Timeout() (timeout, interval time.Duration) {
	return dnsAPIPropagationTimeout, dnsAPIPropagationInterval
}

type manualDNSChallengeProvider struct {
	certID int64
}

func (p *manualDNSChallengeProvider) Present(domain, token, keyAuth string) error {
	info, err := buildDNSChallengeInfo(domain, keyAuth)
	if err != nil {
		return err
	}
	return storeDNSChallengeInfo(p.certID, info)
}

func (p *manualDNSChallengeProvider) CleanUp(domain, token, keyAuth string) error {
	return nil
}

func (p *manualDNSChallengeProvider) Timeout() (timeout, interval time.Duration) {
	return manualDNSPropagationTimeout, manualDNSPropagationInterval
}

func collectChallengeValues(records []dns.DNSRecord, desired dns.DNSRecord) ([]dns.DNSRecord, []string, bool) {
	matches := filterChallengeMatches(records, desired)
	valueSet := map[string]struct{}{}
	values := make([]string, 0, len(matches)+1)
	desiredValue := strings.TrimSpace(desired.Value)
	hasDesired := false
	for _, record := range matches {
		value := strings.TrimSpace(record.Value)
		if value == "" {
			continue
		}
		if _, ok := valueSet[value]; ok {
			continue
		}
		valueSet[value] = struct{}{}
		values = append(values, value)
		if value == desiredValue {
			hasDesired = true
		}
	}
	if desiredValue != "" {
		if _, ok := valueSet[desiredValue]; !ok {
			values = append(values, desiredValue)
		}
	}
	return matches, values, hasDesired
}

func filterChallengeMatches(records []dns.DNSRecord, desired dns.DNSRecord) []dns.DNSRecord {
	if len(records) == 0 {
		return nil
	}
	matches := make([]dns.DNSRecord, 0, len(records))
	for _, record := range records {
		if !matchChallengeRecord(record, desired) {
			continue
		}
		matches = append(matches, record)
	}
	return matches
}

func matchChallengeRecord(record dns.DNSRecord, desired dns.DNSRecord) bool {
	if !strings.EqualFold(strings.TrimSpace(record.Type), strings.TrimSpace(desired.Type)) {
		return false
	}
	if strings.TrimSpace(record.Name) != strings.TrimSpace(desired.Name) {
		return false
	}
	if strings.TrimSpace(desired.Line) != "" && strings.TrimSpace(record.Line) != strings.TrimSpace(desired.Line) {
		return false
	}
	return true
}
