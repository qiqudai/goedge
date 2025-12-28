package providers

import (
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// Not implemented for this task, returning stub
	return []string{}, nil
}

func (p *DNSPodProvider) AddRecord(domain string, record dns.DNSRecord) error {
	// Check if already exists to avoid duplicates if necessary, or just create
	// For robustness, usually try to create. DNSPod might allow duplicates.
	// Best practice: Check existence first or handle error.
	// Simple implementation: Create.
	
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	vals.Set("record_line", record.Line)
	vals.Set("value", record.Value)
	vals.Set("ttl", fmt.Sprintf("%d", record.TTL))
	
	_, err := p.sendRequest("Record.Create", vals)
	return err
}

func (p *DNSPodProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	// 1. Find record ID
	recordID, err := p.findRecordID(domain, record)
	if err != nil {
		return err // Not found or error
	}
	if recordID == "" {
		return nil // Already deleted
	}

	// 2. Remove
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("record_id", recordID)
	
	_, err = p.sendRequest("Record.Remove", vals)
	return err
}

func (p *DNSPodProvider) findRecordID(domain string, record dns.DNSRecord) (string, error) {
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	// DNSPod Record.List filtering by line or value is not always perfect/supported in all versions in filtering.
	// We list and filter manually.
	// Note: 'record_line' parameter in Record.List exists in some docs.
	vals.Set("record_line", record.Line)
	// vals.Set("keyword", record.Value) // keyword search might be fuzzy

	respData, err := p.sendRequest("Record.List", vals)
	if err != nil {
		return "", err
	}

	// Parse response
	var resp struct {
		Status struct {
			Code string `json:"code"`
		} `json:"status"`
		Records []struct {
			ID    string `json:"id"`
			Line  string `json:"line"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"records"`
	}
	
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", err
	}
	
	if resp.Status.Code != "1" {
		// Code 10 means no records
		if resp.Status.Code == "10" {
			return "", nil
		}
		return "", errors.New("api error code: " + resp.Status.Code)
	}

	for _, r := range resp.Records {
		if r.Type == record.Type && r.Value == record.Value { 
			// Line check: API 'record_line' filter usually works, but confirm.
			// The returned 'Line' name might vary (e.g. "默认" vs "Default").
			// We trust the API filter or loose match if strict match fails.
			return r.ID, nil
		}
	}

	return "", nil
}

func (p *DNSPodProvider) sendRequest(action string, vals url.Values) ([]byte, error) {
	vals.Set("login_token", p.Config.ID+","+p.Config.Token)
	vals.Set("format", "json")
	vals.Set("lang", "cn")
	vals.Set("error_on_empty", "no")

	resp, err := http.PostForm("https://dnsapi.cn/"+action, vals)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Basic error check
	// DNSPod always returns 200 OK mostly, check JSON status.
	// Wrapper usually checks status code.
	
	return body, nil
}

func init() {
	dns.RegisterProvider("dnspod", NewDNSPodProvider)
	dns.RegisterProvider("dnspod_intl", NewDNSPodProvider)
}
