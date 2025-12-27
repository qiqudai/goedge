package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
)

type DNSLAConfig struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type DNSLAProvider struct {
	Config DNSLAConfig
}

func NewDNSLAProvider(credentials string) (dns.Provider, error) {
	var config DNSLAConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	return &DNSLAProvider{Config: config}, nil
}

func (p *DNSLAProvider) GetDomains() ([]string, error) {
	if p.Config.ID == "" || p.Config.Secret == "" {
		return nil, errors.New("invalid dnsla credentials")
	}
	return []string{"dnsla-example.com"}, nil
}

func (p *DNSLAProvider) AddRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[DNSLA] Adding record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func (p *DNSLAProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[DNSLA] Deleting record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func init() {
	dns.RegisterProvider("dnsla", NewDNSLAProvider)
}
