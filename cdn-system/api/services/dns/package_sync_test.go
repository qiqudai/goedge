package dns

import (
	"errors"
	"testing"
)

type packageCNAMEProvider struct {
	records []DNSRecord
	adds    int
	deletes int
}

func (p *packageCNAMEProvider) GetDomains() ([]string, error) {
	return []string{"311779.cc"}, nil
}

func (p *packageCNAMEProvider) GetRecords(domain string) ([]DNSRecord, error) {
	out := make([]DNSRecord, len(p.records))
	copy(out, p.records)
	return out, nil
}

func (p *packageCNAMEProvider) AddRecord(domain string, record DNSRecord) error {
	if record.Value == "" {
		return errors.New("empty value")
	}
	p.adds++
	p.records = append(p.records, record)
	return nil
}

func (p *packageCNAMEProvider) DeleteRecord(domain string, record DNSRecord) error {
	p.deletes++
	next := make([]DNSRecord, 0, len(p.records))
	removed := false
	for _, item := range p.records {
		if !removed && item.Type == record.Type && item.Name == record.Name && item.Value == record.Value {
			removed = true
			continue
		}
		next = append(next, item)
	}
	p.records = next
	return nil
}

func TestReconcileLineRecordSetKeepsEquivalentCNAMEWithTrailingDot(t *testing.T) {
	provider := &packageCNAMEProvider{records: []DNSRecord{{
		Type:   "CNAME",
		Name:   "8klh0jkn",
		Value:  "uxt9f6bk.311779.cc.",
		Line:   "Default",
		TTL:    100,
		Status: "ENABLE",
	}}}

	record := DNSRecord{Type: "CNAME", Name: "8klh0jkn", Line: "默认", TTL: 100}
	if err := ReconcileLineRecordSet(provider, "311779.cc", record, []string{"uxt9f6bk.311779.cc"}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if provider.adds != 0 || provider.deletes != 0 {
		t.Fatalf("equivalent cname must not be rebuilt, adds=%d deletes=%d", provider.adds, provider.deletes)
	}
}

func TestVerifyPackageCNAMERecordRequiresActiveRecord(t *testing.T) {
	provider := &packageCNAMEProvider{records: []DNSRecord{{
		Type:   "CNAME",
		Name:   "8klh0jkn",
		Value:  "uxt9f6bk.311779.cc.",
		Line:   "Default",
		Status: "DISABLE",
	}}}
	err := verifyPackageCNAMERecord(provider, "311779.cc", "8klh0jkn", "默认", "uxt9f6bk.311779.cc")
	if err == nil {
		t.Fatalf("disabled CNAME record must not pass verification")
	}
}

func TestVerifyPackageCNAMERecordAcceptsActiveDefaultLineAliases(t *testing.T) {
	provider := &packageCNAMEProvider{records: []DNSRecord{{
		Type:   "CNAME",
		Name:   "8klh0jkn",
		Value:  "uxt9f6bk.311779.cc.",
		Line:   "Default",
		Status: "ENABLE",
	}}}
	if err := verifyPackageCNAMERecord(provider, "311779.cc", "8klh0jkn", "默认", "uxt9f6bk.311779.cc"); err != nil {
		t.Fatalf("active CNAME should pass verification: %v", err)
	}
}

func TestReconcileLineRecordSetTreatsDisabledDesiredRecordAsMissing(t *testing.T) {
	provider := &packageCNAMEProvider{records: []DNSRecord{{
		Type:   "CNAME",
		Name:   "8klh0jkn",
		Value:  "uxt9f6bk.311779.cc.",
		Line:   "Default",
		Status: "DISABLE",
	}}}

	record := DNSRecord{Type: "CNAME", Name: "8klh0jkn", Line: "Default", TTL: 100}
	if err := ReconcileLineRecordSet(provider, "311779.cc", record, []string{"uxt9f6bk.311779.cc."}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if provider.adds != 1 {
		t.Fatalf("disabled desired record should be re-added/enabled, adds=%d", provider.adds)
	}
}
