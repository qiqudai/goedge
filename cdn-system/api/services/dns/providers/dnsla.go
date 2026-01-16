package providers

import (
	"bytes"
	"cdn-api/services/dns"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	dnslaBaseURL    = "https://api.dns.la"
	dnslaTimeout    = 30 * time.Second
	dnslaPageSize   = 200
	dnslaDefaultTTL = 600
)

var dnslaTypeNameToCode = map[string]int{
	"A":     1,
	"NS":    2,
	"CNAME": 5,
	"SOA":   6,
	"PTR":   12,
	"MX":    15,
	"TXT":   16,
	"AAAA":  28,
	"SRV":   33,
	"NAPTR": 35,
	"SPF":   99,
	"SVCB":  64,
	"HTTPS": 65,
	"CAA":   257,
}

var dnslaTypeCodeToName = map[int]string{}

func init() {
	for name, code := range dnslaTypeNameToCode {
		dnslaTypeCodeToName[code] = name
	}
	dns.RegisterProvider("dnsla", NewDNSLAProvider)
}

type DNSLAConfig struct {
	ID     string `json:"api_id"`
	Secret string `json:"api_pass"`
}

type DNSLAProvider struct {
	Config DNSLAConfig
	token  string
	client *http.Client
}

type dnslaRecord struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Type     int    `json:"type"`
	Data     string `json:"data"`
	TTL      int    `json:"ttl"`
	LineID   string `json:"lineId"`
	LineCode string `json:"lineCode"`
	LineName string `json:"lineName"`
}

type dnslaRecordListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total   int           `json:"total"`
		Results []dnslaRecord `json:"results"`
	} `json:"data"`
}

type dnslaCreateRecordResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type dnslaCommonResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func NewDNSLAProvider(credentials string) (dns.Provider, error) {
	var config DNSLAConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.ID) == "" || strings.TrimSpace(config.Secret) == "" {
		var legacy struct {
			ID     string `json:"id"`
			Secret string `json:"secret"`
		}
		if legacyErr := json.Unmarshal([]byte(credentials), &legacy); legacyErr == nil {
			if strings.TrimSpace(config.ID) == "" {
				config.ID = legacy.ID
			}
			if strings.TrimSpace(config.Secret) == "" {
				config.Secret = legacy.Secret
			}
		}
	}
	config.ID = strings.TrimSpace(config.ID)
	config.Secret = strings.TrimSpace(config.Secret)
	if config.ID == "" || config.Secret == "" {
		return nil, errors.New("dnsla credentials missing api_id/api_pass")
	}

	token := base64.StdEncoding.EncodeToString([]byte(config.ID + ":" + config.Secret))
	return &DNSLAProvider{
		Config: config,
		token:  token,
		client: dns.NewHTTPClient(dnslaTimeout),
	}, nil
}

func (p *DNSLAProvider) GetDomains() ([]string, error) {
	return []string{}, nil
}

func (p *DNSLAProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	domain = normalizeDNSLADomain(domain)
	if domain == "" {
		return nil, errors.New("dnsla domain required")
	}

	pageIndex := 1
	pageSize := dnslaPageSize
	var records []dns.DNSRecord

	for {
		resp, err := p.listRecords(domain, pageIndex, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Data.Results {
			line := strings.TrimSpace(item.LineCode)
			if line == "" {
				line = strings.TrimSpace(item.LineName)
			}
			records = append(records, dns.DNSRecord{
				Type:  dnslaTypeName(item.Type),
				Name:  normalizeDNSLARecordName(domain, item.Host),
				Value: strings.TrimSpace(item.Data),
				TTL:   item.TTL,
				Line:  line,
			})
		}

		if len(resp.Data.Results) == 0 {
			break
		}
		if resp.Data.Total > 0 && len(records) >= resp.Data.Total {
			break
		}
		if len(resp.Data.Results) < pageSize {
			break
		}
		pageIndex++
	}

	return records, nil
}

func (p *DNSLAProvider) AddRecord(domain string, record dns.DNSRecord) error {
	domain = normalizeDNSLADomain(domain)
	if domain == "" {
		return errors.New("dnsla domain required")
	}

	typeCode, ok := dnslaTypeCode(record.Type)
	if !ok {
		return fmt.Errorf("dnsla unsupported record type: %s", record.Type)
	}

	host := normalizeDNSLARecordName(domain, record.Name)
	if host == "" {
		host = "@"
	}

	ttl := record.TTL
	if ttl <= 0 {
		ttl = dnslaDefaultTTL
	}

	payload := map[string]any{
		"domain": domain,
		"host":   host,
		"type":   typeCode,
		"data":   strings.TrimSpace(record.Value),
		"ttl":    ttl,
	}
	if strings.TrimSpace(record.Line) != "" {
		payload["lineCode"] = strings.TrimSpace(record.Line)
		payload["lineId"] = strings.TrimSpace(record.Line)
	}
	if record.Weight > 0 {
		payload["weight"] = record.Weight
	}

	body, err := p.doRequest(http.MethodPost, "/api/record", nil, payload)
	if err != nil {
		return err
	}

	var resp dnslaCreateRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("dnsla add record decode failed: %w (body=%s)", err, string(body))
	}
	if resp.Code != 200 {
		msg := strings.TrimSpace(resp.Msg)
		if msg != "" {
			lower := strings.ToLower(msg)
			if strings.Contains(msg, "冲突") || strings.Contains(msg, "已存在") || strings.Contains(lower, "exists") {
				return nil
			}
		}
		return fmt.Errorf("dnsla add record error: %d - %s", resp.Code, resp.Msg)
	}
	return nil
}

