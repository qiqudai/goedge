package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
)

type HuaweiConfig struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type HuaweiProvider struct {
	Config HuaweiConfig
}

func NewHuaweiProvider(credentials string) (dns.Provider, error) {
	var config HuaweiConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	return &HuaweiProvider{Config: config}, nil
}

func (p *HuaweiProvider) GetDomains() ([]string, error) {
	if p.Config.ID == "" || p.Config.Secret == "" {
		return nil, errors.New("invalid huawei credentials")
	}
	return []string{"huawei-example.com"}, nil
}

func (p *HuaweiProvider) AddRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[Huawei] Adding record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func (p *HuaweiProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[Huawei] Deleting record: %s %s -> %s line=%s\n", record.Type, record.Name, record.Value, record.Line)
	return nil
}

func init() {
	dns.RegisterProvider("huawei", NewHuaweiProvider)
}
