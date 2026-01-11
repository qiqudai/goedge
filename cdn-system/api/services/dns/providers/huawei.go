package providers

import (
	"bytes"
	"cdn-api/services/dns"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type HuaweiConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"` // Optional, default cn-north-1
}

type HuaweiProvider struct {
	Config HuaweiConfig
}

func NewHuaweiProvider(credentials string) (dns.Provider, error) {
	var config HuaweiConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	if config.Region == "" {
		config.Region = "cn-north-1"
	}
	return &HuaweiProvider{Config: config}, nil
}

func (p *HuaweiProvider) GetDomains() ([]string, error) {
	return []string{}, nil
}

func (p *HuaweiProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	return []dns.DNSRecord{}, nil // Not implemented
}

func (p *HuaweiProvider) AddRecord(domain string, record dns.DNSRecord) error {
	// 1. Get Zone ID
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	// 2. Create Record Set
	// POST /v2.1/zones/{zone_id}/recordsets
	url := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2.1/zones/%s/recordsets", p.Config.Region, zoneID)
	
	payload := map[string]interface{}{
		"name":        record.Name + "." + domain + ".",
		"type":        record.Type,
		"ttl":         record.TTL,
		"description": "Created by CDN",
		"records":     []string{record.Value},
	}
	if record.Name == "@" {
		payload["name"] = domain + "."
	}
	if record.TTL == 0 {
		payload["ttl"] = 300
	}

	body, _ := json.Marshal(payload)
	resp, err := p.sendRequest("POST", url, body)
	if err != nil {
		return err
	}

	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		ID      string `json:"id"`
	}
	_ = json.Unmarshal(resp, &parsed)
	
	// Huawei returns 2xx on success. If error, body usually contains code/message
	// Need to check for duplicate (idempotency)
	if parsed.Code != "" {
		if strings.Contains(parsed.Code, "Duplicate") || strings.Contains(parsed.Message, "already exists") {
			return nil
		}
		return fmt.Errorf("huawei error: %s - %s", parsed.Code, parsed.Message)
	}

	return nil
}