func (p *DNSLAProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	domain = normalizeDNSLADomain(domain)
	if domain == "" {
		return errors.New("dnsla domain required")
	}

	ids, err := p.findRecordIDs(domain, record)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		query := url.Values{}
		query.Set("id", id)
		body, err := p.doRequest(http.MethodDelete, "/api/record", query, nil)
		if err != nil {
			return err
		}
		var resp dnslaCommonResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("dnsla delete record decode failed: %w (body=%s)", err, string(body))
		}
		if resp.Code != 200 {
			return fmt.Errorf("dnsla delete record error: %d - %s", resp.Code, resp.Msg)
		}
	}

	return nil
}

func (p *DNSLAProvider) listRecords(domain string, pageIndex int, pageSize int) (*dnslaRecordListResponse, error) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = dnslaPageSize
	}

	query := url.Values{}
	query.Set("domain", domain)
	query.Set("pageIndex", strconv.Itoa(pageIndex))
	query.Set("pageSize", strconv.Itoa(pageSize))

	body, err := p.doRequest(http.MethodGet, "/api/recordList", query, nil)
	if err != nil {
		return nil, err
	}

	var resp dnslaRecordListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("dnsla recordList decode failed: %w (body=%s)", err, string(body))
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("dnsla recordList error: %d - %s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

func (p *DNSLAProvider) findRecordIDs(domain string, record dns.DNSRecord) ([]string, error) {
	typeCode, ok := dnslaTypeCode(record.Type)
	if !ok {
		return nil, fmt.Errorf("dnsla unsupported record type: %s", record.Type)
	}
	desiredName := normalizeDNSLARecordName(domain, record.Name)
	desiredValue := strings.TrimSpace(record.Value)
	desiredLine := strings.TrimSpace(record.Line)

	pageIndex := 1
	pageSize := dnslaPageSize
	var matches []string

	for {
		resp, err := p.listRecords(domain, pageIndex, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Data.Results {
			if normalizeDNSLARecordName(domain, item.Host) != desiredName {
				continue
			}
			if item.Type != typeCode {
				continue
			}
			if desiredValue != "" && strings.TrimSpace(item.Data) != desiredValue {
				continue
			}
			if desiredLine != "" && desiredLine != strings.TrimSpace(item.LineCode) && desiredLine != strings.TrimSpace(item.LineName) {
				continue
			}
			if item.ID != "" {
				matches = append(matches, item.ID)
			}
		}
		if len(resp.Data.Results) == 0 {
			break
		}
		if resp.Data.Total > 0 && pageIndex*pageSize >= resp.Data.Total {
			break
		}
		if len(resp.Data.Results) < pageSize {
			break
		}
		pageIndex++
	}

	return matches, nil
}

func (p *DNSLAProvider) doRequest(method string, path string, query url.Values, payload any) ([]byte, error) {
	if p.client == nil {
		p.client = dns.NewHTTPClient(dnslaTimeout)
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	base, err := url.Parse(dnslaBaseURL)
	if err != nil {
		return nil, err
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + path
	if query != nil {
		base.RawQuery = query.Encode()
	}

	var reader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, base.String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+p.token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("dnsla empty response, status=%s", resp.Status)
	}
	return body, nil
}

func normalizeDNSLADomain(domain string) string {
	return strings.TrimSuffix(strings.TrimSpace(domain), ".")
}

func normalizeDNSLARecordName(domain string, name string) string {
	domain = normalizeDNSLADomain(domain)
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	if name == "" {
		return "@"
	}
	if domain != "" {
		if name == domain {
			return "@"
		}
		if strings.HasSuffix(name, "."+domain) {
			trimmed := strings.TrimSuffix(name, "."+domain)
			if trimmed == "" {
				return "@"
			}
			return trimmed
		}
	}
	return name
}

func dnslaTypeCode(typeName string) (int, bool) {
	if typeName == "" {
		return 0, false
	}
	code, ok := dnslaTypeNameToCode[strings.ToUpper(strings.TrimSpace(typeName))]
	return code, ok
}

func dnslaTypeName(code int) string {
	if name, ok := dnslaTypeCodeToName[code]; ok {
		return name
	}
	return strconv.Itoa(code)
}
