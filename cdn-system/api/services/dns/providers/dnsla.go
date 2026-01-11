package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type DNSLAConfig struct {
	ID     string `json:"api_id"`
	Secret string `json:"api_pass"`
}

type DNSLAProvider struct {
	Config DNSLAConfig
}

func NewDNSLAProvider(credentials string) (dns.Provider, error) {
	var config DNSLAConfig
	err := json.Unmarshal([]byte(credentials), &config)
	if err != nil {
		return nil, err
	}
	return &DNSLAProvider{Config: config}, nil
}

func (p *DNSLAProvider) GetDomains() ([]string, error) {
	return []string{}, nil
}

func (p *DNSLAProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	vals := url.Values{}
	vals.Set("apiid", p.Config.ID)
	vals.Set("apipass", p.Config.Secret)
	vals.Set("domain", domain)

	respBody, err := p.post("api/recordList", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code int    `json:"code"`
		Data []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Value string `json:"data"`
			Line  string `json:"line"`
			Name  string `json:"host"`
			TTL   int    `json:"ttl"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	// DNSLA code 200 = success, 300 = success? Need to verify. Based on existing code 200 checks.
	if resp.Code != 200 {
		return nil, nil // Return empty if failed or no records? Better safe to return error if not 200? 
		// Existing findRecordID ignores error? No it returns err.
		// Let's assume 200 is strict success.
	}

	var results []dns.DNSRecord
	for _, r := range resp.Data {
		results = append(results, dns.DNSRecord{
			Type:  r.Type,
			Name:  r.Name,
			Value: r.Value,
			Line:  r.Line,
			TTL:   r.TTL,
		})
	}
	return results, nil
}

func (p *DNSLAProvider) AddRecord(domain string, record dns.DNSRecord) error {
	// DNS.LA Add Record
	// API: https://api.dns.la/api/recordCreate
	
	vals := url.Values{}
	vals.Set("apiid", p.Config.ID)
	vals.Set("apipass", p.Config.Secret)
	vals.Set("domain", domain) 
	vals.Set("host", record.Name)
	vals.Set("recordType", record.Type)
	vals.Set("recordLine", record.Line)
	vals.Set("recordValue", record.Value)
	vals.Set("ttl", fmt.Sprintf("%d", record.TTL))

	respBody, err := p.post("api/recordCreate", vals)
	if err != nil {
		return err
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"` // Usually 'code': 200 is success
	}
	// DNS.LA might return int or string code, assuming int 200 based on standard
	// If unmarshal fails on Code, try struct with string Code
	
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 200 {
		// 309 = Record Exists
		if resp.Code == 309 {
			return nil
		}
		return fmt.Errorf("dnsla error: %d - %s", resp.Code, resp.Msg)
	}

	return nil
}

func (p *DNSLAProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	// 1. Get List to find ID
	recordID, err := p.findRecordID(domain, record)
	if err != nil {
		return err
	}
	if recordID == "" {
		return nil
	}

	// 2. Remove
	vals := url.Values{}
	vals.Set("apiid", p.Config.ID)
	vals.Set("apipass", p.Config.Secret)
	vals.Set("domain", domain)
	vals.Set("recordId", recordID)

	respBody, err := p.post("api/recordRemove", vals)
	if err != nil {
		return err
	}
	
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(respBody, &resp)
	if resp.Code != 200 {
		return fmt.Errorf("dnsla error: %d - %s", resp.Code, resp.Msg)
	}

	return nil
}

func (p *DNSLAProvider) findRecordID(domain string, record dns.DNSRecord) (string, error) {
	vals := url.Values{}
	vals.Set("apiid", p.Config.ID)
	vals.Set("apipass", p.Config.Secret)
	vals.Set("domain", domain)
	vals.Set("host", record.Name)

	respBody, err := p.post("api/recordList", vals)
	if err != nil {
		return "", err
	}

	var resp struct {
		Code int    `json:"code"`
		Data []struct {
			ID    string `json:"id"` // or int? usually string ID in json
			Type  string `json:"type"`
			Value string `json:"data"` // field name might be 'data' or 'value'
			Line  string `json:"line"`
		} `json:"data"`
	}
	
	// Try standard unmarshal
	if err := json.Unmarshal(respBody, &resp); err != nil {
		// If ID is int, standard json decoder handles string->int conversion if unquoting?
		// But let's assume string for ID.
		return "", err
	}
	
	for _, r := range resp.Data {
		if r.Type == record.Type && r.Value == record.Value {
			return r.ID, nil
		}
	}
	return "", nil
}

func (p *DNSLAProvider) post(endpoint string, vals url.Values) ([]byte, error) {
	api := "https://api.dns.la/" + endpoint
	resp, err := http.PostForm(api, vals)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func init() {
	dns.RegisterProvider("dnsla", NewDNSLAProvider)
}
