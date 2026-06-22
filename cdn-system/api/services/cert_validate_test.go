package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"cdn-api/models"
)

func testCertAndKeyPEM(t *testing.T, commonName string, dnsNames ...string) (string, string) {
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
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	return certPEM, keyPEM
}

func TestValidateUploadCertKeyPairTrimsWhitespace(t *testing.T) {
	certPEM, keyPEM := testCertAndKeyPEM(t, "www.example.com")
	if err := ValidateUploadCertKeyPair("  \n"+certPEM+"\n  ", "\t"+keyPEM+"\n"); err != nil {
		t.Fatalf("expected trimmed pair to validate, got %v", err)
	}
}

func TestValidateUploadCertKeyPairRejectsMismatch(t *testing.T) {
	certPEM, _ := testCertAndKeyPEM(t, "www.example.com")
	_, otherKey := testCertAndKeyPEM(t, "api.example.com")
	if err := ValidateUploadCertKeyPair(certPEM, otherKey); err == nil {
		t.Fatal("expected mismatch error")
	} else if !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatCertCoverageErrorApexHint(t *testing.T) {
	cert := testCertPEM(t, "", "h5.icztev.cam", "web.icztev.cam")
	result := CertificateCoversDomain(cert, "icztev.cam")
	got := FormatCertCoverageError("icztev.cam", result)
	if !strings.Contains(got, "certificate names: h5.icztev.cam, web.icztev.cam") {
		t.Fatalf("unexpected message: %s", got)
	}
	if !strings.Contains(got, "apex domain requires exact match or *.icztev.cam wildcard") {
		t.Fatalf("expected apex hint, got: %s", got)
	}
}

func TestIsSiteHTTPSEnabledUsesEnableFlag(t *testing.T) {
	site := &models.Site{
		CertID:         9,
		HttpsListenRaw: `["443"]`,
		Settings: map[string]interface{}{
			"https": map[string]interface{}{
				"enable": false,
			},
		},
	}
	if IsSiteHTTPSEnabled(site) {
		t.Fatal("https enable=false must not count as enabled")
	}
	site.Settings = map[string]interface{}{
		"https": map[string]interface{}{
			"enable": true,
		},
	}
	if !IsSiteHTTPSEnabled(site) {
		t.Fatal("https enable=true must count as enabled")
	}
}

func TestNormalizeUploadCertKeyInputsSplitsBundle(t *testing.T) {
	certPEM, keyPEM := testCertAndKeyPEM(t, "www.example.com")
	bundle := certPEM + "\n" + keyPEM
	gotCert, gotKey := NormalizeUploadCertKeyInputs(bundle, "")
	if gotKey == "" {
		t.Fatalf("expected private key extracted from bundle")
	}
	if _, err := ParsePrivateKeyPEM(gotKey); err != nil {
		t.Fatalf("extracted key invalid: %v", err)
	}
	if !strings.Contains(gotCert, "BEGIN CERTIFICATE") {
		t.Fatalf("expected certificate in cert field")
	}
}

func TestExposeStoredPrivateKeyRejectsCertificate(t *testing.T) {
	certPEM, _ := testCertAndKeyPEM(t, "www.example.com")
	if got := ExposeStoredPrivateKey(certPEM); got != "" {
		t.Fatalf("expected empty key exposure for certificate PEM")
	}
}

func TestExposeStoredPrivateKeyReturnsPlainKey(t *testing.T) {
	_, keyPEM := testCertAndKeyPEM(t, "www.example.com")
	if got := ExposeStoredPrivateKey(keyPEM); strings.TrimSpace(got) != strings.TrimSpace(keyPEM) {
		t.Fatalf("expected plaintext private key round-trip")
	}
}

func TestValidateUploadRejectsCertificateAsKey(t *testing.T) {
	certPEM, _ := testCertAndKeyPEM(t, "www.example.com")
	if err := ValidateUploadCertKeyPair(certPEM, certPEM); err == nil {
		t.Fatal("expected certificate-as-key to fail")
	} else if !strings.Contains(err.Error(), "private key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
