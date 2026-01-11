package providers

import (
	"bytes"
	"cdn-api/services/dns"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
	return []string{}, nil
}

func (p *CloudFlareProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	return []dns.DNSRecord{}, nil // Not implemented
}

func (p *CloudFlareProvider) getZoneID(domain string) (string, error) {
	// Simple domain validation and extracting search domain
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid domain: %s", domain)
	}

	// First try exact match
	zoneID, err := p.lookupZoneID(domain)
	if err == nil && zoneID != "" {
		return zoneID, nil
	}

	// Try parent domain (e.g. searching for 'sub.example.com', if not found, try 'example.com')
	// This is a common strategy as Zone might be 'example.com'
	if len(parts) > 2 {
		searchDomain := strings.Join(parts[len(parts)-2:], ".")
		zoneID, err = p.lookupZoneID(searchDomain)
		if err == nil && zoneID != "" {
			return zoneID, nil
		}
	}
	
	return "", fmt.Errorf("zone not found for domain: %s", domain)
}

func (p *CloudFlareProvider) lookupZoneID(domain string) (string, error) {
	api := "https://api.cloudflare.com/client/v4/zones?name=" + domain
	req, _ := http.NewRequest("GET", api, nil)
	p.setHeaders(req)
	
	respBody, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	var resp struct {
		Success bool `json:"success"`
		Result  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}
	
	if !resp.Success {
		msg := "unknown error"
		if len(resp.Errors) > 0 {
			msg = resp.Errors[0].Message
		}
		return "", errors.New(msg)
	}
	
	if len(resp.Result) == 0 {
		return "", nil // Not found
	}
	
	return resp.Result[0].ID, nil
}

func (p *CloudFlareProvider) AddRecord(domain string, record dns.DNSRecord) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	// CloudFlare expects 'name' to be the FQDN usually
	fullRecordName := record.Name
	if record.Name == "@" || record.Name == "" {
		fullRecordName = domain
	} else if !strings.HasSuffix(record.Name, domain) {
		fullRecordName = record.Name + "." + domain
	}

	// Construct payload
	payload := map[string]interface{}{
		"type":    record.Type,
		"name":    fullRecordName,
		"content": record.Value,
		"ttl":     record.TTL,
		"proxied": false, // Default to DNS only for flexibility
	}
	if record.TTL == 0 {
		payload["ttl"] = 1 // Auto
	}

	body, _ := json.Marshal(payload)
	api := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	req, _ := http.NewRequest("POST", api, bytes.NewBuffer(body))
	p.setHeaders(req)

	respBody, err := p.doRequest(req)
	if err != nil {
		return err
	}
	
	var resp struct {
		Success bool `json:"success"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(respBody, &resp) // safe to ignore error here as we check success
	
	if !resp.Success {
		// Ignore if record already exists
		msg := ""
		if len(resp.Errors) > 0 {
			msg = resp.Errors[0].Message
		}
		if strings.Contains(strings.ToLower(msg), "already exists") {
			return nil
		}
		return fmt.Errorf("cloudflare error: %s", msg)
	}
	
	return nil
}

func (p *CloudFlareProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	fullRecordName := record.Name
	if record.Name == "@" || record.Name == "" {
		fullRecordName = domain
	} else if !strings.HasSuffix(record.Name, domain) {
		fullRecordName = record.Name + "." + domain
	}

	// List records to find ID
	api := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s&content=%s", 
		zoneID, record.Type, fullRecordName, record.Value)
	
	req, _ := http.NewRequest("GET", api, nil)
	p.setHeaders(req)
	respBody, err := p.doRequest(req)
	if err != nil {
		return err
	}

	var listResp struct {
		Success bool `json:"success"`
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return err
	}
	
	// Delete each found record
	for _, rec := range listResp.Result {
		delApi := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, rec.ID)
		delReq, _ := http.NewRequest("DELETE", delApi, nil)
		p.setHeaders(delReq)
		_, _ = p.doRequest(delReq)
	}

	return nil
}

func (p *CloudFlareProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.Config.Email != "" {
		req.Header.Set("X-Auth-Email", p.Config.Email)
		req.Header.Set("X-Auth-Key", p.Config.ApiKey)
	} else {
		// If Email is empty, treat ApiKey as Bearer Token (API Token)
		req.Header.Set("Authorization", "Bearer " + p.Config.ApiKey)
	}
}

func (p *CloudFlareProvider) doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func init() {
	dns.RegisterProvider("cloudflare", NewCloudFlareProvider)
}
