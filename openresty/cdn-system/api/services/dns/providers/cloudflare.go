package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
)

type CloudFlareConfig struct {
	Email  string `json:"email"`
	ApiKey string `json:"api_key"`
}

type CloudFlareProvider struct {
	Config CloudFlareConfig
}

func NewCloudFlareProvider(credentials string) (dns.Provider, error) {
	var config CloudFlareConfig
	err := json.Unmarshal([]byte(credentials), &config)
	if err != nil {
		return nil, err
	}
	return &CloudFlareProvider{Config: config}, nil
}

func (p *CloudFlareProvider) GetDomains() ([]string, error) {
	// Mock implementation
	if p.Config.ApiKey == "" {
		return nil, errors.New("invalid api key")
	}
	return []string{"example.com", "cloudflare-test.com"}, nil
}

func (p *CloudFlareProvider) AddRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[CloudFlare] Adding record: %s %s -> %s (TTL: %d) line=%s\n", record.Type, record.Name, record.Value, record.TTL, record.Line)
	return nil
}

func (p *CloudFlareProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[CloudFlare] Deleting record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func init() {
	dns.RegisterProvider("cloudflare", NewCloudFlareProvider)
}
