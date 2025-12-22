package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
)

type AliyunConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
}

type AliyunProvider struct {
	Config AliyunConfig
}

func NewAliyunProvider(credentials string) (dns.Provider, error) {
	var config AliyunConfig
	err := json.Unmarshal([]byte(credentials), &config)
	if err != nil {
		return nil, err
	}
	return &AliyunProvider{Config: config}, nil
}

func (p *AliyunProvider) GetDomains() ([]string, error) {
	if p.Config.AccessKeyID == "" {
		return nil, errors.New("invalid access key")
	}
	return []string{"aliyun-example.com"}, nil
}

func (p *AliyunProvider) AddRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[Aliyun] Adding record: %s %s -> %s\n", record.Type, record.Name, record.Value)
	return nil
}

func (p *AliyunProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	fmt.Printf("[Aliyun] Deleting record: %s %s -> %s\n", record.Type, record.Name, record.Value)
	return nil
}

func init() {
	dns.RegisterProvider("aliyun", NewAliyunProvider)
}
