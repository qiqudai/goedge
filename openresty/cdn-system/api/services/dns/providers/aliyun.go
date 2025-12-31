package providers

import (
	"cdn-api/services/dns"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
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
	return []string{}, nil
}

func (p *AliyunProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	return []dns.DNSRecord{}, nil // Not implemented
}

func (p *AliyunProvider) AddRecord(domain string, record dns.DNSRecord) error {
	params := p.newParams("AddDomainRecord")
	params.Set("DomainName", domain)
	params.Set("RR", record.Name)
	params.Set("Type", record.Type)
	params.Set("Value", record.Value)
	if record.TTL > 0 {
		params.Set("TTL", fmt.Sprintf("%d", record.TTL))
	}
	// Line mostly not supported or complex in Aliyun (default is default)
	if record.Line != "" {
		params.Set("Line", record.Line)
	}

	respBody, err := p.doRequest(params)
	if err != nil {
		return err
	}

	var resp struct {
		RecordId string `json:"RecordId"`
		Code     string `json:"Code"`
		Message  string `json:"Message"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}
	if resp.Code != "" {
		// Ignore duplicates
		if resp.Code == "DomainRecordDuplicate" {
			return nil
		}
		return fmt.Errorf("aliyun error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

func (p *AliyunProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	// 1. Find Record ID
	recordID, err := p.findRecordID(domain, record)
	if err != nil {
		return err
	}
	if recordID == "" {
		return nil
	}

	// 2. Delete
	params := p.newParams("DeleteDomainRecord")
	params.Set("RecordId", recordID)

	respBody, err := p.doRequest(params)
	if err != nil {
		return err
	}
	
	var resp struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	_ = json.Unmarshal(respBody, &resp)
	if resp.Code != "" {
		return fmt.Errorf("aliyun error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

func (p *AliyunProvider) findRecordID(domain string, record dns.DNSRecord) (string, error) {
	params := p.newParams("DescribeDomainRecords")
	params.Set("DomainName", domain)
	// Filter by RR keyword, Aliyun allows filtering by RR and Type
	params.Set("RRKeyWord", record.Name)
	params.Set("TypeKeyWord", record.Type)
    params.Set("PageSize", "500")

	respBody, err := p.doRequest(params)
	if err != nil {
		return "", err
	}

	var resp struct {
		DomainRecords struct {
			Record []struct {
				RecordId string `json:"RecordId"`
				RR       string `json:"RR"`
				Type     string `json:"Type"`
				Value    string `json:"Value"`
			} `json:"Record"`
		} `json:"DomainRecords"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	for _, r := range resp.DomainRecords.Record {
		// Aliyun values might not exactly match (e.g. trailing dot), strict check can be loose
		if r.RR == record.Name && r.Type == record.Type && r.Value == record.Value {
			return r.RecordId, nil
		}
	}
	return "", nil
}

func (p *AliyunProvider) newParams(action string) url.Values {
	v := url.Values{}
	v.Set("Action", action)
	v.Set("Format", "JSON")
	v.Set("Version", "2015-01-09")
	v.Set("AccessKeyId", p.Config.AccessKeyID)
	v.Set("SignatureMethod", "HMAC-SHA1")
	v.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	v.Set("SignatureVersion", "1.0")
	v.Set("SignatureNonce", uuid.New().String())
	return v
}

func (p *AliyunProvider) doRequest(params url.Values) ([]byte, error) {
	p.sign(params)
	encodedParams := params.Encode()
	reqUrl := "https://alidns.aliyuncs.com/?" + encodedParams
	
	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *AliyunProvider) sign(params url.Values) {
	// Canonicalize
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalizedQueryString string
	for _, k := range keys {
		v := params.Get(k)
		if canonicalizedQueryString != "" {
			canonicalizedQueryString += "&"
		}
		canonicalizedQueryString += p.percentEncode(k) + "=" + p.percentEncode(v)
	}

	stringToSign := "GET&" + p.percentEncode("/") + "&" + p.percentEncode(canonicalizedQueryString)
	
	mac := hmac.New(sha1.New, []byte(p.Config.AccessKeySecret + "&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	
	params.Set("Signature", signature)
}

func (p *AliyunProvider) percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.Replace(s, "+", "%20", -1)
	s = strings.Replace(s, "*", "%2A", -1)
	s = strings.Replace(s, "%7E", "~", -1)
	return s
}

func init() {
	dns.RegisterProvider("aliyun", NewAliyunProvider)
}
