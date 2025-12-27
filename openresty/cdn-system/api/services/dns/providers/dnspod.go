package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
)

type DNSPodConfig struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type DNSPodProvider struct {
	Config DNSPodConfig
}

func NewDNSPodProvider(credentials string) (dns.Provider, error) {
	var config DNSPodConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	return &DNSPodProvider{Config: config}, nil
}

func (p *DNSPodProvider) GetDomains() ([]string, error) {
	if p.Config.ID == "" || p.Config.Token == "" {
		return nil, errors.New("invalid dnspod credentials")
	}
	return []string{"dnspod-example.com"}, nil
}

func (p *DNSPodProvider) AddRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[DNSPod] Adding record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func (p *DNSPodProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[DNSPod] Deleting record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func init() {
	dns.RegisterProvider("dnspod", NewDNSPodProvider)
	dns.RegisterProvider("dnspod_intl", NewDNSPodProvider)
}
