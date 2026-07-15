package controllers

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

func testApplyCertPEM(t *testing.T, commonName string, dnsNames ...string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
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

func TestFindCertForUserDomainReusesWildcardIssuedCert(t *testing.T) {
	now := time.Now()
	cert := models.Cert{
		ID:       1001,
		UserID:   7,
		Domain:   "*.icztev.cam",
		Cert:     testApplyCertPEM(t, "", "*.icztev.cam"),
		State:    "ready",
		Enable:   true,
		CreateAt: now,
		UpdateAt: now,
	}

	got := findCertForDomainInList([]models.Cert{cert}, "web.icztev.cam")
	if got == nil || got.ID != cert.ID {
		t.Fatalf("expected wildcard cert id=%d, got %#v", cert.ID, got)
	}
}

func TestFindCertForUserDomainDoesNotReuseWildcardForApex(t *testing.T) {
	now := time.Now()
	cert := models.Cert{
		ID:       1002,
		UserID:   7,
		Domain:   "*.icztev.cam",
		Cert:     testApplyCertPEM(t, "", "*.icztev.cam"),
		State:    "ready",
		Enable:   true,
		CreateAt: now,
		UpdateAt: now,
	}

	got := findCertForDomainInList([]models.Cert{cert}, "icztev.cam")
	if got != nil {
		t.Fatalf("wildcard cert must not cover apex, got %#v", got)
	}
}

func TestFindCertForUserDomainUsesPEMBeforeMetadata(t *testing.T) {
	now := time.Now()
	cert := models.Cert{
		ID:       1003,
		UserID:   7,
		Domain:   "web.icztev.cam",
		Cert:     testApplyCertPEM(t, "", "other.icztev.cam"),
		State:    "ready",
		Enable:   true,
		CreateAt: now,
		UpdateAt: now,
	}

	got := findCertForDomainInList([]models.Cert{cert}, "web.icztev.cam")
	if got != nil {
		t.Fatalf("metadata must not override certificate SAN mismatch, got %#v", got)
	}
}

func TestIssuedCertCoversSiteRequiresPEMAndValidWildcardCoverage(t *testing.T) {
	now := time.Now()
	cert := models.Cert{
		ID:       1004,
		UserID:   7,
		Domain:   "*.icztev.cam",
		Cert:     testApplyCertPEM(t, "", "*.icztev.cam"),
		State:    "ready",
		Enable:   true,
		CreateAt: now,
		UpdateAt: now,
	}

	if !issuedCertCoversSite(&cert, []string{"web.icztev.cam"}) {
		t.Fatal("expected issued wildcard certificate to cover a one-label subdomain")
	}
	if issuedCertCoversSite(&cert, []string{"icztev.cam"}) {
		t.Fatal("wildcard certificate must not activate the apex domain")
	}
	cert.Cert = ""
	if issuedCertCoversSite(&cert, []string{"web.icztev.cam"}) {
		t.Fatal("issued certificate without PEM must not be attached to a site")
	}
}
