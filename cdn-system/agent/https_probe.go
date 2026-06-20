package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type httpsProbePayload struct {
	SiteID         int64    `json:"site_id"`
	CertID         int64    `json:"cert_id"`
	Domains        []string `json:"domains"`
	Ports          []string `json:"ports"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type httpsProbeResult struct {
	Domain       string `json:"domain"`
	Port         string `json:"port"`
	OK           bool   `json:"ok"`
	TLSVersion   string `json:"tls_version,omitempty"`
	CertSubject  string `json:"cert_subject,omitempty"`
	CertNotAfter string `json:"cert_not_after,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	Error        string `json:"error,omitempty"`
}

func runHTTPSProbeTask(raw string) (string, error) {
	var payload httpsProbePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("invalid https probe payload")
	}
	timeout := time.Duration(payload.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if len(payload.Ports) == 0 {
		payload.Ports = []string{"443"}
	}
	results := make([]httpsProbeResult, 0, len(payload.Domains)*len(payload.Ports))
	var failures []string
	for _, domain := range payload.Domains {
		domain = normalizeEdgeDomainName(domain)
		if domain == "" {
			continue
		}
		for _, port := range payload.Ports {
			port = strings.TrimSpace(port)
			if port == "" {
				continue
			}
			result := runSingleHTTPSProbe(domain, port, timeout)
			results = append(results, result)
			if !result.OK {
				failures = append(failures, fmt.Sprintf("%s:%s %s", domain, port, result.Error))
			}
		}
	}
	if len(results) == 0 {
		return "", fmt.Errorf("https probe has no valid targets")
	}
	rawResults, _ := json.Marshal(results)
	if len(failures) > 0 {
		return string(rawResults), fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return string(rawResults), nil
}

func runSingleHTTPSProbe(domain, port string, timeout time.Duration) httpsProbeResult {
	result := httpsProbeResult{Domain: domain, Port: port}
	addr := net.JoinHostPort("127.0.0.1", port)
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true,
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	state := conn.ConnectionState()
	result.TLSVersion = tlsVersionName(state.Version)
	if len(state.PeerCertificates) == 0 {
		result.Error = "missing peer certificate"
		return result
	}
	leaf := state.PeerCertificates[0]
	result.CertSubject = leaf.Subject.String()
	result.CertNotAfter = leaf.NotAfter.Format(time.RFC3339)
	if err := leaf.VerifyHostname(domain); err != nil {
		result.Error = err.Error()
		return result
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	req, err := http.NewRequest(http.MethodGet, "https://"+domain+"/", nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Host = domain
	if err := req.Write(conn); err != nil {
		result.Error = err.Error()
		return result
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	result.OK = true
	return result
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}
