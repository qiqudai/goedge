package providers

import (
	"bytes"
	"cdn-api/services/dns"
	"cdn-common/i18n"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	dnsPodInternational = "international"
)

type DNSPodConfig struct {
	APIType   string `json:"apiType"`
	ID        string `json:"id"`
	Token     string `json:"token"`
	SecretId  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
	AppID     string `json:"app_id"`
	Region    string `json:"region"`
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

func (p *DNSPodProvider) useTC3() bool {
	if strings.EqualFold(strings.TrimSpace(p.Config.APIType), "tencentDNS") {
		return true
	}
	return strings.TrimSpace(p.Config.SecretId) != "" && strings.TrimSpace(p.Config.SecretKey) != ""
}

func (p *DNSPodProvider) GetDomains() ([]string, error) {
	// Not implemented for this task, returning stub
	return []string{}, nil
}

func (p *DNSPodProvider) GetRecords(domain string) ([]dns.DNSRecord, error) {
	if p.useTC3() {
		return p.getRecordsTC3(domain)
	}
	return p.getRecordsV2(domain)
}

func (p *DNSPodProvider) AddRecord(domain string, record dns.DNSRecord) error {
	if p.useTC3() {
		return p.addRecordTC3(domain, record)
	}
	return p.addRecordV2(domain, record)
}

func (p *DNSPodProvider) DeleteRecord(domain string, record dns.DNSRecord) error {
	if p.useTC3() {
		return p.deleteRecordTC3(domain, record)
	}
	return p.deleteRecordV2(domain, record)
}

func (p *DNSPodProvider) DeleteRecordsByLine(domain string, record dns.DNSRecord) error {
	if p.useTC3() {
		return p.deleteRecordsByLineTC3(domain, record)
	}
	return p.deleteRecordsByLineV2(domain, record)
}

func (p *DNSPodProvider) ReplaceRecordValue(domain string, record dns.DNSRecord, newValue string) error {
	if p.useTC3() {
		return p.replaceRecordTC3(domain, record, newValue)
	}
	return p.replaceRecordV2(domain, record, newValue)
}

func (p *DNSPodProvider) addRecordV2(domain string, record dns.DNSRecord) error {
	if strings.TrimSpace(p.Config.ID) == "" || strings.TrimSpace(p.Config.Token) == "" {
		return errors.New("dnspod id/token required")
	}
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	vals.Set("record_line", record.Line)
	vals.Set("value", record.Value)
	vals.Set("ttl", fmt.Sprintf("%d", record.TTL))
	if record.Weight > 0 {
		vals.Set("weight", fmt.Sprintf("%d", record.Weight))
	}

	resp, err := p.sendRequestV2("Record.Create", vals)
	if err != nil {
		return err
	}

	var r struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}
	if r.Status.Code != "1" {
		if p.isIgnorableV2(r.Status.Code, r.Status.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", r.Status.Code, r.Status.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) replaceRecordV2(domain string, record dns.DNSRecord, newValue string) error {
	if strings.TrimSpace(p.Config.ID) == "" || strings.TrimSpace(p.Config.Token) == "" {
		return errors.New("dnspod id/token required")
	}
	recordID, err := p.findRecordIDByNameV2(domain, record)
	if err != nil {
		return err
	}
	if strings.TrimSpace(record.Value) != "" {
		if exactID, err := p.findRecordIDV2(domain, record); err != nil {
			return err
		} else if exactID != "" {
			recordID = exactID
		}
	}
	if recordID == "" {
		return errors.New("record not found")
	}

	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("record_id", recordID)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	vals.Set("record_line", record.Line)
	vals.Set("value", newValue)
	vals.Set("ttl", fmt.Sprintf("%d", record.TTL))
	if record.Weight > 0 {
		vals.Set("weight", fmt.Sprintf("%d", record.Weight))
	}

	resp, err := p.sendRequestV2("Record.Modify", vals)
	if err != nil {
		return err
	}

	var r struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}
	if r.Status.Code != "1" {
		if p.isIgnorableV2(r.Status.Code, r.Status.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", r.Status.Code, r.Status.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) deleteRecordV2(domain string, record dns.DNSRecord) error {
	recordID, err := p.findRecordIDV2(domain, record)
	if err != nil {
		return err
	}
	if recordID == "" {
		return nil
	}

	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("record_id", recordID)

	resp, err := p.sendRequestV2("Record.Remove", vals)
	if err != nil {
		return err
	}

	var r struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}
	if r.Status.Code != "1" {
		if p.isIgnorableV2(r.Status.Code, r.Status.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", r.Status.Code, r.Status.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) deleteRecordsByLineV2(domain string, record dns.DNSRecord) error {
	vals := url.Values{}
	vals.Set("domain", domain)
	if record.Name != "" {
		vals.Set("sub_domain", record.Name)
	}
	if record.Type != "" {
		vals.Set("record_type", record.Type)
	}
	if record.Line != "" {
		vals.Set("record_line", record.Line)
	}

	respData, err := p.sendRequestV2("Record.List", vals)
	if err != nil {
		return err
	}

	var resp struct {
		Status struct {
			Code string `json:"code"`
		} `json:"status"`
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	if resp.Status.Code != "1" {
		if resp.Status.Code == "10" {
			return nil
		}
		return fmt.Errorf("api error code: %s response: %s", resp.Status.Code, string(respData))
	}
	for _, r := range resp.Records {
		if r.ID == "" {
			continue
		}
		delVals := url.Values{}
		delVals.Set("domain", domain)
		delVals.Set("record_id", r.ID)
		deleteResp, err := p.sendRequestV2("Record.Remove", delVals)
		if err != nil {
			return err
		}
		var delParsed struct {
			Status struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"status"`
		}
		if err := json.Unmarshal(deleteResp, &delParsed); err != nil {
			return err
		}
		if delParsed.Status.Code != "1" {
			return fmt.Errorf("api error code: %s message: %s response: %s", delParsed.Status.Code, delParsed.Status.Message, string(deleteResp))
		}
	}
	return nil
}

func (p *DNSPodProvider) findRecordIDByNameV2(domain string, record dns.DNSRecord) (string, error) {
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	vals.Set("record_line", record.Line)

	respData, err := p.sendRequestV2("Record.List", vals)
	if err != nil {
		return "", err
	}

	var resp struct {
		Status struct {
			Code string `json:"code"`
		} `json:"status"`
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", err
	}

	if resp.Status.Code != "1" {
		if resp.Status.Code == "10" {
			return "", nil
		}
		return "", fmt.Errorf("api error code: %s response: %s", resp.Status.Code, string(respData))
	}
	for _, r := range resp.Records {
		if r.ID != "" {
			return r.ID, nil
		}
	}
	return "", nil
}

func (p *DNSPodProvider) findRecordIDV2(domain string, record dns.DNSRecord) (string, error) {
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("sub_domain", record.Name)
	vals.Set("record_type", record.Type)
	vals.Set("record_line", record.Line)

	respData, err := p.sendRequestV2("Record.List", vals)
	if err != nil {
		return "", err
	}

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
		if resp.Status.Code == "10" {
			return "", nil
		}
		return "", fmt.Errorf("api error code: %s response: %s", resp.Status.Code, string(respData))
	}

	for _, r := range resp.Records {
		if r.Type == record.Type && r.Value == record.Value {
			return r.ID, nil
		}
	}
	return "", nil
}

func (p *DNSPodProvider) sendRequestV2(action string, vals url.Values) ([]byte, error) {
	apiHost := "https://dnsapi.cn"
	lang := "cn"
	if strings.EqualFold(strings.TrimSpace(p.Config.Region), dnsPodInternational) {
		apiHost = "https://api.dnspod.com"
		lang = "en"
	}

	vals.Set("login_token", p.Config.ID+","+p.Config.Token)
	vals.Set("format", "json")
	vals.Set("lang", lang)
	vals.Set("error_on_empty", "no")

	resp, err := http.PostForm(apiHost+"/"+action, vals)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (p *DNSPodProvider) addRecordTC3(domain string, record dns.DNSRecord) error {
	line := strings.TrimSpace(record.Line)
	if line == "" {
		line = i18n.T("dns.line_default")
	}
	payload := map[string]interface{}{
		"Domain":     domain,
		"SubDomain":  record.Name,
		"RecordType": record.Type,
		"RecordLine": line,
		"Value":      record.Value,
		"TTL":        record.TTL,
	}
	if record.Weight > 0 {
		payload["Weight"] = record.Weight
	}
	resp, err := p.sendRequestTC3("CreateRecord", payload)
	if err != nil {
		return err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return err
	}
	if parsed.Response.Error != nil {
		if p.isIgnorableTC3(parsed.Response.Error.Code, parsed.Response.Error.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) replaceRecordTC3(domain string, record dns.DNSRecord, newValue string) error {
	line := strings.TrimSpace(record.Line)
	if line == "" {
		line = i18n.T("dns.line_default")
	}
	recordID, err := p.findRecordIDByNameTC3(domain, record)
	if err != nil {
		return err
	}
	if strings.TrimSpace(record.Value) != "" {
		if exactID, err := p.findRecordIDTC3(domain, record); err != nil {
			return err
		} else if exactID != 0 {
			recordID = exactID
		}
	}
	if recordID == 0 {
		return errors.New("record not found")
	}
	payload := map[string]interface{}{
		"Domain":     domain,
		"RecordId":   recordID,
		"SubDomain":  record.Name,
		"RecordType": record.Type,
		"RecordLine": line,
		"Value":      newValue,
		"TTL":        record.TTL,
	}
	if record.Weight > 0 {
		payload["Weight"] = record.Weight
	}
	resp, err := p.sendRequestTC3("ModifyRecord", payload)
	if err != nil {
		return err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return err
	}
	if parsed.Response.Error != nil {
		if p.isIgnorableTC3(parsed.Response.Error.Code, parsed.Response.Error.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) deleteRecordTC3(domain string, record dns.DNSRecord) error {
	recordID, err := p.findRecordIDTC3(domain, record)
	if err != nil {
		return err
	}
	if recordID == 0 {
		return nil
	}
	payload := map[string]interface{}{
		"Domain":   domain,
		"RecordId": recordID,
	}
	resp, err := p.sendRequestTC3("DeleteRecord", payload)
	if err != nil {
		return err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return err
	}
	if parsed.Response.Error != nil {
		if p.isIgnorableTC3(parsed.Response.Error.Code, parsed.Response.Error.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	return nil
}

func (p *DNSPodProvider) deleteRecordsByLineTC3(domain string, record dns.DNSRecord) error {
	line := strings.TrimSpace(record.Line)
	if line == "" {
		line = i18n.T("dns.line_default")
	}
	payload := map[string]interface{}{
		"Domain":     domain,
		"Subdomain":  record.Name,
		"RecordType": record.Type,
		"RecordLine": line,
		"Offset":     0,
		"Limit":      200,
	}
	resp, err := p.sendRequestTC3("DescribeRecordList", payload)
	if err != nil {
		return err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RecordList []struct {
				RecordId uint64 `json:"RecordId"`
			} `json:"RecordList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return err
	}
	if parsed.Response.Error != nil {
		if parsed.Response.Error.Code == "ResourceNotFound.NoDataOfRecord" {
			return nil
		}
		if p.isIgnorableTC3(parsed.Response.Error.Code, parsed.Response.Error.Message) {
			return nil
		}
		return fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	for _, item := range parsed.Response.RecordList {
		if item.RecordId == 0 {
			continue
		}
		delPayload := map[string]interface{}{
			"Domain":   domain,
			"RecordId": item.RecordId,
		}
		delResp, err := p.sendRequestTC3("DeleteRecord", delPayload)
		if err != nil {
			return err
		}
		var delParsed struct {
			Response struct {
				Error *struct {
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"Error"`
			} `json:"Response"`
		}
		if err := json.Unmarshal(delResp, &delParsed); err != nil {
			return err
		}
		if delParsed.Response.Error != nil {
			if p.isIgnorableTC3(delParsed.Response.Error.Code, delParsed.Response.Error.Message) {
				continue
			}
			return fmt.Errorf("api error code: %s message: %s response: %s", delParsed.Response.Error.Code, delParsed.Response.Error.Message, string(delResp))
		}
	}
	return nil
}

func (p *DNSPodProvider) findRecordIDByNameTC3(domain string, record dns.DNSRecord) (uint64, error) {
	line := strings.TrimSpace(record.Line)
	if line == "" {
		line = i18n.T("dns.line_default")
	}
	payload := map[string]interface{}{
		"Domain":     domain,
		"Subdomain":  record.Name,
		"RecordType": record.Type,
		"RecordLine": line,
		"Offset":     0,
		"Limit":      100,
	}
	resp, err := p.sendRequestTC3("DescribeRecordList", payload)
	if err != nil {
		return 0, err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RecordList []struct {
				RecordId uint64 `json:"RecordId"`
			} `json:"RecordList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return 0, err
	}
	if parsed.Response.Error != nil {
		if parsed.Response.Error.Code == "ResourceNotFound.NoDataOfRecord" {
			return 0, nil
		}
		return 0, fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	for _, item := range parsed.Response.RecordList {
		if item.RecordId != 0 {
			return item.RecordId, nil
		}
	}
	return 0, nil
}

func (p *DNSPodProvider) findRecordIDTC3(domain string, record dns.DNSRecord) (uint64, error) {
	line := strings.TrimSpace(record.Line)
	if line == "" {
		line = i18n.T("dns.line_default")
	}
	payload := map[string]interface{}{
		"Domain":     domain,
		"Subdomain":  record.Name,
		"RecordType": record.Type,
		"RecordLine": line,
		"Offset":     0,
		"Limit":      100,
	}
	resp, err := p.sendRequestTC3("DescribeRecordList", payload)
	if err != nil {
		return 0, err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RecordList []struct {
				RecordId uint64 `json:"RecordId"`
				Name     string `json:"Name"`
				Type     string `json:"Type"`
				Value    string `json:"Value"`
				Line     string `json:"Line"`
			} `json:"RecordList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return 0, err
	}
	if parsed.Response.Error != nil {
		if parsed.Response.Error.Code == "ResourceNotFound.NoDataOfRecord" {
			return 0, nil
		}
		return 0, fmt.Errorf("api error code: %s message: %s response: %s", parsed.Response.Error.Code, parsed.Response.Error.Message, string(resp))
	}
	for _, item := range parsed.Response.RecordList {
		if item.Type == record.Type && item.Value == record.Value {
			if strings.TrimSpace(record.Line) == "" || item.Line == record.Line {
				return item.RecordId, nil
			}
		}
	}
	return 0, nil
}

func (p *DNSPodProvider) sendRequestTC3(action string, payload interface{}) ([]byte, error) {
	const (
		host    = "dnspod.tencentcloudapi.com"
		version = "2021-03-23"
		service = "dnspod"
	)
	if strings.TrimSpace(p.Config.SecretId) == "" || strings.TrimSpace(p.Config.SecretKey) == "" {
		return nil, errors.New("dnspod secret_id/secret_key required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := map[string]string{
		"content-type": "application/json; charset=utf-8",
		"host":         host,
	}
	signedHeaders := []string{"content-type", "host"}
	sort.Strings(signedHeaders)
	canonicalHeaderLines := make([]string, 0, len(signedHeaders))
	for _, key := range signedHeaders {
		canonicalHeaderLines = append(canonicalHeaderLines, key+":"+canonicalHeaders[key])
	}
	canonicalHeadersStr := strings.Join(canonicalHeaderLines, "\n") + "\n"
	signedHeadersStr := strings.Join(signedHeaders, ";")

	hashedPayload := sha256Hex(body)
	canonicalRequest := strings.Join([]string{
		"POST",
		canonicalURI,
		canonicalQueryString,
		canonicalHeadersStr,
		signedHeadersStr,
		hashedPayload,
	}, "\n")

	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := hmacSHA256([]byte("TC3"+p.Config.SecretKey), date)
	signingKey = hmacSHA256(signingKey, service)
	signingKey = hmacSHA256(signingKey, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.Config.SecretId, credentialScope, signedHeadersStr, signature,
	)

	req, err := http.NewRequest("POST", "https://"+host, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", version)
	if region := strings.TrimSpace(p.Config.Region); region != "" && !strings.EqualFold(region, dnsPodInternational) {
		req.Header.Set("X-TC-Region", region)
	}
	req.Header.Set("Authorization", authHeader)

	client := dns.NewHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}

func (p *DNSPodProvider) isIgnorableTC3(code, message string) bool {
	code = strings.TrimSpace(code)
	switch code {
	case "InvalidParameter.DomainRecordExist",
		"ResourceNotFound.NoDataOfRecord",
		"InvalidParameter.RecordLineInvalid":
		return true
	}
	_ = message
	return false
}

func (p *DNSPodProvider) isIgnorableV2(code, message string) bool {
	code = strings.TrimSpace(code)
	switch code {
	case "10", "9":
		return true
	}
	if strings.Contains(message, i18n.T("dns.record_exists")) || strings.Contains(strings.ToLower(message), "record already exists") {
		return true
	}
	if strings.Contains(message, i18n.T("dns.line_word")) && strings.Contains(message, i18n.T("dns.not_word")) {
		return true
	}
	return false
}

func NewDNSPodProviderIntl(credentials string) (dns.Provider, error) {
	provider, err := NewDNSPodProvider(credentials)
	if err != nil {
		return nil, err
	}
	p := provider.(*DNSPodProvider)
	p.Config.Region = dnsPodInternational // Force international
	return p, nil
}

func init() {
	dns.RegisterProvider("dnspod", NewDNSPodProvider)
	dns.RegisterProvider("dnspod_intl", NewDNSPodProviderIntl)
}

func (p *DNSPodProvider) getRecordsV2(domain string) ([]dns.DNSRecord, error) {
	vals := url.Values{}
	vals.Set("domain", domain)
	vals.Set("length", "3000") // Max fetch

	respData, err := p.sendRequestV2("Record.List", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status struct {
			Code string `json:"code"`
		} `json:"status"`
		Records []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Value  string `json:"value"`
			Line   string `json:"line"`
			TTL    string `json:"ttl"`
			Weight string `json:"weight"`
		} `json:"records"`
	}

	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}
	if resp.Status.Code != "1" {
		if resp.Status.Code == "10" {
			return []dns.DNSRecord{}, nil
		}
		return nil, fmt.Errorf("api error code: %s", resp.Status.Code)
	}

	var results []dns.DNSRecord
	for _, r := range resp.Records {
		ttl := 600
		fmt.Sscanf(r.TTL, "%d", &ttl)
		weight := 0
		if strings.TrimSpace(r.Weight) != "" {
			fmt.Sscanf(r.Weight, "%d", &weight)
		}
		results = append(results, dns.DNSRecord{
			Type:   r.Type,
			Name:   r.Name,
			Value:  r.Value,
			Line:   r.Line,
			TTL:    ttl,
			Weight: weight,
		})
	}
	return results, nil
}

func (p *DNSPodProvider) getRecordsTC3(domain string) ([]dns.DNSRecord, error) {
	payload := map[string]interface{}{
		"Domain": domain,
		"Limit":  3000,
	}
	resp, err := p.sendRequestTC3("DescribeRecordList", payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RecordList []struct {
				Name   string `json:"Name"`
				Type   string `json:"Type"`
				Value  string `json:"Value"`
				Line   string `json:"Line"`
				TTL    uint64 `json:"TTL"`
				Weight uint64 `json:"Weight"`
			} `json:"RecordList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return nil, err
	}
	if parsed.Response.Error != nil {
		if parsed.Response.Error.Code == "ResourceNotFound.NoDataOfRecord" {
			return []dns.DNSRecord{}, nil
		}
		return nil, fmt.Errorf("api error code: %s", parsed.Response.Error.Code)
	}

	var results []dns.DNSRecord
	for _, r := range parsed.Response.RecordList {
		results = append(results, dns.DNSRecord{
			Type:   r.Type,
			Name:   r.Name,
			Value:  r.Value,
			Line:   r.Line,
			TTL:    int(r.TTL),
			Weight: int(r.Weight),
		})
	}
	return results, nil
}
