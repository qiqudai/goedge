package controllers

import "strings"

func resolveDNSSyncErrorMessage(err error) (string, string) {
	if err == nil {
		return T("dns sync failed"), ""
	}
	return resolveDNSSyncErrorMessageFromString(err.Error())
}

func resolveDNSSyncErrorMessageFromString(errMsg string) (string, string) {
	raw := strings.TrimSpace(errMsg)
	if raw == "" {
		return T("dns sync failed"), errMsg
	}
	lower := strings.ToLower(raw)

	switch {
	case strings.Contains(lower, "cname domains not configured"):
		return T("dns.sync.cname_domain_not_configured"), raw
	case strings.Contains(lower, "cname domain is empty"):
		return T("dns.sync.cname_domain_empty"), raw
	case strings.Contains(lower, "cname hostname is empty"):
		return T("dns.sync.cname_hostname_empty"), raw
	case strings.Contains(lower, "line cname domain is empty"):
		return T("dns.sync.line_domain_empty"), raw
	case strings.Contains(lower, "line cname hostname is empty"):
		return T("dns.sync.line_hostname_empty"), raw
	case strings.Contains(lower, "dns provider not configured"):
		return T("dns.sync.dns_provider_not_configured"), raw
	case strings.Contains(lower, "dns provider not available"):
		return T("dns.sync.dns_provider_unavailable"), raw
	case strings.Contains(lower, "dnspod id/token required"),
		strings.Contains(lower, "secret_id/secret_key"),
		strings.Contains(lower, "invalid credentials"),
		strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "unauthorized"):
		return T("dns.sync.invalid_credentials"), raw
	case strings.Contains(lower, "recordlineinvalid"),
		(strings.Contains(lower, "record line") && strings.Contains(lower, "invalid")),
		strings.Contains(lower, "record_line"):
		return T("dns.sync.invalid_line"), raw
	case strings.Contains(lower, "recordttllimit"),
		strings.Contains(lower, "ttl limit"),
		(strings.Contains(lower, "ttl") && strings.Contains(lower, "exceed")):
		return T("dns.sync.ttl_limit"), raw
	case strings.Contains(lower, "node list empty"):
		return T("dns.sync.node_list_empty"), raw
	case strings.Contains(lower, "record not found"):
		return T("dns.sync.record_not_found"), raw
	case strings.Contains(lower, "domain is empty"):
		return T("dns.sync.domain_empty"), raw
	case strings.Contains(lower, "hostname is empty"):
		return T("dns.sync.hostname_empty"), raw
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "timed out"),
		strings.Contains(lower, "context deadline exceeded"):
		return T("dns.sync.timeout"), raw
	case strings.Contains(lower, "connection refused"):
		return T("dns.sync.connection_refused"), raw
	case strings.Contains(lower, "no such host"):
		return T("dns.sync.dns_lookup_failed"), raw
	default:
		return T("dns sync failed"), raw
	}
}
