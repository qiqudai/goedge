package services

import (
	"crypto/x509"
	"encoding/pem"
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
