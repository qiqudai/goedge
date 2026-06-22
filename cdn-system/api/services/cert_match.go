package services

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

type CertCoverageResult struct {
	OK     bool
	Reason string
	Names  []string
}

func CertificateCoversDomain(certPEM string, domain string) CertCoverageResult {
	domain = normalizeDomainHostForEdge(domain)
	if domain == "" {
		return CertCoverageResult{Reason: "domain is empty"}
	}
	block, _ := pem.Decode([]byte(strings.TrimSpace(certPEM)))
	if block == nil {
		return CertCoverageResult{Reason: "invalid PEM certificate"}
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertCoverageResult{Reason: "failed to parse certificate"}
	}
	names := make([]string, 0, len(cert.DNSNames)+1)
	names = append(names, cert.DNSNames...)
	if strings.TrimSpace(cert.Subject.CommonName) != "" {
		names = append(names, cert.Subject.CommonName)
	}
	for _, name := range names {
		if certNameMatchesDomain(name, domain) {
			return CertCoverageResult{OK: true, Names: names}
		}
	}
	return CertCoverageResult{Reason: "certificate does not cover domain", Names: names}
}

func FormatCertCoverageError(domain string, result CertCoverageResult) string {
	if result.OK {
		return ""
	}
	domain = normalizeDomainHostForEdge(domain)
	switch result.Reason {
	case "domain is empty":
		return "domain is empty"
	case "invalid PEM certificate":
		return "invalid PEM certificate"
	case "failed to parse certificate":
		return "failed to parse certificate"
	case "certificate does not cover domain":
		return formatCertDoesNotCoverDomain(domain, result.Names)
	default:
		if strings.TrimSpace(result.Reason) == "" {
			return fmt.Sprintf("certificate does not cover domain: %s", domain)
		}
		return result.Reason
	}
}

func formatCertDoesNotCoverDomain(domain string, names []string) string {
	sans := formatCertNameList(names)
	if sans == "" {
		return fmt.Sprintf("certificate does not cover domain: %s (certificate has no DNS SAN/CN)", domain)
	}
	if hasSubdomainSANsOnly(domain, names) {
		return fmt.Sprintf("certificate does not cover domain: %s (certificate names: %s; apex domain requires exact match or *.%s wildcard)", domain, sans, domain)
	}
	for _, name := range names {
		normalized := normalizeDomainHostForEdge(name)
		if !strings.HasPrefix(normalized, "*.") {
			continue
		}
		suffix := strings.TrimPrefix(normalized, "*.")
		if suffix != "" && strings.HasSuffix(domain, "."+suffix) {
			left := strings.TrimSuffix(domain, "."+suffix)
			if left != "" && strings.Contains(left, ".") {
				return fmt.Sprintf("certificate does not cover domain: %s (wildcard %s only covers one label; certificate names: %s)", domain, normalized, sans)
			}
		}
	}
	return fmt.Sprintf("certificate does not cover domain: %s (certificate names: %s)", domain, sans)
}

func formatCertNameList(names []string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = normalizeDomainHostForEdge(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return strings.Join(out, ", ")
}

func hasSubdomainSANsOnly(domain string, names []string) bool {
	domain = normalizeDomainHostForEdge(domain)
	if domain == "" || strings.HasPrefix(domain, "*.") {
		return false
	}
	hasName := false
	for _, name := range names {
		normalized := normalizeDomainHostForEdge(name)
		if normalized == "" {
			continue
		}
		hasName = true
		if certNameMatchesDomain(normalized, domain) {
			return false
		}
	}
	return hasName
}

func certNameMatchesDomain(certName string, domain string) bool {
	certName = normalizeDomainHostForEdge(certName)
	domain = normalizeDomainHostForEdge(domain)
	if certName == "" || domain == "" {
		return false
	}
	if certName == domain {
		return true
	}
	if !strings.HasPrefix(certName, "*.") {
		return false
	}
	suffix := strings.TrimPrefix(certName, "*.")
	if suffix == "" || !strings.HasSuffix(domain, "."+suffix) {
		return false
	}
	left := strings.TrimSuffix(domain, "."+suffix)
	return left != "" && !strings.Contains(left, ".")
}
