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

	"golang.org/x/net/publicsuffix"
)

type HuaweiConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	ID              string `json:"id"`
	Secret          string `json:"secret"`
	Region          string `json:"region"` // Optional, default cn-north-1
}

type HuaweiProvider struct {
	Config         HuaweiConfig
	RegionProvided bool
}

func NewHuaweiProvider(credentials string) (dns.Provider, error) {
	var config HuaweiConfig
	if err := json.Unmarshal([]byte(credentials), &config); err != nil {
		return nil, err
	}
	regionProvided := config.Region != ""
	if config.AccessKeyID == "" && config.ID != "" {
		config.AccessKeyID = config.ID
	}
	if config.SecretAccessKey == "" && config.Secret != "" {
		config.SecretAccessKey = config.Secret
	}
	if config.Region == "" {
		config.Region = "cn-north-1"
	}
	return &HuaweiProvider{Config: config, RegionProvided: regionProvided}, nil
}

func (p *HuaweiProvider) GetDomains() ([]string, error) {
	return []string{}, nil
}

func (p *HuaweiProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	return []dns.DNSRecord{}, nil // Not implemented
}

func (p *HuaweiProvider) UpsertRecordSet(domain string, record dns.DNSRecord, values []string) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	name := record.Name + "." + domain + "."
	if record.Name == "@" {
		name = domain + "."
	}

	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	listUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets?name=%s&type=%s",
		p.Config.Region, zoneID, url.QueryEscape(name), record.Type)
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

	if len(normalized) == 0 {
		for _, rs := range listResp.Recordsets {
			delUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
			if _, err := p.sendRequest("DELETE", delUrl, nil); err != nil {
				return err
			}
		}
		return nil
	}

	ttl := record.TTL
	if ttl == 0 {
		ttl = 300
	}

	if len(listResp.Recordsets) > 0 {
		rs := listResp.Recordsets[0]
		updateUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
		payload := map[string]interface{}{
			"name":    rs.Name,
			"type":    record.Type,
			"ttl":     ttl,
			"records": normalized,
		}
		body, _ := json.Marshal(payload)
		if _, err := p.sendRequest("PUT", updateUrl, body); err != nil {
			return err
		}
		for _, extra := range listResp.Recordsets[1:] {
			delUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets/%s", p.Config.Region, zoneID, extra.ID)
			if _, err := p.sendRequest("DELETE", delUrl, nil); err != nil {
				return err
			}
		}
		return nil
	}

	createUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets", p.Config.Region, zoneID)
	payload := map[string]interface{}{
		"name":        name,
		"type":        record.Type,
		"ttl":         ttl,
		"description": "Created by CDN",
		"records":     normalized,
	}
	body, _ := json.Marshal(payload)
	if _, err := p.sendRequest("POST", createUrl, body); err != nil {
		return err
	}
	return nil
}

func (p *HuaweiProvider) AddRecord(domain string, record dns.DNSRecord) error {
	// 1. Get Zone ID
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}

	// 2. Create Record Set
	// POST /v2/zones/{zone_id}/recordsets
	url := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets", p.Config.Region, zoneID)

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
	// GET /v2/zones/{zone_id}/recordsets?name=...&type=...
	name := record.Name + "." + domain + "."
	if record.Name == "@" {
		name = domain + "."
	}

	listUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets?name=%s&type=%s",
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
				delUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
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
				// PUT /v2/zones/{zone_id}/recordsets/{recordset_id}
				updateUrl := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones/%s/recordsets/%s", p.Config.Region, zoneID, rs.ID)
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
	trimmed := strings.TrimSpace(strings.TrimSuffix(domain, "."))
	if trimmed == "" {
		return "", fmt.Errorf("invalid domain: %s", domain)
	}

	candidates := []string{
		trimmed,
		trimmed + ".",
	}

	if base, err := publicsuffix.EffectiveTLDPlusOne(trimmed); err == nil && base != "" && base != trimmed {
		candidates = append(candidates, base, base+".")
	} else {
		parts := strings.Split(trimmed, ".")
		if len(parts) > 2 {
			base := strings.Join(parts[len(parts)-2:], ".")
			if base != trimmed {
				candidates = append(candidates, base, base+".")
			}
		}
	}

	region := p.Config.Region
	regions := []string{region}
	if !p.RegionProvided {
		for _, r := range []string{"cn-north-4", "cn-east-2", "cn-east-3", "cn-south-1", "cn-south-4"} {
			if r != region {
				regions = append(regions, r)
			}
		}
	}

	for _, r := range regions {
		p.Config.Region = r
		id, err := p.lookupZoneIDCandidates(trimmed, candidates)
		if err != nil {
			return "", err
		}
		if id != "" {
			return id, nil
		}
	}

	p.Config.Region = region
	return "", fmt.Errorf("zone not found for domain: %s", domain)
}

