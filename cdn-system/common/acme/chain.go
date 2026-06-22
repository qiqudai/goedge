package acme

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"

	legoacme "github.com/go-acme/lego/v4/acme"
	"github.com/go-acme/lego/v4/acme/api"
)

const PreferredLetsEncryptChain = "ISRG Root X1"

var incompatibleLEChainSubjectCN = []string{
	"YR2",
	"Root YR",
}

func IsLetsEncryptDirectory(caDirURL string) bool {
	dir := strings.ToLower(strings.TrimSpace(caDirURL))
	return strings.Contains(dir, "letsencrypt.org")
}

func ResolvePreferredChain(caDirURL, explicit string) string {
	if v := strings.TrimSpace(explicit); v != "" {
		return v
	}
	if IsLetsEncryptDirectory(caDirURL) {
		return PreferredLetsEncryptChain
	}
	return ""
}

func ParseCertificateBundle(certPEM string) ([]*x509.Certificate, error) {
	data := []byte(certPEM)
	out := make([]*x509.Certificate, 0, 3)
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		out = append(out, cert)
	}
	if len(out) == 0 {
		return nil, errors.New("certificate bundle is empty")
	}
	return out, nil
}

func UsesIncompatibleLetsEncryptChain(certPEM string) bool {
	certs, err := ParseCertificateBundle(certPEM)
	if err != nil {
		return false
	}
	for _, cert := range certs {
		if hasIncompatibleLetsEncryptCN(cert.Subject.CommonName) {
			return true
		}
	}
	return false
}

func hasIncompatibleLetsEncryptCN(commonName string) bool {
	cn := strings.TrimSpace(commonName)
	for _, marker := range incompatibleLEChainSubjectCN {
		if cn == marker || strings.Contains(cn, marker) {
			return true
		}
	}
	return false
}

func selectCompatibleLetsEncryptBundle(certs map[string]*legoacme.RawCertificate) ([]byte, error) {
	if len(certs) == 0 {
		return nil, errors.New("no certificate bundles available")
	}

	var best []byte
	bestSize := int(^uint(0) >> 1)
	for _, raw := range certs {
		if raw == nil || len(raw.Cert) == 0 {
			continue
		}
		bundle := bytes.TrimSpace(raw.Cert)
		if UsesIncompatibleLetsEncryptChain(string(bundle)) {
			continue
		}
		size, err := countCertificates(bundle)
		if err != nil {
			continue
		}
		if size < bestSize {
			best = append([]byte(nil), bundle...)
			bestSize = size
		}
	}
	if len(best) > 0 {
		return best, nil
	}
	return nil, errors.New("no compatible Let's Encrypt chain found (avoid YR2 / ISRG Root YR)")
}

func countCertificates(pemBytes []byte) (int, error) {
	certs, err := ParseCertificateBundle(string(pemBytes))
	if err != nil {
		return 0, err
	}
	return len(certs), nil
}

func fetchCompatibleLetsEncryptChain(core *api.Core, certURL string) ([]byte, error) {
	if core == nil {
		return nil, errors.New("acme core is nil")
	}
	if strings.TrimSpace(certURL) == "" {
		return nil, errors.New("certificate url is empty")
	}
	certs, err := core.Certificates.GetAll(certURL, true)
	if err != nil {
		return nil, err
	}
	return selectCompatibleLetsEncryptBundle(certs)
}