func (p *HuaweiProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	// 1. Get Zone ID
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	// 2. List Record Sets to find ID
	// GET /v2.1/zones/{zone_id}/recordsets?name=...&type=...
	name := record.Name + "." + domain + "."
	if record.Name == "@" {
		name = domain + "."
	}
	
	listUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2.1/zones/%s/recordsets?name=%s&type=%s", 
		p.Config.Region, zoneID, name, record.Type)
	
	resp, err := p.sendRequest("GET", listUrl, nil)
	if err != nil {
		return err
	}

	var listResp struct {
		Recordsets []struct {
			ID      string   `json:"id"`
			Name    string   `json:"name"`
			Records []string `json:"records"`
		} `json:"recordsets"`
	}
	if err := json.Unmarshal(resp, &listResp); err != nil {
		return err
	}

	// Find exact match for value?
	// Huawei groups records in a recordset. If we delete, we delete the WHOLE recordset (all values).
	// Safest is to find the recordset, allow deletion if it contains our value.
	// But if it contains OTHER values, we should UPDATE it to remove just ours.
	
	for _, rs := range listResp.Recordsets {
		// Found the recordset
		contains := false
		newRecords := []string{}
		for _, v := range rs.Records {
			if v == record.Value {
				contains = true
			} else {
				newRecords = append(newRecords, v)
			}
		}

		if contains {
			if len(newRecords) == 0 {
				// Delete entire recordset
				delUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2.1/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
				delResp, err := p.sendRequest("DELETE", delUrl, nil)
				if err != nil {
					return err
				}
				// Check error
				var delParsed struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				_ = json.Unmarshal(delResp, &delParsed)
				if delParsed.Code != "" {
					return fmt.Errorf("huawei delete error: %s - %s", delParsed.Code, delParsed.Message)
				}
			} else {
				// Update recordset (remove one value)
				// PUT /v2.1/zones/{zone_id}/recordsets/{recordset_id}
				updateUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2.1/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
				payload := map[string]interface{}{
					"name":    rs.Name,
					"type":    record.Type,
					"records": newRecords,
				}
				body, _ := json.Marshal(payload)
				_, err := p.sendRequest("PUT", updateUrl, body)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *HuaweiProvider) getZoneID(domain string) (string, error) {
	// GET /v2.1/zones?name={domain}
	// Note: trailing dot often needed in DB, but query might not?
	url := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2.1/zones?name=%s", p.Config.Region, domain)
	resp, err := p.sendRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Zones []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return "", err
	}

	for _, z := range parsed.Zones {
		// Huawei returns name with dot, e.g. "example.com."
		if strings.TrimSuffix(z.Name, ".") == strings.TrimSuffix(domain, ".") {
			return z.ID, nil
		}
	}
	return "", fmt.Errorf("zone not found for domain: %s", domain)
}

func (p *HuaweiProvider) sendRequest(method, urlStr string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	
	// Headers
	req.Header.Set("content-type", "application/json")
	// Host is set automatically by Go but required for signing
	u, _ := url.Parse(urlStr)
	req.Header.Set("host", u.Host)
	
	// Sign
	p.sign(req, body)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *HuaweiProvider) sign(req *http.Request, body []byte) {
	// SDK-HMAC-SHA256
	const (
		Algorithm    = "SDK-HMAC-SHA256"
		HeaderPrefix = "SDK-HMAC-SHA256"
		Terminator   = "sdk_request"
	)
	
	t := time.Now().UTC()
	xSdkDate := t.Format("20060102T150405Z")
	date := t.Format("20060102")
	
	req.Header.Set("X-Sdk-Date", xSdkDate)
	
	// 1. Canonical Request
	// Method
	canonicalRequest := req.Method + "\n"
	
	// URI (Path)
	// Must be normalized. Go's req.URL.Path should be sufficient if simple.
	uri := req.URL.Path
	if uri == "" {
		uri = "/"
	}
	if !strings.HasSuffix(uri, "/") && strings.Count(uri, "/") == 0 {
		uri += "/"
	}
	// Note: Huawei requires careful normalization, assuming simple paths here
	canonicalRequest += uri + "\n"
	
	// Query
	// Must be sorted
	q := req.URL.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	encodedKeys := []string{}
	for _, k := range keys {
		// Keys/Values must be url encoded (path escaped?)
		// Huawei spec: UriEncode()
		// standard url.Query uses safe encoding
		vals := q[k]
		sort.Strings(vals) // Spec requires sorting values too? Usually yes
		for _, v := range vals {
			encodedKeys = append(encodedKeys, k+"="+url.QueryEscape(v))
		}
	}
	canonicalRequest += strings.Join(encodedKeys, "&") + "\n"

	// Headers
	// Lowercase, sorted
	// We MUST include Host and X-Sdk-Date
	signedHeaders := []string{"content-type", "host", "x-sdk-date"}
	sort.Strings(signedHeaders)
	
	canonicalHeaders := ""
	for _, h := range signedHeaders {
		canonicalHeaders += h + ":" + strings.TrimSpace(req.Header.Get(h)) + "\n"
	}
	canonicalRequest += canonicalHeaders + "\n"
	
	// Signed Headers
	canonicalRequest += strings.Join(signedHeaders, ";") + "\n"
	
	// Payload Hash
	payloadHash := huaweiSha256Hex(body)
	canonicalRequest += payloadHash
	
	// 2. String to Sign
	credentialScope := date + "/" + p.Config.Region + "/dns/" + Terminator
	stringToSign := Algorithm + "\n" + xSdkDate + "\n" + credentialScope + "\n" + huaweiSha256Hex([]byte(canonicalRequest))
	
	// 3. Signature
	// kSecret = "SDK" + SecretKey
	// kDate = HMAC(kSecret, Date)
	// kRegion = HMAC(kDate, Region)
	// kService = HMAC(kRegion, Service)
	// kSigning = HMAC(kService, Terminator)
	
	kSecret := []byte("SDK" + p.Config.SecretAccessKey)
	kDate := huaweiHmacSHA256(kSecret, date)
	kRegion := huaweiHmacSHA256(kDate, p.Config.Region)
	kService := huaweiHmacSHA256(kRegion, "dns")
	kSigning := huaweiHmacSHA256(kService, Terminator)
	
	signature := hex.EncodeToString(huaweiHmacSHA256(kSigning, stringToSign))
	
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		HeaderPrefix, p.Config.AccessKeyID, credentialScope, strings.Join(signedHeaders, ";"), signature)
		
	req.Header.Set("Authorization", authHeader)
}

func huaweiSha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func huaweiHmacSHA256(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}

func init() {
	dns.RegisterProvider("huawei", NewHuaweiProvider)
}
