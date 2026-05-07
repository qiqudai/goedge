package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"cdn-api/models"
)

func testCertPEM(t *testing.T, commonName string, dnsNames ...string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		DNSNames:              dnsNames,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestCertificateCoversDomainExactSAN(t *testing.T) {
	cert := testCertPEM(t, "", "www.example.com")
	if got := CertificateCoversDomain(cert, "www.example.com"); !got.OK {
		t.Fatalf("expected cert to cover domain, got reason=%s names=%v", got.Reason, got.Names)
	}
}

func TestCertificateCoversDomainWildcardSingleLabelOnly(t *testing.T) {
	cert := testCertPEM(t, "", "*.example.com")
	if got := CertificateCoversDomain(cert, "a.example.com"); !got.OK {
		t.Fatalf("expected wildcard cert to cover one label, got reason=%s", got.Reason)
	}
	if got := CertificateCoversDomain(cert, "a.b.example.com"); got.OK {
		t.Fatalf("wildcard cert must not cover multiple labels")
	}
	if got := CertificateCoversDomain(cert, "example.com"); got.OK {
		t.Fatalf("wildcard cert must not cover apex")
	}
}

func TestCertificateCoversDomainInvalidPEM(t *testing.T) {
	if got := CertificateCoversDomain("not a cert", "www.example.com"); got.OK {
		t.Fatalf("invalid PEM must not cover domain")
	}
}

func TestFindCertForSiteDomainRejectsSelectedMismatch(t *testing.T) {
	cert := models.Cert{
		ID:     12,
		Domain: "www.example.com",
		Cert:   testCertPEM(t, "", "www.example.com"),
	}
	if got := findCertForSiteDomain(12, "api.example.com", []models.Cert{cert}); got != nil {
		t.Fatalf("selected cert that does not cover domain must be rejected")
	}
	if got := findCertForSiteDomain(12, "www.example.com", []models.Cert{cert}); got == nil {
		t.Fatalf("selected cert that covers domain must be accepted")
	}
}

func TestFindCertForDomainUsesPEMOverMetadata(t *testing.T) {
	cert := models.Cert{
		ID:     21,
		Domain: "api.example.com",
		Cert:   testCertPEM(t, "", "www.example.com"),
	}
	if got := findCertForDomain("api.example.com", []models.Cert{cert}); got != nil {
		t.Fatalf("metadata domain must not override certificate SAN mismatch")
	}
	if got := findCertForDomain("www.example.com", []models.Cert{cert}); got == nil {
		t.Fatalf("certificate SAN should be source of truth")
	}
}
