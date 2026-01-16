package services

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type NotifyTemplate struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type NotifyItemConfig struct {
	Enable          int            `json:"enable"`
	Methods         []string       `json:"methods"`
	ContinuousTimes int            `json:"continuous_times"`
	Interval        int            `json:"interval"`
	EmailTemplate   NotifyTemplate `json:"email_template"`
	SMSTemplate     NotifyTemplate `json:"sms_template"`
}

type notifyPolicy struct {
	enabled    bool
	allowEmail bool
	allowSMS   bool
	inPeriod   bool
	emailTitle string
	emailBody  string
	smsBody    string
}

var notifyConfigKeyMap = map[string]string{
	"traffic-exceed":    "traffic-exceed-notify",
	"traffic-exceeding": "traffic-exceeding-notify",
	"package-expire":    "package-expire-notify",
	"package-expiring":  "package-expiring-notify",
	"cc-switch":         "cc-switch-notify",
	"bandwidth-exceed":  "bandwidth-exceed-notify",
	"conn-exceed":       "conn-exceed-notify",
	"cert-expire":       "cert-expire-notify",
	"cert-expiring":     "cert-expiring-notify",
	"account-auth2":     "account-auth2-notify",
}

func ResolveNotifyPolicy(msgType string, now time.Time) notifyPolicy {
	cfg, err := LoadSystemConfig()
	if err != nil {
		return notifyPolicy{enabled: true, allowEmail: true, inPeriod: true}
	}
	policy := notifyPolicy{enabled: true, allowEmail: true, inPeriod: true}
	policy.inPeriod = isWithinNotificationPeriod(cfg, now)

	globalEmail, globalSMS := parseGlobalNotifyMethods(cfg["notify-method"])
	if !globalEmail {
		policy.allowEmail = false
	}
	if !globalSMS {
		policy.allowSMS = false
	}
	// SMS sending is disabled; only site + email notifications are used.
	policy.allowSMS = false

	key := notifyConfigKeyMap[msgType]
	if key == "" {
		return policy
	}
	raw := strings.TrimSpace(cfg[key])
	if raw == "" {
		return policy
	}

	item := parseNotifyItemConfig(raw)
	if !item.enabled {
		policy.enabled = false
		return policy
	}
	if len(item.methods) > 0 {
		policy.allowEmail = item.methods["email"] && policy.allowEmail
		policy.allowSMS = item.methods["sms"] && policy.allowSMS
	}
	policy.emailTitle = item.emailTitle
	policy.emailBody = item.emailBody
	policy.smsBody = item.smsBody
	return policy
}

type parsedNotifyItem struct {
	enabled    bool
	methods    map[string]bool
	emailTitle string
	emailBody  string
	smsBody    string
}

func parseNotifyItemConfig(raw string) parsedNotifyItem {
	item := parsedNotifyItem{
		enabled: true,
		methods: map[string]bool{},
	}
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		return item
	}
	if _, ok := rawMap["enable"]; ok || rawMap["methods"] != nil || rawMap["email_template"] != nil || rawMap["sms_template"] != nil {
		var modern NotifyItemConfig
		_ = json.Unmarshal([]byte(raw), &modern)
		if val, ok := rawMap["enable"]; ok {
			item.enabled = parseBoolValue(val, true)
		}
		for _, m := range modern.Methods {
			item.methods[strings.ToLower(strings.TrimSpace(m))] = true
		}
		item.emailTitle = modern.EmailTemplate.Title
		item.emailBody = modern.EmailTemplate.Content
		item.smsBody = modern.SMSTemplate.Content
		return item
	}

	if val, ok := rawMap["state"]; ok {
		item.enabled = parseBoolValue(val, true)
	}
	if val, ok := rawMap["phone-templ"]; ok {
		item.smsBody = parseString(val)
	}
	if val, ok := rawMap["email-templ"]; ok {
		item.emailBody = parseString(val)
	}
	// Legacy configs always allow email/sms if enabled.
	item.methods["email"] = true
	item.methods["sms"] = true
	return item
}

func parseGlobalNotifyMethods(raw string) (bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true, false
	}
	if strings.HasPrefix(raw, "{") {
		var payload struct {
			Email bool `json:"email"`
			Phone bool `json:"phone"`
		}
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			return payload.Email, payload.Phone
		}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	email := false
	sms := false
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "email":
			email = true
		case "sms", "phone":
			sms = true
		}
	}
	if !email && !sms {
		return true, false
	}
	return email, sms
}

func isWithinNotificationPeriod(cfg map[string]string, now time.Time) bool {
	mode := strings.TrimSpace(strings.ToLower(cfg["notification-period"]))
	if mode == "" || mode == "all" {
		return true
	}
	if mode != "custom" {
		return true
	}
	raw := strings.TrimSpace(cfg["notification-period-custom"])
	if raw == "" {
		return true
	}
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return true
	}
	start, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	if start < 0 || start > 23 || end < 0 || end > 23 {
		return true
	}
	hour := now.Hour()
	if start <= end {
		return hour >= start && hour <= end
	}
	return hour >= start || hour <= end
}
