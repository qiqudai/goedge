package providers

import (
	"cdn-api/services/dns"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryProvider is an in-memory DNS provider for tests and local chaos runs.
type MemoryProvider struct {
	store *memoryStore
}

type memoryStore struct {
	mu      sync.Mutex
	records map[string][]dns.DNSRecord
}

var sharedMemoryStore = &memoryStore{records: map[string][]dns.DNSRecord{}}

func NewMemoryProvider(_ string) (dns.Provider, error) {
	return &MemoryProvider{store: sharedMemoryStore}, nil
}

func (p *MemoryProvider) lockRecords() (map[string][]dns.DNSRecord, func()) {
	p.store.mu.Lock()
	return p.store.records, func() { p.store.mu.Unlock() }
}

func (p *MemoryProvider) GetDomains() ([]string, error) {
	records, unlock := p.lockRecords()
	defer unlock()
	domains := make([]string, 0, len(records))
	for domain := range records {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains, nil
}

func (p *MemoryProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	records, unlock := p.lockRecords()
	defer unlock()
	items := records[domain]
	out := make([]dns.DNSRecord, len(items))
	copy(out, items)
	return out, nil
}

func (p *MemoryProvider) AddRecord(domain string, record dns.DNSRecord) error {
	records, unlock := p.lockRecords()
	defer unlock()
	records[domain] = append(records[domain], cloneRecord(record))
	return nil
}

func (p *MemoryProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	records, unlock := p.lockRecords()
	defer unlock()
	items := records[domain]
	next := make([]dns.DNSRecord, 0, len(items))
	removed := false
	for _, item := range items {
		if !removed && recordMatches(item, record) {
			removed = true
			continue
		}
		next = append(next, item)
	}
	records[domain] = next
	return nil
}

func (p *MemoryProvider) DeleteRecordsByLine(domain string, record dns.DNSRecord) error {
	records, unlock := p.lockRecords()
	defer unlock()
	items := records[domain]
	next := make([]dns.DNSRecord, 0, len(items))
	for _, item := range items {
		if item.Type != record.Type || item.Name != record.Name {
			next = append(next, item)
			continue
		}
		if strings.TrimSpace(record.Line) != "" && item.Line != record.Line {
			next = append(next, item)
			continue
		}
	}
	records[domain] = next
	return nil
}

func (p *MemoryProvider) ReplaceRecordValue(domain string, record dns.DNSRecord, newValue string) error {
	records, unlock := p.lockRecords()
	defer unlock()
	items := records[domain]
	for i, item := range items {
		if !recordMatches(item, record) {
			continue
		}
		items[i].Value = newValue
		records[domain] = items
		return nil
	}
	return fmt.Errorf("record not found")
}

func (p *MemoryProvider) UpsertRecordSet(domain string, record dns.DNSRecord, values []string) error {
	return dns.ReconcileLineRecordSet(p, domain, record, values)
}

// LineAValues returns sorted A record values for a hostname and line.
func (p *MemoryProvider) LineAValues(domain, name, line string) []string {
	records, unlock := p.lockRecords()
	defer unlock()
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, item := range records[domain] {
		if item.Type != "A" || item.Name != name {
			continue
		}
		if strings.TrimSpace(line) != "" && item.Line != line {
			continue
		}
		if _, ok := seen[item.Value]; ok {
			continue
		}
		seen[item.Value] = struct{}{}
		out = append(out, item.Value)
	}
	sort.Strings(out)
	return out
}

func ResetMemoryStore() {
	sharedMemoryStore.mu.Lock()
	defer sharedMemoryStore.mu.Unlock()
	sharedMemoryStore.records = map[string][]dns.DNSRecord{}
}

func recordMatches(existing, target dns.DNSRecord) bool {
	if existing.Type != target.Type || existing.Name != target.Name || existing.Value != target.Value {
		return false
	}
	if strings.TrimSpace(target.Line) != "" && existing.Line != target.Line {
		return false
	}
	return true
}

func cloneRecord(record dns.DNSRecord) dns.DNSRecord {
	return dns.DNSRecord{
		Type:   record.Type,
		Name:   record.Name,
		Value:  record.Value,
		TTL:    record.TTL,
		Line:   record.Line,
		Weight: record.Weight,
		Status: record.Status,
	}
}

func init() {
	dns.RegisterProvider("memory", NewMemoryProvider)
}