func (p *HuaweiProvider) lookupZoneIDCandidates(domain string, candidates []string) (string, error) {
	seen := map[string]struct{}{}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		zones, err := p.listZones(name)
		if err != nil {
			return "", err
		}
		if id := matchZoneID(zones, domain); id != "" {
			return id, nil
		}
	}

	zones, err := p.listZones("")
	if err != nil {
		return "", err
	}
	if id := matchZoneID(zones, domain); id != "" {
		return id, nil
	}

	return "", nil
}

type huaweiZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type huaweiZonesResp struct {
	Zones     []huaweiZone      `json:"zones"`
	Links     map[string]string `json:"links"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	ErrorCode string            `json:"error_code"`
	ErrorMsg  string            `json:"error_msg"`
}

func (p *HuaweiProvider) listZones(name string) ([]huaweiZone, error) {
	baseURL := fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2/zones", p.Config.Region)
	if name != "" {
		baseURL = baseURL + "?name=" + url.QueryEscape(name)
	}

	zones := []huaweiZone{}
	nextURL := baseURL
	for nextURL != "" {
		resp, err := p.sendRequest("GET", nextURL, nil)
		if err != nil {
			return nil, err
		}
		var parsed huaweiZonesResp
		if err := json.Unmarshal(resp, &parsed); err != nil {
			return nil, err
		}
		if msg := huaweiErrorMessage(parsed.Code, parsed.Message, parsed.ErrorCode, parsed.ErrorMsg); msg != "" {
			return nil, fmt.Errorf("huawei api error: %s", msg)
		}
		zones = append(zones, parsed.Zones...)
		nextURL = ""
		if parsed.Links != nil {
			if parsed.Links["next"] != "" {
				nextURL = parsed.Links["next"]
			}
		}
	}
	return zones, nil
}

func matchZoneID(zones []huaweiZone, domain string) string {
	domain = strings.TrimSuffix(domain, ".")
	best := ""
	bestLen := -1
	for _, z := range zones {
		zoneName := strings.TrimSuffix(z.Name, ".")
		if zoneName == "" {
			continue
		}
		if domain == zoneName || strings.HasSuffix(domain, "."+zoneName) {
			if len(zoneName) > bestLen {
				best = z.ID
				bestLen = len(zoneName)
			}
		}
	}
	return best
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
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if msg := huaweiErrorMessageFromBody(respBody); msg != "" {
			return nil, fmt.Errorf("huawei api error: %s", msg)
		}
		return nil, fmt.Errorf("huawei api error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func huaweiErrorMessage(code, message, errorCode, errorMsg string) string {
	if code != "" || message != "" {
		if message == "" {
			return code
		}
		if code == "" {
			return message
		}
		return code + " - " + message
	}
	if errorCode != "" || errorMsg != "" {
		if errorMsg == "" {
			return errorCode
		}
		if errorCode == "" {
			return errorMsg
		}
		return errorCode + " - " + errorMsg
	}
	return ""
}

func huaweiErrorMessageFromBody(body []byte) string {
	var parsed struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return huaweiErrorMessage(parsed.Code, parsed.Message, parsed.ErrorCode, parsed.ErrorMsg)
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
	if !strings.HasSuffix(uri, "/") {
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
